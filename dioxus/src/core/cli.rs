// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//! The vfox client. Read-only queries run directly and are parsed into
//! models; state-changing operations are returned as [`VfoxCommand`] values,
//! so that the caller can queue them, stream their output and retry them.
//! Every command line built here is documented in
//! `docs/vfox-cli-contract.md`.

use std::cell::RefCell;
use std::collections::BTreeMap;
use std::rc::Rc;

use super::command::{VfoxCommand, VfoxError, VfoxOperation};
use super::models::{AvailablePlugin, AvailableVersion, ConfigEntry, InstalledSdk, PluginInfo, UseScope};
use super::output::{
    PLUGIN_INFO_TEMPLATE, merge_installed_sdks, parse_available_plugins, parse_config_list, parse_current,
    parse_install_path, parse_installed_tree, parse_plugin_info, parse_search, parse_vfox_version, summarize_error,
};
use super::process::{ProcessOptions, ProcessOutput, ProcessRunner, ProcessStartError, RunningProcess};

/// The name of the vfox executable, looked up in `PATH` when no path is known.
pub const VFOX_EXECUTABLE_NAME: &str = "vfox";

/// Variables that tie a process to an activated shell session. They are
/// removed so that a GUI started from an activated terminal does not act on
/// that terminal's session.
pub const SESSION_VARIABLES: [&str; 4] = ["__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH", "__VFOX_INITIALIZED"];

/// How vfox processes are launched.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct VfoxLaunchConfig {
    /// Path to the vfox executable, or a bare name to look up in `PATH`.
    pub executable: String,
    /// The environment inherited from the application.
    pub base_environment: BTreeMap<String, String>,
    /// Variables added by the user, such as mirrors; they override the base
    /// environment.
    pub extra_environment: BTreeMap<String, String>,
    /// Working directory of the commands that are not about a project. vfox
    /// reads a project `.vfox.toml` (and legacy files such as `.nvmrc`) from
    /// its working directory, so global queries run where none can be: in `/`.
    pub neutral_directory: String,
}

impl Default for VfoxLaunchConfig {
    fn default() -> Self {
        Self {
            executable: VFOX_EXECUTABLE_NAME.to_owned(),
            base_environment: BTreeMap::new(),
            extra_environment: BTreeMap::new(),
            neutral_directory: "/".to_owned(),
        }
    }
}

impl VfoxLaunchConfig {
    /// The complete environment of a vfox child process.
    pub fn environment(&self, hook_shell: bool) -> BTreeMap<String, String> {
        let mut environment: BTreeMap<String, String> = self
            .base_environment
            .iter()
            .chain(&self.extra_environment)
            .filter(|(key, _)| !SESSION_VARIABLES.contains(&key.as_str()))
            .map(|(key, value)| (key.clone(), value.clone()))
            .collect();
        // Disable colours; vfox (through pterm) honours https://no-color.org.
        environment.insert("NO_COLOR".into(), "1".into());
        if hook_shell {
            environment.insert("__VFOX_SHELL".into(), "bash".into());
        }
        environment
    }
}

/// Whether `name` is accepted as a plugin name or alias:
/// `[A-Za-z0-9][A-Za-z0-9._+-]*`.
pub fn is_valid_plugin_name(name: &str) -> bool {
    let mut chars = name.chars();
    chars.next().is_some_and(|c| c.is_ascii_alphanumeric())
        && chars.all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '_' | '+' | '-'))
}

/// `[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*`
fn is_valid_config_key(key: &str) -> bool {
    key.split('.').all(|segment| {
        let mut chars = segment.chars();
        chars.next().is_some_and(|c| c.is_ascii_alphabetic()) && chars.all(|c| c.is_ascii_alphanumeric())
    })
}

// Arguments never pass through a shell, so the checks only guard against
// option injection and typos.
fn check_name(name: &str) -> Result<&str, VfoxError> {
    if is_valid_plugin_name(name) {
        Ok(name)
    } else {
        Err(VfoxError::Invalid(format!("Not a valid plugin name: {name}")))
    }
}

fn check_version(version: &str) -> Result<&str, VfoxError> {
    if version.is_empty() || version.starts_with('-') || version.chars().any(|c| c.is_whitespace() || c == '@') {
        Err(VfoxError::Invalid(format!("Not a valid version: {version}")))
    } else {
        Ok(version)
    }
}

fn check_config_key(key: &str) -> Result<&str, VfoxError> {
    if is_valid_config_key(key) {
        Ok(key)
    } else {
        Err(VfoxError::Invalid(format!("Not a valid configuration key: {key}")))
    }
}

fn trimmed(value: Option<&str>) -> Option<String> {
    value.map(str::trim).filter(|value| !value.is_empty()).map(str::to_owned)
}

fn arguments(parts: &[&str]) -> Vec<String> {
    parts.iter().map(|part| (*part).to_owned()).collect()
}

/// Drives the vfox CLI.
#[derive(Clone)]
pub struct VfoxCli {
    runner: Rc<dyn ProcessRunner>,
    launch_config: Rc<RefCell<VfoxLaunchConfig>>,
}

impl VfoxCli {
    /// A client that runs vfox with `runner`, as `config` says.
    pub fn new(runner: Rc<dyn ProcessRunner>, config: VfoxLaunchConfig) -> Self {
        Self { runner, launch_config: Rc::new(RefCell::new(config)) }
    }

    /// How vfox is launched.
    pub fn launch_config(&self) -> VfoxLaunchConfig {
        self.launch_config.borrow().clone()
    }

    /// Changes how vfox is launched, for the following invocations.
    pub fn set_launch_config(&self, config: VfoxLaunchConfig) {
        *self.launch_config.borrow_mut() = config;
    }

    async fn query(&self, arguments: Vec<String>, allow_failure: bool) -> Result<ProcessOutput, VfoxError> {
        let config = self.launch_config();
        let options = ProcessOptions {
            working_directory: config.neutral_directory.clone(),
            environment: config.environment(false),
        };
        let output = self.runner.run(&config.executable, &arguments, &options).await.map_err(not_found)?;
        if output.exit_code != 0 && !allow_failure {
            return Err(failure(&arguments, &output));
        }
        Ok(output)
    }

    /// The version of vfox, for example `1.0.12`.
    pub async fn version(&self) -> Result<String, VfoxError> {
        let arguments = arguments(&["--version"]);
        let output = self.query(arguments.clone(), false).await?;
        parse_vfox_version(&format!("{}\n{}", output.stdout, output.stderr))
            .ok_or_else(|| VfoxError::failed("Unrecognised version output", &arguments, Some(output.exit_code)))
    }

    /// All installed plugins with their installed and current versions (`vfox
    /// list` and `vfox current`).
    pub async fn installed_sdks(&self) -> Result<Vec<InstalledSdk>, VfoxError> {
        let (list, current) =
            tokio::join!(self.query(arguments(&["list"]), true), self.query(arguments(&["current"]), false));
        let current = parse_current(&current?.stdout);
        let list = list?;
        if list.exit_code != 0 {
            // `vfox list` fails when no plugin is installed; `vfox current`
            // then prints nothing.
            if current.is_empty() {
                return Ok(Vec::new());
            }
            return Err(failure(&arguments(&["list"]), &list));
        }
        Ok(merge_installed_sdks(&parse_installed_tree(&list.stdout), &current))
    }

    /// Metadata of the installed plugin `name`.
    pub async fn plugin_info(&self, name: &str) -> Result<PluginInfo, VfoxError> {
        let arguments = arguments(&["info", check_name(name)?, "--format", PLUGIN_INFO_TEMPLATE]);
        let output = self.query(arguments.clone(), false).await?;
        parse_plugin_info(&output.stdout)
            .ok_or_else(|| VfoxError::failed("Unrecognised plugin information", &arguments, Some(output.exit_code)))
    }

    /// The directory of an installed version, if it is installed.
    pub async fn install_path(&self, name: &str, version: &str) -> Result<Option<String>, VfoxError> {
        let subject = format!("{}@{}", check_name(name)?, check_version(version)?);
        Ok(parse_install_path(&self.query(arguments(&["info", &subject]), false).await?.stdout))
    }

    /// Versions of `name` that can be installed, newest first; may take a while.
    pub async fn search_versions(&self, name: &str) -> Result<Vec<AvailableVersion>, VfoxError> {
        Ok(parse_search(&self.query(arguments(&["search", check_name(name)?]), false).await?.stdout))
    }

    /// Plugins listed in the vfox registry.
    pub async fn available_plugins(&self) -> Result<Vec<AvailablePlugin>, VfoxError> {
        Ok(parse_available_plugins(&self.query(arguments(&["available"]), false).await?.stdout))
    }

    /// The vfox configuration as `key = value` entries.
    pub async fn config_entries(&self) -> Result<Vec<ConfigEntry>, VfoxError> {
        Ok(parse_config_list(&self.query(arguments(&["config", "--list"]), false).await?.stdout))
    }

    /// Sets the configuration `key` to `value`.
    pub async fn set_config(&self, key: &str, value: &str) -> Result<(), VfoxError> {
        self.query(arguments(&["config", check_config_key(key)?, value]), false).await.map(|_| ())
    }

    /// Resets the configuration `key` to its default value.
    pub async fn unset_config(&self, key: &str) -> Result<(), VfoxError> {
        self.query(arguments(&["config", "--unset", check_config_key(key)?]), false).await.map(|_| ())
    }

    /// Installs `version` of `name`.
    pub fn install(&self, name: &str, version: &str) -> Result<VfoxCommand, VfoxError> {
        let subject = format!("{}@{}", check_name(name)?, check_version(version)?);
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            version: Some(version.into()),
            ..VfoxCommand::new(VfoxOperation::Install, arguments(&["install", "--yes", &subject]))
        })
    }

    /// Uninstalls `version` of `name`.
    pub fn uninstall(&self, name: &str, version: &str) -> Result<VfoxCommand, VfoxError> {
        let subject = format!("{}@{}", check_name(name)?, check_version(version)?);
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            version: Some(version.into()),
            ..VfoxCommand::new(VfoxOperation::Uninstall, arguments(&["uninstall", &subject]))
        })
    }

    /// Makes `version` of `name` the current version in `scope`; the project
    /// scope needs `project_directory`.
    pub fn use_version(
        &self,
        name: &str,
        version: &str,
        scope: UseScope,
        project_directory: Option<&str>,
    ) -> Result<VfoxCommand, VfoxError> {
        let subject = format!("{}@{}", check_name(name)?, check_version(version)?);
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            version: Some(version.into()),
            working_directory: scope_directory(scope, project_directory)?,
            requires_hook_environment: true,
            ..VfoxCommand::new(VfoxOperation::Use, arguments(&["use", scope_flag(scope), &subject]))
        })
    }

    /// Removes the current version of `name` from `scope`.
    pub fn unuse(
        &self,
        name: &str,
        scope: UseScope,
        project_directory: Option<&str>,
    ) -> Result<VfoxCommand, VfoxError> {
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            working_directory: scope_directory(scope, project_directory)?,
            ..VfoxCommand::new(VfoxOperation::Unuse, arguments(&["unuse", scope_flag(scope), check_name(name)?]))
        })
    }

    /// Adds the plugin `name` from the registry, or from `source` (a URL or a
    /// local path) when given; `alias` installs the plugin under another name.
    pub fn add_plugin(&self, name: &str, source: Option<&str>, alias: Option<&str>) -> Result<VfoxCommand, VfoxError> {
        let name = trimmed(Some(name));
        let source = trimmed(source);
        let alias = trimmed(alias);
        if name.is_none() && source.is_none() {
            return Err(VfoxError::Invalid("A plugin name or a source is required".into()));
        }
        let mut arguments = vec!["add".to_owned()];
        if let Some(source) = &source {
            arguments.extend(["--source".to_owned(), source.clone()]);
        }
        if let Some(alias) = &alias {
            arguments.extend(["--alias".to_owned(), check_name(alias)?.to_owned()]);
        }
        if let Some(name) = &name {
            arguments.push(check_name(name)?.to_owned());
        }
        Ok(VfoxCommand { sdk: alias.or(name), ..VfoxCommand::new(VfoxOperation::AddPlugin, arguments) })
    }

    /// Removes the plugin `name` together with all of its installed versions.
    pub fn remove_plugin(&self, name: &str) -> Result<VfoxCommand, VfoxError> {
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            ..VfoxCommand::new(VfoxOperation::RemovePlugin, arguments(&["remove", "--yes", check_name(name)?]))
        })
    }

    /// Updates the plugin `name` to its latest version.
    pub fn update_plugin(&self, name: &str) -> Result<VfoxCommand, VfoxError> {
        Ok(VfoxCommand {
            sdk: Some(name.into()),
            ..VfoxCommand::new(VfoxOperation::UpdatePlugin, arguments(&["update", check_name(name)?]))
        })
    }

    /// Updates every installed plugin.
    pub fn update_all_plugins(&self) -> VfoxCommand {
        VfoxCommand::new(VfoxOperation::UpdateAllPlugins, arguments(&["update", "--all"]))
    }

    /// Starts `command`.
    pub fn start(&self, command: &VfoxCommand) -> Result<RunningProcess, VfoxError> {
        let config = self.launch_config();
        let options = ProcessOptions {
            working_directory: command.working_directory.clone().unwrap_or_else(|| config.neutral_directory.clone()),
            environment: config.environment(command.requires_hook_environment),
        };
        self.runner.start(&config.executable, &command.arguments, &options).map_err(not_found)
    }
}

fn not_found(error: ProcessStartError) -> VfoxError {
    VfoxError::NotFound { executable: error.executable, message: error.message }
}

fn failure(arguments: &[String], output: &ProcessOutput) -> VfoxError {
    let summary = summarize_error(&output.stdout, &output.stderr);
    let message = if summary.is_empty() { format!("vfox exited with code {}", output.exit_code) } else { summary };
    VfoxError::failed(message, arguments, Some(output.exit_code))
}

fn scope_flag(scope: UseScope) -> &'static str {
    match scope {
        UseScope::Global => "--global",
        UseScope::Project => "--project",
    }
}

fn scope_directory(scope: UseScope, project_directory: Option<&str>) -> Result<Option<String>, VfoxError> {
    match scope {
        UseScope::Global => Ok(None),
        UseScope::Project => trimmed(project_directory)
            .map(Some)
            .ok_or_else(|| VfoxError::Invalid("A project directory is required for the project scope".into())),
    }
}

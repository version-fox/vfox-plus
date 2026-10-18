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

//! State-changing vfox invocations and the errors of vfox.

use std::fmt;

/// The kinds of state-changing operations the application performs.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum VfoxOperation {
    /// `vfox install --yes <name>@<version>`
    Install,
    /// `vfox uninstall <name>@<version>`
    Uninstall,
    /// `vfox use --global|--project <name>@<version>`
    Use,
    /// `vfox unuse --global|--project <name>`
    Unuse,
    /// `vfox add [--source <url>] [--alias <alias>] <name>`
    AddPlugin,
    /// `vfox remove --yes <name>`
    RemovePlugin,
    /// `vfox update <name>`
    UpdatePlugin,
    /// `vfox update --all`
    UpdateAllPlugins,
}

/// A state-changing vfox invocation. Commands are values: the vfox client
/// builds them, the task queue executes them, and running one again retries
/// it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct VfoxCommand {
    /// What the command does.
    pub operation: VfoxOperation,
    /// Arguments passed to the vfox executable.
    pub arguments: Vec<String>,
    /// The plugin the command acts on, if any.
    pub sdk: Option<String>,
    /// The SDK version the command acts on, if any.
    pub version: Option<String>,
    /// Directory to run in; `None` means a neutral directory, so that no
    /// project configuration is picked up.
    pub working_directory: Option<String>,
    /// Whether vfox must believe that it runs inside an activated shell. `vfox
    /// use` refuses to run on Unix without shell integration, but with
    /// `--global` or `--project` it only edits `.vfox.toml` and symlinks. See
    /// `docs/vfox-cli-contract.md`.
    pub requires_hook_environment: bool,
}

impl VfoxCommand {
    /// A command of `operation` with `arguments` and nothing else.
    pub fn new(operation: VfoxOperation, arguments: Vec<String>) -> Self {
        Self {
            operation,
            arguments,
            sdk: None,
            version: None,
            working_directory: None,
            requires_hook_environment: false,
        }
    }

    /// `name@version`, or just the name.
    pub fn subject(&self) -> Option<String> {
        match (&self.sdk, &self.version) {
            (Some(sdk), Some(version)) => Some(format!("{sdk}@{version}")),
            (Some(sdk), None) => Some(sdk.clone()),
            _ => None,
        }
    }

    /// The command line as shown in the task output.
    pub fn command_line(&self) -> String {
        format!("vfox {}", self.arguments.join(" "))
    }
}

/// A vfox invocation failed.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum VfoxError {
    /// The vfox executable could not be started, for example because it is
    /// missing.
    NotFound {
        /// The path or name that could not be executed.
        executable: String,
        /// Why it could not be started.
        message: String,
    },
    /// vfox ran but failed, or printed something unexpected.
    Failed {
        /// A short, human readable description.
        message: String,
        /// The arguments vfox was started with.
        arguments: Vec<String>,
        /// The exit code, if the process ran at all.
        exit_code: Option<i32>,
    },
    /// The arguments were rejected before vfox ran, such as a name that looks
    /// like an option.
    Invalid(String),
}

impl VfoxError {
    /// A failure of vfox with `message`.
    pub fn failed(message: impl Into<String>, arguments: &[String], exit_code: Option<i32>) -> Self {
        Self::Failed { message: message.into(), arguments: arguments.to_vec(), exit_code }
    }

    /// The short, human readable description.
    pub fn message(&self) -> &str {
        match self {
            Self::NotFound { message, .. } | Self::Failed { message, .. } | Self::Invalid(message) => message,
        }
    }
}

impl fmt::Display for VfoxError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.message())
    }
}

impl std::error::Error for VfoxError {}

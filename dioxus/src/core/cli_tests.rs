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

//! Tests of the vfox client with a scripted process runner.

use std::collections::BTreeMap;
use std::rc::Rc;

use super::cli::{SESSION_VARIABLES, VfoxCli, VfoxLaunchConfig, is_valid_plugin_name};
use super::command::VfoxError;
use super::models::{InstalledSdk, UseScope};
use super::testing::{FakeProcessRunner, fixture_output, fixture_runner, output};

fn cli(runner: &FakeProcessRunner) -> VfoxCli {
    let map = |pairs: &[(&str, &str)]| {
        pairs.iter().map(|(k, v)| ((*k).to_owned(), (*v).to_owned())).collect::<BTreeMap<_, _>>()
    };
    VfoxCli::new(
        Rc::new(runner.clone()),
        VfoxLaunchConfig {
            executable: "/usr/local/bin/vfox".into(),
            base_environment: map(&[
                ("HOME", "/home/user"),
                ("PATH", "/usr/bin"),
                ("__VFOX_SHELL", "bash"),
                ("__VFOX_PID", "42"),
                ("__VFOX_CURTMPPATH", "/home/user/.vfox/tmp/x"),
            ]),
            extra_environment: map(&[("VFOX_NODEJS_MIRROR", "https://mirror"), ("PATH", "/opt/bin")]),
            neutral_directory: "/".into(),
        },
    )
}

#[test]
fn the_environment_drops_the_shell_session_disables_colours_and_lets_user_variables_win() {
    let environment = cli(&FakeProcessRunner::new()).launch_config().environment(false);
    for variable in SESSION_VARIABLES {
        assert!(!environment.contains_key(variable), "{variable}");
    }
    assert_eq!(environment["NO_COLOR"], "1");
    assert_eq!(environment["HOME"], "/home/user");
    assert_eq!(environment["PATH"], "/opt/bin");
    assert_eq!(environment["VFOX_NODEJS_MIRROR"], "https://mirror");
    assert_eq!(cli(&FakeProcessRunner::new()).launch_config().environment(true)["__VFOX_SHELL"], "bash");
}

#[tokio::test]
async fn queries_run_in_the_neutral_directory_with_the_configured_executable() {
    let runner = fixture_runner();
    assert_eq!(cli(&runner).version().await.unwrap(), "1.0.13");
    let call = &runner.calls()[0];
    assert_eq!(call.executable, "/usr/local/bin/vfox");
    assert_eq!(call.options.working_directory, "/");
    assert_eq!(call.options.environment["NO_COLOR"], "1");
    runner.on("--version", output(0, "hello\n", ""));
    assert_eq!(cli(&runner).version().await.unwrap_err().message(), "Unrecognised version output");
}

#[tokio::test]
async fn installed_sdks_merge_list_and_current() {
    let runner = fixture_runner();
    let sdk = |name: &str, versions: &[&str], current: Option<&str>| InstalledSdk {
        name: name.into(),
        versions: versions.iter().map(|v| (*v).into()).collect(),
        current: current.map(Into::into),
    };
    assert_eq!(
        cli(&runner).installed_sdks().await.unwrap(),
        vec![
            sdk("maven", &["3.9.16", "3.8.9"], Some("3.9.16")),
            sdk("mvn2", &[], None),
            sdk("nodejs", &["24.11.1", "22.21.0"], Some("24.11.1")),
        ]
    );
    runner.on("list", fixture_output("list-empty", 1)).on("current", fixture_output("current-empty", 0));
    assert_eq!(cli(&runner).installed_sdks().await.unwrap(), Vec::new());
    runner.on("list", output(1, "boom\n", "")).on("current", fixture_output("current", 0));
    let error = cli(&runner).installed_sdks().await.unwrap_err();
    assert_eq!(error, VfoxError::failed("boom", &["list".into()], Some(1)));
}

#[tokio::test]
async fn plugin_information_versions_registry_and_install_paths() {
    let runner = fixture_runner();
    let cli = cli(&runner);
    assert_eq!(cli.plugin_info("nodejs").await.unwrap().version, "0.5.1");
    assert_eq!(runner.calls().last().unwrap().arguments[..3], ["info", "nodejs", "--format"]);
    assert!(cli.plugin_info("maven").await.is_err());
    assert!(cli.search_versions("maven").await.unwrap().iter().any(|v| v.version == "3.9.16"));
    assert_eq!(cli.available_plugins().await.unwrap().len(), 43);
    runner
        .on("info nodejs@24.11.1", fixture_output("info-version-path", 0))
        .on("info nodejs@1.0.0", fixture_output("info-version-not-installed", 0));
    assert_eq!(
        cli.install_path("nodejs", "24.11.1").await.unwrap().as_deref(),
        Some("/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1")
    );
    assert_eq!(cli.install_path("nodejs", "1.0.0").await.unwrap(), None);
}

#[tokio::test]
async fn the_configuration_is_listed_set_and_reset() {
    let runner = fixture_runner();
    let cli = cli(&runner);
    runner
        .on("config proxy.url http://proxy:8080", output(0, "", ""))
        .on("config --unset proxy.url", output(0, "", ""));
    assert_eq!(cli.config_entries().await.unwrap().len(), 10);
    cli.set_config("proxy.url", "http://proxy:8080").await.unwrap();
    cli.unset_config("proxy.url").await.unwrap();
    let lines: Vec<String> = runner.calls_starting("config").iter().map(|call| call.arguments.join(" ")).collect();
    assert_eq!(lines, ["config --list", "config proxy.url http://proxy:8080", "config --unset proxy.url"]);
    runner.on("config cache.availableHookDuration bad", fixture_output("config-set-invalid-duration", 1));
    assert_eq!(
        cli.set_config("cache.availableHookDuration", "bad").await.unwrap_err().message(),
        "time: invalid duration \"notaduration\""
    );
    runner.on("config --unset proxy.url", output(3, "", ""));
    assert_eq!(cli.unset_config("proxy.url").await.unwrap_err().message(), "vfox exited with code 3");
    assert!(matches!(cli.set_config("proxy..url", "x").await, Err(VfoxError::Invalid(_))));
}

#[tokio::test]
async fn a_missing_executable_is_reported_as_such() {
    let runner = fixture_runner();
    runner.set_missing_executable(true);
    let cli = cli(&runner);
    assert!(matches!(cli.version().await, Err(VfoxError::NotFound { .. })));
    assert!(matches!(cli.start(&cli.update_all_plugins()), Err(VfoxError::NotFound { .. })));
}

#[test]
fn commands() {
    let cli = cli(&FakeProcessRunner::new());
    let install = cli.install("nodejs", "24.11.1").unwrap();
    assert_eq!(install.arguments, ["install", "--yes", "nodejs@24.11.1"]);
    assert_eq!(install.subject().as_deref(), Some("nodejs@24.11.1"));
    assert_eq!(install.command_line(), "vfox install --yes nodejs@24.11.1");
    assert_eq!(install.working_directory, None);

    let global = cli.use_version("nodejs", "24.11.1", UseScope::Global, None).unwrap();
    assert_eq!(global.arguments, ["use", "--global", "nodejs@24.11.1"]);
    assert!(global.requires_hook_environment);
    let project = cli.use_version("java", "21", UseScope::Project, Some("/work/app")).unwrap();
    assert_eq!(project.arguments, ["use", "--project", "java@21"]);
    assert_eq!(project.working_directory.as_deref(), Some("/work/app"));
    assert!(cli.use_version("java", "21", UseScope::Project, None).is_err());
    assert!(cli.unuse("java", UseScope::Project, Some(" ")).is_err());

    assert_eq!(cli.unuse("java", UseScope::Global, None).unwrap().arguments, ["unuse", "--global", "java"]);
    assert_eq!(cli.unuse("java", UseScope::Project, Some("/p")).unwrap().working_directory.as_deref(), Some("/p"));
    assert_eq!(cli.uninstall("java", "21").unwrap().arguments, ["uninstall", "java@21"]);
    assert_eq!(cli.remove_plugin("java").unwrap().arguments, ["remove", "--yes", "java"]);
    assert_eq!(cli.update_plugin("java").unwrap().arguments, ["update", "java"]);
    assert_eq!(cli.update_all_plugins().arguments, ["update", "--all"]);
    assert_eq!(cli.update_all_plugins().subject(), None);
}

#[test]
fn plugins_are_added_from_the_registry_or_from_a_source() {
    let cli = cli(&FakeProcessRunner::new());
    assert_eq!(cli.add_plugin("nodejs", None, None).unwrap().arguments, ["add", "nodejs"]);
    let custom = cli.add_plugin("", Some(" https://x/p.zip "), Some("mine")).unwrap();
    assert_eq!(custom.arguments, ["add", "--source", "https://x/p.zip", "--alias", "mine"]);
    assert_eq!(custom.subject().as_deref(), Some("mine"));
    assert_eq!(cli.add_plugin("", Some("/p.lua"), None).unwrap().sdk, None);
    assert!(cli.add_plugin("  ", None, None).is_err());
}

#[test]
fn names_and_versions_that_look_like_options_are_rejected() {
    let cli = cli(&FakeProcessRunner::new());
    for (name, version) in [("--all", "1"), ("java", "-1"), ("java", "1 2"), ("java", "a@b"), ("java", "")] {
        assert!(cli.install(name, version).is_err(), "{name} {version}");
    }
    assert!(is_valid_plugin_name("my-sdk.2"));
    assert!(!is_valid_plugin_name("-x"));
    assert!(!is_valid_plugin_name(""));
}

#[test]
fn start_launches_commands_with_the_right_environment() {
    let runner = FakeProcessRunner::new();
    let cli = cli(&runner);
    let _process = cli.start(&cli.use_version("java", "21", UseScope::Project, Some("/work/app")).unwrap()).unwrap();
    let call = &runner.calls()[0];
    assert!(call.streamed);
    assert_eq!(call.options.working_directory, "/work/app");
    assert_eq!(call.options.environment["__VFOX_SHELL"], "bash");
}

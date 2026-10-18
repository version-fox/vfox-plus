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

//! The end-to-end scenarios, only in builds with the `e2e` feature. The
//! application runs the scenario named by `VFOX_PLUS_E2E_SCENARIO`, drives
//! its own page like a user with the driver of the web front end
//! (`web/src/e2e/driver.ts`, bundled into `assets/e2e-driver.js`), checks the
//! state, reports every step on its standard output (`e2e: …`) and exits with
//! code 0 when the scenario passed, 1 when it failed. `web/e2e/run.sh` and
//! `shared/tools/measure.sh` prepare the environment, as for the web front
//! end, whose scenarios (`web/src/e2e/scenarios.ts`) these are.

use std::io::Write as _;
use std::time::{Duration, Instant};

use dioxus::prelude::*;
use serde::de::DeserializeOwned;
use tokio::io::{AsyncReadExt as _, AsyncWriteExt as _};

use super::bridge;
use super::context::AppContext;
use crate::state::tasks::{TaskStatus, VfoxTask};

type Outcome = Result<(), String>;

/// Reports progress on the standard output.
fn step(message: &str) {
    println!("e2e: {message}");
    let _ = std::io::stdout().flush();
}

fn check(condition: bool, message: &str) -> Outcome {
    if condition { Ok(()) } else { Err(format!("Check failed: {message}")) }
}

fn environment(name: &str) -> String {
    std::env::var(name).unwrap_or_default()
}

fn js(value: &impl serde::Serialize) -> String {
    serde_json::to_string(value).unwrap_or_else(|_| "null".to_owned())
}

/// Calls the driver: `call` is a call of a function of `window.vfoxPlusDriver`.
async fn driver<T: DeserializeOwned>(call: &str) -> Result<T, String> {
    bridge::query(&format!("return await window.vfoxPlusDriver.{call};"))
        .await
        .map_err(|error| format!("{call}: {error}"))
}

async fn wait_for_text(text: &str) -> Outcome {
    driver::<serde_json::Value>(&format!("waitForText({}).then(() => true)", js(&text))).await.map(drop)
}

async fn wait_for_text_within(text: &str, milliseconds: u32) -> Outcome {
    driver::<serde_json::Value>(&format!("waitForText({}, {milliseconds}).then(() => true)", js(&text))).await.map(drop)
}

async fn click_button(name: &str) -> Outcome {
    driver::<serde_json::Value>(&format!("clickButton({}).then(() => true)", js(&name))).await.map(drop)
}

async fn click_button_near(text: &str, name: &str) -> Outcome {
    driver::<serde_json::Value>(&format!("clickButtonNear({}, {}).then(() => true)", js(&text), js(&name)))
        .await
        .map(drop)
}

/// Clicks the visible element matching `selector` whose text is `text`, or
/// starts with it.
async fn click_with(selector: &str, text: &str, prefix: bool) -> Outcome {
    driver::<serde_json::Value>(&format!(
        "clickWith({}, {}, {{ prefix: {prefix} }}).then(() => true)",
        js(&selector),
        js(&text)
    ))
    .await
    .map(drop)
}

async fn fill(label: &str, value: &str, submit: bool) -> Outcome {
    driver::<serde_json::Value>(&format!(
        "fill({}, {}, {{ submit: {submit} }}).then(() => true)",
        js(&label),
        js(&value)
    ))
    .await
    .map(drop)
}

async fn count_text(text: &str) -> Result<usize, String> {
    driver(&format!("countText({})", js(&text))).await
}

async fn pause(milliseconds: u64) {
    tokio::time::sleep(Duration::from_millis(milliseconds)).await;
}

/// Waits until `condition` holds; fails when it reports an error.
async fn wait_for(description: &str, mut condition: impl FnMut() -> Result<bool, String>) -> Outcome {
    let end = Instant::now() + Duration::from_secs(30);
    loop {
        if condition()? {
            return Ok(());
        }
        if Instant::now() > end {
            return Err(format!("Timed out waiting for {description}"));
        }
        pause(100).await;
    }
}

/// Waits until the task with `title` (in English) has succeeded; fails if it failed.
async fn wait_for_task(app: AppContext, title: &str) -> Outcome {
    wait_for(&format!("the task \"{title}\" to finish"), || {
        let task = app.model.tasks.tasks().into_iter().find(|task| app.task_title(&task.command) == title);
        let Some(task) = task else { return Ok(false) };
        let state = task.state.peek();
        match state.status {
            TaskStatus::Failed => Err(format!("{title} failed: {}", state.error_message.clone().unwrap_or_default())),
            status => Ok(status == TaskStatus::Succeeded),
        }
    })
    .await
}

/// Runs vfox as the tests see it, without colours, and returns its output.
async fn vfox(arguments: &[&str]) -> Result<String, String> {
    let output = tokio::process::Command::new(environment("VFOX_PLUS_E2E_VFOX"))
        .args(arguments)
        .current_dir("/")
        .env("NO_COLOR", "1")
        .stdin(std::process::Stdio::null())
        .output()
        .await
        .map_err(|error| error.to_string())?;
    Ok(String::from_utf8_lossy(&output.stdout).into_owned())
}

/// Asks the test server of `web/e2e/test-server.mjs` to hold or release the downloads.
async fn control(action: &str) -> Outcome {
    let server = environment("VFOX_PLUS_TEST_SERVER");
    let address = server.trim_start_matches("http://").trim_end_matches('/');
    let mut stream = tokio::net::TcpStream::connect(address).await.map_err(|error| format!("{server}: {error}"))?;
    let request = format!("GET /control/{action} HTTP/1.0\r\nHost: {address}\r\n\r\n");
    stream.write_all(request.as_bytes()).await.map_err(|error| error.to_string())?;
    let mut response = String::new();
    stream.read_to_string(&mut response).await.map_err(|error| error.to_string())?;
    check(
        response.starts_with("HTTP/1.1 200") || response.starts_with("HTTP/1.0 200"),
        &format!("the server can {action}"),
    )
}

/// The application against the fake vfox of `shared/fixtures/fake-vfox.sh`, which replays the fixtures.
async fn fake(app: AppContext) -> Outcome {
    let model = app.model;
    wait_for_text("Versions in use").await?;
    step("home");
    let executable = model.vfox.state.peek().executable.clone().unwrap_or_default();
    wait_for_text(&format!("vfox 1.0.13 · {executable}")).await?;
    check(executable == environment("VFOX_PLUS_E2E_VFOX"), "the executable from the settings is used")?;

    click_button("SDKs").await?;
    click_with("m3e-list-action", "nodejs", true).await?;
    wait_for_text("Node.js runtime environment.").await?;
    step("detail");

    // Select 22.21.0 as the global default.
    click_button("Use").await?;
    wait_for_task(app, "Use nodejs@22.21.0").await?;

    // Install a version from the available tab.
    click_button("Available").await?;
    wait_for_text("872 versions").await?;
    fill("Filter versions", "24.11.0", false).await?;
    wait_for_text("1 version").await?;
    click_button_near("24.11.0", "Install").await?;
    wait_for_task(app, "Install nodejs@24.11.0").await?;

    let tasks = model.tasks.tasks();
    check(tasks.iter().all(|task| task.status() == TaskStatus::Succeeded), "every task succeeded")?;
    check(tasks.first().and_then(VfoxTaskExt::progress_value) == Some(1.0), "the download progress reached 100 %")?;
    check(
        tasks
            .first()
            .is_some_and(|task| task.state.peek().log.iter().any(|line| line == "Install nodejs@24.11.0 success!")),
        "the output of vfox is kept",
    )
}

/// The progress of a task, without subscribing to it.
trait VfoxTaskExt {
    fn progress_value(&self) -> Option<f64>;
}

impl VfoxTaskExt for VfoxTask {
    fn progress_value(&self) -> Option<f64> {
        self.state.peek().progress
    }
}

/// Every operation against the real vfox and the offline test plugin of `shared/test-plugin`.
async fn real(app: AppContext) -> Outcome {
    let model = app.model;
    let server = environment("VFOX_PLUS_TEST_SERVER");
    let home = environment("HOME");
    let local_plugin = environment("VFOX_PLUS_E2E_LOCAL_PLUGIN");

    wait_for_text("Versions in use").await?;
    check(model.installed.value().is_some_and(|sdks| sdks.is_empty()), "no plugin is installed yet")?;

    // Point vfox to the local registry through the vfox configuration editor.
    step("configure the registry");
    click_button("Settings").await?;
    fill("Plugin registry", &format!("{server}/registry"), true).await?;
    wait_for_text("Saved").await?;
    check(
        vfox(&["config", "registry.address"]).await?.trim() == format!("{server}/registry"),
        "vfox stored the registry",
    )?;

    // Add the plugin from the registry.
    step("add a plugin from the registry");
    click_button("Plugins").await?;
    wait_for_text("testsdk").await?;
    click_button("Add").await?;
    wait_for_task(app, "Add plugin testsdk").await?;
    wait_for_text("Added").await?;

    // Add a local copy of the plugin under another name. vfox moves local .lua
    // files away, so the application must hand it a copy.
    step("add a plugin from a local file");
    click_button("Add from source…").await?;
    wait_for_text("Add a plugin from source").await?;
    fill("Source", &local_plugin, false).await?;
    fill("Name (optional)", "localsdk", false).await?;
    click_button("Add plugin").await?;
    wait_for_task(app, "Add plugin localsdk").await?;
    check(std::path::Path::new(&local_plugin).exists(), "the source file survives vfox add --source")?;

    // Search and install both versions.
    step("install");
    click_button("SDKs").await?;
    click_with("m3e-list-action", "testsdk", true).await?;
    wait_for_text("An offline SDK for the vfox-plus end-to-end tests.").await?;
    click_button("Available").await?;
    wait_for_text("2 versions").await?;
    // The list renders its rows after the count.
    wait_for_text("LTS").await?;
    wait_for_text("testtool 2.0").await?;
    check(count_text("LTS").await? == 1, "the note is shown once")?;
    click_button_near("2.0.0", "Install").await?;
    wait_for_task(app, "Install testsdk@2.0.0").await?;
    check(
        model.tasks.tasks().first().and_then(VfoxTaskExt::progress_value) == Some(1.0),
        "vfox reports the download progress on standard error",
    )?;

    // Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
    // leaves the partial installation behind, but must neither list it nor
    // refuse to install the version again.
    step("cancel an installation");
    control("hold").await?;
    click_button_near("1.0.0", "Install").await?;
    wait_for("the download to start", || {
        Ok(model.tasks.tasks().first().and_then(VfoxTaskExt::progress_value).unwrap_or(0.0) > 0.0)
    })
    .await?;
    click_button("Cancel").await?;
    wait_for("vfox to exit", || {
        Ok(model.tasks.tasks().first().is_some_and(|task| task.state.peek().finished_at.is_some()))
    })
    .await?;
    check(model.tasks.tasks().first().map(VfoxTask::status) == Some(TaskStatus::Cancelled), "the task is cancelled")?;
    check(!vfox(&["list", "testsdk"]).await?.contains("1.0.0"), "vfox does not list the partial version")?;
    control("release").await?;
    pause(500).await;
    click_button_near("1.0.0", "Install").await?;
    wait_for_task(app, "Install testsdk@1.0.0").await?;

    // Make 2.0.0 the global default.
    step("use");
    click_button("Installed (2)").await?;
    wait_for_text("1.0.0").await?;
    click_button_near("2.0.0", "Use").await?;
    wait_for_task(app, "Use testsdk@2.0.0").await?;
    wait_for_text("In use: 2.0.0").await?;
    check(
        std::fs::read_to_string(format!("{home}/.vfox/.vfox.toml")).unwrap_or_default().contains("testsdk = \"2.0.0\""),
        "the global .vfox.toml pins 2.0.0",
    )?;
    check(std::path::Path::new(&format!("{home}/.vfox/sdks/testsdk")).exists(), "the SDK is linked")?;

    // Uninstall 1.0.0.
    step("uninstall");
    click_button("More actions for 1.0.0").await?;
    click_with("m3e-menu-item", "Uninstall", false).await?;
    wait_for_text("Uninstall testsdk 1.0.0?").await?;
    click_with("m3e-button", "Uninstall", false).await?;
    wait_for_task(app, "Uninstall testsdk@1.0.0").await?;
    wait_for_text("Installed (1)").await?;

    // Remove the second plugin.
    step("remove");
    click_with("m3e-list-action", "localsdk", true).await?;
    if wait_for_text_within("localsdk runtime environment.", 5000).await.is_err() {
        wait_for_text("localsdk").await?;
    }
    click_button("More actions").await?;
    click_with("m3e-menu-item", "Remove plugin", false).await?;
    wait_for_text("Remove localsdk?").await?;
    click_with("m3e-button", "Remove", false).await?;
    wait_for_task(app, "Remove plugin localsdk").await?;
    wait_for_text("Select an SDK to manage its versions.").await?;
    check(
        !model.installed.value().unwrap_or_default().iter().any(|sdk| sdk.name == "localsdk"),
        "the removed plugin is no longer listed",
    )?;

    // Every task but the cancelled one succeeded, and vfox agrees with what
    // the application shows.
    let statuses: Vec<TaskStatus> =
        model.tasks.tasks().iter().map(VfoxTask::status).filter(|status| *status != TaskStatus::Succeeded).collect();
    check(statuses == [TaskStatus::Cancelled], &format!("only the cancelled task did not succeed: {statuses:?}"))?;
    check(vfox(&["current"]).await? == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0")?;
    check(vfox(&["list", "testsdk"]).await?.contains("2.0.0"), "vfox lists testsdk 2.0.0")?;
    // The copy of the local plugin is deleted once its task has finished.
    let copies: Vec<String> = std::fs::read_dir(format!("{home}/.vfox/tmp"))
        .map(|entries| {
            entries
                .filter_map(Result::ok)
                .map(|entry| entry.file_name().to_string_lossy().into_owned())
                .filter(|name| name.starts_with("vfox-plus-plugin-"))
                .collect()
        })
        .unwrap_or_default();
    check(copies.is_empty(), &format!("the copy of the local plugin is deleted: {copies:?}"))
}

/// Opens the Home page and the versions of Node.js, so that the harness can
/// measure the memory of the application at both points (see
/// `docs/new-implementation.md`).
async fn measure(_app: AppContext) -> Outcome {
    wait_for_text("Versions in use").await?;
    pause(3000).await;
    step("MEASURE home");
    pause(5000).await;
    click_button("SDKs").await?;
    click_with("m3e-list-action", "nodejs", true).await?;
    click_button("Available").await?;
    wait_for_text("872 versions").await?;
    pause(3000).await;
    step("MEASURE nodejs");
    pause(5000).await;
    let _ = click_button("Home").await;
    Ok(())
}

/// Asks the harness for a screenshot named `name` once the page has settled:
/// it watches for the line and captures the window.
async fn shot(name: &str) {
    pause(1500).await;
    step(&format!("SHOT {name}"));
    pause(1500).await;
}

/// Shows every page against the fake vfox, for screenshots (`sandbox/sandbox.sh
/// screenshots`); with `VFOX_PLUS_FAKE_DELAY`, an installation in progress.
async fn screenshots(app: AppContext) -> Outcome {
    use crate::core::settings::{AppSettings, LanguagePreference, ThemePreference};
    use dioxus::desktop::LogicalSize;

    let model = app.model;
    wait_for_text("Versions in use").await?;
    if !environment("VFOX_PLUS_FAKE_DELAY").is_empty() {
        click_button("SDKs").await?;
        click_with("m3e-list-action", "nodejs", true).await?;
        click_button("Available").await?;
        wait_for_text("872 versions").await?;
        click_button_near("26.12.0", "Install").await?;
        wait_for("the download to start", || {
            Ok(model.tasks.tasks().first().and_then(VfoxTaskExt::progress_value).unwrap_or(0.0) > 0.0)
        })
        .await?;
        shot("installing").await;
        return Ok(());
    }
    shot("home").await;
    click_button("SDKs").await?;
    click_with("m3e-list-action", "nodejs", true).await?;
    wait_for_text("Node.js runtime environment.").await?;
    shot("sdks").await;
    click_button("More actions for 22.21.0").await?;
    wait_for_text("Copy installation path").await?;
    shot("menu").await;
    click_with("m3e-menu-item", "Copy installation path", false).await?;
    click_button("Available").await?;
    wait_for_text("872 versions").await?;
    shot("available").await;
    fill("Filter versions", "24.11.0", false).await?;
    wait_for_text("1 version").await?;
    click_button_near("24.11.0", "Install").await?;
    wait_for_task(app, "Install nodejs@24.11.0").await?;
    click_button("Tasks").await?;
    click_button("Show output").await?;
    wait_for_text("Install nodejs@24.11.0 success!").await.or(Ok::<(), String>(()))?;
    shot("tasks").await;
    click_button("Plugins").await?;
    wait_for_text("dotnet").await?;
    shot("plugins").await;
    click_button("Add from source…").await?;
    wait_for_text("Add a plugin from source").await?;
    shot("dialog").await;
    click_button("Cancel").await?;
    click_button("Settings").await?;
    wait_for_text("Plugin registry").await?;
    shot("settings").await;
    model.update_settings(|settings| AppSettings { theme: ThemePreference::Dark, rail_expanded: true, ..settings });
    click_button("Home").await?;
    shot("dark-expanded").await;
    click_button("SDKs").await?;
    click_with("m3e-list-action", "maven", true).await?;
    wait_for_text("3.8.9").await?;
    shot("dark-sdks").await;
    model.update_settings(|settings| AppSettings {
        theme: ThemePreference::Light,
        rail_expanded: false,
        language: LanguagePreference::SimplifiedChinese,
        ..settings
    });
    click_button("首页").await?;
    wait_for_text("正在使用的版本").await?;
    shot("zh-home").await;
    click_button("设置").await?;
    wait_for_text("插件仓库").await?;
    shot("zh-settings").await;
    model.update_settings(|settings| AppSettings { language: LanguagePreference::English, ..settings });
    dioxus::desktop::window().set_inner_size(LogicalSize::new(412.0, 800.0));
    click_button("Home").await?;
    wait_for_text("Versions in use").await?;
    shot("compact-home").await;
    click_button("SDKs").await?;
    click_with("m3e-list-action", "nodejs", true).await?;
    wait_for_text("Node.js runtime environment.").await?;
    shot("compact-detail").await;
    Ok(())
}

/// Runs the scenario named by the environment, if one is, and exits.
pub fn start(app: AppContext) {
    let Ok(name) = std::env::var("VFOX_PLUS_E2E_SCENARIO") else { return };
    spawn(async move {
        let started = Instant::now();
        step(&format!("scenario {name} started"));
        let outcome = match name.as_str() {
            "fake" => fake(app).await,
            "real" => real(app).await,
            "measure" => measure(app).await,
            "screenshots" => screenshots(app).await,
            _ => {
                step(&format!("unknown scenario \"{name}\"; use one of fake, real, measure, screenshots"));
                std::process::exit(2);
            }
        };
        let code = match outcome {
            Ok(()) => {
                step(&format!("scenario {name} passed in {} s", started.elapsed().as_secs()));
                0
            }
            Err(error) => {
                step(&format!("scenario {name} FAILED: {error}"));
                1
            }
        };
        app.model.dispose();
        std::process::exit(code);
    });
}

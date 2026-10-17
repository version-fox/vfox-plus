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

//! Tests of the state, in a virtual DOM without a user interface.

use std::cell::{Cell, RefCell};
use std::collections::{BTreeMap, HashSet};
use std::rc::Rc;
use std::time::Duration;

use dioxus::core::NoOpMutations;
use dioxus::prelude::*;

use super::controllers::{
    SdkDetailController, SdkTab, SettingsController, VersionFilter, VfoxAvailability, VfoxConfigController,
    VfoxController,
};
use super::model::AppModel;
use super::resource::AsyncResource;
use super::tasks::{TaskQueue, TaskStatus, VfoxTask};
use crate::core::cli::{VfoxCli, VfoxLaunchConfig};
use crate::core::command::{VfoxError, VfoxOperation};
use crate::core::locator::VfoxLocator;
use crate::core::settings::{AppSettings, MemorySettingsStore, SettingsStore, ThemePreference};
use crate::core::testing::{FakeProcessRunner, fixture_output, fixture_runner, output};

/// A virtual DOM whose tasks the tests drive.
struct Dom {
    dom: VirtualDom,
}

impl Dom {
    fn new() -> Self {
        let mut dom = VirtualDom::new(|| rsx! {});
        dom.rebuild_in_place();
        Self { dom }
    }

    /// Runs `f` in the scope of the application.
    fn run<R>(&self, f: impl FnOnce() -> R) -> R {
        self.dom.in_scope(ScopeId::APP, f)
    }

    /// Runs the tasks until none has anything to do.
    async fn settle(&mut self) {
        loop {
            tokio::select! {
                () = self.dom.wait_for_work() => self.dom.render_immediate(&mut NoOpMutations),
                () = tokio::time::sleep(Duration::from_millis(20)) => break,
            }
        }
    }
}

fn cli(runner: &FakeProcessRunner) -> VfoxCli {
    VfoxCli::new(Rc::new(runner.clone()), VfoxLaunchConfig::default())
}

fn queue(dom: &Dom, runner: &FakeProcessRunner, finished: &Rc<RefCell<Vec<u32>>>) -> TaskQueue {
    let finished = finished.clone();
    dom.run(|| {
        let queue = TaskQueue::with_limits(cli(runner), 3, 2);
        queue.on_finished(move |task| finished.borrow_mut().push(task.id));
        queue
    })
}

#[tokio::test]
async fn the_queue_runs_one_command_at_a_time_in_order() {
    let mut dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.hold("install --yes nodejs@24").hold("install --yes java@21");
    let finished = Rc::new(RefCell::new(Vec::new()));
    let queue = queue(&dom, &runner, &finished);
    let (a, b) = dom.run(|| {
        let cli = cli(&runner);
        (queue.enqueue(cli.install("nodejs", "24").unwrap()), queue.enqueue(cli.install("java", "21").unwrap()))
    });
    dom.settle().await;
    dom.run(|| {
        assert_eq!(a.status(), TaskStatus::Running);
        assert_eq!(b.status(), TaskStatus::Queued);
        assert_eq!(queue.active_count(), 2);
    });
    assert_eq!(runner.calls().len(), 1);
    runner.processes("install --yes nodejs@24")[0].exit(0);
    dom.settle().await;
    dom.run(|| {
        assert_eq!(a.status(), TaskStatus::Succeeded);
        assert_eq!(b.status(), TaskStatus::Running);
    });
    runner.processes("install --yes java@21")[0].exit(0);
    dom.settle().await;
    assert_eq!(*finished.borrow(), vec![a.id, b.id]);
    dom.run(|| {
        assert_eq!(queue.tasks().iter().map(|task| task.id).collect::<Vec<_>>(), vec![b.id, a.id]);
        assert_eq!(queue.active_count(), 0);
        let state = a.state.peek();
        assert!(state.started_at.is_some() && state.finished_at.is_some());
    });
}

#[tokio::test]
async fn an_identical_active_command_is_not_queued_twice() {
    let dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.hold("update --all");
    let queue = queue(&dom, &runner, &Rc::default());
    dom.run(|| {
        let cli = cli(&runner);
        let a = queue.enqueue(cli.update_all_plugins());
        assert_eq!(queue.enqueue(cli.update_all_plugins()).id, a.id);
        assert_eq!(queue.tasks().len(), 1);
    });
}

#[tokio::test]
async fn progress_bars_become_progress_and_the_log_is_bounded() {
    let mut dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.hold("install --yes maven@3.9.16");
    let queue = queue(&dom, &runner, &Rc::default());
    let task = dom.run(|| queue.enqueue(cli(&runner).install("maven", "3.9.16").unwrap()));
    dom.settle().await;
    let process = runner.processes("install --yes maven@3.9.16")[0].clone();
    process.stdout("Preinstalling maven@3.9.16...\n");
    process.stderr("\rDownloading...  42% [===>  ] (1 MB/s)\r");
    dom.settle().await;
    dom.run(|| {
        assert_eq!(task.progress(), Some(0.42));
        assert_eq!(task.state.peek().log, ["Preinstalling maven@3.9.16..."]);
    });
    process.stdout("one\ntwo\nthree");
    process.exit(0);
    dom.settle().await;
    dom.run(|| {
        let state = task.state.peek();
        assert_eq!(state.log, ["one", "two", "three"]);
        assert_eq!(state.dropped_lines, 1);
        assert_eq!(state.status, TaskStatus::Succeeded);
        assert_eq!(state.exit_code, Some(0));
    });
}

#[tokio::test]
async fn failures_are_summarised_with_the_message_of_vfox() {
    let mut dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.on("install --yes maven@3.9.16", fixture_output("install-failure", 1)).on("update java", output(4, "", ""));
    let finished = Rc::new(RefCell::new(Vec::new()));
    let queue = queue(&dom, &runner, &finished);
    let task = dom.run(|| queue.enqueue(cli(&runner).install("maven", "3.9.16").unwrap()));
    dom.settle().await;
    dom.run(|| {
        assert_eq!(task.status(), TaskStatus::Failed);
        assert_eq!(
            task.state.peek().error_message.as_deref(),
            Some("mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: permission denied")
        );
    });
    assert_eq!(*finished.borrow(), vec![task.id]);
    let silent = dom.run(|| queue.enqueue(cli(&runner).update_plugin("java").unwrap()));
    dom.settle().await;
    dom.run(|| assert_eq!(silent.state.peek().error_message.as_deref(), Some("vfox exited with code 4")));
    runner.set_missing_executable(true);
    let missing = dom.run(|| queue.enqueue(cli(&runner).update_plugin("go").unwrap()));
    dom.settle().await;
    dom.run(|| {
        assert_eq!(missing.status(), TaskStatus::Failed);
        assert_ne!(missing.state.peek().error_message.clone().unwrap_or_default(), "");
    });
}

#[tokio::test]
async fn cancelling_drops_queued_tasks_and_kills_running_ones() {
    let mut dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.hold("install --yes go@1");
    let finished = Rc::new(RefCell::new(Vec::new()));
    let queue = queue(&dom, &runner, &finished);
    let (running, queued) = dom.run(|| {
        let cli = cli(&runner);
        (queue.enqueue(cli.install("go", "1").unwrap()), queue.enqueue(cli.update_plugin("java").unwrap()))
    });
    dom.settle().await;
    dom.run(|| {
        queue.cancel(&queued);
        assert_eq!(queued.status(), TaskStatus::Cancelled);
        queue.cancel(&running);
    });
    dom.settle().await;
    assert!(runner.processes("install --yes go@1")[0].killed());
    dom.run(|| assert_eq!(running.status(), TaskStatus::Cancelled));
    assert_eq!(*finished.borrow(), vec![queued.id, running.id]);
    assert_eq!(runner.calls_starting("update java"), Vec::new());
    // A finished task is not cancelled again.
    dom.run(|| queue.cancel(&running));
    assert_eq!(finished.borrow().len(), 2);
}

#[tokio::test]
async fn a_failed_task_is_retried_finished_tasks_are_cleared_and_the_oldest_forgotten() {
    let mut dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.on("update java", output(1, "java plugin not installed\n", ""));
    let queue = queue(&dom, &runner, &Rc::default());
    let failed = dom.run(|| queue.enqueue(cli(&runner).update_plugin("java").unwrap()));
    dom.settle().await;
    dom.run(|| assert_eq!(failed.status(), TaskStatus::Failed));
    runner.on("update java", fixture_output("update", 0));
    let retried = dom.run(|| queue.retry(&failed));
    dom.settle().await;
    dom.run(|| {
        assert_ne!(failed.id, retried.id);
        assert_eq!(retried.status(), TaskStatus::Succeeded);
        assert!(queue.has_finished());
        queue.clear_finished();
        assert!(queue.tasks().is_empty());
    });
    for name in ["a", "b", "c", "d"] {
        dom.run(|| queue.enqueue(cli(&runner).update_plugin(name).unwrap()));
        dom.settle().await;
    }
    dom.run(|| {
        let names: Vec<Option<String>> = queue.tasks().iter().map(|task| task.command.sdk.clone()).collect();
        assert_eq!(names, [Some("d".into()), Some("c".into()), Some("b".into())]);
    });
}

#[tokio::test]
async fn the_active_task_of_a_version_is_found() {
    let dom = Dom::new();
    let runner = FakeProcessRunner::new();
    runner.hold("install --yes go@1");
    let queue = queue(&dom, &runner, &Rc::default());
    dom.run(|| {
        let task = queue.enqueue(cli(&runner).install("go", "1").unwrap());
        assert_eq!(queue.active_task(Some(VfoxOperation::Install), Some("go"), Some("1")).map(|t| t.id), Some(task.id));
        assert!(queue.active_task(Some(VfoxOperation::Uninstall), Some("go"), None).is_none());
        let id = queue.on_finished(|_| {});
        queue.remove_listener(id);
        queue.dispose();
    });
}

#[tokio::test]
async fn a_resource_keeps_its_value_while_reloading_and_records_errors() {
    let mut dom = Dom::new();
    let calls = Rc::new(Cell::new(0));
    let counter = calls.clone();
    let resource = dom.run(|| {
        AsyncResource::new(move || {
            counter.set(counter.get() + 1);
            let call = counter.get();
            Box::pin(async move { if call == 2 { Err(VfoxError::Invalid("boom".into())) } else { Ok(call) } })
        })
    });
    dom.run(|| resource.load());
    dom.settle().await;
    dom.run(|| {
        assert_eq!(resource.value(), Some(1));
        assert!(resource.has_value());
        resource.load();
    });
    dom.settle().await;
    dom.run(|| {
        assert_eq!(resource.error(), Some(VfoxError::Invalid("boom".into())));
        assert_eq!(resource.value(), Some(1));
        assert!(!resource.is_loading());
        resource.ensure_loaded();
        resource.replace(9);
        assert_eq!(resource.value(), Some(9));
        assert_eq!(resource.error(), None);
    });
    dom.settle().await;
    assert_eq!(calls.get(), 2, "ensure_loaded does not load what is loaded");
}

#[tokio::test]
async fn the_latest_of_overlapping_loads_wins() {
    let mut dom = Dom::new();
    let senders: Rc<RefCell<Vec<tokio::sync::oneshot::Sender<&'static str>>>> = Rc::default();
    let pending = senders.clone();
    let resource = dom.run(|| {
        AsyncResource::new(move || {
            let (sender, receiver) = tokio::sync::oneshot::channel();
            pending.borrow_mut().push(sender);
            Box::pin(async move { Ok(receiver.await.unwrap_or("dropped")) })
        })
    });
    dom.run(|| {
        resource.load();
        resource.load();
        resource.ensure_loaded();
    });
    dom.settle().await;
    let mut senders = std::mem::take(&mut *senders.borrow_mut());
    assert_eq!(senders.len(), 2, "a load in progress is not started again");
    let _ = senders.pop().unwrap().send("new");
    let _ = senders.pop().unwrap().send("old");
    dom.settle().await;
    dom.run(|| assert_eq!(resource.value(), Some("new")));
}

#[tokio::test]
async fn settings_are_persisted_and_no_ops_skipped() {
    let dom = Dom::new();
    let store =
        Rc::new(MemorySettingsStore::new(AppSettings { theme: ThemePreference::Dark, ..AppSettings::default() }));
    let changes = Rc::new(RefCell::new(Vec::new()));
    let recorded = changes.clone();
    dom.run(|| {
        let controller = SettingsController::new(store.clone());
        controller.on_change(move |_, current| recorded.borrow_mut().push(current.theme));
        controller.load();
        assert_eq!(controller.get().theme, ThemePreference::Dark);
        controller.update(|settings| AppSettings { theme: ThemePreference::Dark, ..settings });
        assert_eq!(store.save_count(), 0);
        controller.update(|settings| AppSettings { theme: ThemePreference::Light, ..settings });
    });
    assert_eq!(store.save_count(), 1);
    assert_eq!(store.load().theme, ThemePreference::Light);
    assert_eq!(*changes.borrow(), vec![ThemePreference::Light]);
}

fn vfox_controller(dom: &Dom, runner: &FakeProcessRunner, found: Option<&'static str>) -> (VfoxCli, VfoxController) {
    let cli = cli(runner);
    let locator = VfoxLocator::with_check(BTreeMap::new(), move |path| Some(path) == found);
    let controller =
        dom.run(|| VfoxController::new(cli.clone(), locator, BTreeMap::from([("HOME".into(), "/home/u".into())])));
    (cli, controller)
}

#[tokio::test]
async fn vfox_is_detected_or_chosen_and_its_version_read() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    let (cli, controller) = vfox_controller(&dom, &runner, Some("/usr/local/bin/vfox"));
    let settings =
        AppSettings { extra_environment: BTreeMap::from([("A".into(), "1".into())]), ..AppSettings::default() };
    dom.run(|| spawn(controller.apply(settings)));
    dom.settle().await;
    dom.run(|| {
        let state = controller.state.peek();
        assert_eq!(state.availability, VfoxAvailability::Ready);
        assert_eq!(state.version.as_deref(), Some("1.0.13"));
        assert_eq!(state.executable.as_deref(), Some("/usr/local/bin/vfox"));
        assert!(!state.is_custom_executable);
        assert_eq!(state.error, None);
    });
    assert_eq!(cli.launch_config().extra_environment, BTreeMap::from([("A".into(), "1".into())]));
    dom.run(|| {
        spawn(controller.apply(AppSettings { vfox_executable: Some("/opt/vfox".into()), ..AppSettings::default() }))
    });
    dom.settle().await;
    dom.run(|| {
        assert_eq!(controller.state.peek().executable.as_deref(), Some("/opt/vfox"));
        assert!(controller.state.peek().is_custom_executable);
    });
}

#[tokio::test]
async fn a_missing_and_a_broken_vfox_are_told_apart() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    let (_, controller) = vfox_controller(&dom, &runner, None);
    runner.set_missing_executable(true);
    dom.run(|| spawn(controller.apply(AppSettings::default())));
    dom.settle().await;
    dom.run(|| {
        assert_eq!(controller.availability(), VfoxAvailability::NotFound);
        assert_eq!(controller.state.peek().executable, None);
    });
    runner.set_missing_executable(false);
    runner.on("--version", output(0, "hello", ""));
    dom.run(|| spawn(controller.check()));
    dom.settle().await;
    dom.run(|| {
        assert_eq!(controller.availability(), VfoxAvailability::Broken);
        assert!(matches!(controller.state.peek().error, Some(VfoxError::Failed { .. })));
    });
}

#[tokio::test]
async fn available_versions_are_filtered_by_text_note_and_installation() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    let controller = dom.run(|| SdkDetailController::new(cli(&runner), "nodejs"));
    dom.run(|| controller.show_tab(SdkTab::Available));
    dom.settle().await;
    let installed = HashSet::from(["24.11.1".to_owned()]);
    dom.run(|| {
        let mut filter = controller.filter;
        let mut query = controller.query;
        assert_eq!(controller.visible_versions(&installed).len(), 872);
        filter.set(VersionFilter::WithNote);
        assert!(controller.visible_versions(&installed).iter().all(|version| version.note.as_deref() == Some("LTS")));
        filter.set(VersionFilter::All);
        query.set("24.11.".into());
        let versions: Vec<String> =
            controller.visible_versions(&installed).into_iter().map(|version| version.version).collect();
        assert_eq!(versions, ["24.11.1", "24.11.0"]);
        query.set("lts".into());
        assert!(controller.visible_versions(&installed).len() > 10);
        query.set(String::new());
        filter.set(VersionFilter::Installed);
        assert_eq!(controller.visible_versions(&installed).len(), 1);
    });
}

#[tokio::test]
async fn the_vfox_configuration_is_saved_and_reloaded() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    runner
        .on("config gitignore.enable false", output(0, "", ""))
        .on("config cache.availableHookDuration x", fixture_output("config-set-invalid-duration", 1));
    let controller = dom.run(|| VfoxConfigController::new(cli(&runner)));
    dom.run(|| controller.entries.load());
    dom.settle().await;
    dom.run(|| {
        assert_eq!(controller.value_for("gitignore.enable").as_deref(), Some("true"));
        assert_eq!(controller.value_for("missing"), None);
    });
    let result = Rc::new(RefCell::new(None));
    let saved = result.clone();
    dom.run(|| {
        spawn(
            async move { *saved.borrow_mut() = Some(controller.set("gitignore.enable".into(), "false".into()).await) },
        )
    });
    dom.settle().await;
    assert_eq!(*result.borrow(), Some(Ok(())));
    dom.run(|| assert!(!controller.is_saving("gitignore.enable")));
    assert_eq!(runner.calls_starting("config --list").len(), 2);
    let failed = result.clone();
    dom.run(|| {
        spawn(async move {
            *failed.borrow_mut() = Some(controller.set("cache.availableHookDuration".into(), "x".into()).await);
        })
    });
    dom.settle().await;
    assert!(matches!(*result.borrow(), Some(Err(VfoxError::Failed { .. }))));
    dom.run(|| assert!(controller.entries.has_value()));
}

fn model(dom: &Dom, runner: &FakeProcessRunner, home: &str) -> AppModel {
    dom.run(|| {
        AppModel::new(
            Rc::new(runner.clone()),
            Rc::new(MemorySettingsStore::default()),
            BTreeMap::from([("HOME".into(), home.into())]),
            VfoxLocator::with_check(BTreeMap::new(), |path| path == "/usr/bin/vfox"),
        )
    })
}

#[tokio::test]
async fn the_model_probes_vfox_loads_the_sdks_and_reloads_them_after_tasks() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    let model = model(&dom, &runner, "/home/u");
    dom.run(|| model.start());
    dom.settle().await;
    dom.run(|| {
        assert_eq!(model.vfox.availability(), VfoxAvailability::Ready);
        let names: Vec<String> = model.installed.value().unwrap().into_iter().map(|sdk| sdk.name).collect();
        assert_eq!(names, ["maven", "mvn2", "nodejs"]);
        assert_eq!(model.installed_sdk("nodejs").unwrap().current.as_deref(), Some("24.11.1"));
        assert!(model.installed_sdk("python").is_none());
    });
    let lists = runner.calls_starting("list").len();
    let task: VfoxTask = dom.run(|| {
        model.run(model.cli().use_version("nodejs", "22.21.0", crate::core::models::UseScope::Global, None).unwrap())
    });
    dom.settle().await;
    assert_eq!(runner.calls_starting("list").len(), lists + 1);
    dom.run(|| assert_eq!(task.status(), TaskStatus::Succeeded));
    // A task that never ran changes nothing.
    let (running, queued) = dom.run(|| {
        let running = model.run(model.cli().update_plugin("a").unwrap());
        let queued = model.run(model.cli().update_plugin("b").unwrap());
        model.tasks.cancel(&queued);
        (running, queued)
    });
    dom.settle().await;
    dom.run(|| {
        assert_eq!(running.status(), TaskStatus::Succeeded);
        assert_eq!(queued.status(), TaskStatus::Cancelled);
    });
    assert_eq!(runner.calls_starting("list").len(), lists + 2);
    dom.run(|| {
        assert!(model.run_checked(model.cli().install("--x", "1")).is_none());
        model.dispose();
    });
}

#[tokio::test]
async fn changing_the_executable_or_the_environment_applies_them_again() {
    let mut dom = Dom::new();
    let runner = fixture_runner();
    let model = model(&dom, &runner, "/home/u");
    dom.run(|| model.start());
    dom.settle().await;
    dom.run(|| {
        model.registry.load();
        model.vfox_config.entries.load();
    });
    dom.settle().await;
    dom.run(|| {
        model.update_settings(|settings| AppSettings {
            extra_environment: BTreeMap::from([("X".into(), "1".into())]),
            ..settings
        });
    });
    dom.settle().await;
    assert_eq!(runner.calls_starting("available").len(), 2);
    assert_eq!(runner.calls_starting("config --list").len(), 2);
    assert_eq!(runner.calls().last().unwrap().options.environment.get("X").map(String::as_str), Some("1"));
    dom.run(|| model.update_settings(|settings| AppSettings { vfox_executable: Some("/opt/vfox".into()), ..settings }));
    dom.settle().await;
    dom.run(|| assert_eq!(model.vfox.state.peek().executable.as_deref(), Some("/opt/vfox")));
    assert_eq!(runner.calls().last().unwrap().executable, "/opt/vfox");
    // Other settings do not touch vfox.
    let count = runner.calls().len();
    dom.run(|| model.update_settings(|settings| AppSettings { theme: ThemePreference::Dark, ..settings }));
    dom.settle().await;
    assert_eq!(runner.calls().len(), count);
    dom.run(|| assert_eq!(model.environment()["HOME"], "/home/u"));
}

#[tokio::test]
async fn a_local_lua_plugin_is_handed_to_vfox_as_a_copy_that_is_deleted_afterwards() {
    let mut dom = Dom::new();
    let home = tempfile::tempdir().unwrap();
    let original = home.path().join("mine.lua");
    std::fs::write(&original, "PLUGIN = {}").unwrap();
    let runner = fixture_runner();
    runner.hold("add --source");
    let model = model(&dom, &runner, &home.path().to_string_lossy());
    dom.run(|| model.start());
    dom.settle().await;
    let task = dom.run(|| model.add_plugin_from_source(&original.to_string_lossy(), Some("mine")).unwrap());
    dom.settle().await;
    let arguments = runner.calls_starting("add").pop().unwrap().arguments;
    let copy = arguments[arguments.iter().position(|argument| argument == "--source").unwrap() + 1].clone();
    // vfox renames the file into its temporary directory, which only works on
    // the same filesystem.
    assert!(copy.starts_with(&format!("{}/.vfox/tmp/vfox-plus-plugin-", home.path().display())), "{copy}");
    assert_eq!(arguments[arguments.len() - 2..], ["--alias", "mine"]);
    assert!(std::path::Path::new(&copy).exists(), "the copy exists while vfox runs");
    runner.processes("add --source")[0].exit(0);
    dom.settle().await;
    dom.run(|| assert_eq!(task.status(), TaskStatus::Succeeded));
    assert!(original.exists());
    assert!(!std::path::Path::new(&copy).exists());
    dom.run(|| model.add_plugin_from_source("https://example.com/p.zip", None));
    dom.settle().await;
    assert_eq!(runner.calls_starting("add").pop().unwrap().arguments, ["add", "--source", "https://example.com/p.zip"]);
}

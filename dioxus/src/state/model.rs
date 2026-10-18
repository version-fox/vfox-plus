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

//! The application state, wired together.

use std::collections::BTreeMap;
use std::rc::Rc;

use dioxus::prelude::*;

use super::controllers::{
    SdkDetailController, SettingsController, VfoxAvailability, VfoxConfigController, VfoxController,
};
use super::resource::AsyncResource;
use super::tasks::{TaskQueue, TaskStatus, VfoxTask};
use crate::core::cli::{VfoxCli, VfoxLaunchConfig};
use crate::core::command::{VfoxCommand, VfoxError};
use crate::core::locator::VfoxLocator;
use crate::core::models::{AvailablePlugin, InstalledSdk};
use crate::core::plugin_source::{PreparedPluginSource, prepare_plugin_source, vfox_temporary_directory};
use crate::core::process::ProcessRunner;
use crate::core::settings::{AppSettings, SettingsStore};

/// The application state: every controller, wired together. The model owns
/// the [`VfoxCli`] and keeps the derived data consistent: the installed SDKs
/// are reloaded whenever vfox becomes available and whenever a task that
/// changes them finishes.
#[derive(Clone, Copy, PartialEq)]
pub struct AppModel {
    cli: CopyValue<VfoxCli>,
    environment: CopyValue<BTreeMap<String, String>>,
    /// Application preferences.
    pub settings: SettingsController,
    /// Availability of the vfox executable.
    pub vfox: VfoxController,
    /// Installed plugins and their versions.
    pub installed: AsyncResource<Vec<InstalledSdk>>,
    /// Plugins available in the registry.
    pub registry: AsyncResource<Vec<AvailablePlugin>>,
    /// The vfox configuration.
    pub vfox_config: VfoxConfigController,
    /// State-changing operations.
    pub tasks: TaskQueue,
    plugin_copies: CopyValue<Vec<(u32, PreparedPluginSource)>>,
}

impl AppModel {
    /// The state of the current scope, which runs vfox with `runner` in
    /// `environment` and keeps the settings in `store`.
    pub fn new(
        runner: Rc<dyn ProcessRunner>,
        store: Rc<dyn SettingsStore>,
        environment: BTreeMap<String, String>,
        locator: VfoxLocator,
    ) -> Self {
        let cli = VfoxCli::new(
            runner,
            VfoxLaunchConfig { base_environment: environment.clone(), ..VfoxLaunchConfig::default() },
        );
        let (installed_cli, registry_cli) = (cli.clone(), cli.clone());
        let model = Self {
            cli: CopyValue::new(cli.clone()),
            environment: CopyValue::new(environment.clone()),
            settings: SettingsController::new(store),
            vfox: VfoxController::new(cli.clone(), locator, environment),
            installed: AsyncResource::new(move || {
                let cli = installed_cli.clone();
                Box::pin(async move { cli.installed_sdks().await })
            }),
            registry: AsyncResource::new(move || {
                let cli = registry_cli.clone();
                Box::pin(async move { cli.available_plugins().await })
            }),
            vfox_config: VfoxConfigController::new(cli.clone()),
            tasks: TaskQueue::new(cli),
            plugin_copies: CopyValue::new(Vec::new()),
        };
        model.tasks.on_finished(move |task| model.on_task_finished(task));
        model.settings.on_change(move |previous, current| {
            if previous.vfox_executable != current.vfox_executable
                || previous.extra_environment != current.extra_environment
            {
                model.spawn(model.apply_settings());
            }
        });
        model
    }

    /// The vfox client that every controller shares.
    pub fn cli(&self) -> VfoxCli {
        self.cli.peek().clone()
    }

    /// The environment of the application.
    pub fn environment(&self) -> BTreeMap<String, String> {
        self.environment.peek().clone()
    }

    /// Runs `future` as a task of the scope of the model, which outlives the
    /// pages that start it.
    pub fn spawn(&self, future: impl Future<Output = ()> + 'static) {
        super::spawn_in(self.cli.origin_scope(), future);
    }

    /// Loads the settings and probes vfox in the background.
    pub fn start(&self) {
        self.settings.load();
        self.spawn(self.apply_settings());
    }

    /// Points vfox at the executable and the environment of the settings and
    /// reloads what depends on it.
    pub async fn apply_settings(self) {
        self.vfox.apply(self.settings.get()).await;
        if self.vfox.availability() == VfoxAvailability::Ready {
            self.installed.load();
            if self.registry.has_value() || self.registry.error().is_some() {
                self.registry.load();
            }
            if self.vfox_config.entries.has_value() || self.vfox_config.entries.error().is_some() {
                self.vfox_config.entries.load();
            }
        }
    }

    /// Changes the settings and saves them.
    pub fn update_settings(&self, change: impl FnOnce(AppSettings) -> AppSettings) {
        self.settings.update(change);
    }

    /// The installed SDK called `name`, if it is installed.
    pub fn installed_sdk(&self, name: &str) -> Option<InstalledSdk> {
        self.installed.with(|sdks| sdks.and_then(|sdks| sdks.iter().find(|sdk| sdk.name == name).cloned()))
    }

    /// The state of the detail view of `name`, in the current scope.
    pub fn sdk_detail(&self, name: &str) -> SdkDetailController {
        SdkDetailController::new(self.cli(), name)
    }

    /// Queues `command` and returns its task.
    pub fn run(&self, command: VfoxCommand) -> VfoxTask {
        self.tasks.enqueue(command)
    }

    /// Queues the command that `command` builds, unless it is invalid.
    pub fn run_checked(&self, command: Result<VfoxCommand, VfoxError>) -> Option<VfoxTask> {
        match command {
            Ok(command) => Some(self.run(command)),
            Err(error) => {
                eprintln!("vfox+: {error}");
                None
            }
        }
    }

    /// Adds a plugin from `source`, a URL or a local `.zip` or `.lua` file,
    /// optionally under `alias`. vfox moves a local `.lua` file away instead of
    /// reading it, so such a file is copied into the temporary directory of
    /// vfox and vfox gets the copy, which is deleted once the task has
    /// finished.
    pub fn add_plugin_from_source(&self, source: &str, alias: Option<&str>) -> Option<VfoxTask> {
        let cli = self.cli();
        let prepared =
            prepare_plugin_source(source, &vfox_temporary_directory(&cli.launch_config().environment(false)));
        let task = self.run_checked(cli.add_plugin("", Some(&prepared.path), alias))?;
        if prepared.is_copy() {
            let mut copies = self.plugin_copies;
            copies.write().push((task.id, prepared));
        }
        Some(task)
    }

    fn on_task_finished(&self, task: &VfoxTask) {
        // Dropping the copy deletes it.
        let mut copies = self.plugin_copies;
        copies.write().retain(|(id, _)| *id != task.id);
        // A task that never ran changed nothing.
        let state = task.state.peek();
        if state.status == TaskStatus::Cancelled && state.started_at.is_none() {
            return;
        }
        drop(state);
        self.installed.load();
    }

    /// Releases every resource when the application quits.
    pub fn dispose(&self) {
        self.tasks.dispose();
        let mut copies = self.plugin_copies;
        copies.write().clear();
    }
}

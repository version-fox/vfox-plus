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

//! The controllers of the application state: the settings, vfox itself, the
//! vfox configuration and the detail of an SDK.

use std::collections::{BTreeMap, BTreeSet, HashSet};
use std::rc::Rc;

use dioxus::prelude::*;

use super::resource::AsyncResource;
use crate::core::cli::{VFOX_EXECUTABLE_NAME, VfoxCli};
use crate::core::command::VfoxError;
use crate::core::locator::VfoxLocator;
use crate::core::models::{AvailableVersion, ConfigEntry, PluginInfo};
use crate::core::settings::{AppSettings, SettingsStore};

type SettingsListener = Rc<dyn Fn(&AppSettings, &AppSettings)>;

/// Holds the [`AppSettings`] and persists every change.
#[derive(Clone, Copy, PartialEq)]
pub struct SettingsController {
    store: CopyValue<Rc<dyn SettingsStore>>,
    /// The current settings.
    pub settings: Signal<AppSettings>,
    listeners: CopyValue<Vec<SettingsListener>>,
}

impl SettingsController {
    /// A controller of the current scope that keeps the settings in `store`.
    pub fn new(store: Rc<dyn SettingsStore>) -> Self {
        Self {
            store: CopyValue::new(store),
            settings: Signal::new(AppSettings::default()),
            listeners: CopyValue::new(Vec::new()),
        }
    }

    /// The current settings.
    pub fn get(&self) -> AppSettings {
        self.settings.read().clone()
    }

    /// Calls `listener` with the previous and the new settings after every change.
    pub fn on_change(&self, listener: impl Fn(&AppSettings, &AppSettings) + 'static) {
        let mut listeners = self.listeners;
        listeners.write().push(Rc::new(listener));
    }

    /// Loads the persisted settings.
    pub fn load(&self) {
        let settings = self.store.peek().load();
        let mut signal = self.settings;
        signal.set(settings);
    }

    /// Applies `change` to the settings and saves the result.
    pub fn update(&self, change: impl FnOnce(AppSettings) -> AppSettings) {
        let previous = self.settings.peek().clone();
        let updated = change(previous.clone());
        if updated == previous {
            return;
        }
        let mut signal = self.settings;
        signal.set(updated.clone());
        let listeners: Vec<SettingsListener> = self.listeners.peek().clone();
        for listener in listeners {
            listener(&previous, &updated);
        }
        if let Err(error) = self.store.peek().save(&updated) {
            eprintln!("vfox+: cannot save the settings: {error}");
        }
    }
}

/// Whether vfox can be used.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum VfoxAvailability {
    /// Not checked yet.
    Unknown,
    /// The executable is being probed.
    Checking,
    /// vfox runs and reported its version.
    Ready,
    /// No vfox executable was found.
    NotFound,
    /// An executable was found but did not behave like vfox.
    Broken,
}

/// What [`VfoxController`] knows about vfox.
#[derive(Clone, Debug, PartialEq)]
pub struct VfoxState {
    /// Whether vfox can be used.
    pub availability: VfoxAvailability,
    /// The executable in use, if one was found.
    pub executable: Option<String>,
    /// Whether the executable was chosen by the user rather than detected.
    pub is_custom_executable: bool,
    /// The vfox version, once vfox is ready.
    pub version: Option<String>,
    /// Why vfox is not available, if it is not.
    pub error: Option<VfoxError>,
}

/// Locates vfox, configures the [`VfoxCli`] and checks that vfox works.
#[derive(Clone, Copy, PartialEq)]
pub struct VfoxController {
    cli: CopyValue<VfoxCli>,
    locator: CopyValue<VfoxLocator>,
    environment: CopyValue<BTreeMap<String, String>>,
    /// What is known about vfox.
    pub state: Signal<VfoxState>,
}

impl VfoxController {
    /// A controller of the current scope.
    pub fn new(cli: VfoxCli, locator: VfoxLocator, environment: BTreeMap<String, String>) -> Self {
        Self {
            cli: CopyValue::new(cli),
            locator: CopyValue::new(locator),
            environment: CopyValue::new(environment),
            state: Signal::new(VfoxState {
                availability: VfoxAvailability::Unknown,
                executable: None,
                is_custom_executable: false,
                version: None,
                error: None,
            }),
        }
    }

    /// Whether vfox can be used.
    pub fn availability(&self) -> VfoxAvailability {
        self.state.read().availability
    }

    /// Points the CLI at the executable and the environment of `settings`
    /// and checks the result.
    pub async fn apply(self, settings: AppSettings) {
        let executable = settings.vfox_executable.clone().or_else(|| self.locator.peek().locate());
        {
            let mut state = self.state;
            let mut state = state.write();
            state.is_custom_executable = settings.vfox_executable.is_some();
            state.executable.clone_from(&executable);
        }
        let cli = self.cli.peek().clone();
        let mut config = cli.launch_config();
        config.executable = executable.unwrap_or_else(|| VFOX_EXECUTABLE_NAME.to_owned());
        config.base_environment = self.environment.peek().clone();
        config.extra_environment = settings.extra_environment;
        cli.set_launch_config(config);
        self.check().await;
    }

    /// Runs `vfox --version` to check that vfox works.
    pub async fn check(self) {
        let mut state = self.state;
        state.write().availability = VfoxAvailability::Checking;
        let result = self.cli.peek().clone().version().await;
        let mut state = state.write();
        match result {
            Ok(version) => {
                state.version = Some(version);
                state.error = None;
                state.availability = VfoxAvailability::Ready;
            }
            Err(error) => {
                state.version = None;
                state.availability = if matches!(error, VfoxError::NotFound { .. }) {
                    VfoxAvailability::NotFound
                } else {
                    VfoxAvailability::Broken
                };
                state.error = Some(error);
            }
        }
    }
}

/// The vfox configuration (`vfox config`), editable key by key.
#[derive(Clone, Copy, PartialEq)]
pub struct VfoxConfigController {
    cli: CopyValue<VfoxCli>,
    /// The configuration entries.
    pub entries: AsyncResource<Vec<ConfigEntry>>,
    saving: Signal<BTreeSet<String>>,
}

impl VfoxConfigController {
    /// A controller of the current scope.
    pub fn new(cli: VfoxCli) -> Self {
        let loader = cli.clone();
        Self {
            cli: CopyValue::new(cli),
            entries: AsyncResource::new(move || {
                let cli = loader.clone();
                Box::pin(async move { cli.config_entries().await })
            }),
            saving: Signal::new(BTreeSet::new()),
        }
    }

    /// The value of `key`, if vfox knows the key.
    pub fn value_for(&self, key: &str) -> Option<String> {
        self.entries.with(|entries| {
            entries.and_then(|entries| entries.iter().find(|entry| entry.key == key)).map(|entry| entry.value.clone())
        })
    }

    /// Whether `key` is being saved.
    pub fn is_saving(&self, key: &str) -> bool {
        self.saving.read().contains(key)
    }

    /// Sets `key` to `value` and reloads.
    ///
    /// # Errors
    ///
    /// When vfox rejects the value.
    pub async fn set(self, key: String, value: String) -> Result<(), VfoxError> {
        let cli = self.cli.peek().clone();
        self.save(key.clone(), async move { cli.set_config(&key, &value).await }).await
    }

    /// Resets `key` to its default and reloads.
    ///
    /// # Errors
    ///
    /// When vfox fails.
    pub async fn reset(self, key: String) -> Result<(), VfoxError> {
        let cli = self.cli.peek().clone();
        self.save(key.clone(), async move { cli.unset_config(&key).await }).await
    }

    async fn save(self, key: String, action: impl Future<Output = Result<(), VfoxError>>) -> Result<(), VfoxError> {
        let mut saving = self.saving;
        saving.write().insert(key.clone());
        let result = action.await;
        if let Ok(mut saving) = saving.try_write() {
            saving.remove(&key);
        }
        self.entries.refresh().await;
        result
    }
}

/// Which versions the available tab lists.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum VersionFilter {
    /// Every version.
    All,
    /// Versions with a note such as `LTS`.
    WithNote,
    /// Versions that are installed already.
    Installed,
}

/// The two tabs of the detail of an SDK.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SdkTab {
    /// The installed versions.
    Installed,
    /// The versions that can be installed.
    Available,
}

/// State of the detail view of one installed plugin.
#[derive(Clone, Copy, PartialEq)]
pub struct SdkDetailController {
    /// The plugin name.
    pub name: CopyValue<String>,
    /// Plugin metadata.
    pub info: AsyncResource<PluginInfo>,
    /// Installable versions, loaded when they are first shown because the
    /// search can be slow.
    pub versions: AsyncResource<Vec<AvailableVersion>>,
    /// The visible tab.
    pub tab: Signal<SdkTab>,
    /// The text the available versions are filtered by.
    pub query: Signal<String>,
    /// Which available versions are shown.
    pub filter: Signal<VersionFilter>,
}

impl SdkDetailController {
    /// The state of the detail of `name`, in the current scope.
    pub fn new(cli: VfoxCli, name: &str) -> Self {
        let (info_cli, info_name) = (cli.clone(), name.to_owned());
        let (versions_cli, versions_name) = (cli, name.to_owned());
        Self {
            name: CopyValue::new(name.to_owned()),
            info: AsyncResource::new(move || {
                let (cli, name) = (info_cli.clone(), info_name.clone());
                Box::pin(async move { cli.plugin_info(&name).await })
            }),
            versions: AsyncResource::new(move || {
                let (cli, name) = (versions_cli.clone(), versions_name.clone());
                Box::pin(async move { cli.search_versions(&name).await })
            }),
            tab: Signal::new(SdkTab::Installed),
            query: Signal::new(String::new()),
            filter: Signal::new(VersionFilter::All),
        }
    }

    /// Shows `tab`, loading the available versions on first use.
    pub fn show_tab(&self, tab: SdkTab) {
        let mut signal = self.tab;
        signal.set(tab);
        if tab == SdkTab::Available {
            self.versions.ensure_loaded();
        }
    }

    /// The available versions matching the query and the filter. `installed`
    /// holds the installed versions; it is passed in rather than taken from
    /// the search output, so that it stays current after installations.
    pub fn visible_versions(&self, installed: &HashSet<String>) -> Vec<AvailableVersion> {
        let needle = self.query.read().trim().to_lowercase();
        let filter = *self.filter.read();
        self.versions.with(|versions| {
            versions
                .into_iter()
                .flatten()
                .filter(|version| {
                    let matches_filter = match filter {
                        VersionFilter::All => true,
                        VersionFilter::WithNote => version.note.is_some(),
                        VersionFilter::Installed => installed.contains(&version.version),
                    };
                    matches_filter
                        && (needle.is_empty()
                            || version.version.to_lowercase().contains(&needle)
                            || version.note.as_ref().is_some_and(|note| note.to_lowercase().contains(&needle)))
                })
                .cloned()
                .collect()
        })
    }
}

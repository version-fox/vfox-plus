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

//! What every component of the user interface needs: the application model,
//! the navigation state, the translations and the dialogs, in one context.

use dioxus::prelude::*;

use super::bridge;
use super::components::dialogs::Dialogs;
use crate::core::command::{VfoxCommand, VfoxOperation};
use crate::core::i18n::{Localizations, MessageArgument};
use crate::state::model::AppModel;

/// The version of the application.
pub const APP_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Links that the user interface opens.
pub mod links {
    /// How to install vfox.
    pub const VFOX_INSTALL: &str = "https://vfox.dev/guides/quick-start.html";
    /// The documentation of vfox.
    pub const VFOX_DOCS: &str = "https://vfox.dev/";
    /// The repository of vfox-plus.
    pub const REPOSITORY: &str = "https://github.com/version-fox/vfox-plus";
}

/// The line that activates vfox in Bash.
pub const ACTIVATION_COMMAND: &str = r#"eval "$(vfox activate bash)""#;

/// The top-level destinations of the application.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Destination {
    /// The overview.
    Home,
    /// The installed SDKs.
    Sdks,
    /// The plugin registry.
    Plugins,
    /// The task queue.
    Tasks,
    /// The settings.
    Settings,
}

impl Destination {
    /// The destinations in the order of the navigation.
    pub const ALL: [Self; 5] = [Self::Home, Self::Sdks, Self::Plugins, Self::Tasks, Self::Settings];

    /// The message key of the label.
    pub fn label(self) -> &'static str {
        match self {
            Self::Home => "navHome",
            Self::Sdks => "navSdks",
            Self::Plugins => "navPlugins",
            Self::Tasks => "navTasks",
            Self::Settings => "navSettings",
        }
    }

    /// The icon meaning.
    pub fn icon(self) -> &'static str {
        match self {
            Self::Home => "home",
            Self::Sdks => "sdks",
            Self::Plugins => "plugins",
            Self::Tasks => "tasks",
            Self::Settings => "settings",
        }
    }
}

/// Navigation state of the application shell.
#[derive(Clone, Copy, PartialEq)]
pub struct Navigation {
    /// The visible destination.
    pub destination: Signal<Destination>,
    /// The SDK shown on the SDKs page, if any.
    pub selected_sdk: Signal<Option<String>>,
}

impl Navigation {
    /// The navigation of the current scope, on the Home page.
    pub fn new() -> Self {
        Self { destination: Signal::new(Destination::Home), selected_sdk: Signal::new(None) }
    }

    /// Shows `destination`.
    pub fn go(&self, destination: Destination) {
        let mut visible = self.destination;
        if *visible.peek() != destination {
            visible.set(destination);
        }
    }

    /// Shows the SDKs page with `name` selected, or with no selection.
    pub fn show_sdk(&self, name: Option<String>) {
        let mut selected = self.selected_sdk;
        selected.set(name);
        self.go(Destination::Sdks);
    }
}

impl Default for Navigation {
    fn default() -> Self {
        Self::new()
    }
}

/// The running application, which every component reaches with [`use_app`].
#[derive(Clone, Copy, PartialEq)]
pub struct AppContext {
    /// The application state.
    pub model: AppModel,
    /// Where the user is.
    pub navigation: Navigation,
    /// The translations of the current language.
    pub l10n: Memo<Localizations>,
    /// Whether the colours are dark.
    pub dark: Memo<bool>,
    /// The dialog that is shown.
    pub dialogs: Dialogs,
}

/// The running application.
pub fn use_app() -> AppContext {
    use_context::<AppContext>()
}

impl AppContext {
    /// The message `key` of the current language.
    pub fn t(&self, key: &str) -> String {
        self.l10n.read().t(key, &[])
    }

    /// The message `key` of the current language, formatted with `arguments`.
    pub fn tf(&self, key: &str, arguments: &[(&str, MessageArgument)]) -> String {
        self.l10n.read().t(key, arguments)
    }

    /// The localized title of a task that runs `command`.
    pub fn task_title(&self, command: &VfoxCommand) -> String {
        let subject = command.subject().unwrap_or_default();
        let key = match command.operation {
            VfoxOperation::Install => "taskInstall",
            VfoxOperation::Uninstall => "taskUninstall",
            VfoxOperation::Use => "taskUse",
            VfoxOperation::Unuse => "taskUnuse",
            VfoxOperation::AddPlugin if subject.is_empty() => return self.t("taskAddPluginFromSource"),
            VfoxOperation::AddPlugin => "taskAddPlugin",
            VfoxOperation::RemovePlugin => "taskRemovePlugin",
            VfoxOperation::UpdatePlugin => "taskUpdatePlugin",
            VfoxOperation::UpdateAllPlugins => return self.t("taskUpdateAllPlugins"),
        };
        self.tf(key, &[("subject", subject.into())])
    }

    /// Shows `message` in a snack bar.
    pub fn notify(&self, message: String) {
        bridge::notify(message, self.t("actionClose"));
    }
}

/// The value of a boolean attribute of a custom element: present or absent.
/// Dioxus only removes the boolean attributes of HTML when they are false.
pub fn flag(on: bool) -> Option<&'static str> {
    on.then_some("")
}

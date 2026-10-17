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

//! The dialogs of the application, on `m3e-dialog`. One dialog at a time is
//! shown by [`DialogHost`]; [`Dialogs::show`] resolves with what the user
//! chose once the dialog has closed, by a button, the Escape key or the scrim.

use std::path::Path;

use dioxus::prelude::*;
use tokio::sync::oneshot;

use super::parts::{IconButton, icon};
use crate::core::cli::is_valid_plugin_name;
use crate::ui::bridge;
use crate::ui::context::{flag, use_app};
use crate::ui::desktop::{self, Choose};

/// A dialog to show.
#[derive(Clone, Debug, PartialEq)]
pub enum DialogRequest {
    /// Asks the user to confirm an action.
    Confirm {
        /// The question.
        title: String,
        /// What happens.
        body: String,
        /// The label of the confirming button.
        confirm_label: String,
        /// The icon above the question.
        icon: Option<&'static str>,
        /// Whether the action destroys something, in the error colour.
        destructive: bool,
    },
    /// Asks for the project directory in which to pin `name` `version`.
    ProjectDirectory {
        /// The SDK.
        name: String,
        /// The version.
        version: String,
    },
    /// Asks for the source of a plugin that is not in the registry.
    CustomPlugin,
    /// Shows a long, monospaced text such as the open-source licenses.
    Text {
        /// The heading.
        title: String,
        /// The text.
        text: String,
    },
}

/// What the user chose in a dialog.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DialogResult {
    /// The dialog was closed without a choice.
    Dismissed,
    /// The action was confirmed.
    Confirmed,
    /// A project directory.
    Directory(String),
    /// The source of a plugin, with an optional alias.
    Plugin {
        /// A URL or a local path.
        source: String,
        /// The name to install the plugin under.
        alias: Option<String>,
    },
}

/// The dialog that is shown and whoever waits for it.
#[derive(Clone, Copy, PartialEq)]
pub struct Dialogs {
    current: Signal<Option<(u32, DialogRequest)>>,
    waiting: CopyValue<Option<oneshot::Sender<DialogResult>>>,
    counter: CopyValue<u32>,
}

impl Dialogs {
    /// The dialogs of the current scope.
    pub fn new() -> Self {
        Self { current: Signal::new(None), waiting: CopyValue::new(None), counter: CopyValue::new(0) }
    }

    /// Shows `request` and resolves with the choice of the user once the
    /// dialog has closed. A dialog that is still shown is dismissed.
    pub async fn show(self, request: DialogRequest) -> DialogResult {
        let (sender, receiver) = oneshot::channel();
        let mut waiting = self.waiting;
        if let Some(previous) = waiting.write().replace(sender) {
            let _ = previous.send(DialogResult::Dismissed);
        }
        let mut counter = self.counter;
        *counter.write() += 1;
        let mut current = self.current;
        current.set(Some((*counter.peek(), request)));
        receiver.await.unwrap_or(DialogResult::Dismissed)
    }

    /// Asks the user to confirm an action.
    pub async fn confirm(
        self,
        title: String,
        body: String,
        confirm_label: String,
        icon: &'static str,
        destructive: bool,
    ) -> bool {
        self.show(DialogRequest::Confirm { title, body, confirm_label, icon: Some(icon), destructive }).await
            == DialogResult::Confirmed
    }

    fn finish(self, id: u32, result: DialogResult) {
        if self.current.peek().as_ref().is_some_and(|(current, _)| *current == id) {
            let mut current = self.current;
            current.set(None);
            let mut waiting = self.waiting;
            if let Some(sender) = waiting.write().take() {
                let _ = sender.send(result);
            }
        }
    }
}

impl Default for Dialogs {
    fn default() -> Self {
        Self::new()
    }
}

/// Shows the current dialog.
#[component]
pub fn DialogHost() -> Element {
    let current = use_app().dialogs.current.read().clone();
    match current {
        Some((id, request)) => rsx! { DialogView { key: "{id}", id, request } },
        None => rsx! {},
    }
}

#[component]
fn DialogView(id: u32, request: DialogRequest) -> Element {
    let app = use_app();
    let mut open = use_signal(|| true);
    let mut result = use_signal(|| DialogResult::Dismissed);
    let element = format!("dialog-{id}");
    let mut close = move |value: DialogResult| {
        result.set(value);
        open.set(false);
    };
    let watched = element.clone();
    let onmounted = move |_| {
        let element = watched.clone();
        spawn(async move {
            bridge::dialog_closed(&element).await;
            app.dialogs.finish(id, result.peek().clone());
        });
    };
    let cancel = app.t("actionCancel");
    let body = match request {
        DialogRequest::Confirm { title, body, confirm_label, icon: meaning, destructive } => rsx! {
            div { "slot": "header", class: if destructive { "dialog-header destructive" } else { "dialog-header" },
                if let Some(meaning) = meaning {
                    {icon(meaning, false, None)}
                }
                span { class: "type-headline-small", "{title}" }
            }
            p { class: "dialog-body type-body-medium", "{body}" }
            div { "slot": "actions", class: "dialog-actions",
                m3e-button { "variant": "text", onclick: move |_| close(DialogResult::Dismissed), "{cancel}" }
                m3e-button {
                    "variant": "filled",
                    class: if destructive { "destructive-button" } else { "" },
                    "autofocus": true,
                    onclick: move |_| close(DialogResult::Confirmed),
                    "{confirm_label}"
                }
            }
        },
        DialogRequest::ProjectDirectory { name, version } => rsx! {
            ProjectDirectoryForm { id, name, version, onclose: close }
        },
        DialogRequest::CustomPlugin => rsx! {
            CustomPluginForm { id, onclose: close }
        },
        DialogRequest::Text { title, text } => rsx! {
            div { "slot": "header", class: "dialog-header", span { class: "type-headline-small", "{title}" } }
            div { class: "text-dialog", pre { class: "mono type-body-small", "{text}" } }
            div { "slot": "actions", class: "dialog-actions",
                m3e-button { "variant": "text", onclick: move |_| close(DialogResult::Dismissed), {app.t("actionClose")} }
            }
        },
    };
    rsx! {
        m3e-dialog { id: "{element}", "open": open(), onmounted, {body} }
    }
}

/// The fields and buttons of the dialog that asks for a project directory.
#[component]
fn ProjectDirectoryForm(id: u32, name: String, version: String, onclose: EventHandler<DialogResult>) -> Element {
    let app = use_app();
    let mut directory = use_signal(String::new);
    let mut error = use_signal(String::new);
    let input = format!("directory-{id}");
    let mut submit = move || {
        let path = directory.peek().trim().to_owned();
        if path.is_empty() || !Path::new(&path).is_dir() {
            error.set(app.t("useProjectInvalid"));
        } else {
            onclose.call(DialogResult::Directory(path));
        }
    };
    let browse = move |_| {
        spawn(async move {
            let current = directory.peek().trim().to_owned();
            let initial = if current.is_empty() { app.model.environment().get("HOME").cloned() } else { Some(current) };
            let title = app.t("useProjectDirectory");
            if let Some(chosen) = desktop::choose(Choose::Directory, &title, initial.as_deref(), None).await {
                directory.set(chosen);
                error.set(String::new());
            }
        });
    };
    rsx! {
        div { "slot": "header", class: "dialog-header",
            {icon("projectScope", false, None)}
            span { class: "type-headline-small",
                {app.tf("useProjectTitle", &[("name", name.into()), ("version", version.into())])}
            }
        }
        p { class: "dialog-body type-body-medium", {app.t("useProjectBody")} }
        div { class: "field-row",
            m3e-form-field { "variant": "outlined", "error": flag(!error.read().is_empty()),
                label { "slot": "label", r#for: "{input}", {app.t("useProjectDirectory")} }
                input {
                    id: "{input}",
                    autofocus: true,
                    value: "{directory}",
                    oninput: move |event| {
                        directory.set(event.value());
                        error.set(String::new());
                    },
                    onkeydown: move |event| {
                        if event.key() == Key::Enter {
                            submit();
                        }
                    },
                }
                span { "slot": "error", "{error}" }
            }
            IconButton { icon: "folder", variant: "tonal", label: app.t("actionBrowse"), onclick: browse }
        }
        div { "slot": "actions", class: "dialog-actions",
            m3e-button { "variant": "text", onclick: move |_| onclose.call(DialogResult::Dismissed), {app.t("actionCancel")} }
            m3e-button { "variant": "filled", onclick: move |_| submit(), {app.t("useProjectConfirm")} }
        }
    }
}

/// The fields and buttons of the dialog that asks for the source of a plugin.
#[component]
fn CustomPluginForm(id: u32, onclose: EventHandler<DialogResult>) -> Element {
    let app = use_app();
    let mut source = use_signal(String::new);
    let mut alias = use_signal(String::new);
    let mut source_error = use_signal(String::new);
    let mut alias_error = use_signal(String::new);
    let (source_input, alias_input) = (format!("source-{id}"), format!("alias-{id}"));
    let mut submit = move || {
        let chosen = source.peek().trim().to_owned();
        let name = alias.peek().trim().to_owned();
        source_error.set(if chosen.is_empty() { app.t("pluginsCustomRequired") } else { String::new() });
        alias_error.set(if !name.is_empty() && !is_valid_plugin_name(&name) {
            app.t("pluginsCustomInvalidName")
        } else {
            String::new()
        });
        if source_error.peek().is_empty() && alias_error.peek().is_empty() {
            onclose.call(DialogResult::Plugin { source: chosen, alias: Some(name).filter(|name| !name.is_empty()) });
        }
    };
    let browse = move |_| {
        spawn(async move {
            let home = app.model.environment().get("HOME").cloned();
            let (title, label) = (app.t("pluginsCustomSource"), app.t("pluginsCustomFileType"));
            if let Some(file) =
                desktop::choose(Choose::File, &title, home.as_deref(), Some((&label, &["zip", "lua"]))).await
            {
                source.set(file);
                source_error.set(String::new());
            }
        });
    };
    let on_key = move |event: KeyboardEvent| {
        if event.key() == Key::Enter {
            submit();
        }
    };
    rsx! {
        div { "slot": "header", class: "dialog-header",
            {icon("plugins", false, None)}
            span { class: "type-headline-small", {app.t("pluginsCustomTitle")} }
        }
        div { class: "field-row",
            m3e-form-field { "variant": "outlined", "error": flag(!source_error.read().is_empty()),
                label { "slot": "label", r#for: "{source_input}", {app.t("pluginsCustomSource")} }
                input {
                    id: "{source_input}",
                    autofocus: true,
                    value: "{source}",
                    oninput: move |event| {
                        source.set(event.value());
                        source_error.set(String::new());
                    },
                    onkeydown: on_key,
                }
                span { "slot": "hint", {app.t("pluginsCustomSourceHelp")} }
                span { "slot": "error", "{source_error}" }
            }
            IconButton { icon: "folder", variant: "tonal", label: app.t("actionBrowse"), onclick: browse }
        }
        div { class: "field-row",
            m3e-form-field { "variant": "outlined", "error": flag(!alias_error.read().is_empty()),
                label { "slot": "label", r#for: "{alias_input}", {app.t("pluginsCustomAlias")} }
                input {
                    id: "{alias_input}",
                    value: "{alias}",
                    oninput: move |event| {
                        alias.set(event.value());
                        alias_error.set(String::new());
                    },
                    onkeydown: on_key,
                }
                span { "slot": "hint", {app.t("pluginsCustomAliasHelp")} }
                span { "slot": "error", "{alias_error}" }
            }
        }
        div { "slot": "actions", class: "dialog-actions",
            m3e-button { "variant": "text", onclick: move |_| onclose.call(DialogResult::Dismissed), {app.t("actionCancel")} }
            m3e-button { "variant": "filled", onclick: move |_| submit(), {app.t("pluginsCustomConfirm")} }
        }
    }
}

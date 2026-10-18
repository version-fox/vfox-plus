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

//! Everything about one installed plugin: its metadata, its installed versions
//! and the versions that can be installed.

use std::collections::HashSet;

use dioxus::core::Task;
use dioxus::prelude::*;

use crate::core::command::VfoxOperation;
use crate::core::models::{AvailableVersion, InstalledSdk, UseScope, VersionAddition};
use crate::state::controllers::{SdkDetailController, SdkTab, VersionFilter};
use crate::state::tasks::{TaskStatus, VfoxTask};
use crate::ui::bridge;
use crate::ui::components::dialogs::{DialogRequest, DialogResult};
use crate::ui::components::parts::{
    IconButton, StatusMessage, avatar, choice_group, error_message, icon, loading, unique_id,
};
use crate::ui::context::{AppContext, flag, use_app};
use crate::ui::desktop;
use crate::ui::icons::symbol;

/// The distance between the rows of the available versions, which all have
/// the same height (`.version-row` in assets/app.css), so that only the rows
/// in view are rendered: Node.js alone has close to 900 versions.
pub const VERSION_ROW_PITCH: f64 = 66.0;

/// Rows rendered beyond those in view, on either side.
const OVERSCAN: usize = 6;

/// A fraction as a whole percentage.
pub fn percent(fraction: f64) -> u32 {
    // Between 0 and 100.
    #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
    let value = (fraction.clamp(0.0, 1.0) * 100.0).round() as u32;
    value
}

/// The detail of the installed plugin `name`; with `on_back`, it has a back
/// button.
#[component]
pub fn SdkDetail(name: String, #[props(default)] on_back: Option<EventHandler<()>>) -> Element {
    let app = use_app();
    let model = app.model;
    let controller = use_hook(|| {
        let controller = model.sdk_detail(&name);
        controller.info.load();
        controller
    });
    // Updating the plugin changes its information.
    let listener = use_hook(|| {
        let name = name.clone();
        model.tasks.on_finished(move |task| {
            if task.command.sdk.as_deref() == Some(name.as_str())
                && task.command.operation == VfoxOperation::UpdatePlugin
            {
                controller.info.load();
            }
        })
    });
    use_drop(move || model.tasks.remove_listener(listener));
    let mut width = use_signal(|| 1000.0_f64);
    let menu_id = use_hook(|| unique_id("sdk-menu"));

    let sdk =
        model.installed_sdk(&name).unwrap_or(InstalledSdk { name: name.clone(), versions: Vec::new(), current: None });
    let tab = *controller.tab.read();
    let tabs = vec![
        (SdkTab::Installed, format!("{} ({})", app.t("sdkTabInstalled"), sdk.versions.len()), Some("versions")),
        (SdkTab::Available, app.t("sdkTabAvailable"), Some("install")),
    ];
    rsx! {
        div {
            class: "sdk-detail",
            onresize: move |event| {
                // Resize events bubble, and the SDKs page has its own width.
                event.stop_propagation();
                if let Ok(size) = event.get_content_box_size()
                    && (size.width - *width.peek()).abs() > 0.5
                {
                    width.set(size.width);
                }
            },
            {header(app, controller, &sdk, width() < 520.0, on_back, &menu_id)}
            div { class: "tabs", {choice_group(tabs, tab, Callback::new(move |tab| controller.show_tab(tab)))} }
            if tab == SdkTab::Installed {
                {installed(app, controller, &sdk, &menu_id)}
            } else {
                AvailableVersions { controller, sdk: sdk.clone() }
            }
        }
    }
}

fn header(
    app: AppContext,
    controller: SdkDetailController,
    sdk: &InstalledSdk,
    narrow: bool,
    on_back: Option<EventHandler<()>>,
    menu_id: &str,
) -> Element {
    let model = app.model;
    let info = controller.info.value();
    let updating = model.tasks.active_task(Some(VfoxOperation::UpdatePlugin), Some(&sdk.name), None).is_some();
    let name = sdk.name.clone();
    let homepage = info.as_ref().map(|info| info.homepage.clone()).filter(|homepage| !homepage.is_empty());
    let count = sdk.versions.len();
    let back = on_back.map(|handler| {
        rsx! {
            IconButton { icon: "back", label: app.t("actionClose"), onclick: move |_| handler.call(()) }
        }
    });
    let actions = rsx! {
        if updating {
            m3e-loading-indicator { class: "small", "aria-label": app.t("loading") }
        } else {
            IconButton {
                icon: "update",
                label: app.t("sdkUpdatePlugin"),
                onclick: {
                    let name = name.clone();
                    move |_| {
                        model.run_checked(model.cli().update_plugin(&name));
                    }
                },
            }
        }
        IconButton { icon: "more", label: app.t("sdkMoreActions"),
            m3e-menu-trigger { "for": "{menu_id}" }
        }
        m3e-menu { id: "{menu_id}", "position-x": "before",
            if let Some(homepage) = homepage {
                m3e-menu-item { onclick: move |_| desktop::open_url(&homepage),
                    {icon("openInNew", false, Some("icon"))}
                    {app.t("actionOpenHomepage")}
                }
            }
            m3e-menu-item {
                class: "danger",
                onclick: move |_| {
                    let name = name.clone();
                    spawn(async move {
                        let confirmed = app
                            .dialogs
                            .confirm(
                                app.tf("sdkRemoveTitle", &[("name", name.as_str().into())]),
                                app.tf("sdkRemoveBody", &[("count", count.into())]),
                                app.t("sdkRemoveConfirm"),
                                "removePlugin",
                                true,
                            )
                            .await;
                        if confirmed {
                            model.run_checked(model.cli().remove_plugin(&name));
                            app.navigation.show_sdk(None);
                        }
                    });
                },
                {icon("removePlugin", false, Some("icon"))}
                {app.t("sdkRemovePlugin")}
            }
        }
    };
    let current_tag = match &sdk.current {
        Some(version) => rsx! {
            span { class: "tag emphasized",
                {icon("current", true, None)}
                {app.tf("sdksCurrent", &[("version", version.into())])}
            }
        },
        None => rsx! {
            span { class: "tag", {app.t("sdksNoCurrent")} }
        },
    };
    let text = rsx! {
        div { class: "text",
            h1 { class: "type-headline-medium-emphasized", role: "heading", "aria-level": "1", "{sdk.name}" }
            div { class: "tags",
                {current_tag}
                if let Some(version) = info.as_ref().map(|info| info.version.clone()).filter(|version| !version.is_empty()) {
                    span { class: "tag", {app.tf("sdkPluginVersion", &[("version", version.into())])} }
                }
            }
            if let Some(description) = info.as_ref().map(|info| info.description.clone()).filter(|description| !description.is_empty()) {
                p { class: "description type-body-medium", "{description}" }
            }
        }
    };
    if narrow {
        rsx! {
            header { class: "header narrow",
                div { class: "actions",
                    {back}
                    span { class: "spacer" }
                    {actions}
                }
                div { class: "identity",
                    {avatar(&sdk.name, 56, None, None)}
                    {text}
                }
            }
        }
    } else {
        rsx! {
            header { class: "header",
                div { class: "identity",
                    {back}
                    {avatar(&sdk.name, 72, None, None)}
                    {text}
                }
                div { class: "actions", {actions} }
            }
        }
    }
}

fn installed(app: AppContext, controller: SdkDetailController, sdk: &InstalledSdk, menu_id: &str) -> Element {
    if sdk.versions.is_empty() {
        return rsx! {
            StatusMessage { icon: "install", shape: "pill", heading: app.tf("sdkInstalledEmpty", &[("name", sdk.name.as_str().into())]),
                m3e-button { "variant": "filled", onclick: move |_| controller.show_tab(SdkTab::Available),
                    {icon("install", false, Some("icon"))}
                    {app.t("sdkBrowseVersions")}
                }
            }
        };
    }
    rsx! {
        div { class: "list",
            m3e-list { "variant": "segmented",
                for (index, version) in sdk.versions.iter().enumerate() {
                    {installed_version(app, sdk, version, index, menu_id)}
                }
            }
        }
    }
}

/// Uses `version` of `name` in a project that the user chooses.
fn use_in_project(app: AppContext, name: String, version: String) {
    spawn(async move {
        let request = DialogRequest::ProjectDirectory { name: name.clone(), version: version.clone() };
        if let DialogResult::Directory(directory) = app.dialogs.show(request).await {
            app.model.run_checked(app.model.cli().use_version(&name, &version, UseScope::Project, Some(&directory)));
        }
    });
}

fn installed_version(app: AppContext, sdk: &InstalledSdk, version: &str, index: usize, menu_id: &str) -> Element {
    let model = app.model;
    let current = sdk.current.as_deref() == Some(version);
    let task = model.tasks.active_task(None, Some(&sdk.name), Some(version));
    // Identifiers of menus must be valid in CSS selectors, which versions are not.
    let menu = format!("{menu_id}-version-{index}");
    let use_menu = format!("{menu}-use");
    let (name, version) = (sdk.name.clone(), version.to_owned());
    let use_globally = {
        let (name, version) = (name.clone(), version.clone());
        move |_| {
            model.run_checked(model.cli().use_version(&name, &version, UseScope::Global, None));
        }
    };
    let actions = if let Some(task) = task {
        rsx! {
            m3e-loading-indicator { class: "small", "aria-label": app.task_title(&task.command) }
        }
    } else {
        rsx! {
            div { class: "version-actions",
                if current {
                    span { class: "tag emphasized",
                        {icon("current", true, None)}
                        {app.t("versionInUse")}
                    }
                } else {
                    m3e-split-button { "variant": "tonal",
                        m3e-button { "slot": "leading-button", onclick: use_globally.clone(),
                            {icon("check", false, Some("icon"))}
                            {app.t("versionUse")}
                        }
                        m3e-icon-button { "slot": "trailing-button", "aria-label": app.t("versionUseOptions"),
                            m3e-icon { "variant": "rounded", "name": symbol("expandMore") }
                            m3e-menu-trigger { "for": "{use_menu}" }
                        }
                    }
                    m3e-menu { id: "{use_menu}", "position-x": "before",
                        m3e-menu-item { onclick: use_globally,
                            {icon("globalScope", false, Some("icon"))}
                            {app.t("versionUseGlobally")}
                        }
                        m3e-menu-item {
                            onclick: {
                                let (name, version) = (name.clone(), version.clone());
                                move |_| use_in_project(app, name.clone(), version.clone())
                            },
                            {icon("projectScope", false, Some("icon"))}
                            {app.t("versionUseInProject")}
                        }
                    }
                }
                IconButton { icon: "more", label: app.tf("versionMoreActions", &[("version", version.as_str().into())]),
                    m3e-menu-trigger { "for": "{menu}" }
                }
                m3e-menu { id: "{menu}", "position-x": "before",
                    if current {
                        m3e-menu-item {
                            onclick: {
                                let (name, version) = (name.clone(), version.clone());
                                move |_| use_in_project(app, name.clone(), version.clone())
                            },
                            {icon("projectScope", false, Some("icon"))}
                            {app.t("versionUseInProject")}
                        }
                        m3e-menu-item {
                            onclick: {
                                let name = name.clone();
                                move |_| {
                                    model.run_checked(model.cli().unuse(&name, UseScope::Global, None));
                                }
                            },
                            {icon("cancel", false, Some("icon"))}
                            {app.t("versionUnuse")}
                        }
                    }
                    m3e-menu-item {
                        onclick: {
                            let (name, version) = (name.clone(), version.clone());
                            move |_| {
                                let (name, version) = (name.clone(), version.clone());
                                spawn(async move {
                                    if let Ok(Some(path)) = model.cli().install_path(&name, &version).await {
                                        desktop::copy_text(&path);
                                        app.notify(app.t("actionCopied"));
                                    }
                                });
                            }
                        },
                        {icon("copy", false, Some("icon"))}
                        {app.t("versionCopyPath")}
                    }
                    m3e-menu-item {
                        class: "danger",
                        onclick: {
                            let (name, version) = (name.clone(), version.clone());
                            move |_| {
                                let (name, version) = (name.clone(), version.clone());
                                spawn(async move {
                                    let body = if current { app.t("versionUninstallCurrentBody") } else { app.t("versionUninstallBody") };
                                    let title = app.tf("versionUninstallTitle", &[("name", name.as_str().into()), ("version", version.as_str().into())]);
                                    if app.dialogs.confirm(title, body, app.t("versionUninstall"), "uninstall", true).await {
                                        model.run_checked(model.cli().uninstall(&name, &version));
                                    }
                                });
                            }
                        },
                        {icon("uninstall", false, Some("icon"))}
                        {app.t("versionUninstall")}
                    }
                }
            }
        }
    };
    rsx! {
        m3e-list-item { key: "{version}",
            m3e-icon {
                "slot": "leading",
                "variant": "rounded",
                class: if current { "current-icon" } else { "" },
                "name": symbol(if current { "current" } else { "versions" }),
                "filled": flag(current),
            }
            span { class: if current { "type-body-large-emphasized" } else { "" }, "{version}" }
            div { "slot": "trailing", {actions} }
        }
    }
}

/// The versions that can be installed, filtered; only the rows in view are
/// rendered.
#[component]
fn AvailableVersions(controller: SdkDetailController, sdk: InstalledSdk) -> Element {
    let app = use_app();
    let mut rows = use_signal(|| (0_usize, 24_usize));
    let mut watcher = use_signal(|| None::<Task>);
    let list_id = use_hook(|| unique_id("versions"));
    let versions = controller.versions;
    if !versions.has_value() {
        return match versions.error() {
            Some(error) => {
                let retry = Callback::new(move |()| versions.load());
                error_message(app, app.tf("availableError", &[("name", sdk.name.as_str().into())]), &error, Some(retry))
            }
            None => loading(app, Some(app.t("availableLoading"))),
        };
    }
    let installed: HashSet<String> = sdk.versions.iter().cloned().collect();
    let visible = controller.visible_versions(&installed);
    let count = visible.len();
    let (start, end) = *rows.read();
    let end = end.min(count);
    let start = start.min(end);
    let filters = vec![
        (VersionFilter::All, app.t("availableFilterAll"), None),
        (VersionFilter::WithNote, app.t("availableFilterTagged"), None),
        (VersionFilter::Installed, app.t("availableFilterInstalled"), None),
    ];
    let hint = app.t("availableSearchHint");
    let mut query = controller.query;
    let mut filter = controller.filter;
    #[allow(clippy::cast_precision_loss)]
    let (height, offset) = ((count as f64 * VERSION_ROW_PITCH - 2.0).max(0.0), start as f64 * VERSION_ROW_PITCH);
    let watched = list_id.clone();
    rsx! {
        div { class: "toolbar",
            m3e-form-field { "variant": "outlined", "hide-subscript": "always",
                {icon("search", false, Some("prefix"))}
                input {
                    placeholder: "{hint}",
                    "aria-label": "{hint}",
                    value: "{query}",
                    oninput: move |event| query.set(event.value()),
                }
            }
            {choice_group(filters, *filter.read(), Callback::new(move |value| filter.set(value)))}
            span { class: "count type-label-large", {app.tf("availableCount", &[("count", count.into())])} }
        }
        if count == 0 {
            StatusMessage { icon: "notFound", shape: "9-sided-cookie", heading: app.t("availableEmpty") }
        }
        div { class: "list", hidden: count == 0,
            div {
                id: "{list_id}",
                style: "height: {height}px",
                onmounted: move |_| {
                    if let Some(task) = watcher.write().take() {
                        task.cancel();
                    }
                    let id = watched.clone();
                    watcher.set(Some(spawn(async move {
                        bridge::watch_visible_rows(&id, VERSION_ROW_PITCH, OVERSCAN, move |start, end| rows.set((start, end))).await;
                    })));
                },
                div { style: "transform: translateY({offset}px)",
                    for (index, version) in visible[start..end].iter().enumerate() {
                        {version_row(app, &sdk.name, version, installed.contains(&version.version), start + index, count)}
                    }
                }
            }
        }
    }
}

fn version_row(
    app: AppContext,
    name: &str,
    version: &AvailableVersion,
    installed: bool,
    index: usize,
    count: usize,
) -> Element {
    let model = app.model;
    let task = model.tasks.active_task(Some(VfoxOperation::Install), Some(name), Some(&version.version));
    let additions = version.additions.iter().map(VersionAddition::label).collect::<Vec<_>>().join(" · ");
    let trailing = if let Some(task) = task {
        rsx! { InstallProgress { task } }
    } else if installed {
        rsx! {
            m3e-button { "variant": "text", "disabled": true,
                {icon("check", false, Some("icon"))}
                {app.t("versionInstalled")}
            }
        }
    } else {
        let (name, number) = (name.to_owned(), version.version.clone());
        rsx! {
            m3e-button {
                "variant": "tonal",
                onclick: move |_| {
                    model.run_checked(model.cli().install(&name, &number));
                },
                {icon("install", false, Some("icon"))}
                {app.t("versionInstall")}
            }
        }
    };
    let position = match (index == 0, index + 1 == count) {
        (true, true) => "first last",
        (true, false) => "first",
        (false, true) => "last",
        (false, false) => "",
    };
    rsx! {
        div { key: "{version.version}", class: "version-row {position}",
            div { class: "text",
                div { class: "available",
                    span { class: "title type-body-large", "{version.version}" }
                    if let Some(note) = &version.note {
                        span { class: "tag emphasized", "{note}" }
                    }
                }
                if !additions.is_empty() {
                    div { class: "additions type-body-medium", "{additions}" }
                }
            }
            {trailing}
        }
    }
}

/// The download progress of an installation, with a cancel button.
#[component]
fn InstallProgress(task: VfoxTask) -> Element {
    let app = use_app();
    let state = task.state.read();
    let label = match state.progress {
        Some(progress) => app.tf("taskDownloading", &[("percent", percent(progress).into())]),
        None if state.status == TaskStatus::Queued => app.t("taskStatusQueued"),
        None => app.t("taskStatusRunning"),
    };
    let value = percent(state.progress.unwrap_or(0.0));
    let mode = if state.progress.is_some() { "determinate" } else { "indeterminate" };
    drop(state);
    let title = app.task_title(&task.command);
    let cancel = move |_| app.model.tasks.cancel(&task);
    rsx! {
        div { class: "install-progress",
            div { class: "bar",
                m3e-linear-progress-indicator {
                    "variant": "wavy",
                    "mode": mode,
                    "value": "{value}",
                    "aria-label": "{title}",
                }
                span { class: "type-label-small", "{label}" }
            }
            IconButton { icon: "cancel", label: app.t("actionCancel"), onclick: cancel }
        }
    }
}

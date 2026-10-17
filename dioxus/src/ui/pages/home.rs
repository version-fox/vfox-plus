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

//! The overview: what is installed, what is in use and where to go next.

use dioxus::prelude::*;

use crate::ui::components::parts::{IconButton, avatar, error_message, icon, loading, page_header};
use crate::ui::context::{ACTIVATION_COMMAND, AppContext, Destination, links, use_app};
use crate::ui::desktop;

/// A statistic of the overview.
struct Tile {
    value: usize,
    label: String,
    icon: &'static str,
    shape: &'static str,
    role: &'static str,
}

/// The Home page.
#[component]
pub fn HomePage() -> Element {
    let app = use_app();
    let model = app.model;
    let reload = Callback::new(move |()| model.installed.load());
    let Some(sdks) = model.installed.value() else {
        let body = match model.installed.error() {
            Some(error) => error_message(app, app.t("sdksLoadError"), &error, Some(reload)),
            None => loading(app, None),
        };
        return rsx! {
            div { class: "page-root home-page", div { class: "pane", {body} } }
        };
    };
    let active: Vec<_> = sdks.iter().filter(|sdk| sdk.current.is_some()).cloned().collect();
    let versions: usize = sdks.iter().map(|sdk| sdk.versions.len()).sum();
    let vfox = model.vfox.state.read().clone();
    let subtitle = [
        Some(app.tf("homeSubtitle", &[("version", vfox.version.unwrap_or_else(|| "?".into()).into())])),
        vfox.executable,
    ]
    .into_iter()
    .flatten()
    .collect::<Vec<_>>()
    .join(" · ");
    let tiles = [
        Tile {
            value: sdks.len(),
            label: app.tf("homeStatPlugins", &[("count", sdks.len().into())]),
            icon: "plugins",
            shape: "9-sided-cookie",
            role: "primary",
        },
        Tile {
            value: versions,
            label: app.tf("homeStatVersions", &[("count", versions.into())]),
            icon: "versions",
            shape: "4-leaf-clover",
            role: "secondary",
        },
        Tile { value: active.len(), label: app.t("homeStatActive"), icon: "current", shape: "sunny", role: "tertiary" },
    ];
    let actions = rsx! {
        IconButton {
            icon: "refresh",
            label: app.t("actionRefresh"),
            disabled: model.installed.is_loading(),
            onclick: move |_| model.installed.load(),
        }
    };

    rsx! {
        div { class: "page-root home-page",
            div { class: "pane",
                {page_header(app.t("appTitle"), Some(subtitle), actions)}
                div { class: "page-content",
                    div { class: "stats",
                        for tile in tiles {
                            button {
                                class: "tile",
                                style: "--tile-container: var(--md-sys-color-{tile.role}-container); --tile-on-container: var(--md-sys-color-on-{tile.role}-container)",
                                onclick: move |_| app.navigation.show_sdk(None),
                                m3e-shape { "name": tile.shape, "aria-hidden": "true" }
                                {icon(tile.icon, true, None)}
                                span { class: "value", "{tile.value}" }
                                span { class: "label type-label-large-emphasized", "{tile.label}" }
                            }
                        }
                    }
                    h2 { class: "group-header", role: "heading", "aria-level": "2", {app.t("homeActiveTitle")} }
                    m3e-action-list { "variant": "segmented",
                        if active.is_empty() {
                            m3e-list-action { onclick: move |_| app.navigation.show_sdk(None),
                                {icon("info", false, Some("leading"))}
                                {app.t("homeActiveEmpty")}
                            }
                        }
                        for sdk in active {
                            m3e-list-action {
                                key: "{sdk.name}",
                                onclick: {
                                    let name = sdk.name.clone();
                                    move |_| app.navigation.show_sdk(Some(name.clone()))
                                },
                                {avatar(&sdk.name, 40, None, Some("leading"))}
                                "{sdk.name}"
                                span { "slot": "supporting-text", {sdk.current.clone().unwrap_or_default()} }
                                {icon("chevronRight", false, Some("trailing"))}
                            }
                        }
                    }
                    div { class: "shortcuts",
                        section { class: "shell-card",
                            div { class: "heading",
                                {icon("terminal", false, None)}
                                h2 { class: "type-title-medium-emphasized", {app.t("homeShellTitle")} }
                            }
                            p { class: "type-body-medium muted", {app.t("homeShellBody")} }
                            div { class: "command",
                                code { class: "mono type-body-medium", "{ACTIVATION_COMMAND}" }
                                IconButton {
                                    icon: "copy",
                                    label: app.t("actionCopy"),
                                    onclick: move |_| {
                                        desktop::copy_text(ACTIVATION_COMMAND);
                                        app.notify(app.t("actionCopied"));
                                    },
                                }
                            }
                        }
                        {shortcut(app, "explore", "homeExploreTitle", "homeExploreBody", "chevronRight", Callback::new(move |()| app.navigation.go(Destination::Plugins)))}
                        {shortcut(app, "docs", "homeDocsTitle", "homeDocsBody", "openInNew", Callback::new(|()| desktop::open_url(links::VFOX_DOCS)))}
                    }
                }
            }
        }
    }
}

/// A large button that leads somewhere else.
fn shortcut(
    app: AppContext,
    meaning: &str,
    title: &str,
    body: &str,
    trailing: &str,
    activate: Callback<()>,
) -> Element {
    rsx! {
        button { class: "shortcut", onclick: move |_| activate.call(()),
            span { class: "shape", "aria-hidden": "true",
                m3e-shape { "name": "6-sided-cookie" }
                {icon(meaning, false, None)}
            }
            span { class: "text",
                span { class: "type-title-small-emphasized", {app.t(title)} }
                p { class: "type-body-small muted", {app.t(body)} }
            }
            {icon(trailing, false, None)}
        }
    }
}

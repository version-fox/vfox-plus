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

//! Installed SDKs, in the list-detail canonical layout: wide pages show the
//! list and the selected SDK side by side, and the detail fades through when
//! the selection changes; narrow pages show one of them at a time and move
//! between them along the horizontal shared axis.

use dioxus::prelude::*;

use super::sdk_detail::SdkDetail;
use crate::core::models::InstalledSdk;
use crate::ui::components::parts::{IconButton, StatusMessage, avatar, error_message, icon, loading, page_header};
use crate::ui::context::{AppContext, Destination, use_app};

/// The narrowest page that shows list and detail side by side.
pub const TWO_PANE_WIDTH: f64 = 840.0;

/// What a narrow page shows.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Shown {
    List,
    Detail,
}

/// The SDKs page.
#[component]
pub fn SdksPage() -> Element {
    let app = use_app();
    let model = app.model;
    let mut width = use_signal(|| 0.0_f64);
    let query = use_signal(String::new);
    // On narrow pages: what was shown, and how the last change moved.
    let mut axis = use_hook(|| CopyValue::new((None::<Shown>, "")));
    let onresize = move |event: Event<ResizeData>| {
        event.stop_propagation();
        if let Ok(size) = event.get_content_box_size()
            && (size.width - *width.peek()).abs() > 0.5
        {
            width.set(size.width);
        }
    };
    let reload = Callback::new(move |()| model.installed.load());
    let Some(sdks) = model.installed.value() else {
        let body = match model.installed.error() {
            Some(error) => error_message(app, app.t("sdksLoadError"), &error, Some(reload)),
            None => loading(app, None),
        };
        return rsx! {
            div { class: "page-root sdks-page", onresize, div { class: "pane", {body} } }
        };
    };
    // The selected SDK, if it is installed.
    let selected = app.navigation.selected_sdk.read().clone().filter(|name| sdks.iter().any(|sdk| sdk.name == *name));
    let width = width();
    if width >= TWO_PANE_WIDTH || width == 0.0 {
        axis.set((None, ""));
        return rsx! {
            div { class: "page-root sdks-page", onresize,
                div { class: "layout",
                    div { class: "pane list-pane", {list(app, &sdks, selected.as_deref(), false, query)} }
                    div { class: "pane detail-pane",
                        // Keyed, so that the detail fades through when the selection changes.
                        div { key: "{selected:?}", class: "detail-host changed",
                            if let Some(name) = selected {
                                SdkDetail { key: "{name}", name: name.clone() }
                            } else {
                                StatusMessage { icon: "sdks", shape: "gem", heading: app.t("sdksSelectPrompt") }
                            }
                        }
                    }
                }
            }
        };
    }
    let shown = if selected.is_some() { Shown::Detail } else { Shown::List };
    let (last, mut direction) = *axis.peek();
    if last != Some(shown) {
        // Shared axis, horizontal: forwards into the detail, backwards out of it.
        direction = match (last, shown) {
            (None, _) => "",
            (Some(_), Shown::Detail) => "forward",
            (Some(_), Shown::List) => "backward",
        };
        axis.set((Some(shown), direction));
    }
    rsx! {
        div { class: "page-root sdks-page", onresize,
            div { key: "{selected:?}", class: "single {direction}",
                if let Some(name) = selected {
                    div { class: "pane",
                        SdkDetail { key: "{name}", name: name.clone(), on_back: move |()| app.navigation.show_sdk(None) }
                    }
                } else {
                    div { class: "pane", {list(app, &sdks, None, true, query)} }
                }
            }
        }
    }
}

/// The installed SDKs, filtered by `query`.
fn list(
    app: AppContext,
    sdks: &[InstalledSdk],
    selected: Option<&str>,
    chevrons: bool,
    mut query: Signal<String>,
) -> Element {
    let model = app.model;
    if sdks.is_empty() {
        return rsx! {
            StatusMessage { icon: "empty", shape: "flower", heading: app.t("sdksEmptyTitle"), body: app.t("sdksEmptyBody"),
                m3e-button { "variant": "filled", onclick: move |_| app.navigation.go(Destination::Plugins),
                    {icon("add", false, Some("icon"))}
                    {app.t("navAddPlugin")}
                }
            }
        };
    }
    let needle = query.read().trim().to_lowercase();
    let visible: Vec<InstalledSdk> =
        sdks.iter().filter(|sdk| needle.is_empty() || sdk.name.to_lowercase().contains(&needle)).cloned().collect();
    let actions = rsx! {
        IconButton {
            icon: "refresh",
            label: app.t("actionRefresh"),
            disabled: model.installed.is_loading(),
            onclick: move |_| model.installed.load(),
        }
    };
    let hint = app.t("sdksSearchHint");
    rsx! {
        {page_header(app.t("navSdks"), None, actions)}
        div { class: "search",
            m3e-search-bar { "clearable": "", "clear-label": app.t("actionClose"),
                {icon("search", false, Some("leading"))}
                input {
                    "slot": "input",
                    placeholder: "{hint}",
                    "aria-label": "{hint}",
                    value: "{query}",
                    oninput: move |event| query.set(event.value()),
                }
            }
        }
        if visible.is_empty() {
            StatusMessage { icon: "notFound", shape: "9-sided-cookie", heading: app.tf("sdksNoMatch", &[("query", query.read().trim().into())]) }
        } else {
            div { class: "items",
                m3e-action-list { "variant": "segmented",
                    for sdk in visible {
                        m3e-list-action {
                            key: "{sdk.name}",
                            class: if selected == Some(sdk.name.as_str()) { "selected" } else { "" },
                            "aria-current": if selected == Some(sdk.name.as_str()) { "true" } else { "false" },
                            onclick: {
                                let name = sdk.name.clone();
                                move |_| app.navigation.show_sdk(Some(name.clone()))
                            },
                            {avatar(&sdk.name, 40, None, Some("leading"))}
                            "{sdk.name}"
                            span { "slot": "supporting-text", class: "two-lines",
                                {match &sdk.current {
                                    Some(version) => app.tf("sdksCurrent", &[("version", version.into())]),
                                    None => app.t("sdksNoCurrent"),
                                }}
                                "\n"
                                {app.tf("sdksVersionCount", &[("count", sdk.versions.len().into())])}
                            }
                            if chevrons {
                                {icon("chevronRight", false, Some("trailing"))}
                            }
                        }
                    }
                }
            }
        }
    }
}

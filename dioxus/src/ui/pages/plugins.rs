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

//! The plugin registry: every plugin that `vfox add` can install.

use std::collections::HashSet;

use dioxus::prelude::*;

use crate::core::command::VfoxOperation;
use crate::core::models::AvailablePlugin;
use crate::ui::components::dialogs::{DialogRequest, DialogResult};
use crate::ui::components::parts::{
    IconButton, StatusMessage, avatar, choice_group, error_message, icon, loading, page_header,
};
use crate::ui::context::{AppContext, use_app};
use crate::ui::desktop;

/// Which plugins the page lists.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum PluginFilter {
    All,
    Official,
    Community,
}

/// The repository path of a homepage, such as `version-fox/vfox-nodejs`.
pub fn repository_path(homepage: &str) -> String {
    let Some((_, rest)) = homepage.split_once("://") else { return homepage.to_owned() };
    let path = rest.split_once('/').map_or("", |(_, path)| path);
    let segments: Vec<&str> =
        path.split(['?', '#']).next().unwrap_or_default().split('/').filter(|segment| !segment.is_empty()).collect();
    if segments.is_empty() { homepage.to_owned() } else { segments.join("/") }
}

/// The Plugins page.
#[component]
pub fn PluginsPage() -> Element {
    let app = use_app();
    let model = app.model;
    let mut query = use_signal(String::new);
    let mut filter = use_signal(|| PluginFilter::All);
    use_hook(|| model.registry.ensure_loaded());

    let installed: HashSet<String> = model
        .installed
        .with(|sdks| sdks.map(|sdks| sdks.iter().map(|sdk| sdk.name.clone()).collect()).unwrap_or_default());
    let needle = query.read().trim().to_lowercase();
    let chosen = *filter.read();
    let updating_all = model.tasks.active_task(Some(VfoxOperation::UpdateAllPlugins), None, None).is_some();
    let filters = vec![
        (PluginFilter::All, app.t("pluginsFilterAll"), None),
        (PluginFilter::Official, app.t("pluginsFilterOfficial"), Some("verified")),
        (PluginFilter::Community, app.t("pluginsFilterCommunity"), None),
    ];
    let body = match model.registry.value() {
        None => match model.registry.error() {
            Some(error) => error_message(
                app,
                app.t("pluginsLoadError"),
                &error,
                Some(Callback::new(move |()| model.registry.load())),
            ),
            None => loading(app, None),
        },
        Some(plugins) => {
            let visible: Vec<AvailablePlugin> = plugins
                .into_iter()
                .filter(|plugin| {
                    (chosen == PluginFilter::All || (chosen == PluginFilter::Official) == plugin.official)
                        && (needle.is_empty()
                            || plugin.name.to_lowercase().contains(&needle)
                            || plugin.homepage.contains(&needle))
                })
                .collect();
            if visible.is_empty() {
                rsx! { StatusMessage { icon: "notFound", shape: "9-sided-cookie", heading: app.t("pluginsNoMatch") } }
            } else {
                rsx! {
                    div { class: "grid",
                        for plugin in visible {
                            {card(app, &plugin, installed.contains(&plugin.name))}
                        }
                    }
                }
            }
        }
    };
    let actions = rsx! {
        if updating_all {
            m3e-loading-indicator { "aria-label": app.t("pluginsUpdateAll") }
        } else {
            IconButton {
                icon: "update",
                label: app.t("pluginsUpdateAll"),
                disabled: installed.is_empty(),
                onclick: move |_| {
                    model.run(model.cli().update_all_plugins());
                },
            }
        }
        IconButton { icon: "refresh", label: app.t("actionRefresh"), disabled: model.registry.is_loading(), onclick: move |_| model.registry.load() }
        m3e-button {
            "variant": "tonal",
            onclick: move |_| {
                spawn(async move {
                    if let DialogResult::Plugin { source, alias } = app.dialogs.show(DialogRequest::CustomPlugin).await {
                        model.add_plugin_from_source(&source, alias.as_deref());
                    }
                });
            },
            {icon("add", false, Some("icon"))}
            {app.t("pluginsAddCustom")}
        }
    };
    let hint = app.t("pluginsSearchHint");
    rsx! {
        div { class: "page-root plugins-page",
            div { class: "pane",
                {page_header(app.t("pluginsTitle"), Some(app.t("pluginsSubtitle")), actions)}
                div { class: "filters",
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
                    {choice_group(filters, chosen, Callback::new(move |value| filter.set(value)))}
                }
                {body}
            }
        }
    }
}

fn card(app: AppContext, plugin: &AvailablePlugin, installed: bool) -> Element {
    let model = app.model;
    let adding = model.tasks.active_task(Some(VfoxOperation::AddPlugin), Some(&plugin.name), None).is_some();
    let name = plugin.name.clone();
    let action = if adding {
        rsx! { m3e-loading-indicator { "aria-label": app.t("loading") } }
    } else if installed {
        rsx! {
            m3e-button { "variant": "text", onclick: move |_| app.navigation.show_sdk(Some(name.clone())),
                {icon("check", false, Some("icon"))}
                {app.t("pluginsAdded")}
            }
        }
    } else {
        rsx! {
            m3e-button {
                "variant": "tonal",
                onclick: move |_| {
                    model.run_checked(model.cli().add_plugin(&name, None, None));
                },
                {icon("add", false, Some("icon"))}
                {app.t("pluginsAdd")}
            }
        }
    };
    let official = format!("official-{}", plugin.name);
    let homepage = plugin.homepage.clone();
    rsx! {
        article { key: "{plugin.name}", class: if installed { "card installed" } else { "card" },
            div { class: "identity",
                {avatar(&plugin.name, 44, None, None)}
                div { class: "names",
                    div { class: "name",
                        span { class: "type-title-medium-emphasized ellipsis", "{plugin.name}" }
                        if plugin.official {
                            m3e-icon {
                                id: "{official}",
                                class: "verified",
                                "variant": "rounded",
                                "filled": "",
                                "name": "verified",
                                "aria-label": app.t("pluginsOfficial"),
                            }
                            m3e-tooltip { "for": "{official}", {app.t("pluginsOfficial")} }
                        }
                    }
                    div { class: "source type-body-small ellipsis", {repository_path(&plugin.homepage)} }
                }
            }
            div { class: "bottom",
                IconButton { icon: "openInNew", label: app.t("actionOpenHomepage"), onclick: move |_| desktop::open_url(&homepage) }
                {action}
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn repository_paths() {
        assert_eq!(repository_path("https://github.com/version-fox/vfox-nodejs"), "version-fox/vfox-nodejs");
        assert_eq!(repository_path("https://github.com/a/b/?tab=readme#x"), "a/b");
        assert_eq!(repository_path("https://example.com"), "https://example.com");
        assert_eq!(repository_path("not a url"), "not a url");
    }
}

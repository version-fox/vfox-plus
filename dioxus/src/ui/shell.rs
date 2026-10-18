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

//! The adaptive frame around the pages: a navigation bar at the bottom of
//! compact windows, the expressive navigation rail at the start of wider
//! ones. Switching destinations fades through.

use std::time::Duration;

use dioxus::prelude::*;

use super::bridge;
use super::components::parts::icon;
use super::context::{AppContext, Destination, flag, use_app};
use super::icons::symbol;
use super::motion::{EXPRESSIVE, settle_duration};
use super::pages::home::HomePage;
use super::pages::plugins::PluginsPage;
use super::pages::sdks::SdksPage;
use super::pages::settings::SettingsPage;
use super::pages::tasks::TasksPage;
use super::pages::unavailable::UnavailablePage;
use crate::state::controllers::VfoxAvailability;

/// Material 3 window size classes: compact windows are narrower than 600 dp.
const COMPACT_QUERY: &str = "(max-width: 599.98px)";

/// What the stage shows.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum Page {
    Destination(Destination),
    /// Shown instead of the pages, except the settings, while vfox is not ready.
    Unavailable,
}

/// The page shown for the destination; Settings stay reachable without vfox.
fn page_for(app: AppContext) -> Page {
    let destination = *app.navigation.destination.read();
    if app.model.vfox.availability() != VfoxAvailability::Ready && destination != Destination::Settings {
        Page::Unavailable
    } else {
        Page::Destination(destination)
    }
}

/// The page that is shown and the one that fades out.
#[derive(Clone, Copy, Debug, PartialEq)]
struct Transition {
    shown: Page,
    leaving: Option<Page>,
    generation: u32,
}

/// The navigation and the pages.
#[component]
pub fn Shell() -> Element {
    let app = use_app();
    let mut compact = use_signal(|| false);
    use_hook(|| spawn(bridge::watch_media(COMPACT_QUERY, move |matches| compact.set(matches))));
    let page = use_memo(move || page_for(app));
    let mut transition = use_signal(|| Transition { shown: *page.peek(), leaving: None, generation: 0 });
    let reduced_motion = use_context::<ReducedMotion>().0;
    use_effect(move || {
        let next = page();
        let current = *transition.peek();
        if current.shown == next {
            return;
        }
        // A transition that still runs ends at once.
        let generation = current.generation + 1;
        transition.set(Transition { shown: next, leaving: Some(current.shown), generation });
        let duration = if *reduced_motion.peek() { 0 } else { settle_duration(EXPRESSIVE.slow_effects) };
        spawn(async move {
            tokio::time::sleep(Duration::from_millis(u64::from(duration))).await;
            if transition.peek().generation == generation {
                transition.write().leaving = None;
            }
        });
    });

    let Transition { shown, leaving, .. } = transition();
    // Keyed, so that the element of the old page fades out as it is.
    let pages: Vec<Page> = leaving.filter(|leaving| *leaving != shown).into_iter().chain([shown]).collect();
    let animating = pages.len() > 1;
    let stage = rsx! {
        div { class: "stage",
            for key in pages {
                div {
                    key: "{key:?}",
                    class: if key == shown && animating {
                        "page entering"
                    } else if key == shown {
                        "page"
                    } else {
                        "page leaving"
                    },
                    "aria-hidden": if key == shown { "false" } else { "true" },
                    "inert": flag(key != shown),
                    {page_view(key)}
                }
            }
        }
    };
    if compact() {
        rsx! {
            div { class: "shell compact",
                {stage}
                m3e-nav-bar { {items(app)} }
            }
        }
    } else {
        rsx! {
            div { class: "shell",
                Rail {}
                {stage}
            }
        }
    }
}

fn page_view(page: Page) -> Element {
    match page {
        Page::Destination(Destination::Home) => rsx! { HomePage {} },
        Page::Destination(Destination::Sdks) => rsx! { SdksPage {} },
        Page::Destination(Destination::Plugins) => rsx! { PluginsPage {} },
        Page::Destination(Destination::Tasks) => rsx! { TasksPage {} },
        Page::Destination(Destination::Settings) => rsx! { SettingsPage {} },
        Page::Unavailable => rsx! { UnavailablePage {} },
    }
}

/// Whether the platform asks for reduced motion.
#[derive(Clone, Copy, PartialEq)]
pub struct ReducedMotion(pub Signal<bool>);

/// The navigation rail of wider windows, with the button that expands it and
/// the button that adds plugins.
#[component]
fn Rail() -> Element {
    let app = use_app();
    let expanded = app.model.settings.settings.read().rail_expanded;
    let ready = app.model.vfox.availability() == VfoxAvailability::Ready;
    let add = app.t("navAddPlugin");
    rsx! {
        m3e-nav-rail { "mode": if expanded { "expanded" } else { "compact" },
            m3e-icon-button {
                "toggle": "",
                "data-controlled": "",
                "selected": expanded,
                "aria-label": if expanded { app.t("navCollapse") } else { app.t("navExpand") },
                onclick: move |_| app.model.update_settings(|settings| crate::core::settings::AppSettings { rail_expanded: !expanded, ..settings }),
                m3e-icon { "variant": "rounded", "name": symbol("menu") }
                m3e-icon { "variant": "rounded", "slot": "selected", "name": symbol("menuOpen") }
            }
            if ready {
                m3e-fab { "size": "small", "aria-label": "{add}", onclick: move |_| app.navigation.go(Destination::Plugins),
                    {icon("add", false, None)}
                    span { "slot": "label", "{add}" }
                }
            }
            {items(app)}
        }
    }
}

/// The destinations, with the number of active tasks on the tasks.
fn items(app: AppContext) -> Element {
    let active = app.model.tasks.active_count();
    let selected = *app.navigation.destination.read();
    rsx! {
        for destination in Destination::ALL {
            m3e-nav-item {
                key: "{destination:?}",
                "selected": selected == destination,
                "aria-label": app.t(destination.label()),
                onclick: move |_| app.navigation.go(destination),
                span { class: "badge-host", "slot": "icon",
                    {icon(destination.icon(), false, None)}
                    if destination == Destination::Tasks && active > 0 {
                        span { class: "badge", "{active}" }
                    }
                }
                span { class: "badge-host", "slot": "selected-icon",
                    {icon(destination.icon(), true, None)}
                    if destination == Destination::Tasks && active > 0 {
                        span { class: "badge", "{active}" }
                    }
                }
                {app.t(destination.label())}
            }
        }
    }
}

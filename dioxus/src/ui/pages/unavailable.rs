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

//! Shown instead of the pages while vfox is missing or broken.

use dioxus::prelude::*;

use crate::core::settings::AppSettings;
use crate::state::controllers::VfoxAvailability;
use crate::ui::components::parts::{StatusMessage, icon, loading};
use crate::ui::context::{links, use_app};
use crate::ui::desktop::{self, Choose};

/// The page that helps to install or locate vfox.
#[component]
pub fn UnavailablePage() -> Element {
    let app = use_app();
    let model = app.model;
    let state = model.vfox.state.read().clone();
    let body = match state.availability {
        VfoxAvailability::Unknown | VfoxAvailability::Checking | VfoxAvailability::Ready => {
            loading(app, Some(app.t("vfoxCheckingTitle")))
        }
        availability => {
            let missing = availability == VfoxAvailability::NotFound;
            let body = if missing {
                app.t("vfoxMissingBody")
            } else {
                app.tf(
                    "vfoxBrokenBody",
                    &[("path", state.executable.clone().unwrap_or_else(|| "vfox".to_owned()).into())],
                )
            };
            let (meaning, shape, heading) = if missing {
                ("notFound", "4-leaf-clover", app.t("vfoxMissingTitle"))
            } else {
                ("warning", "soft-burst", app.t("vfoxBrokenTitle"))
            };
            rsx! {
                StatusMessage {
                    icon: meaning,
                    shape,
                    error: !missing,
                    heading,
                    body,
                    m3e-button { "variant": "filled", onclick: move |_| desktop::open_url(links::VFOX_INSTALL),
                        {icon("openInNew", false, Some("icon"))}
                        {app.t("vfoxInstallGuide")}
                    }
                    m3e-button {
                        "variant": "tonal",
                        onclick: move |_| {
                            spawn(async move {
                                let title = app.t("vfoxLocate");
                                if let Some(path) = desktop::choose(Choose::File, &title, Some("/usr/local/bin"), None).await {
                                    model.update_settings(|settings| AppSettings { vfox_executable: Some(path), ..settings });
                                }
                            });
                        },
                        {icon("folder", false, Some("icon"))}
                        {app.t("vfoxLocate")}
                    }
                    m3e-button { "variant": "outlined", onclick: move |_| model.spawn(model.apply_settings()),
                        {icon("retry", false, Some("icon"))}
                        {app.t("actionRetry")}
                    }
                }
            }
        }
    };
    rsx! {
        div { class: "page-root", div { class: "pane", {body} } }
    }
}

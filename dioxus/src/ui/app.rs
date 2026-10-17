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

//! The root component: the application state, the theme, the language, the
//! shell, the dialogs and the snack bars that report finished tasks.

use std::collections::BTreeMap;
use std::rc::Rc;

use dioxus::desktop::tao::event::{Event, WindowEvent};
use dioxus::desktop::use_wry_event_handler;
use dioxus::prelude::*;

use super::bridge;
use super::components::dialogs::{DialogHost, Dialogs};
use super::context::{AppContext, Destination, Navigation};
use super::desktop;
use super::motion::{EXPRESSIVE, motion_tokens};
use super::shell::{ReducedMotion, Shell};
use super::theme::color_tokens;
use crate::core::i18n::{Localizations, resolve_language, system_locales};
use crate::core::locator::VfoxLocator;
use crate::core::process::TokioProcessRunner;
use crate::core::settings::{JsonFileSettingsStore, ThemePreference, settings_path};
use crate::state::model::AppModel;
use crate::state::tasks::{TaskStatus, VfoxTask};

/// The application.
#[component]
pub fn VfoxPlusApp() -> Element {
    let environment = use_hook(|| std::env::vars().collect::<BTreeMap<String, String>>());
    let model = use_hook(|| {
        let model = AppModel::new(
            Rc::new(TokioProcessRunner),
            Rc::new(JsonFileSettingsStore::new(settings_path(&environment))),
            environment.clone(),
            VfoxLocator::new(environment.clone()),
        );
        model.start();
        model
    });
    let locales = use_hook(|| system_locales(&environment));
    let language = use_memo(move || resolve_language(model.settings.settings.read().language, &locales));
    let l10n = use_memo(move || Localizations::for_language(language()));

    // The appearance of the desktop: the accent colour and the colour scheme
    // of the XDG desktop portal, and the media queries of the web view.
    let mut accent = use_signal(|| None::<u32>);
    let mut portal_dark = use_signal(|| None::<bool>);
    let mut media_dark = use_signal(|| false);
    let mut reduced_motion = use_signal(|| false);
    use_hook(|| {
        spawn(async move { accent.set(desktop::accent_color().await) });
        spawn(async move { portal_dark.set(desktop::prefers_dark().await) });
        spawn(bridge::watch_media("(prefers-color-scheme: dark)", move |matches| media_dark.set(matches)));
        spawn(bridge::watch_media("(prefers-reduced-motion: reduce)", move |matches| reduced_motion.set(matches)));
    });
    let dark = use_memo(move || match model.settings.settings.read().theme {
        ThemePreference::Dark => true,
        ThemePreference::Light => false,
        ThemePreference::System => portal_dark().unwrap_or_else(&*media_dark),
    });
    use_effect(move || {
        let settings = model.settings.settings.read();
        let seed =
            if settings.use_system_color { accent().unwrap_or(settings.seed_color) } else { settings.seed_color };
        let mut tokens: Vec<(String, String)> = color_tokens(seed, dark())
            .into_iter()
            .map(|(name, value)| (format!("--md-sys-color-{name}"), value))
            .collect();
        tokens.extend(motion_tokens(&EXPRESSIVE, reduced_motion()));
        let l10n = l10n.read();
        bridge::apply_theme(&tokens, dark(), l10n.language_tag(), &l10n.t("appTitle", &[]));
    });

    let navigation = use_hook(Navigation::new);
    let dialogs = use_hook(Dialogs::new);
    let app = use_context_provider(|| AppContext { model, navigation, l10n, dark, dialogs });
    use_context_provider(|| ReducedMotion(reduced_motion));
    use_hook(|| {
        model.tasks.on_finished(move |task| announce(app, task));
    });
    // Running vfox processes end with the application.
    use_wry_event_handler(move |event, _| {
        if let Event::WindowEvent { event: WindowEvent::CloseRequested | WindowEvent::Destroyed, .. } = event {
            model.dispose();
        }
    });
    #[cfg(feature = "e2e")]
    use_hook(|| super::e2e::start(app));

    // The page puts the application inside m3e-theme (see style::INDEX).
    rsx! {
        div { class: "app", Shell {} }
        DialogHost {}
    }
}

/// Tells the user how a task ended, unless they are watching the task list.
fn announce(app: AppContext, task: &VfoxTask) {
    let state = task.state.peek();
    if state.status == TaskStatus::Cancelled {
        return;
    }
    let title = app.task_title(&task.command);
    if state.status == TaskStatus::Failed {
        // Failures stay until dismissed, so that they can be read in full.
        let message = format!(
            "{}\n{}",
            app.tf("taskFailed", &[("title", title.into())]),
            state.error_message.clone().unwrap_or_default()
        )
        .trim()
        .to_owned();
        let (action, close) = (app.t("taskView"), app.t("actionClose"));
        spawn(async move {
            if bridge::snackbar(&message, Some(&action), true, &close).await {
                app.navigation.go(Destination::Tasks);
            }
        });
    } else if *app.navigation.destination.peek() != Destination::Tasks {
        app.notify(app.tf("taskSucceeded", &[("title", title.into())]));
    }
}

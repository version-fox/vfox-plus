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

//! Settings of the application and of vfox.

use std::collections::HashMap;

use dioxus::prelude::*;

use crate::core::command::VfoxError;
use crate::core::settings::{
    AppSettings, LanguagePreference, ThemePreference, format_environment_lines, parse_environment_lines,
};
use crate::state::controllers::{VfoxAvailability, VfoxConfigController};
use crate::ui::bridge;
use crate::ui::components::dialogs::DialogRequest;
use crate::ui::components::parts::{IconButton, avatar, choice_group, error_message, icon, loading, page_header};
use crate::ui::context::{APP_VERSION, AppContext, links, use_app};
use crate::ui::desktop::{self, Choose};
use crate::ui::theme::{SEED_COLORS, hex, primary_of};

/// The configuration keys of vfox that get a dedicated editor.
pub mod config_keys {
    /// Whether vfox uses the proxy.
    pub const PROXY_ENABLE: &str = "proxy.enable";
    /// The proxy.
    pub const PROXY_URL: &str = "proxy.url";
    /// The plugin registry.
    pub const REGISTRY: &str = "registry.address";
    /// Where the SDKs are installed.
    pub const STORAGE: &str = "storage.sdkPath";
    /// Whether legacy version files such as `.nvmrc` are read.
    pub const LEGACY_ENABLE: &str = "legacyVersionFile.enable";
    /// How legacy version files are resolved.
    pub const LEGACY_STRATEGY: &str = "legacyVersionFile.strategy";
    /// How long the available versions are cached.
    pub const CACHE: &str = "cache.availableHookDuration";
    /// The timeout of the requests of plugins.
    pub const HTTP_TIMEOUT: &str = "plugin.http.timeout";
    /// The timeout of downloads.
    pub const DOWNLOAD_TIMEOUT: &str = "plugin.http.downloadTimeout";
    /// Whether vfox adds its files to `.gitignore`.
    pub const GITIGNORE: &str = "gitignore.enable";

    /// Every key with a dedicated editor.
    pub const ALL: [&str; 10] = [
        PROXY_ENABLE,
        PROXY_URL,
        REGISTRY,
        STORAGE,
        LEGACY_ENABLE,
        LEGACY_STRATEGY,
        CACHE,
        HTTP_TIMEOUT,
        DOWNLOAD_TIMEOUT,
        GITIGNORE,
    ];
}

/// The licenses of everything the application carries: the npm packages and
/// the font (`components/build.mjs`) and the Rust crates
/// (`shared/tools/cargo_licenses.py`).
pub fn license_text() -> String {
    const NPM: &str = include_str!("../../../assets/npm-licenses.txt");
    const CRATES: &str = include_str!("../../../assets/crate-licenses.txt");
    format!("{NPM}\n{}\n\n{CRATES}", "-".repeat(72))
}

/// The values of the configuration fields that the user is editing.
#[derive(Clone, Copy, PartialEq)]
struct Drafts {
    /// Typed and not saved yet, by key.
    typed: Signal<HashMap<String, String>>,
    /// Being saved, by key, so that a value is not saved twice.
    saving: Signal<HashMap<String, String>>,
    /// Counts the saves, which show the stored values again afterwards.
    revision: Signal<u32>,
}

/// The Settings page.
#[component]
pub fn SettingsPage() -> Element {
    let app = use_app();
    let model = app.model;
    use_hook(|| {
        if model.vfox.availability() == VfoxAvailability::Ready {
            model.vfox_config.entries.load();
        }
    });
    let drafts =
        Drafts { typed: use_signal(HashMap::new), saving: use_signal(HashMap::new), revision: use_signal(|| 0) };
    let environment_text = use_signal(|| None::<String>);
    let environment_error = use_signal(String::new);
    rsx! {
        div { class: "page-root settings-page",
            div { class: "pane",
                {page_header(app.t("navSettings"), None, rsx! {})}
                div { class: "page-content",
                    {appearance(app)}
                    {vfox_section(app, environment_text, environment_error)}
                    {configuration(app, drafts)}
                    {about(app)}
                }
            }
        }
    }
}

fn group_header(title: String) -> Element {
    rsx! {
        h2 { class: "group-header", role: "heading", "aria-level": "2", "{title}" }
    }
}

fn update(app: AppContext, change: impl FnOnce(AppSettings) -> AppSettings) {
    app.model.update_settings(change);
}

fn appearance(app: AppContext) -> Element {
    let settings = app.model.settings.get();
    let dark = (app.dark)();
    let themes = vec![
        (ThemePreference::System, app.t("themeSystem"), Some("systemMode")),
        (ThemePreference::Light, app.t("themeLight"), Some("lightMode")),
        (ThemePreference::Dark, app.t("themeDark"), Some("darkMode")),
    ];
    // Language names are always shown in their own language.
    let languages = vec![
        (LanguagePreference::System, app.t("languageSystem"), None),
        (LanguagePreference::English, "English".to_owned(), None),
        (LanguagePreference::SimplifiedChinese, "简体中文".to_owned(), None),
    ];
    let system_color = app.t("settingsSystemColor");
    rsx! {
        {group_header(app.t("settingsAppearance"))}
        div { class: "group",
            div { class: "row-item",
                {icon("darkMode", false, None)}
                div { class: "text", div { class: "title type-body-large", {app.t("settingsTheme")} } }
                div { class: "trailing",
                    {choice_group(themes, settings.theme, Callback::new(move |theme| update(app, |s| AppSettings { theme, ..s })))}
                }
            }
            div {
                class: "row-item clickable",
                onclick: move |_| update(app, |s| AppSettings { use_system_color: !s.use_system_color, ..s }),
                {icon("palette", false, None)}
                div { class: "text", div { class: "title type-body-large", "{system_color}" } }
                div { class: "trailing",
                    m3e-switch { "data-controlled": "", "checked": settings.use_system_color, "aria-label": "{system_color}" }
                }
            }
            div { class: "row-item",
                span { class: "icon-space" }
                div { class: "text",
                    div { class: "title type-body-large", {app.t("settingsColor")} }
                    div { class: "swatches",
                        for (index, seed) in SEED_COLORS.into_iter().enumerate() {
                            {swatch(app, &settings, index, seed, dark)}
                        }
                    }
                }
            }
            div { class: "row-item",
                {icon("language", false, None)}
                div { class: "text", div { class: "title type-body-large", {app.t("settingsLanguage")} } }
                div { class: "trailing",
                    {choice_group(languages, settings.language, Callback::new(move |language| update(app, |s| AppSettings { language, ..s })))}
                }
            }
        }
    }
}

fn swatch(app: AppContext, settings: &AppSettings, index: usize, seed: u32, dark: bool) -> Element {
    let selected = !settings.use_system_color && settings.seed_color == seed;
    let label = app.tf("settingsColorOption", &[("index", (index + 1).into())]);
    rsx! {
        button {
            key: "{seed}",
            class: "swatch",
            "aria-label": "{label}",
            title: "{label}",
            "aria-pressed": if selected { "true" } else { "false" },
            "data-seed": hex(seed),
            onclick: move |_| update(app, |s| AppSettings { seed_color: seed, use_system_color: false, ..s }),
            m3e-shape {
                "name": if selected { "flower" } else { "circle" },
                style: "--m3e-shape-container-color: {primary_of(seed, dark)}",
            }
            m3e-icon { "variant": "rounded", "name": "check", style: if dark { "color: #000" } else { "color: #fff" } }
        }
    }
}

fn vfox_section(
    app: AppContext,
    mut environment_text: Signal<Option<String>>,
    mut environment_error: Signal<String>,
) -> Element {
    let model = app.model;
    let vfox = model.vfox.state.read().clone();
    let source = match (&vfox.executable, vfox.is_custom_executable) {
        (None, _) => app.t("settingsExecutableMissing"),
        (Some(_), true) => app.t("settingsExecutableCustom"),
        (Some(_), false) => app.t("settingsExecutableDetected"),
    };
    let mut details = vec![vfox.executable.clone().unwrap_or_else(|| "vfox".to_owned()), source];
    if let Some(version) = &vfox.version {
        details.push(app.tf("settingsVfoxVersion", &[("version", version.into())]));
    }
    let details = details.join(" · ");
    let stored = format_environment_lines(&model.settings.settings.read().extra_environment);
    let text = environment_text.read().clone().unwrap_or(stored);
    let error = environment_error.read().clone();
    let save = move |_| {
        let text = environment_text
            .peek()
            .clone()
            .unwrap_or_else(|| format_environment_lines(&model.settings.get().extra_environment));
        let parsed = parse_environment_lines(&text);
        if parsed.invalid_lines.is_empty() {
            environment_error.set(String::new());
            update(app, |s| AppSettings { extra_environment: parsed.variables, ..s });
            app.notify(app.t("settingsSaved"));
        } else {
            let lines = parsed.invalid_lines.iter().map(ToString::to_string).collect::<Vec<_>>().join(", ");
            environment_error.set(app.tf(
                "settingsEnvironmentInvalid",
                &[("count", parsed.invalid_lines.len().into()), ("lines", lines.into())],
            ));
        }
    };
    let choose_executable = move |_| {
        spawn(async move {
            let title = app.t("vfoxLocate");
            if let Some(path) = desktop::choose(Choose::File, &title, Some("/usr/local/bin"), None).await {
                update(app, |s| AppSettings { vfox_executable: Some(path), ..s });
            }
        });
    };
    rsx! {
        {group_header(app.t("settingsVfox"))}
        div { class: "group",
            div { class: "row-item",
                {icon("terminal", false, None)}
                div { class: "text",
                    div { class: "title type-body-large", {app.t("settingsExecutable")} }
                    div { class: "supporting type-body-medium", "{details}" }
                }
                div { class: "trailing",
                    if vfox.availability == VfoxAvailability::Checking {
                        m3e-loading-indicator { class: "small", "aria-label": app.t("vfoxCheckingTitle") }
                    } else if vfox.is_custom_executable {
                        IconButton {
                            icon: "refresh",
                            label: app.t("settingsExecutableDetect"),
                            onclick: move |_| update(app, |s| AppSettings { vfox_executable: None, ..s }),
                        }
                    }
                    m3e-button { "variant": "tonal", onclick: choose_executable, {app.t("actionBrowse")} }
                }
            }
            div { class: "row-item environment",
                {icon("environment", false, None)}
                div { class: "text",
                    div { class: "title type-body-large", {app.t("settingsEnvironment")} }
                    div { class: "supporting type-body-medium", {app.t("settingsEnvironmentHelp")} }
                }
                div { class: "wide",
                    m3e-form-field { "variant": "outlined", "error": crate::ui::context::flag(!error.is_empty()),
                        textarea {
                            class: "mono",
                            rows: "3",
                            "aria-label": app.t("settingsEnvironment"),
                            placeholder: "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node",
                            value: "{text}",
                            oninput: move |event| environment_text.set(Some(event.value())),
                        }
                        span { "slot": "error", "{error}" }
                    }
                    div { class: "save",
                        m3e-button { "variant": "filled", onclick: save, {app.t("actionSave")} }
                    }
                }
            }
        }
    }
}

/// Reports how saving a vfox setting went.
fn report(app: AppContext, result: Result<(), VfoxError>) {
    match result {
        Ok(()) => app.notify(app.t("settingsSaved")),
        Err(error) => app.notify(error.message().to_owned()),
    }
}

/// Saves `value` for `key`, or resets the key when the value is empty. The
/// save runs in the scope of the model, so that it completes when the page
/// goes.
fn save_config(app: AppContext, drafts: Drafts, key: String, value: String) {
    let config = app.model.vfox_config;
    let stored = config.value_for(&key).unwrap_or_default();
    let mut typed = drafts.typed;
    typed.write().remove(&key);
    if value == stored || drafts.saving.peek().get(&key) == Some(&value) {
        return;
    }
    let mut saving = drafts.saving;
    saving.write().insert(key.clone(), value.clone());
    app.model.spawn(async move {
        // Clearing a field resets the key. If vfox rejects the value, the
        // reload shows the stored value again.
        let result =
            if value.is_empty() { config.reset(key.clone()).await } else { config.set(key.clone(), value).await };
        if let Ok(mut saving) = saving.try_write() {
            saving.remove(&key);
        }
        let mut revision = drafts.revision;
        if let Ok(mut revision) = revision.try_write() {
            *revision += 1;
        }
        report(app, result);
    });
}

fn configuration(app: AppContext, drafts: Drafts) -> Element {
    let model = app.model;
    let config = model.vfox_config;
    let body = if model.vfox.availability() != VfoxAvailability::Ready {
        rsx! {}
    } else if !config.entries.has_value() {
        match config.entries.error() {
            Some(error) => error_message(
                app,
                app.t("settingsVfoxConfigError"),
                &error,
                Some(Callback::new(move |()| config.entries.load())),
            ),
            None => loading(app, None),
        }
    } else {
        let other: Vec<String> = config.entries.with(|entries| {
            entries
                .into_iter()
                .flatten()
                .filter(|entry| !config_keys::ALL.contains(&entry.key.as_str()))
                .map(|entry| entry.key.clone())
                .collect()
        });
        let strategies = vec![
            ("specified", app.t("strategySpecified")),
            ("latest_installed", app.t("strategyLatestInstalled")),
            ("latest_available", app.t("strategyLatestAvailable")),
        ];
        rsx! {
            div { class: "group",
                {config_switch(app, drafts, config_keys::PROXY_ENABLE, app.t("configProxyEnable"), "proxy", None)}
                {config_text(app, drafts, config_keys::PROXY_URL, app.t("configProxyUrl"), None, "http://127.0.0.1:7890".to_owned(), None, false)}
                {config_text(app, drafts, config_keys::REGISTRY, app.t("configRegistry"), Some("registry"), app.t("configRegistryHint"), None, false)}
                {config_text(app, drafts, config_keys::STORAGE, app.t("configStorage"), Some("storage"), app.t("configStorageHint"), None, true)}
            }
            div { class: "group",
                {config_switch(app, drafts, config_keys::LEGACY_ENABLE, app.t("configLegacyEnable"), "legacyFiles", Some(app.t("configLegacyEnableHelp")))}
                {config_choice(app, drafts, config_keys::LEGACY_STRATEGY, app.t("configLegacyStrategy"), strategies)}
                {config_switch(app, drafts, config_keys::GITIGNORE, app.t("configGitignore"), "gitignore", Some(app.t("configGitignoreHelp")))}
            }
            div { class: "group",
                {config_text(app, drafts, config_keys::CACHE, app.t("configCache"), Some("cache"), "12h".to_owned(), Some(app.t("configCacheHelp")), false)}
                {config_text(app, drafts, config_keys::HTTP_TIMEOUT, app.t("configHttpTimeout"), Some("timeout"), "30s".to_owned(), Some(app.t("configTimeoutHelp")), false)}
                {config_text(app, drafts, config_keys::DOWNLOAD_TIMEOUT, app.t("configDownloadTimeout"), None, "30m".to_owned(), Some(app.t("configTimeoutHelp")), false)}
            }
            if !other.is_empty() {
                {group_header(app.t("configOther"))}
                div { class: "group",
                    for key in other {
                        {config_text(app, drafts, &key, key.clone(), None, String::new(), None, false)}
                    }
                }
            }
        }
    };
    rsx! {
        {group_header(app.t("settingsVfoxConfig"))}
        p { class: "help type-body-small", {app.t("settingsVfoxConfigHelp")} }
        {body}
    }
}

fn config_switch(
    app: AppContext,
    drafts: Drafts,
    key: &str,
    title: String,
    meaning: &str,
    help: Option<String>,
) -> Element {
    let config: VfoxConfigController = app.model.vfox_config;
    let value = config.value_for(key).as_deref() == Some("true");
    let saving = config.is_saving(key);
    let key = key.to_owned();
    let toggle = move |_| {
        if !saving {
            save_config(app, drafts, key.clone(), (!value).to_string());
        }
    };
    rsx! {
        div { class: "row-item clickable", onclick: toggle,
            {icon(meaning, false, None)}
            div { class: "text",
                div { class: "title type-body-large", "{title}" }
                if let Some(help) = help {
                    div { class: "supporting type-body-medium", "{help}" }
                }
            }
            div { class: "trailing",
                m3e-switch { "data-controlled": "", "checked": value, "disabled": saving, "aria-label": "{title}" }
            }
        }
    }
}

fn config_choice(
    app: AppContext,
    drafts: Drafts,
    key: &str,
    title: String,
    choices: Vec<(&'static str, String)>,
) -> Element {
    let config = app.model.vfox_config;
    let value = config.value_for(key).unwrap_or_default();
    let id = format!("config-{}", key.replace('.', "-"));
    let revision = *drafts.revision.read();
    let (key, select) = (key.to_owned(), id.clone());
    let stored = value.clone();
    rsx! {
        // A new select after every save shows the stored value.
        div { key: "{key}-{revision}", class: "row-item",
            span { class: "icon-space" }
            div { class: "text", div { class: "title type-body-large", "{title}" } }
            div { class: "trailing",
                m3e-form-field { "variant": "outlined", "hide-subscript": "always",
                    m3e-select {
                        id: "{id}",
                        "aria-label": "{title}",
                        "disabled": config.is_saving(&key),
                        oninput: move |_| {
                            let (key, select, stored) = (key.clone(), select.clone(), stored.clone());
                            spawn(async move {
                                let selection = bridge::value_of(&select).await;
                                if !selection.is_empty() && selection != stored {
                                    save_config(app, drafts, key, selection);
                                }
                            });
                        },
                        for (choice, label) in choices {
                            m3e-option { "value": choice, "selected": choice == value, "{label}" }
                        }
                    }
                }
            }
        }
    }
}

#[allow(clippy::too_many_arguments)]
fn config_text(
    app: AppContext,
    drafts: Drafts,
    key: &str,
    title: String,
    meaning: Option<&str>,
    hint: String,
    help: Option<String>,
    pick_directory: bool,
) -> Element {
    let config = app.model.vfox_config;
    let value = drafts.typed.read().get(key).cloned().or_else(|| config.value_for(key)).unwrap_or_default();
    let id = format!("config-{}", key.replace('.', "-"));
    let key = key.to_owned();
    let mut typed = drafts.typed;
    let (on_input, on_key, on_change, on_browse) = (key.clone(), key.clone(), key.clone(), key.clone());
    rsx! {
        div { class: "row-item",
            if let Some(meaning) = meaning {
                {icon(meaning, false, None)}
            } else {
                span { class: "icon-space" }
            }
            div { class: "text",
                div { class: "title type-body-large", "{title}" }
                if let Some(help) = help {
                    div { class: "supporting type-body-medium", "{help}" }
                }
            }
            div { class: "trailing",
                m3e-form-field { "variant": "outlined", "hide-subscript": "always",
                    input {
                        id: "{id}",
                        "aria-label": "{title}",
                        placeholder: "{hint}",
                        value: "{value}",
                        disabled: config.is_saving(&key),
                        oninput: move |event| {
                            typed.write().insert(on_input.clone(), event.value());
                        },
                        onkeydown: move |event| {
                            if event.key() == Key::Enter {
                                let value = drafts.typed.peek().get(&on_key).cloned().unwrap_or_default();
                                save_config(app, drafts, on_key.clone(), value.trim().to_owned());
                            }
                        },
                        onchange: move |event| save_config(app, drafts, on_change.clone(), event.value().trim().to_owned()),
                    }
                    if pick_directory {
                        IconButton {
                            slot: "suffix",
                            icon: "folder",
                            label: app.t("actionBrowse"),
                            onclick: move |_| {
                                let key = on_browse.clone();
                                let current = config.value_for(&key).filter(|value| !value.is_empty());
                                spawn(async move {
                                    let title = app.t("configStorage");
                                    if let Some(directory) = desktop::choose(Choose::Directory, &title, current.as_deref(), None).await {
                                        save_config(app, drafts, key, directory);
                                    }
                                });
                            },
                        }
                    }
                }
            }
        }
    }
}

fn about(app: AppContext) -> Element {
    let link = |meaning: &'static str, title: String, url: &'static str| {
        rsx! {
            div {
                class: "row-item clickable",
                role: "link",
                tabindex: "0",
                onclick: move |_| desktop::open_url(url),
                onkeydown: move |event| {
                    if event.key() == Key::Enter {
                        desktop::open_url(url);
                    }
                },
                {icon(meaning, false, None)}
                div { class: "text",
                    div { class: "title type-body-large", "{title}" }
                    div { class: "supporting type-body-medium", "{url}" }
                }
                div { class: "trailing", {icon("openInNew", false, None)} }
            }
        }
    };
    rsx! {
        {group_header(app.t("settingsAbout"))}
        div { class: "group",
            div { class: "row-item",
                {avatar("vfox+", 40, Some("sdks"), None)}
                div { class: "text",
                    div { class: "title type-body-large", {app.t("appTitle")} }
                    div { class: "supporting type-body-medium",
                        {format!("{} · {} · Dioxus", app.tf("aboutVersion", &[("version", APP_VERSION.into())]), app.t("aboutLicense"))}
                    }
                }
            }
            {link("sourceCode", app.t("aboutSource"), links::REPOSITORY)}
            {link("docs", app.t("aboutDocs"), links::VFOX_DOCS)}
            div {
                class: "row-item clickable",
                role: "button",
                tabindex: "0",
                onclick: move |_| {
                    let title = app.t("aboutLicenses");
                    spawn(async move {
                        app.dialogs.show(DialogRequest::Text { title, text: license_text() }).await;
                    });
                },
                {icon("license", false, None)}
                div { class: "text", div { class: "title type-body-large", {app.t("aboutLicenses")} } }
                div { class: "trailing", {icon("chevronRight", false, None)} }
            }
        }
    }
}

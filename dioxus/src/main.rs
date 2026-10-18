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

//! Starts vfox+ in a window.

use std::collections::BTreeMap;

use dioxus::desktop::{Config, LogicalSize, WindowBuilder};
use vfox_plus::core::settings::{JsonFileSettingsStore, SettingsStore, ThemePreference, settings_path};
use vfox_plus::ui::app::VfoxPlusApp;
use vfox_plus::ui::style;
use vfox_plus::ui::theme::color_tokens;

/// The application ID, which names the window class, so that desktops match
/// the window with the desktop entry `dev.vfox.VfoxPlus.desktop`.
const APP_ID: &str = "dev.vfox.VfoxPlus";

fn main() {
    glib::set_prgname(Some(APP_ID));
    glib::set_application_name("vfox+");
    // The first frame already has the colours of the stored settings; the
    // application corrects them once it knows the appearance of the desktop.
    let environment: BTreeMap<String, String> = std::env::vars().collect();
    let settings = JsonFileSettingsStore::new(settings_path(&environment)).load();
    let colors = color_tokens(settings.seed_color, settings.theme == ThemePreference::Dark);
    let surface = colors.iter().find(|(name, _)| *name == "surface").map_or("#ffffff", |(_, value)| value.as_str());
    let channel = |index: usize| u8::from_str_radix(&surface[index..index + 2], 16).unwrap_or(255);
    let config = Config::new()
        .with_window(
            WindowBuilder::new()
                .with_title("vfox+")
                .with_inner_size(LogicalSize::new(1280.0, 820.0))
                .with_min_inner_size(LogicalSize::new(360.0, 480.0)),
        )
        .with_menu(None)
        .with_disable_context_menu(true)
        .with_background_color((channel(1), channel(3), channel(5), 255))
        .with_custom_index(style::INDEX.to_owned())
        .with_custom_head(style::head(&colors));
    dioxus::LaunchBuilder::desktop().with_cfg(config).launch(VfoxPlusApp);
}

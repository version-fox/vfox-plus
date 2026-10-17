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

//! Small building blocks shared by the pages. Parts without state are
//! functions, so that long lists do not create a component per icon.

use std::sync::atomic::{AtomicU32, Ordering};

use dioxus::prelude::*;

use crate::core::command::VfoxError;
use crate::ui::context::{AppContext, flag};
use crate::ui::icons::symbol;

/// The shapes of the Material 3 Expressive shape library that look good as
/// decorative containers of small icons and initials, as named by `@m3e/web`.
pub const DECORATIVE_SHAPES: [&str; 12] = [
    "9-sided-cookie",
    "4-leaf-clover",
    "sunny",
    "soft-burst",
    "6-sided-cookie",
    "gem",
    "pentagon",
    "flower",
    "puffy-diamond",
    "4-sided-cookie",
    "arch",
    "8-leaf-clover",
];

/// A deterministic, platform-independent string hash (FNV-1a, 32 bit) of the
/// UTF-16 code units of `text`, the same as in the other implementations.
pub fn stable_hash(text: &str) -> u32 {
    text.encode_utf16().fold(0x811c_9dc5_u32, |hash, unit| (hash ^ u32::from(unit)).wrapping_mul(0x0100_0193))
}

/// An identifier that is unique in the document, for elements that refer to
/// each other, such as a menu and its trigger. It is valid in CSS selectors.
pub fn unique_id(prefix: &str) -> String {
    static NEXT: AtomicU32 = AtomicU32::new(1);
    format!("{prefix}-{}", NEXT.fetch_add(1, Ordering::Relaxed))
}

/// An `m3e-icon` for the icon `meaning`, filled when `filled` is set.
pub fn icon(meaning: &str, filled: bool, slot: Option<&'static str>) -> Element {
    rsx! {
        m3e-icon { "variant": "rounded", "name": symbol(meaning), "filled": flag(filled), "slot": slot }
    }
}

/// A plugin avatar: an initial on a decorative Material 3 Expressive shape.
///
/// The shape and the colour are derived from the name, so every plugin keeps
/// its look. Hovering spins the shape by 30° on the fast spatial spring.
pub fn avatar(name: &str, size: u32, meaning: Option<&str>, slot: Option<&'static str>) -> Element {
    let hash = stable_hash(name) as usize;
    // Selected list items use the secondary container colour. Avatars avoid it,
    // and the primary container too, which some seeds (the vfox orange among
    // them) make just as light, so that they stay visible on selected items.
    let (background, foreground) = [
        ("primary-fixed-dim", "on-primary-fixed"),
        ("secondary", "on-secondary"),
        ("tertiary-container", "on-tertiary-container"),
    ][hash % 3];
    let shape = DECORATIVE_SHAPES[hash % DECORATIVE_SHAPES.len()];
    let initial: String = name.chars().next().map_or_else(|| "?".to_owned(), |first| first.to_uppercase().collect());
    rsx! {
        span {
            class: "avatar",
            "slot": slot,
            style: "--size: {size}px; color: var(--md-sys-color-{foreground})",
            "aria-hidden": "true",
            m3e-shape { "name": shape, style: "--m3e-shape-container-color: var(--md-sys-color-{background})" }
            if let Some(meaning) = meaning {
                {icon(meaning, false, None)}
            } else {
                // Drawn by the stylesheet, so that it is not part of the text of
                // the list item that holds the avatar.
                span { class: "initial", "data-initial": "{initial}" }
            }
        }
    }
}

/// An icon button with a tooltip, which also serves as its accessible name.
/// Every icon-only button of the application is one. Children go into the
/// button, such as the trigger of a menu.
#[component]
pub fn IconButton(
    icon: &'static str,
    label: String,
    #[props(default)] variant: Option<&'static str>,
    #[props(default)] disabled: bool,
    #[props(default)] filled: bool,
    #[props(default)] class: Option<&'static str>,
    #[props(default)] slot: Option<&'static str>,
    #[props(default)] onclick: Option<EventHandler<MouseEvent>>,
    children: Element,
) -> Element {
    let id = use_hook(|| unique_id("icon-button"));
    let class = format!("icon-button {}", class.unwrap_or_default());
    rsx! {
        span { class, "slot": slot,
            m3e-icon-button {
                id: "{id}",
                "variant": variant.unwrap_or("standard"),
                "aria-label": "{label}",
                "disabled": disabled,
                onclick: move |event| {
                    if let Some(handler) = onclick {
                        handler.call(event);
                    }
                },
                {self::icon(icon, filled, None)}
                {children}
            }
            m3e-tooltip { "for": "{id}", "show-delay": "500", "{label}" }
        }
    }
}

/// A friendly placeholder for empty, unavailable or failed content, with an
/// expressive shape behind its icon. Buttons go into the children.
#[component]
pub fn StatusMessage(
    icon: &'static str,
    shape: &'static str,
    heading: String,
    #[props(default)] body: Option<String>,
    #[props(default)] error: bool,
    children: Element,
) -> Element {
    rsx! {
        div { class: if error { "status-message error" } else { "status-message" },
            div { class: "status", role: "status",
                div { class: "shape", "aria-hidden": "true",
                    m3e-shape { "name": shape }
                    {self::icon(icon, true, None)}
                }
                h2 { class: "type-title-large-emphasized", "{heading}" }
                if let Some(body) = body.filter(|body| !body.is_empty()) {
                    p { class: "type-body-medium", "{body}" }
                }
                div { class: "status-actions", {children} }
            }
        }
    }
}

/// A centred loading indicator with an optional label.
pub fn loading(app: AppContext, label: Option<String>) -> Element {
    let name = label.clone().unwrap_or_else(|| app.t("loading"));
    rsx! {
        div { class: "loading-message",
            div { class: "loading", role: "progressbar", "aria-label": "{name}",
                m3e-loading-indicator { "variant": "contained" }
                if let Some(label) = label {
                    span { class: "type-body-medium", "{label}" }
                }
            }
        }
    }
}

/// A status message for an error, with the error text and a retry button.
pub fn error_message(app: AppContext, heading: String, error: &VfoxError, retry: Option<Callback<()>>) -> Element {
    rsx! {
        StatusMessage { icon: "error", shape: "soft-burst", error: true, heading, body: error.message().to_owned(),
            if let Some(retry) = retry {
                m3e-button { "variant": "tonal", onclick: move |_| retry.call(()),
                    {icon("retry", false, Some("icon"))}
                    {app.t("actionRetry")}
                }
            }
        }
    }
}

/// One choice of a [`choice_group`]: its value, label and optional icon.
pub type Choice<T> = (T, String, Option<&'static str>);

/// A Material 3 Expressive connected button group with single selection. The
/// buttons are controlled: the selected one cannot be unselected by clicking
/// it again.
pub fn choice_group<T: Copy + PartialEq + 'static>(
    choices: Vec<Choice<T>>,
    selected: T,
    choose: Callback<T>,
) -> Element {
    rsx! {
        m3e-button-group { "variant": "connected",
            for (value, label, meaning) in choices {
                m3e-button {
                    "variant": "tonal",
                    "toggle": "",
                    "data-controlled": "",
                    "selected": value == selected,
                    onclick: move |_| {
                        if value != selected {
                            choose.call(value);
                        }
                    },
                    if let Some(meaning) = meaning {
                        {icon(meaning, false, Some("icon"))}
                    }
                    "{label}"
                }
            }
        }
    }
}

/// The header of a page: its title, an optional subtitle and actions.
pub fn page_header(title: String, subtitle: Option<String>, actions: Element) -> Element {
    rsx! {
        header { class: "page-header",
            div { class: "titles",
                h1 { class: "type-headline-medium-emphasized", role: "heading", "aria-level": "1", "{title}" }
                if let Some(subtitle) = subtitle {
                    p { class: "subtitle type-body-medium", "{subtitle}" }
                }
            }
            div { class: "actions", {actions} }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_hash_is_the_one_of_the_other_implementations() {
        // FNV-1a of the UTF-16 code units, as in web/src/ui/components/parts.ts.
        assert_eq!(stable_hash(""), 0x811c_9dc5);
        assert_eq!(stable_hash("a"), 0xe40c_292c);
        assert_eq!(stable_hash("nodejs") % 3, 0);
        assert_ne!(unique_id("x"), unique_id("x"));
    }
}

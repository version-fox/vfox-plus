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

//! The document: the stylesheet, the icon font and the web components, all
//! built into the executable, so that the application needs no files at run
//! time.

use std::fmt::Write as _;

use base64::Engine;

/// The page. Dioxus renders the application into `#main`. `m3e-theme` is
/// directly beneath the body, as in the web front end: there it writes its
/// tokens to the root element, where the colours and springs of the
/// application (`bridge::apply_theme`) override them, and they reach
/// everything, including the snack bars that `@m3e/web` adds to the body.
pub const INDEX: &str = r##"<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>vfox+</title>
  </head>
  <body>
    <m3e-theme color="#f28c1b" variant="tonal-spot" motion="expressive"><div id="main"></div></m3e-theme>
  </body>
</html>
"##;

/// The styles of the user interface.
const APP_CSS: &str = include_str!("../../assets/app.css");

/// The subset of Material Symbols Rounded with the icons of
/// `shared/icons.json` (`components/build.mjs`). Only the fill axis is kept.
const ICON_FONT: &[u8] = include_bytes!("../../assets/MaterialSymbolsRounded.woff2");

/// The web components of `@m3e/web` and the helpers of `window.vfoxPlus`
/// (`components/index.js`).
const COMPONENTS_JS: &str = include_str!("../../assets/components.js");

/// The Material 3 type scale: size and line height in pixels, the baseline and
/// the emphasized weight, and the tracking in pixels. The emphasized weights
/// are those of `docs/design-system.md#typography`.
const TYPE_SCALE: [(&str, [f64; 5]); 15] = [
    ("display-large", [57.0, 64.0, 400.0, 500.0, -0.25]),
    ("display-medium", [45.0, 52.0, 400.0, 500.0, 0.0]),
    ("display-small", [36.0, 44.0, 400.0, 500.0, 0.0]),
    ("headline-large", [32.0, 40.0, 400.0, 500.0, 0.0]),
    ("headline-medium", [28.0, 36.0, 400.0, 500.0, 0.0]),
    ("headline-small", [24.0, 32.0, 400.0, 500.0, 0.0]),
    ("title-large", [22.0, 28.0, 400.0, 500.0, 0.0]),
    ("title-medium", [16.0, 24.0, 500.0, 700.0, 0.15]),
    ("title-small", [14.0, 20.0, 500.0, 700.0, 0.1]),
    ("body-large", [16.0, 24.0, 400.0, 500.0, 0.5]),
    ("body-medium", [14.0, 20.0, 400.0, 500.0, 0.25]),
    ("body-small", [12.0, 16.0, 400.0, 500.0, 0.4]),
    ("label-large", [14.0, 20.0, 500.0, 700.0, 0.1]),
    ("label-medium", [12.0, 16.0, 500.0, 700.0, 0.5]),
    ("label-small", [11.0, 16.0, 500.0, 700.0, 0.5]),
];

/// The type scale as classes: `.type-title-medium`, `.type-title-medium-emphasized`, …
fn type_classes() -> String {
    let mut rules = String::new();
    for (name, [size, line_height, weight, emphasized_weight, tracking]) in TYPE_SCALE {
        for emphasized in [false, true] {
            let token = format!("--md-sys-typescale-{}{name}", if emphasized { "emphasized-" } else { "" });
            let class = format!(".type-{name}{}", if emphasized { "-emphasized" } else { "" });
            let weight = if emphasized { emphasized_weight } else { weight };
            let _ = writeln!(
                rules,
                "{class} {{ font-size: var({token}-font-size, {size}px); font-weight: var({token}-font-weight, {weight}); \
                 line-height: var({token}-line-height, {line_height}px); letter-spacing: var({token}-tracking, {tracking}px); }}"
            );
        }
    }
    rules
}

/// Every style of the document.
pub fn stylesheet() -> String {
    let font = base64::engine::general_purpose::STANDARD.encode(ICON_FONT);
    format!(
        "@font-face {{ font-family: 'Material Symbols Rounded'; font-style: normal; font-weight: 400; font-display: block; \
         src: url(data:font/woff2;base64,{font}) format('woff2'); }}\n{APP_CSS}\n{}",
        type_classes()
    )
}

/// What goes into the head of the document: the styles with the initial
/// `colors` of the scheme, and the components. The components are defined
/// before the application renders, so that its elements upgrade as soon as
/// they are created.
pub fn head(colors: &[(&str, String)]) -> String {
    let tokens = colors.iter().fold(String::new(), |mut tokens, (name, value)| {
        let _ = write!(tokens, "--md-sys-color-{name}: {value}; ");
        tokens
    });
    let mut head = format!("<style>:root {{ {tokens}}}\n{}</style><script>{COMPONENTS_JS}</script>", stylesheet());
    // The end-to-end driver of the web front end, in builds for the end-to-end tests.
    #[cfg(feature = "e2e")]
    let _ = write!(head, "<script>{}</script>", include_str!("../../assets/e2e-driver.js"));
    head
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_type_scale_has_a_baseline_and_an_emphasized_class_per_role() {
        let classes = type_classes();
        assert_eq!(classes.lines().count(), 30);
        assert!(classes.contains(
            ".type-label-large-emphasized { font-size: var(--md-sys-typescale-emphasized-label-large-font-size, 14px); \
             font-weight: var(--md-sys-typescale-emphasized-label-large-font-weight, 700);"
        ));
        assert!(classes.contains("letter-spacing: var(--md-sys-typescale-display-large-tracking, -0.25px)"));
    }

    #[test]
    fn the_document_carries_the_font_and_the_components() {
        let head = head(&crate::ui::theme::color_tokens(crate::core::settings::DEFAULT_SEED_COLOR, false));
        assert!(head.contains(":root { --md-sys-color-background: #fff8f5; "));
        assert!(head.contains("src: url(data:font/woff2;base64,d09G"), "a WOFF2 font starts with wOF2");
        assert!(head.contains("customElements"));
        assert!(head.contains(".shell .page.entering"));
    }
}

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

//! The user interface: Dioxus components that render the Material 3
//! Expressive web components of `@m3e/web` in the web view of the system,
//! with the theme, the motion, the icons and the desktop integration.

pub mod app;
pub mod bridge;
pub mod components;
pub mod context;
pub mod desktop;
#[cfg(feature = "e2e")]
pub mod e2e;
pub mod icons;
pub mod motion;
pub mod pages;
pub mod shell;
pub mod style;
pub mod theme;

#[cfg(test)]
mod tests {
    use std::collections::BTreeSet;
    use std::path::Path;

    use crate::core::i18n::Catalogue;

    /// The string literals of the sources of the user interface.
    fn literals() -> Vec<String> {
        let mut found = Vec::new();
        let mut pending = vec![Path::new(env!("CARGO_MANIFEST_DIR")).join("src/ui")];
        while let Some(directory) = pending.pop() {
            for entry in std::fs::read_dir(&directory).unwrap().filter_map(Result::ok) {
                let path = entry.path();
                if path.is_dir() {
                    pending.push(path);
                } else if path.extension().is_some_and(|extension| extension == "rs") {
                    let text = std::fs::read_to_string(&path).unwrap();
                    found.extend(text.split('"').skip(1).step_by(2).map(str::to_owned));
                }
            }
        }
        found
    }

    /// Names of the page that are camel case, too.
    const JAVASCRIPT_NAMES: [&str; 3] = ["closeLabel", "customElements", "valueOf"];

    /// Message keys and icon meanings are camel case, such as `navHome` and
    /// `chevronRight`: every such literal must be one of them, so that a typo
    /// fails here rather than showing the key or no icon.
    #[test]
    fn every_message_key_and_icon_exists() {
        let catalogue = Catalogue::builtin("en").unwrap();
        let keys: BTreeSet<&String> = catalogue.keys().into_iter().collect();
        let camel_case = |literal: &str| {
            literal.chars().next().is_some_and(|c| c.is_ascii_lowercase())
                && literal.chars().any(|c| c.is_ascii_uppercase())
                && literal.chars().all(|c| c.is_ascii_alphanumeric())
        };
        let unknown: Vec<String> = literals()
            .into_iter()
            .filter(|literal| camel_case(literal))
            .filter(|literal| !keys.contains(literal) && !super::icons::is_known(literal))
            .filter(|literal| !JAVASCRIPT_NAMES.contains(&literal.as_str()))
            .collect();
        assert!(unknown.is_empty(), "neither message keys nor icons: {unknown:?}");
        let used = literals().into_iter().filter(|literal| keys.contains(literal)).count();
        assert!(used > 150, "the message keys are found: {used}");
    }

    #[test]
    fn the_version_is_the_one_of_the_package() {
        assert_eq!(super::context::APP_VERSION, env!("CARGO_PKG_VERSION"));
    }
}

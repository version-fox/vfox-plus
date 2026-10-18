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

//! The Material Symbols of the user interface, by meaning, read from
//! `shared/icons.json`. The icon font contains exactly these symbols and their
//! ligatures, so an `m3e-icon` named after the symbol draws it.

use std::collections::HashMap;
use std::sync::LazyLock;

use serde_json::Value;

static ICONS: LazyLock<HashMap<String, String>> = LazyLock::new(|| {
    let manifest: Value = serde_json::from_str(include_str!("../../../shared/icons.json")).unwrap_or(Value::Null);
    manifest["icons"]
        .as_object()
        .map(|icons| {
            icons.iter().filter_map(|(meaning, symbol)| Some((meaning.clone(), symbol.as_str()?.to_owned()))).collect()
        })
        .unwrap_or_default()
});

/// The Material Symbol of the icon for `meaning`, such as `arrow_back` for
/// `back`. An unknown meaning, which the tests catch, is drawn as itself.
pub fn symbol(meaning: &str) -> &str {
    ICONS.get(meaning).map_or(meaning, String::as_str)
}

/// Whether `shared/icons.json` has an icon for `meaning`.
pub fn is_known(meaning: &str) -> bool {
    ICONS.contains_key(meaning)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn meanings_map_to_symbols() {
        assert_eq!(symbol("back"), "arrow_back");
        assert_eq!(symbol("home"), "home");
        assert!(is_known("sdks"));
        assert!(!is_known("nonsense"));
        assert_eq!(symbol("nonsense"), "nonsense");
        assert!(ICONS.len() > 50);
    }
}

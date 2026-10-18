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

//! Preferences of the application itself, not of vfox. The JSON keys are part
//! of the shared specification in `docs/architecture.md#settings`, so that
//! every implementation of the GUI reads the same file.

use std::cell::{Cell, RefCell};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

use serde_json::{Map, Value};

/// Light or dark appearance; `System` follows the desktop.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ThemePreference {
    /// Follows the desktop.
    System,
    /// Always light.
    Light,
    /// Always dark.
    Dark,
}

impl ThemePreference {
    const ALL: [Self; 3] = [Self::System, Self::Light, Self::Dark];

    /// The value in the settings file.
    pub fn json(self) -> &'static str {
        match self {
            Self::System => "system",
            Self::Light => "light",
            Self::Dark => "dark",
        }
    }
}

/// The language of the user interface; `System` follows the desktop.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LanguagePreference {
    /// Follows the desktop.
    System,
    /// English.
    English,
    /// Simplified Chinese.
    SimplifiedChinese,
}

impl LanguagePreference {
    const ALL: [Self; 3] = [Self::System, Self::English, Self::SimplifiedChinese];

    /// The value in the settings file.
    pub fn json(self) -> &'static str {
        match self {
            Self::System => "system",
            Self::English => "english",
            Self::SimplifiedChinese => "simplifiedChinese",
        }
    }

    /// The language code of the catalogue, or `None` to follow the desktop.
    pub fn code(self) -> Option<&'static str> {
        match self {
            Self::System => None,
            Self::English => Some("en"),
            Self::SimplifiedChinese => Some("zh"),
        }
    }
}

/// The brand colour when the desktop accent colour is not used: the orange of
/// the vfox logo.
pub const DEFAULT_SEED_COLOR: u32 = 0xFFF2_8C1B;

/// Preferences of the application, persisted as JSON.
#[derive(Clone, Debug, PartialEq)]
pub struct AppSettings {
    /// Light, dark or the desktop's appearance.
    pub theme: ThemePreference,
    /// Whether to derive the colour scheme from the desktop accent colour.
    pub use_system_color: bool,
    /// ARGB seed of the colour scheme when the accent colour is not used.
    pub seed_color: u32,
    /// The language of the user interface.
    pub language: LanguagePreference,
    /// Path of the vfox executable chosen by the user, or `None` to detect it.
    pub vfox_executable: Option<String>,
    /// Additional environment variables for vfox, such as mirror settings.
    pub extra_environment: BTreeMap<String, String>,
    /// Whether the navigation rail shows labels next to the icons.
    pub rail_expanded: bool,
    /// Keys this version does not know, kept so that settings of other
    /// implementations survive.
    pub unknown: Map<String, Value>,
}

impl Default for AppSettings {
    fn default() -> Self {
        Self {
            theme: ThemePreference::System,
            use_system_color: true,
            seed_color: DEFAULT_SEED_COLOR,
            language: LanguagePreference::System,
            vfox_executable: None,
            extra_environment: BTreeMap::new(),
            rail_expanded: false,
            unknown: Map::new(),
        }
    }
}

const KNOWN_KEYS: [&str; 7] =
    ["theme", "useSystemColor", "seedColor", "language", "vfoxExecutable", "extraEnvironment", "railExpanded"];

impl AppSettings {
    /// The settings as JSON, with the keys in the order of the specification.
    pub fn to_json(&self) -> Value {
        let mut map = Map::new();
        map.insert("theme".into(), self.theme.json().into());
        map.insert("useSystemColor".into(), self.use_system_color.into());
        map.insert("seedColor".into(), self.seed_color.into());
        map.insert("language".into(), self.language.json().into());
        if let Some(executable) = &self.vfox_executable {
            map.insert("vfoxExecutable".into(), executable.clone().into());
        }
        map.insert(
            "extraEnvironment".into(),
            Value::Object(
                self.extra_environment.iter().map(|(key, value)| (key.clone(), value.clone().into())).collect(),
            ),
        );
        map.insert("railExpanded".into(), self.rail_expanded.into());
        map.extend(self.unknown.clone());
        Value::Object(map)
    }

    /// Restores settings from `json`, ignoring malformed values.
    pub fn from_json(json: &Value) -> Self {
        let Some(map) = json.as_object() else { return Self::default() };
        let string = |key: &str| map.get(key).and_then(Value::as_str);
        let defaults = Self::default();
        Self {
            theme: ThemePreference::ALL
                .into_iter()
                .find(|theme| Some(theme.json()) == string("theme"))
                .unwrap_or(defaults.theme),
            use_system_color: map.get("useSystemColor").and_then(Value::as_bool).unwrap_or(defaults.use_system_color),
            seed_color: map
                .get("seedColor")
                .and_then(Value::as_u64)
                .and_then(|seed| u32::try_from(seed).ok())
                .unwrap_or(defaults.seed_color),
            language: LanguagePreference::ALL
                .into_iter()
                .find(|language| Some(language.json()) == string("language"))
                .unwrap_or(defaults.language),
            vfox_executable: string("vfoxExecutable").filter(|path| !path.is_empty()).map(str::to_owned),
            extra_environment: map
                .get("extraEnvironment")
                .and_then(Value::as_object)
                .map(|variables| {
                    variables
                        .iter()
                        .filter_map(|(key, value)| value.as_str().map(|value| (key.clone(), value.to_owned())))
                        .collect()
                })
                .unwrap_or_default(),
            rail_expanded: map.get("railExpanded").and_then(Value::as_bool).unwrap_or(defaults.rail_expanded),
            unknown: map
                .iter()
                .filter(|(key, _)| !KNOWN_KEYS.contains(&key.as_str()))
                .map(|(k, v)| (k.clone(), v.clone()))
                .collect(),
        }
    }
}

/// The variables of the valid lines of [`parse_environment_lines`], and the
/// 1-based numbers of the others.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ParsedEnvironment {
    /// The variables of the valid lines.
    pub variables: BTreeMap<String, String>,
    /// The numbers of the invalid lines.
    pub invalid_lines: Vec<usize>,
}

fn is_environment_key(key: &str) -> bool {
    let mut chars = key.chars();
    chars.next().is_some_and(|c| c.is_ascii_alphabetic() || c == '_')
        && chars.all(|c| c.is_ascii_alphanumeric() || c == '_')
}

/// Parses `KEY=VALUE` lines into environment variables. Blank lines and lines
/// starting with `#` are ignored; keys must be valid POSIX variable names.
pub fn parse_environment_lines(text: &str) -> ParsedEnvironment {
    let mut variables = BTreeMap::new();
    let mut invalid_lines = Vec::new();
    for (index, line) in text.split('\n').enumerate() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        match line.split_once('=') {
            Some((key, value)) if is_environment_key(key.trim()) => {
                variables.insert(key.trim().to_owned(), value.trim().to_owned());
            }
            _ => invalid_lines.push(index + 1),
        }
    }
    ParsedEnvironment { variables, invalid_lines }
}

/// Formats environment variables as `KEY=VALUE` lines, sorted by key.
pub fn format_environment_lines(variables: &BTreeMap<String, String>) -> String {
    variables.iter().map(|(key, value)| format!("{key}={value}")).collect::<Vec<_>>().join("\n")
}

/// The settings file for `environment`: `$XDG_CONFIG_HOME/vfox-plus/settings.json`,
/// where `XDG_CONFIG_HOME` defaults to `~/.config` as the XDG Base Directory
/// Specification defines.
pub fn settings_path(environment: &BTreeMap<String, String>) -> PathBuf {
    let base = match environment.get("XDG_CONFIG_HOME") {
        Some(directory) if directory.starts_with('/') => PathBuf::from(directory),
        _ => environment.get("HOME").map_or_else(std::env::temp_dir, PathBuf::from).join(".config"),
    };
    base.join("vfox-plus").join("settings.json")
}

/// Loads and saves [`AppSettings`].
pub trait SettingsStore {
    /// The stored settings, or the defaults if none are stored.
    fn load(&self) -> AppSettings;

    /// Persists `settings`.
    ///
    /// # Errors
    ///
    /// When the settings cannot be written.
    fn save(&self, settings: &AppSettings) -> std::io::Result<()>;
}

/// Stores settings as JSON in a file. Corrupted files are ignored rather than
/// fatal: the application then starts with the defaults.
#[derive(Clone, Debug)]
pub struct JsonFileSettingsStore {
    /// The settings file.
    pub path: PathBuf,
}

impl JsonFileSettingsStore {
    /// A store of the file at `path`.
    pub fn new(path: impl AsRef<Path>) -> Self {
        Self { path: path.as_ref().to_owned() }
    }
}

impl SettingsStore for JsonFileSettingsStore {
    fn load(&self) -> AppSettings {
        std::fs::read_to_string(&self.path)
            .ok()
            .and_then(|text| serde_json::from_str::<Value>(&text).ok())
            .map(|json| AppSettings::from_json(&json))
            .unwrap_or_default()
    }

    fn save(&self, settings: &AppSettings) -> std::io::Result<()> {
        // A temporary file first, so that a crash never leaves a truncated
        // settings file behind.
        if let Some(parent) = self.path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        let temporary = self.path.with_extension("json.tmp");
        let text = serde_json::to_string_pretty(&settings.to_json()).map_err(std::io::Error::other)?;
        std::fs::write(&temporary, format!("{text}\n"))?;
        std::fs::rename(&temporary, &self.path)
    }
}

/// Keeps settings in memory, for tests.
#[derive(Debug, Default)]
pub struct MemorySettingsStore {
    settings: RefCell<AppSettings>,
    saves: Cell<usize>,
}

impl MemorySettingsStore {
    /// A store that holds `settings`.
    pub fn new(settings: AppSettings) -> Self {
        Self { settings: RefCell::new(settings), saves: Cell::new(0) }
    }

    /// How many times [`SettingsStore::save`] was called.
    pub fn save_count(&self) -> usize {
        self.saves.get()
    }
}

impl SettingsStore for MemorySettingsStore {
    fn load(&self) -> AppSettings {
        self.settings.borrow().clone()
    }

    fn save(&self, settings: &AppSettings) -> std::io::Result<()> {
        self.saves.set(self.saves.get() + 1);
        *self.settings.borrow_mut() = settings.clone();
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn settings_round_trip_through_json_with_the_keys_of_the_specification_in_order() {
        let settings = AppSettings {
            theme: ThemePreference::Dark,
            use_system_color: false,
            seed_color: 0xFF12_3456,
            language: LanguagePreference::SimplifiedChinese,
            vfox_executable: Some("/opt/vfox".into()),
            extra_environment: BTreeMap::from([("A".into(), "1".into())]),
            rail_expanded: true,
            unknown: Map::new(),
        };
        let text = settings.to_json().to_string();
        assert_eq!(AppSettings::from_json(&serde_json::from_str(&text).unwrap()), settings);
        let keys: Vec<String> = AppSettings::default().to_json().as_object().unwrap().keys().cloned().collect();
        assert_eq!(keys, ["theme", "useSystemColor", "seedColor", "language", "extraEnvironment", "railExpanded"]);
        assert_eq!(AppSettings::default().to_json()["seedColor"], u64::from(DEFAULT_SEED_COLOR));
    }

    #[test]
    fn malformed_values_are_ignored_and_unknown_keys_kept() {
        let json: Value = serde_json::from_str(
            r##"{"theme":"purple","useSystemColor":"yes","seedColor":"#fff","language":42,"vfoxExecutable":"",
               "extraEnvironment":{"A":"1","B":2},"railExpanded":"true","future":true}"##,
        )
        .unwrap();
        let settings = AppSettings::from_json(&json);
        let mut expected =
            AppSettings { extra_environment: BTreeMap::from([("A".into(), "1".into())]), ..AppSettings::default() };
        expected.unknown.insert("future".into(), true.into());
        assert_eq!(settings, expected);
        assert_eq!(settings.to_json()["future"], true);
        assert_eq!(AppSettings::from_json(&serde_json::json!([1, 2])), AppSettings::default());
        assert!(AppSettings::from_json(&serde_json::json!({"extraEnvironment": ["x"]})).extra_environment.is_empty());
        assert_eq!(
            AppSettings::from_json(&serde_json::json!({"seedColor": 99_999_999_999u64})).seed_color,
            DEFAULT_SEED_COLOR
        );
    }

    #[test]
    fn environment_lines() {
        let parsed = parse_environment_lines("# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n");
        assert_eq!(parsed.variables, BTreeMap::from([("A".into(), "1".into()), ("B".into(), "x=y".into())]));
        assert_eq!(parsed.invalid_lines, vec![5, 6]);
        assert_eq!(
            format_environment_lines(&BTreeMap::from([("B".into(), "2".into()), ("A".into(), "1".into())])),
            "A=1\nB=2"
        );
    }

    #[test]
    fn the_settings_file_follows_xdg_and_is_written_atomically() {
        let environment =
            |pairs: &[(&str, &str)]| pairs.iter().map(|(k, v)| ((*k).to_owned(), (*v).to_owned())).collect();
        assert_eq!(
            settings_path(&environment(&[("XDG_CONFIG_HOME", "/x"), ("HOME", "/h")])),
            PathBuf::from("/x/vfox-plus/settings.json")
        );
        assert_eq!(
            settings_path(&environment(&[("XDG_CONFIG_HOME", "relative"), ("HOME", "/h")])),
            PathBuf::from("/h/.config/vfox-plus/settings.json")
        );
        let directory = tempfile::tempdir().unwrap();
        let store = JsonFileSettingsStore::new(directory.path().join("nested/settings.json"));
        assert_eq!(store.load(), AppSettings::default());
        let settings = AppSettings { theme: ThemePreference::Light, ..AppSettings::default() };
        store.save(&settings).unwrap();
        assert_eq!(store.load(), settings);
        let text = std::fs::read_to_string(&store.path).unwrap();
        assert!(text.ends_with("}\n"));
        assert!(text.contains("\n  \"theme\": \"light\""));
        assert!(!directory.path().join("nested/settings.json.tmp").exists());
        std::fs::write(&store.path, "{not json").unwrap();
        assert_eq!(store.load(), AppSettings::default());
    }

    #[test]
    fn settings_of_another_implementation_keep_their_unknown_keys() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("settings.json");
        std::fs::write(&path, r#"{"theme": "dark", "windowSize": {"width": 1}}"#).unwrap();
        let store = JsonFileSettingsStore::new(&path);
        store.save(&AppSettings { rail_expanded: true, ..store.load() }).unwrap();
        let saved: Value = serde_json::from_str(&std::fs::read_to_string(&path).unwrap()).unwrap();
        assert_eq!(saved["theme"], "dark");
        assert_eq!(saved["windowSize"]["width"], 1);
        let memory = MemorySettingsStore::new(AppSettings::default());
        memory.save(&AppSettings::default()).unwrap();
        assert_eq!(memory.save_count(), 1);
    }
}

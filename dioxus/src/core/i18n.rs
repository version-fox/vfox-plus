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

//! The translations of `shared/i18n`, in the subset of ICU `MessageFormat` that
//! the catalogues use: placeholders (`{name}`) and plurals with exact matches
//! and plural categories (`{count, plural, =0{…} =1{…} other{…}}`), which may
//! contain placeholders themselves. Apostrophes are literal, as in Flutter's
//! gen-l10n without use-escaping. See `docs/i18n.md`.

use std::cell::RefCell;
use std::collections::BTreeMap;
use std::collections::HashMap;
use std::fmt;
use std::rc::Rc;

use serde_json::{Map, Value};

use super::settings::LanguagePreference;

/// The catalogues of `shared/i18n`, built into the application.
const CATALOGUES: [(&str, &str); 2] =
    [("en", include_str!("../../../shared/i18n/app_en.arb")), ("zh", include_str!("../../../shared/i18n/app_zh.arb"))];

/// A part of a parsed message.
#[derive(Clone, Debug, PartialEq)]
pub enum MessagePart {
    /// Literal text.
    Text(String),
    /// A placeholder, replaced by an argument.
    Argument(String),
    /// A plural of the argument `name`, by case.
    Plural {
        /// The argument that chooses the case.
        name: String,
        /// The cases: `=0`, `one`, `other`, …
        cases: Vec<(String, Vec<MessagePart>)>,
    },
}

/// A message could not be parsed.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MessageFormatError(pub String);

impl fmt::Display for MessageFormatError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

/// An argument of a message.
#[derive(Clone, Debug, PartialEq)]
pub enum MessageArgument {
    /// Text, for placeholders.
    Text(String),
    /// A number, for placeholders and plurals.
    Number(f64),
}

impl fmt::Display for MessageArgument {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Text(text) => f.write_str(text),
            Self::Number(number) if number.fract() == 0.0 && number.abs() < 1e15 => write!(f, "{number:.0}"),
            Self::Number(number) => write!(f, "{number}"),
        }
    }
}

impl From<&str> for MessageArgument {
    fn from(text: &str) -> Self {
        Self::Text(text.to_owned())
    }
}

impl From<String> for MessageArgument {
    fn from(text: String) -> Self {
        Self::Text(text)
    }
}

impl From<&String> for MessageArgument {
    fn from(text: &String) -> Self {
        Self::Text(text.clone())
    }
}

impl From<usize> for MessageArgument {
    #[allow(clippy::cast_precision_loss)]
    fn from(number: usize) -> Self {
        Self::Number(number as f64)
    }
}

impl From<u32> for MessageArgument {
    fn from(number: u32) -> Self {
        Self::Number(f64::from(number))
    }
}

impl From<i32> for MessageArgument {
    fn from(number: i32) -> Self {
        Self::Number(f64::from(number))
    }
}

/// Parses `source` into parts.
///
/// # Errors
///
/// When `source` is not a message of the supported subset.
pub fn parse_message(source: &str) -> Result<Vec<MessagePart>, MessageFormatError> {
    let mut parser = Parser { chars: source.chars().collect(), position: 0, source };
    let parts = parser.parts()?;
    if parser.position < parser.chars.len() {
        return Err(parser.error("Unbalanced \"}\""));
    }
    Ok(parts)
}

/// The plural category of `count` in `language`: `one` or `other` (the
/// catalogues only need English and Chinese).
pub fn plural_category(language: &str, count: f64) -> &'static str {
    // Chinese has no grammatical number.
    if language != "zh" && (count - 1.0).abs() < f64::EPSILON { "one" } else { "other" }
}

/// Formats parsed `parts` with `arguments`, choosing plural cases by the rules
/// of `language`.
pub fn format_message(parts: &[MessagePart], arguments: &[(&str, MessageArgument)], language: &str) -> String {
    let mut result = String::new();
    for part in parts {
        match part {
            MessagePart::Text(text) => result.push_str(text),
            MessagePart::Argument(name) => {
                if let Some((_, value)) = arguments.iter().find(|(key, _)| key == name) {
                    result.push_str(&value.to_string());
                } else {
                    result.push('{');
                    result.push_str(name);
                    result.push('}');
                }
            }
            MessagePart::Plural { name, cases } => {
                let count = match arguments.iter().find(|(key, _)| key == name) {
                    Some((_, MessageArgument::Number(number))) => *number,
                    _ => 0.0,
                };
                let exact = format!("={}", MessageArgument::Number(count));
                let category = plural_category(language, count);
                let chosen = [exact.as_str(), category, "other"]
                    .iter()
                    .find_map(|selector| cases.iter().find(|(case, _)| case == selector))
                    .map(|(_, parts)| parts.as_slice())
                    .unwrap_or_default();
                result.push_str(&format_message(chosen, arguments, language));
            }
        }
    }
    result
}

struct Parser<'a> {
    chars: Vec<char>,
    position: usize,
    source: &'a str,
}

impl Parser<'_> {
    fn error(&self, message: &str) -> MessageFormatError {
        MessageFormatError(format!("{message} in \"{}\"", self.source))
    }

    fn peek(&self) -> Option<char> {
        self.chars.get(self.position).copied()
    }

    fn expect(&mut self, expected: char) -> Result<(), MessageFormatError> {
        if self.peek() == Some(expected) {
            self.position += 1;
            Ok(())
        } else {
            Err(self.error(&format!("Expected \"{expected}\" at {}", self.position)))
        }
    }

    fn skip_spaces(&mut self) {
        while self.peek().is_some_and(char::is_whitespace) {
            self.position += 1;
        }
    }

    /// Parses parts until the end or an unmatched `}`, which is not consumed.
    fn parts(&mut self) -> Result<Vec<MessagePart>, MessageFormatError> {
        let mut parts = Vec::new();
        let mut text = String::new();
        while let Some(c) = self.peek() {
            match c {
                '}' => break,
                '{' => {
                    if !text.is_empty() {
                        parts.push(MessagePart::Text(std::mem::take(&mut text)));
                    }
                    parts.push(self.placeholder()?);
                }
                _ => {
                    text.push(c);
                    self.position += 1;
                }
            }
        }
        if !text.is_empty() {
            parts.push(MessagePart::Text(text));
        }
        Ok(parts)
    }

    fn placeholder(&mut self) -> Result<MessagePart, MessageFormatError> {
        self.expect('{')?;
        let name = self.identifier()?;
        self.skip_spaces();
        if self.peek() == Some('}') {
            self.position += 1;
            return Ok(MessagePart::Argument(name));
        }
        self.expect(',')?;
        let kind = self.identifier()?;
        if kind != "plural" {
            return Err(self.error(&format!("Unsupported argument type \"{kind}\"")));
        }
        self.skip_spaces();
        self.expect(',')?;
        let mut cases = Vec::new();
        loop {
            self.skip_spaces();
            if self.peek() == Some('}') {
                self.position += 1;
                break;
            }
            let selector = self.selector()?;
            self.skip_spaces();
            self.expect('{')?;
            cases.push((selector, self.parts()?));
            self.expect('}')?;
        }
        if !cases.iter().any(|(case, _)| case == "other") {
            return Err(self.error("A plural needs an \"other\" case"));
        }
        Ok(MessagePart::Plural { name, cases })
    }

    /// `[A-Za-z_][A-Za-z0-9_]*`
    fn identifier(&mut self) -> Result<String, MessageFormatError> {
        self.skip_spaces();
        let start = self.position;
        if !self.peek().is_some_and(|c| c.is_ascii_alphabetic() || c == '_') {
            return Err(self.error("Expected a name"));
        }
        while self.peek().is_some_and(|c| c.is_ascii_alphanumeric() || c == '_') {
            self.position += 1;
        }
        Ok(self.chars[start..self.position].iter().collect())
    }

    /// `=\d+` or `[a-z]+`
    fn selector(&mut self) -> Result<String, MessageFormatError> {
        let start = self.position;
        if self.peek() == Some('=') {
            self.position += 1;
            let digits = self.position;
            while self.peek().is_some_and(|c| c.is_ascii_digit()) {
                self.position += 1;
            }
            if self.position == digits {
                return Err(self.error("Expected a plural case"));
            }
        } else {
            while self.peek().is_some_and(|c| c.is_ascii_lowercase()) {
                self.position += 1;
            }
            if self.position == start {
                return Err(self.error("Expected a plural case"));
            }
        }
        Ok(self.chars[start..self.position].iter().collect())
    }
}

/// The languages with a catalogue in `shared/i18n`.
pub const SUPPORTED_LANGUAGES: [&str; 2] = ["en", "zh"];

/// Picks the catalogue for `preference`. With the system preference, the
/// locales of the desktop are matched against the catalogues, most preferred
/// first; every Chinese locale gets Simplified Chinese, and English is the
/// fallback.
pub fn resolve_language(preference: LanguagePreference, system_locales: &[String]) -> &'static str {
    if let Some(code) = preference.code() {
        return code;
    }
    for locale in system_locales {
        let language = locale.to_lowercase();
        match language.split(['-', '_', '.', '@']).next() {
            Some("zh") => return "zh",
            Some("en") => return "en",
            _ => {}
        }
    }
    "en"
}

/// The locales of the desktop, most preferred first, as `GLib` reads them from
/// `environment`.
pub fn system_locales(environment: &BTreeMap<String, String>) -> Vec<String> {
    let get = |key: &str| environment.get(key).cloned().unwrap_or_default();
    get("LANGUAGE")
        .split(':')
        .map(str::to_owned)
        .chain([get("LC_ALL"), get("LC_MESSAGES"), get("LANG")])
        .filter(|locale| !locale.is_empty() && locale != "C" && locale != "POSIX" && !locale.starts_with("C."))
        .collect()
}

/// An ARB catalogue: messages and their `@`-prefixed metadata.
#[derive(Clone, Debug, PartialEq)]
pub struct Catalogue {
    /// Every entry, metadata included.
    pub entries: Map<String, Value>,
}

impl Catalogue {
    /// Parses the ARB catalogue `text`.
    ///
    /// # Errors
    ///
    /// When `text` is not a JSON object.
    pub fn parse(text: &str) -> Result<Self, serde_json::Error> {
        let value: Value = serde_json::from_str(text)?;
        Ok(Self { entries: value.as_object().cloned().unwrap_or_default() })
    }

    /// The built-in catalogue of `language`, if there is one.
    pub fn builtin(language: &str) -> Option<Self> {
        CATALOGUES.iter().find(|(code, _)| *code == language).and_then(|(_, text)| Self::parse(text).ok())
    }

    /// The message keys, without metadata.
    pub fn keys(&self) -> Vec<&String> {
        self.entries.keys().filter(|key| !key.starts_with('@')).collect()
    }

    /// The message `key`, if there is one.
    pub fn message(&self, key: &str) -> Option<&str> {
        self.entries.get(key).and_then(Value::as_str)
    }
}

/// The messages of one language, with English as the fallback.
#[derive(Clone, Debug)]
pub struct Localizations {
    language: &'static str,
    messages: Rc<Catalogue>,
    fallback: Rc<Catalogue>,
    cache: Rc<RefCell<HashMap<String, Vec<MessagePart>>>>,
}

impl PartialEq for Localizations {
    fn eq(&self, other: &Self) -> bool {
        self.language == other.language && Rc::ptr_eq(&self.messages, &other.messages)
    }
}

impl Localizations {
    /// The messages of `messages` in `language`, falling back to `fallback`.
    pub fn new(language: &'static str, messages: Catalogue, fallback: Catalogue) -> Self {
        Self { language, messages: Rc::new(messages), fallback: Rc::new(fallback), cache: Rc::default() }
    }

    /// The translations for `preference` and the desktop locales `system_locales`.
    pub fn for_preference(preference: LanguagePreference, system_locales: &[String]) -> Self {
        Self::for_language(resolve_language(preference, system_locales))
    }

    /// The translations of `language`, one of [`SUPPORTED_LANGUAGES`].
    pub fn for_language(language: &'static str) -> Self {
        let english = Catalogue::builtin("en").unwrap_or(Catalogue { entries: Map::new() });
        let messages = if language == "en" {
            english.clone()
        } else {
            Catalogue::builtin(language).unwrap_or_else(|| english.clone())
        };
        Self::new(language, messages, english)
    }

    /// The language code of the messages.
    pub fn language(&self) -> &'static str {
        self.language
    }

    /// The BCP 47 tag of the language.
    pub fn language_tag(&self) -> &'static str {
        if self.language == "zh" { "zh-Hans" } else { "en" }
    }

    /// The message `key` formatted with `arguments`. A key that the language
    /// lacks falls back to English; a key that no catalogue has is returned as
    /// it is, which the tests catch.
    pub fn t(&self, key: &str, arguments: &[(&str, MessageArgument)]) -> String {
        let mut cache = self.cache.borrow_mut();
        let parts = cache.entry(key.to_owned()).or_insert_with(|| {
            match self.messages.message(key).or_else(|| self.fallback.message(key)) {
                Some(source) => parse_message(source).unwrap_or_else(|_| vec![MessagePart::Text(source.to_owned())]),
                None => vec![MessagePart::Text(key.to_owned())],
            }
        });
        format_message(parts, arguments, self.language)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn placeholders(parts: &[MessagePart]) -> Vec<String> {
        let mut names = Vec::new();
        for part in parts {
            match part {
                MessagePart::Text(_) => {}
                MessagePart::Argument(name) => names.push(name.clone()),
                MessagePart::Plural { name, cases } => {
                    names.push(name.clone());
                    for (_, parts) in cases {
                        names.extend(placeholders(parts));
                    }
                }
            }
        }
        names.sort();
        names.dedup();
        names
    }

    #[test]
    fn placeholders_and_plurals_with_exact_matches_first() {
        let parts = parse_message("{count, plural, =0{none} =1{one {name}} other{{count} of {name}}}!").unwrap();
        let format = |count: i32| format_message(&parts, &[("count", count.into()), ("name", "x".into())], "en");
        assert_eq!(format(0), "none!");
        assert_eq!(format(1), "one x!");
        assert_eq!(format(7), "7 of x!");
        let categories = parse_message("{n, plural, one{single} other{many}}").unwrap();
        assert_eq!(format_message(&categories, &[("n", 1.into())], "en"), "single");
        assert_eq!(format_message(&categories, &[("n", 1.into())], "zh"), "many");
        assert_eq!(format_message(&parse_message("don't {x}").unwrap(), &[], "en"), "don't {x}");
    }

    #[test]
    fn malformed_messages_are_rejected() {
        assert!(parse_message("{a, select, x{y} other{z}}").is_err());
        assert!(parse_message("{a, plural, =1{y}}").is_err());
        assert!(parse_message("a } b").is_err());
        assert!(parse_message("{1}").is_err());
        assert!(parse_message("{a, plural, ={y} other{z}}").is_err());
    }

    #[test]
    fn the_catalogues_have_the_same_messages_descriptions_and_placeholders() {
        let en = Catalogue::builtin("en").unwrap();
        let zh = Catalogue::builtin("zh").unwrap();
        assert_eq!(en.entries["@@locale"], "en");
        assert_eq!(zh.entries["@@locale"], "zh");
        let mut en_keys = en.keys();
        let mut zh_keys = zh.keys();
        en_keys.sort();
        zh_keys.sort();
        assert_eq!(en_keys, zh_keys);
        for key in en.keys() {
            let metadata = &en.entries[&format!("@{key}")];
            assert!(metadata["description"].is_string(), "{key} has a description");
            let template = placeholders(&parse_message(en.message(key).unwrap()).unwrap());
            let translated = placeholders(&parse_message(zh.message(key).unwrap()).unwrap());
            assert!(template.iter().all(|name| translated.contains(name)), "{key} lacks placeholders in app_zh.arb");
            let mut declared: Vec<String> = metadata
                .get("placeholders")
                .and_then(Value::as_object)
                .map(|p| p.keys().cloned().collect())
                .unwrap_or_default();
            declared.sort();
            assert_eq!(declared, template, "{key}");
        }
    }

    #[test]
    fn localizations_format_each_language_and_fall_back_to_english() {
        let english = Localizations::for_preference(LanguagePreference::English, &[]);
        let chinese = Localizations::for_preference(LanguagePreference::SimplifiedChinese, &[]);
        assert_eq!(english.t("sdksVersionCount", &[("count", 2.into())]), "2 versions installed");
        assert_eq!(english.t("sdksVersionCount", &[("count", 1.into())]), "1 version installed");
        assert_eq!(chinese.t("sdksVersionCount", &[("count", 2.into())]), "已安装 2 个版本");
        assert_eq!(chinese.t("taskInstall", &[("subject", "maven@3.9.16".into())]), "安装 maven@3.9.16");
        let partial = Localizations::new(
            "zh",
            Catalogue { entries: serde_json::from_str(r#"{"navHome": "首页"}"#).unwrap() },
            Catalogue::builtin("en").unwrap(),
        );
        assert_eq!(partial.t("navSdks", &[]), "SDKs");
        assert_eq!(english.t("noSuchKey", &[]), "noSuchKey");
        assert_eq!(chinese.language_tag(), "zh-Hans");
        assert_eq!(english.language(), "en");
        assert_eq!(english, Localizations { cache: Rc::default(), ..english.clone() });
    }

    #[test]
    fn the_system_language() {
        let locales = |values: &[&str]| values.iter().map(|value| (*value).to_owned()).collect::<Vec<_>>();
        assert_eq!(resolve_language(LanguagePreference::System, &locales(&["zh-TW"])), "zh");
        assert_eq!(resolve_language(LanguagePreference::System, &locales(&["zh_CN.UTF-8"])), "zh");
        assert_eq!(resolve_language(LanguagePreference::System, &locales(&["de-DE", "en-US"])), "en");
        assert_eq!(resolve_language(LanguagePreference::System, &locales(&["fr"])), "en");
        assert_eq!(resolve_language(LanguagePreference::English, &locales(&["zh-CN"])), "en");
        assert_eq!(resolve_language(LanguagePreference::SimplifiedChinese, &locales(&["en"])), "zh");
        let environment: BTreeMap<String, String> =
            [("LANGUAGE", "zh_CN:en_US"), ("LC_ALL", ""), ("LANG", "de_DE.UTF-8")]
                .iter()
                .map(|(k, v)| ((*k).into(), (*v).into()))
                .collect();
        assert_eq!(system_locales(&environment), locales(&["zh_CN", "en_US", "de_DE.UTF-8"]));
        assert_eq!(system_locales(&BTreeMap::from([("LANG".into(), "C.UTF-8".into())])), Vec::<String>::new());
        assert_eq!(MessageArgument::Number(2.5).to_string(), "2.5");
    }
}

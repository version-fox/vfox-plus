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

//! Runs the parsers against the shared vfox output fixtures. The expected
//! results live in `shared/fixtures/vfox-*/cases.json` and are shared by every
//! implementation; see `shared/fixtures/README.md`.

use serde_json::{Map, Value, json};

use super::models::AvailableVersion;
use super::output::{
    parse_available_plugins, parse_config_list, parse_current, parse_download_progress, parse_install_path,
    parse_installed_tree, parse_plugin_info, parse_search, parse_vfox_version, summarize_error,
};
use super::terminal::{split_terminal_lines, strip_ansi};
use super::testing::{read_capture, shared};

/// A JSON value with every number as a float, so that `1` and `1.0` compare equal.
fn canonical(value: &Value) -> Value {
    match value {
        Value::Number(number) => json!(number.as_f64()),
        Value::Array(items) => Value::Array(items.iter().map(canonical).collect()),
        Value::Object(map) => Value::Object(map.iter().map(|(key, value)| (key.clone(), canonical(value))).collect()),
        other => other.clone(),
    }
}

fn indexed(items: Vec<Value>) -> Value {
    let count = items.len();
    let items: Map<String, Value> =
        items.into_iter().enumerate().map(|(index, item)| (index.to_string(), item)).collect();
    json!({ "count": count, "items": items })
}

fn version(version: &AvailableVersion) -> Value {
    json!({
        "version": version.version,
        "note": version.note,
        "additions": version.additions.iter().map(|a| json!({ "name": a.name, "version": a.version })).collect::<Vec<_>>(),
        "installed": version.installed,
    })
}

/// The result of the parser `check` in the shape of cases.json.
fn parse(check: &str, stdout: &str, stderr: &str) -> Value {
    match check {
        "version" => json!(parse_vfox_version(stdout)),
        "current" => {
            Value::Object(parse_current(stdout).into_iter().map(|(name, current)| (name, json!(current))).collect())
        }
        "list" => Value::Object(
            parse_installed_tree(stdout).into_iter().map(|(name, versions)| (name, json!(versions))).collect(),
        ),
        "search" => indexed(parse_search(stdout).iter().map(version).collect()),
        "available" => indexed(
            parse_available_plugins(stdout)
                .iter()
                .map(|p| json!({ "name": p.name, "homepage": p.homepage, "official": p.official }))
                .collect(),
        ),
        "pluginInfo" => parse_plugin_info(stdout).map_or(Value::Null, |info| {
            json!({
                "name": info.name,
                "version": info.version,
                "homepage": info.homepage,
                "installPath": info.install_path,
                "description": info.description,
            })
        }),
        "installPath" => json!(parse_install_path(stdout)),
        "config" => json!(
            parse_config_list(stdout).iter().map(|e| json!({ "key": e.key, "value": e.value })).collect::<Vec<_>>()
        ),
        "finalProgress" => json!(
            split_terminal_lines(&strip_ansi(stderr))
                .iter()
                .filter_map(|line| parse_download_progress(line))
                .next_back()
        ),
        "error" => json!(summarize_error(stdout, stderr)),
        other => panic!("Unknown parser \"{other}\" in cases.json"),
    }
}

/// For search and available, only the listed items are compared.
fn select(check: &str, actual: Value, expected: &Value) -> Value {
    if check != "search" && check != "available" {
        return actual;
    }
    let items: Map<String, Value> = expected["items"]
        .as_object()
        .map(|items| items.keys().map(|key| (key.clone(), actual["items"][key].clone())).collect())
        .unwrap_or_default();
    json!({ "count": actual["count"], "items": items })
}

#[test]
fn every_case_of_every_captured_vfox_version() {
    let fixtures = shared().join("fixtures");
    let mut versions: Vec<_> = std::fs::read_dir(&fixtures)
        .unwrap()
        .filter_map(Result::ok)
        .map(|entry| entry.path())
        .filter(|path| path.join("cases.json").exists())
        .collect();
    versions.sort();
    assert!(!versions.is_empty(), "the fixtures are found");
    let mut checked = 0;
    for directory in versions {
        let manifest: Value =
            serde_json::from_str(&std::fs::read_to_string(directory.join("cases.json")).unwrap()).unwrap();
        for case in manifest["cases"].as_array().unwrap() {
            let name = case["name"].as_str().unwrap();
            let stdout = read_capture(&directory.join(format!("{name}.stdout")));
            let stderr = read_capture(&directory.join(format!("{name}.stderr")));
            for (check, expected) in case["expect"].as_object().unwrap() {
                let actual = select(check, parse(check, &stdout, &stderr), expected);
                assert_eq!(canonical(&actual), canonical(expected), "{} {name}: {check}", directory.display());
                checked += 1;
            }
        }
    }
    assert!(checked > 30, "the cases are checked: {checked}");
}

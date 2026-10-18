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

//! Parsers for the human-oriented output of the vfox CLI. vfox has no
//! machine-readable output mode for most commands, so every format handled
//! here is documented in `docs/vfox-cli-contract.md` and covered by the
//! captured outputs in `shared/fixtures`. The parsers never fail on
//! unexpected lines, which they skip.

use super::models::{AvailablePlugin, AvailableVersion, ConfigEntry, InstalledSdk, PluginInfo, VersionAddition};
use super::terminal::{split_terminal_lines, strip_ansi};

/// The ASCII unit separator between the fields of `vfox info --format`.
pub const FIELD_SEPARATOR: char = '\u{1f}';

/// The Go template passed to `vfox info <name> --format`. The fields are
/// joined with [`FIELD_SEPARATOR`], which cannot appear in any of them, so
/// descriptions may contain new lines and any punctuation.
pub const PLUGIN_INFO_TEMPLATE: &str = "{{.Name}}{{\"\\x1f\"}}{{.Version}}{{\"\\x1f\"}}{{.Homepage}}\
    {{\"\\x1f\"}}{{.InstallPath}}{{\"\\x1f\"}}{{.Description}}";

fn lines(output: &str) -> Vec<String> {
    split_terminal_lines(&strip_ansi(output))
}

/// Parses `vfox --version`, for example `vfox version 1.0.12`.
pub fn parse_vfox_version(output: &str) -> Option<String> {
    let text = strip_ansi(output);
    let chars: Vec<char> = text.chars().collect();
    // The first `<digits>.<digits>.<digits>` and what follows it of
    // `[0-9A-Za-z.+-]`.
    for start in 0..chars.len() {
        if !chars[start].is_ascii_digit() || (start > 0 && chars[start - 1].is_ascii_digit()) {
            continue;
        }
        let mut i = start;
        let mut groups = 0;
        loop {
            let digits = chars[i..].iter().take_while(|c| c.is_ascii_digit()).count();
            if digits == 0 {
                break;
            }
            i += digits;
            groups += 1;
            if groups == 3 || chars.get(i) != Some(&'.') {
                break;
            }
            i += 1;
        }
        if groups == 3 {
            while chars.get(i).is_some_and(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '+' | '-')) {
                i += 1;
            }
            return Some(chars[start..i].iter().collect());
        }
    }
    None
}

fn strip_v_prefix(version: &str) -> String {
    match version.strip_prefix('v') {
        Some(rest) if !rest.is_empty() => rest.to_owned(),
        _ => version.to_owned(),
    }
}

/// Parses `vfox current` without arguments: lines `<name> -> v<version>`, or
/// `<name> -> N/A` when no version is in effect. The `v` prefix that vfox adds
/// is removed again. The entries keep the order of the output.
pub fn parse_current(output: &str) -> Vec<(String, Option<String>)> {
    let mut result: Vec<(String, Option<String>)> = Vec::new();
    for line in lines(output) {
        let Some(arrow) = line.find(" -> ") else { continue };
        if arrow == 0 {
            continue;
        }
        let name = line[..arrow].trim();
        let value = line[arrow + 4..].trim();
        if name.is_empty() || name.contains(' ') {
            continue;
        }
        let current = if value.is_empty() || value == "N/A" { None } else { Some(strip_v_prefix(value)) };
        match result.iter_mut().find(|(existing, _)| existing == name) {
            Some(entry) => entry.1 = current,
            None => result.push((name.to_owned(), current)),
        }
    }
    result
}

/// Parses the tree printed by `vfox list` without arguments:
///
/// ```text
/// All installed sdk versions
/// ├─┬java
/// │ ├──25.4.4.1-graalce
/// │ └──25.3.4.1-graalce
/// └──maven
/// ```
///
/// A plugin without installed versions is drawn as a leaf. The entries and
/// the version lists keep the order of the output.
pub fn parse_installed_tree(output: &str) -> Vec<(String, Vec<String>)> {
    let mut result: Vec<(String, Vec<String>)> = Vec::new();
    let mut current: Option<usize> = None;
    for line in lines(output) {
        let chars: Vec<char> = line.chars().collect();
        let indent = chars.iter().take_while(|c| **c == '│' || **c == ' ').count();
        let rest = &chars[indent..];
        if rest.len() < 3 || !matches!(rest[0], '├' | '└') || rest[1] != '─' || !matches!(rest[2], '─' | '┬')
        {
            continue;
        }
        let text: String = rest[3..].iter().collect::<String>().trim().to_owned();
        if text.is_empty() {
            continue;
        }
        if indent / 2 == 0 {
            current = Some(result.iter().position(|(name, _)| *name == text).unwrap_or_else(|| {
                result.push((text, Vec::new()));
                result.len() - 1
            }));
        } else if let Some(index) = current {
            result[index].1.push(text);
        }
    }
    result
}

/// Combines the outputs of `vfox list` and `vfox current` into SDKs. Plugins
/// that only appear in one of them are still included, so that the result
/// degrades gracefully if one command fails.
pub fn merge_installed_sdks(tree: &[(String, Vec<String>)], current: &[(String, Option<String>)]) -> Vec<InstalledSdk> {
    let mut names: Vec<&String> = tree.iter().map(|(name, _)| name).collect();
    for (name, _) in current {
        if !names.contains(&name) {
            names.push(name);
        }
    }
    names
        .into_iter()
        .map(|name| InstalledSdk {
            name: name.clone(),
            versions: tree.iter().find(|(n, _)| n == name).map(|(_, v)| v.clone()).unwrap_or_default(),
            current: current.iter().find(|(n, _)| n == name).and_then(|(_, c)| c.clone()),
        })
        .collect()
}

/// Parses `vfox search <name>` in a non-interactive terminal:
///
/// ```text
/// Available versions:
///  - 24.22.0 (LTS) [npm 11.19.1]
///  - 3.9.16 (installed)
/// ```
///
/// Every entry is `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
pub fn parse_search(output: &str) -> Vec<AvailableVersion> {
    lines(output)
        .iter()
        .filter_map(|line| line.trim_end().strip_prefix(" - ").and_then(|entry| parse_search_entry(entry.trim())))
        .collect()
}

fn parse_search_entry(entry: &str) -> Option<AvailableVersion> {
    if entry.is_empty() {
        return None;
    }
    let Some(space) = entry.find(' ') else {
        return Some(AvailableVersion {
            version: entry.to_owned(),
            note: None,
            additions: Vec::new(),
            installed: false,
        });
    };
    let version = entry[..space].to_owned();
    let mut rest = entry[space + 1..].trim();
    let mut installed = false;
    if let Some(before) = rest.strip_suffix("(installed)") {
        installed = true;
        rest = before.trim();
    }
    let mut additions = Vec::new();
    if rest.ends_with(']')
        && let Some(open) = rest.rfind('[')
    {
        additions = parse_additions(&rest[open + 1..rest.len() - 1]);
        rest = rest[..open].trim();
    }
    let note = if rest.starts_with('(') && rest.ends_with(')') && rest.len() >= 2 {
        Some(rest[1..rest.len() - 1].trim().to_owned()).filter(|note| !note.is_empty())
    } else if rest.is_empty() {
        None
    } else {
        Some(rest.to_owned())
    };
    Some(AvailableVersion { version, note, additions, installed })
}

fn parse_additions(text: &str) -> Vec<VersionAddition> {
    text.split(',')
        .map(str::trim)
        .filter(|item| !item.is_empty())
        .map(|item| match item.find(' ') {
            Some(space) => {
                VersionAddition { name: item[..space].to_owned(), version: item[space + 1..].trim().to_owned() }
            }
            None => VersionAddition { name: item.to_owned(), version: String::new() },
        })
        .collect()
}

/// Parses `vfox available`:
///
/// ```text
/// AVAILABLE PLUGINS
///
///   nodejs              ✓  https://github.com/version-fox/vfox-nodejs
///   bun                 ✗  https://github.com/arfaWong/vfox-bun
/// ```
pub fn parse_available_plugins(output: &str) -> Vec<AvailablePlugin> {
    lines(output)
        .iter()
        .filter_map(|line| {
            if !line.starts_with(char::is_whitespace) {
                return None;
            }
            let fields: Vec<&str> = line.split_whitespace().collect();
            match fields.as_slice() {
                [name, mark @ ("✓" | "✗"), homepage] => Some(AvailablePlugin {
                    name: (*name).to_owned(),
                    homepage: (*homepage).to_owned(),
                    official: *mark == "✓",
                }),
                _ => None,
            }
        })
        .collect()
}

/// Parses the output of `vfox info <name> --format` with
/// [`PLUGIN_INFO_TEMPLATE`], if it has every field.
pub fn parse_plugin_info(output: &str) -> Option<PluginInfo> {
    let text = strip_ansi(output);
    let fields: Vec<&str> = text.split(FIELD_SEPARATOR).collect();
    if fields.len() < 5 {
        return None;
    }
    Some(PluginInfo {
        name: fields[0].trim().to_owned(),
        version: fields[1].trim().to_owned(),
        homepage: fields[2].trim().to_owned(),
        install_path: fields[3].trim().to_owned(),
        // The description is the last field and may contain separators itself.
        description: fields[4..].join(&FIELD_SEPARATOR.to_string()).trim().to_owned(),
    })
}

/// Parses `vfox info <name>@<version>`, which prints the directory of an
/// installed version, or `notfound`.
pub fn parse_install_path(output: &str) -> Option<String> {
    let text = strip_ansi(output);
    let path = text.trim();
    if path.is_empty() || path == "notfound" { None } else { Some(path.to_owned()) }
}

/// Parses `vfox config --list`, which prints one `<key> = <value>` per line.
pub fn parse_config_list(output: &str) -> Vec<ConfigEntry> {
    let mut result = Vec::new();
    for line in lines(output) {
        match line.find(" = ") {
            Some(equals) if equals > 0 => {
                let key = line[..equals].trim();
                if key.is_empty() || key.contains(' ') {
                    continue;
                }
                result.push(ConfigEntry { key: key.to_owned(), value: line[equals + 3..].trim().to_owned() });
            }
            _ => {
                // `key =` with an empty value loses its trailing space when trimmed.
                let trimmed = line.trim_end();
                if let Some(key) = trimmed.strip_suffix(" =") {
                    result.push(ConfigEntry { key: key.trim().to_owned(), value: String::new() });
                }
            }
        }
    }
    result
}

/// Extracts the download progress from a line of `vfox install` output: the
/// progress bar on standard error, `Downloading...  42% [=====>    ]`. Returns
/// a fraction between 0 and 1.
pub fn parse_download_progress(line: &str) -> Option<f64> {
    let start = line.find("Downloading...")? + "Downloading...".len();
    let rest = &line[start..];
    let digits_start = rest.len() - rest.trim_start().len();
    if digits_start == 0 {
        return None;
    }
    let digits: String = rest[digits_start..].chars().take_while(char::is_ascii_digit).collect();
    if digits.is_empty() || digits.len() > 3 || !rest[digits_start + digits.len()..].starts_with('%') {
        return None;
    }
    let percent: u32 = digits.parse().ok()?;
    Some(f64::from(percent.min(100)) / 100.0)
}

/// Picks a short, human readable error message from a failed command. Usage
/// errors go to standard error. The error a command returns goes to standard
/// output up to vfox 1.0.12 and to standard error from 1.0.13, while what a
/// command prints before failing stays on standard output. So the lines of
/// standard output are read first and those of standard error last; progress
/// bar redraws and blank lines are ignored, and the last meaningful line wins,
/// unless it is the generic `failed to install <name>` trailer, in which case
/// the more specific line before it is used.
pub fn summarize_error(stdout: &str, stderr: &str) -> String {
    let mut all = meaningful_lines(stdout);
    all.extend(meaningful_lines(stderr));
    match all.as_slice() {
        [] => String::new(),
        [.., before, last] if is_install_trailer(last) => before.clone(),
        [.., last] => last.clone(),
    }
}

fn is_install_trailer(line: &str) -> bool {
    line.strip_prefix("failed to install")
        .is_some_and(|rest| rest.is_empty() || !rest.starts_with(|c: char| c.is_alphanumeric() || c == '_'))
}

fn meaningful_lines(text: &str) -> Vec<String> {
    lines(text)
        .iter()
        .map(|line| line.trim().to_owned())
        .filter(|line| !line.is_empty() && parse_download_progress(line).is_none())
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reads_release_and_pre_release_versions() {
        assert_eq!(parse_vfox_version("vfox version 1.0.12\n").as_deref(), Some("1.0.12"));
        assert_eq!(parse_vfox_version("vfox version 2.0.0-beta.1").as_deref(), Some("2.0.0-beta.1"));
        assert_eq!(parse_vfox_version("command not found"), None);
        assert_eq!(parse_vfox_version("v1.2 and 3.4.5"), Some("3.4.5".into()));
    }

    #[test]
    fn current_strips_the_v_prefix_once_and_ignores_colours_and_other_lines() {
        assert_eq!(parse_current("go -> vv1.2\n"), vec![("go".into(), Some("v1.2".into()))]);
        assert_eq!(
            parse_current("Warning: something\njava -> \u{1b}[92mv21\u{1b}[0m\nmaven -> N/A \n"),
            vec![("java".into(), Some("21".into())), ("maven".into(), None)]
        );
    }

    #[test]
    fn the_tree_handles_leaves_headers_and_blank_lines() {
        assert_eq!(parse_installed_tree("All installed sdk versions\n└──maven\n"), vec![("maven".into(), vec![])]);
        assert_eq!(
            parse_installed_tree("All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n"),
            vec![("nodejs".into(), vec!["24.11.1".into()])]
        );
    }

    #[test]
    fn merging_keeps_plugins_that_only_one_command_reported() {
        let sdks = merge_installed_sdks(
            &[("java".into(), vec!["21".into()])],
            &[("java".into(), Some("21".into())), ("go".into(), None)],
        );
        assert_eq!(
            sdks,
            vec![
                InstalledSdk { name: "java".into(), versions: vec!["21".into()], current: Some("21".into()) },
                InstalledSdk { name: "go".into(), versions: vec![], current: None },
            ]
        );
    }

    #[test]
    fn search_reads_notes_additions_and_the_installed_marker() {
        let versions = parse_search(
            "Available versions:\n - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n - 1.0 (release candidate)\n \
             - 0.1.14 [npm ]\n - 2.0\n - 1.0 preview\nPlease select a version\n -\n",
        );
        assert_eq!(versions.len(), 5);
        assert_eq!(versions[0].note.as_deref(), Some("LTS"));
        assert!(versions[0].installed);
        assert_eq!(
            versions[0].additions.iter().map(VersionAddition::label).collect::<Vec<_>>(),
            vec!["npm 11.19.1", "corepack 0.34.0"]
        );
        assert_eq!(versions[1].note.as_deref(), Some("release candidate"));
        assert_eq!(versions[2].additions[0].label(), "npm");
        assert_eq!(versions[3].note, None);
        assert_eq!(versions[4].note.as_deref(), Some("preview"));
    }

    #[test]
    fn plugin_information_keeps_separators_in_the_description() {
        let info = parse_plugin_info("a\u{1f}1.0\u{1f}https://x\u{1f}/p\u{1f}one\u{1f}two").unwrap();
        assert_eq!(info.description, "one\u{1f}two");
        assert_eq!(info.homepage, "https://x");
        assert_eq!(parse_plugin_info("nodejs not supported"), None);
        assert_eq!(parse_install_path("notfound\n"), None);
        assert_eq!(parse_install_path("/opt/sdk/v-1\n").as_deref(), Some("/opt/sdk/v-1"));
    }

    #[test]
    fn configuration_keeps_empty_values_and_values_with_spaces() {
        assert_eq!(
            parse_config_list("proxy.url = \nfoo.bar = a = b\nnot a setting\nregistry.address =\n"),
            vec![
                ConfigEntry { key: "proxy.url".into(), value: String::new() },
                ConfigEntry { key: "foo.bar".into(), value: "a = b".into() },
                ConfigEntry { key: "registry.address".into(), value: String::new() },
            ]
        );
    }

    #[test]
    fn download_progress() {
        assert_eq!(parse_download_progress("Downloading...  42% [===>   ] (1 MB/s) [1s:2s]"), Some(0.42));
        assert_eq!(parse_download_progress("Downloading... 100% [======]"), Some(1.0));
        assert_eq!(parse_download_progress("Downloading...100%"), None);
        assert_eq!(parse_download_progress("Preinstalling maven@3.9.16..."), None);
    }

    #[test]
    fn errors_prefer_the_last_meaningful_line_and_skip_the_install_trailer() {
        assert_eq!(summarize_error("Removing...\n", "Use the -y flag\n"), "Use the -y flag");
        assert_eq!(
            summarize_error(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n",
                "\rDownloading...  3% [>   ]\r      \r"
            ),
            "failed to download x@1 file, err:EOF"
        );
        assert_eq!(
            summarize_error(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n",
                "\rDownloading...  3% [>   ]\r      \rfailed to install x\n"
            ),
            "failed to download x@1 file, err:EOF"
        );
        assert_eq!(summarize_error("", "  \n"), "");
        assert_eq!(summarize_error("failed to install x\n", ""), "failed to install x");
    }
}

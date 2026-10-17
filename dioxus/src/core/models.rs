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

//! Domain models shared by every layer of the application. They mirror the
//! concepts of the vfox CLI that `docs/vfox-cli-contract.md` describes, and are
//! immutable values that compare by value.

/// An installed vfox plugin together with the SDK versions it manages.
///
/// vfox does not distinguish between a plugin and the SDK it installs: `vfox
/// add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24` installs
/// a version of the `nodejs` SDK through that plugin.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct InstalledSdk {
    /// Plugin name, for example `nodejs`.
    pub name: String,
    /// Installed versions, in the order reported by vfox (newest first).
    pub versions: Vec<String>,
    /// The version in effect, if any.
    pub current: Option<String>,
}

/// Metadata of an installed plugin, as reported by `vfox info <name>`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PluginInfo {
    /// The plugin name.
    pub name: String,
    /// Version of the plugin itself, not of the SDK.
    pub version: String,
    /// The homepage of the plugin.
    pub homepage: String,
    /// Directory in which the SDK versions of this plugin are installed.
    pub install_path: String,
    /// What the plugin installs.
    pub description: String,
}

/// A plugin listed in the vfox plugin registry (`vfox available`).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AvailablePlugin {
    /// Plugin name that is passed to `vfox add`.
    pub name: String,
    /// The homepage of the plugin.
    pub homepage: String,
    /// Whether the version-fox organisation maintains the plugin.
    pub official: bool,
}

/// A tool that is bundled with an SDK version, such as `npm` for Node.js.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct VersionAddition {
    /// The name of the tool.
    pub name: String,
    /// May be empty when the plugin omits it.
    pub version: String,
}

impl VersionAddition {
    /// As shown to users: `npm 11.19.1`, or just `npm`.
    pub fn label(&self) -> String {
        if self.version.is_empty() { self.name.clone() } else { format!("{} {}", self.name, self.version) }
    }
}

/// A version that can be installed, as listed by `vfox search <name>`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AvailableVersion {
    /// The version to pass to `vfox install <name>@<version>`.
    pub version: String,
    /// Optional remark from the plugin, for example `LTS`.
    pub note: Option<String>,
    /// Tools bundled with this version.
    pub additions: Vec<VersionAddition>,
    /// Whether vfox reported the version as installed.
    pub installed: bool,
}

/// One `key = value` line of `vfox config --list`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ConfigEntry {
    /// Dotted configuration key, for example `proxy.url`.
    pub key: String,
    /// The value as printed by vfox; empty when the value is unset.
    pub value: String,
}

/// Where a version selected with `vfox use` takes effect. vfox also knows a
/// session scope, which only exists inside an activated shell, so a desktop
/// application does not offer it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum UseScope {
    /// The default for the current user (`~/.vfox/.vfox.toml`).
    Global,
    /// Pinned in the `.vfox.toml` of a project directory.
    Project,
}

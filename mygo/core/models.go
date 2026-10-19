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

// Package core holds the domain models, the process runner, the vfox client
// and the settings store. Nothing in it depends on a user interface
// framework.
package core

// InstalledSdk is an installed vfox plugin together with the SDK versions it
// manages.
//
// vfox does not distinguish between a plugin and the SDK it installs:
// `vfox add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24`
// installs a version of the `nodejs` SDK through that plugin.
type InstalledSdk struct {
	// Plugin name, for example `nodejs`.
	Name string
	// Installed versions, in the order reported by vfox (newest first).
	Versions []string
	// The version that is currently in effect, or empty if none is.
	Current string
	// Whether a version is currently in effect.
	HasCurrent bool
}

// PluginInfo is metadata of an installed plugin, as reported by
// `vfox info <name>`.
type PluginInfo struct {
	// Plugin name.
	Name string
	// Version of the plugin itself (not of the SDK).
	Version string
	// Homepage of the plugin, usually a GitHub repository.
	Homepage string
	// Directory in which the SDK versions of this plugin are installed.
	InstallPath string
	// Human readable description.
	Description string
}

// AvailablePlugin is a plugin listed in the vfox plugin registry
// (`vfox available`).
type AvailablePlugin struct {
	// Plugin name that is passed to `vfox add`.
	Name string
	// Homepage of the plugin.
	Homepage string
	// Whether the plugin is maintained by the version-fox organisation.
	Official bool
}

// VersionAddition is a tool that is bundled with an SDK version, such as
// `npm` for Node.js.
type VersionAddition struct {
	// Name of the bundled tool.
	Name string
	// Version of the bundled tool. May be empty when the plugin omits it.
	Version string
}

// String formats the tool as shown to users: `npm 11.19.1`, or just `npm`.
func (a VersionAddition) String() string {
	if a.Version == "" {
		return a.Name
	}
	return a.Name + " " + a.Version
}

// AvailableVersion is a version that can be installed, as listed by
// `vfox search <name>`.
type AvailableVersion struct {
	// The version string to pass to `vfox install <name>@<version>`.
	Version string
	// Optional remark from the plugin, for example `LTS`.
	Note string
	// Whether the plugin gave a remark.
	HasNote bool
	// Tools bundled with this version.
	Additions []VersionAddition
	// Whether this version is already installed.
	Installed bool
}

// ConfigEntry is one `key = value` line of `vfox config --list`.
type ConfigEntry struct {
	// Dotted configuration key, for example `proxy.url`.
	Key string
	// Current value as printed by vfox. Empty when the value is unset.
	Value string
}

// UseScope is where a version selected with `vfox use` takes effect.
//
// vfox also knows a `session` scope, which only exists inside an activated
// shell. A desktop application has no shell session, so it is not offered.
//
//   - UseScopeGlobal: the default for the current user (`~/.vfox/.vfox.toml`);
//   - UseScopeProject: pinned in the `.vfox.toml` of a project directory.
type UseScope string

// The scopes of `vfox use`.
const (
	UseScopeGlobal  UseScope = "global"
	UseScopeProject UseScope = "project"
)

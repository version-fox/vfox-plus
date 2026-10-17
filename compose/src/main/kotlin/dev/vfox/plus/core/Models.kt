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

package dev.vfox.plus.core

// Domain models shared by every layer of the application. They mirror the
// concepts of the vfox CLI that docs/vfox-cli-contract.md describes, and are
// immutable values that compare by value.

/**
 * An installed vfox plugin together with the SDK versions it manages.
 *
 * vfox does not distinguish between a plugin and the SDK it installs: `vfox
 * add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24` installs
 * a version of the `nodejs` SDK through that plugin.
 */
data class InstalledSdk(
    /** Plugin name, for example `nodejs`. */
    val name: String,
    /** Installed versions, in the order reported by vfox (newest first). */
    val versions: List<String>,
    /** The version in effect, or `null` if none is. */
    val current: String?,
)

/** Metadata of an installed plugin, as reported by `vfox info <name>`. */
data class PluginInfo(
    val name: String,
    /** Version of the plugin itself, not of the SDK. */
    val version: String,
    val homepage: String,
    /** Directory in which the SDK versions of this plugin are installed. */
    val installPath: String,
    val description: String,
)

/** A plugin listed in the vfox plugin registry (`vfox available`). */
data class AvailablePlugin(
    /** Plugin name that is passed to `vfox add`. */
    val name: String,
    val homepage: String,
    /** Whether the version-fox organisation maintains the plugin. */
    val official: Boolean,
)

/** A tool that is bundled with an SDK version, such as `npm` for Node.js. */
data class VersionAddition(
    val name: String,
    /** May be empty when the plugin omits it. */
    val version: String,
) {
    /** As shown to users: `npm 11.19.1`, or just `npm`. */
    val label: String get() = if (version.isEmpty()) name else "$name $version"
}

/** A version that can be installed, as listed by `vfox search <name>`. */
data class AvailableVersion(
    /** The version to pass to `vfox install <name>@<version>`. */
    val version: String,
    /** Optional remark from the plugin, for example `LTS`. */
    val note: String?,
    /** Tools bundled with this version. */
    val additions: List<VersionAddition>,
    /** Whether vfox reported the version as installed. */
    val installed: Boolean,
)

/** One `key = value` line of `vfox config --list`. */
data class ConfigEntry(
    /** Dotted configuration key, for example `proxy.url`. */
    val key: String,
    /** The value as printed by vfox; empty when the value is unset. */
    val value: String,
)

/**
 * Where a version selected with `vfox use` takes effect. vfox also knows a
 * session scope, which only exists inside an activated shell, so a desktop
 * application does not offer it.
 */
enum class UseScope {
    /** The default for the current user (`~/.vfox/.vfox.toml`). */
    GLOBAL,

    /** Pinned in the `.vfox.toml` of a project directory. */
    PROJECT,
}

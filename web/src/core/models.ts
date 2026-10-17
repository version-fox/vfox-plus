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

/**
 * Domain models shared by every layer of the application.
 *
 * The models mirror the concepts of the vfox CLI that are described in
 * `docs/vfox-cli-contract.md`. They are plain immutable values, so that they
 * can be compared in tests and rebuilt cheaply.
 */

/**
 * An installed vfox plugin together with the SDK versions it manages.
 *
 * vfox does not distinguish between a plugin and the SDK it installs:
 * `vfox add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24`
 * installs a version of the `nodejs` SDK through that plugin.
 */
export interface InstalledSdk {
  /** Plugin name, for example `nodejs`. */
  readonly name: string;
  /** Installed versions, in the order reported by vfox (newest first). */
  readonly versions: readonly string[];
  /** The version that is currently in effect, or `null` if none is. */
  readonly current: string | null;
}

/** Metadata of an installed plugin, as reported by `vfox info <name>`. */
export interface PluginInfo {
  /** Plugin name. */
  readonly name: string;
  /** Version of the plugin itself (not of the SDK). */
  readonly version: string;
  /** Homepage of the plugin, usually a GitHub repository. */
  readonly homepage: string;
  /** Directory in which the SDK versions of this plugin are installed. */
  readonly installPath: string;
  /** Human readable description. */
  readonly description: string;
}

/** A plugin listed in the vfox plugin registry (`vfox available`). */
export interface AvailablePlugin {
  /** Plugin name that is passed to `vfox add`. */
  readonly name: string;
  /** Homepage of the plugin. */
  readonly homepage: string;
  /** Whether the plugin is maintained by the version-fox organisation. */
  readonly official: boolean;
}

/** A tool that is bundled with an SDK version, such as `npm` for Node.js. */
export interface VersionAddition {
  /** Name of the bundled tool. */
  readonly name: string;
  /** Version of the bundled tool. May be empty when the plugin omits it. */
  readonly version: string;
}

/** A version that can be installed, as listed by `vfox search <name>`. */
export interface AvailableVersion {
  /** The version string to pass to `vfox install <name>@<version>`. */
  readonly version: string;
  /** Optional remark from the plugin, for example `LTS`. */
  readonly note: string | null;
  /** Tools bundled with this version. */
  readonly additions: readonly VersionAddition[];
  /** Whether this version is already installed. */
  readonly installed: boolean;
}

/** One `key = value` line of `vfox config --list`. */
export interface ConfigEntry {
  /** Dotted configuration key, for example `proxy.url`. */
  readonly key: string;
  /** Current value as printed by vfox. Empty when the value is unset. */
  readonly value: string;
}

/**
 * Where a version selected with `vfox use` takes effect.
 *
 * vfox also knows a `session` scope, which only exists inside an activated
 * shell. A desktop application has no shell session, so it is not offered.
 *
 * - `global`: the default for the current user (`~/.vfox/.vfox.toml`);
 * - `project`: pinned in the `.vfox.toml` of a project directory.
 */
export type UseScope = 'global' | 'project';

/** Formats a bundled tool as shown to users: `npm 11.19.1`, or just `npm`. */
export function formatAddition(addition: VersionAddition): string {
  return addition.version === '' ? addition.name : `${addition.name} ${addition.version}`;
}

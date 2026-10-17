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

/** Sources passed to `vfox add --source`. */

import type { FileSystem } from '../host';

/** Whether `source` is a URL rather than a local path. */
export function isRemotePluginSource(source: string): boolean {
  return source.startsWith('http://') || source.startsWith('https://');
}

/**
 * Whether vfox would move `source` instead of reading it.
 *
 * vfox (still in 1.0.13) renames a local `.lua` plugin into its temporary
 * directory, which deletes the user's file and fails when the file is on
 * another filesystem; local `.zip` archives and URLs are only read. See
 * `docs/vfox-cli-contract.md`.
 */
export function vfoxMovesPluginSource(source: string): boolean {
  return !isRemotePluginSource(source) && source.toLowerCase().endsWith('.lua');
}

/**
 * The temporary directory of vfox when it runs with `environment`:
 * `~/.version-fox/tmp` if the legacy `~/.version-fox` exists, `~/.vfox/tmp`
 * otherwise. Without `HOME`, where vfox does not run at all, the system
 * temporary directory.
 */
export async function vfoxTemporaryDirectory(
  environment: Readonly<Record<string, string>>,
  files: FileSystem,
): Promise<string> {
  const home = environment['HOME'];
  if (home === undefined || home === '') {
    return files.systemTemporaryDirectory();
  }
  const legacy = `${home}/.version-fox`;
  return `${(await files.exists(legacy)) ? legacy : `${home}/.vfox`}/tmp`;
}

/** A plugin source that is safe to hand to vfox. */
export interface PreparedPluginSource {
  /** The path or URL to pass to `vfox add --source`. */
  readonly path: string;
  /** Whether {@link path} is a private copy of the original source. */
  readonly isCopy: boolean;
  /** Deletes the private copy, if one was made. */
  dispose(): Promise<void>;
}

/**
 * Prepares `source` for `vfox add --source`.
 *
 * A local `.lua` file is copied into a new directory under `temporaryRoot`, so
 * that vfox moves the copy and the user's file stays where it is. Pass the
 * {@link vfoxTemporaryDirectory}: vfox moves the file with a rename into a new
 * directory there, which only works on the same filesystem. Other sources are
 * returned as they are. If the copy cannot be made, the original path is
 * returned and vfox reports the problem; it fails to create its own directory
 * in `temporaryRoot` before moving anything.
 */
export async function preparePluginSource(
  source: string,
  temporaryRoot: string,
  files: FileSystem,
): Promise<PreparedPluginSource> {
  const original: PreparedPluginSource = { path: source, isCopy: false, dispose: () => Promise.resolve() };
  if (!vfoxMovesPluginSource(source)) {
    return original;
  }
  let directory: string | null = null;
  try {
    await files.createDirectory(temporaryRoot);
    const created = await files.createTemporaryDirectory(temporaryRoot, 'vfox-plus-plugin-');
    directory = created;
    const copy = `${created}/${source.split('/').pop() ?? 'plugin.lua'}`;
    await files.copyFile(source, copy);
    return { path: copy, isCopy: true, dispose: () => files.remove(created) };
  } catch {
    if (directory !== null) {
      await files.remove(directory).catch(() => undefined);
    }
    return original;
  }
}

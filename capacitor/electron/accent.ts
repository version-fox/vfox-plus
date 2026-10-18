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
 * The accent colour of the desktop, from the XDG desktop portal
 * (`org.freedesktop.appearance` `accent-color`). Electron reads it only on
 * Windows and macOS, so it is read with `gdbus` of GLib.
 */

import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

/** Runs `executable` with `args` and resolves with its standard output. */
export type CommandRunner = (executable: string, args: readonly string[]) => Promise<string>;

const runCommand: CommandRunner = async (executable, args) =>
  (await promisify(execFile)(executable, [...args], { timeout: 3000 })).stdout;

/** The accent colour of the desktop as ARGB, or `null` if it sets none. */
export async function accentColor(run: CommandRunner = runCommand): Promise<number | null> {
  try {
    const reply = await run('gdbus', [
      'call',
      '--session',
      '--timeout',
      '2',
      '--dest',
      'org.freedesktop.portal.Desktop',
      '--object-path',
      '/org/freedesktop/portal/desktop',
      '--method',
      'org.freedesktop.portal.Settings.ReadOne',
      'org.freedesktop.appearance',
      'accent-color',
    ]);
    return parseAccentColor(reply);
  } catch {
    // No portal, no session bus or no gdbus: the seed colour of the settings applies.
    return null;
  }
}

/**
 * Converts the reply of gdbus, such as `(<(0.2, 0.4, 0.6)>,)`, to ARGB. The
 * portal reports components outside of 0–1 when no accent colour is set.
 */
export function parseAccentColor(reply: string): number | null {
  const match = /\(\s*([^\s(),<>]+)\s*,\s*([^\s(),<>]+)\s*,\s*([^\s(),<>]+)\s*\)/.exec(reply);
  if (match === null) {
    return null;
  }
  const channels = match.slice(1, 4).map(Number);
  if (channels.some((channel) => !Number.isFinite(channel) || channel < 0 || channel > 1)) {
    return null;
  }
  const [red = 0, green = 0, blue = 0] = channels.map((channel) => Math.round(channel * 255));
  return (0xff000000 | (red << 16) | (green << 8) | blue) >>> 0;
}

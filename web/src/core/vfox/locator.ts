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

/** Finds the vfox executable. */

import { VFOX_EXECUTABLE_NAME } from './cli';

/**
 * Looks for the vfox executable in `PATH` and in common install locations.
 *
 * Applications started from a desktop launcher do not source the user's shell
 * profile, so directories such as `~/.local/bin` are often missing from their
 * `PATH`. The well-known install locations of the official installers and
 * package managers are therefore searched as well.
 */
export class VfoxLocator {
  constructor(
    private readonly environment: Readonly<Record<string, string>>,
    private readonly isExecutable: (path: string) => Promise<boolean>,
  ) {}

  /** Directories searched in addition to `PATH`, in order. */
  get fallbackDirectories(): string[] {
    const home = this.environment['HOME'];
    return [
      '/usr/local/bin',
      '/usr/bin',
      ...(home === undefined ? [] : [`${home}/.local/bin`, `${home}/bin`, `${home}/.linuxbrew/bin`]),
      '/home/linuxbrew/.linuxbrew/bin',
      '/snap/bin',
    ];
  }

  /** All directories that are searched, without duplicates. */
  get searchDirectories(): string[] {
    const path = (this.environment['PATH'] ?? '').split(':').filter((entry) => entry !== '');
    return [...new Set([...path, ...this.fallbackDirectories])];
  }

  /** Resolves with the path of the first vfox executable found, or `null`. */
  async locate(): Promise<string | null> {
    for (const directory of this.searchDirectories) {
      const candidate = `${directory}/${VFOX_EXECUTABLE_NAME}`;
      if (await this.isExecutable(candidate)) {
        return candidate;
      }
    }
    return null;
  }
}

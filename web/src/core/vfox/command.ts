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

/** Descriptions of vfox invocations that change state, and vfox errors. */

/**
 * The kinds of state-changing operations the application performs:
 *
 * - `install`: `vfox install --yes <name>@<version>`
 * - `uninstall`: `vfox uninstall <name>@<version>`
 * - `use`: `vfox use --global|--project <name>@<version>`
 * - `unuse`: `vfox unuse --global|--project <name>`
 * - `addPlugin`: `vfox add [--source <url>] [--alias <alias>] <name>`
 * - `removePlugin`: `vfox remove --yes <name>`
 * - `updatePlugin`: `vfox update <name>`
 * - `updateAllPlugins`: `vfox update --all`
 */
export type VfoxOperation =
  'install' | 'uninstall' | 'use' | 'unuse' | 'addPlugin' | 'removePlugin' | 'updatePlugin' | 'updateAllPlugins';

/**
 * A state-changing vfox invocation.
 *
 * Commands are values: they are built by `VfoxCli`, queued and executed by the
 * task queue, and can be retried by running them again.
 */
export interface VfoxCommand {
  /** What the command does. */
  readonly operation: VfoxOperation;
  /** Arguments passed to the vfox executable. */
  readonly arguments: readonly string[];
  /** The plugin the command acts on, if any. */
  readonly sdk: string | null;
  /** The SDK version the command acts on, if any. */
  readonly version: string | null;
  /**
   * Directory to run in. `null` means a neutral directory, so that no project
   * configuration is picked up.
   */
  readonly workingDirectory: string | null;
  /**
   * Whether vfox must believe that it runs inside an activated shell.
   *
   * `vfox use` refuses to run on Unix without shell integration, but with
   * `--global` or `--project` it only edits `.vfox.toml` and symlinks, which is
   * exactly what a GUI needs. See `docs/vfox-cli-contract.md`.
   */
  readonly requiresHookEnvironment: boolean;
}

/** Creates a command; the omitted fields take their neutral values. */
export function vfoxCommand(fields: Pick<VfoxCommand, 'operation' | 'arguments'> & Partial<VfoxCommand>): VfoxCommand {
  return {
    sdk: null,
    version: null,
    workingDirectory: null,
    requiresHookEnvironment: false,
    ...fields,
  };
}

/** The SDK and version of `command` as `name@version`, or just the name. */
export function commandSubject(command: VfoxCommand): string | null {
  if (command.sdk !== null && command.version !== null) {
    return `${command.sdk}@${command.version}`;
  }
  return command.sdk;
}

/** The command line of `command`, as shown in the task output. */
export function commandLine(command: VfoxCommand): string {
  return `vfox ${command.arguments.join(' ')}`;
}

/** A vfox invocation failed. */
export class VfoxError extends Error {
  /** The arguments vfox was started with. */
  readonly arguments: readonly string[];
  /** The exit code, if the process ran at all. */
  readonly exitCode: number | null;
  /** Standard output of the failed process. */
  readonly stdout: string;
  /** Standard error of the failed process. */
  readonly stderr: string;

  constructor(
    message: string,
    details: {
      arguments?: readonly string[];
      exitCode?: number | null;
      stdout?: string;
      stderr?: string;
    } = {},
  ) {
    super(message);
    this.name = 'VfoxError';
    this.arguments = details.arguments ?? [];
    this.exitCode = details.exitCode ?? null;
    this.stdout = details.stdout ?? '';
    this.stderr = details.stderr ?? '';
  }
}

/** The vfox executable could not be started. */
export class VfoxNotFoundError extends VfoxError {
  /** The path or name that could not be executed. */
  readonly executable: string;

  constructor(executable: string, message: string) {
    super(message);
    this.name = 'VfoxNotFoundError';
    this.executable = executable;
  }
}

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

/** The client that drives the vfox command line interface. */

import type { AvailablePlugin, AvailableVersion, ConfigEntry, InstalledSdk, PluginInfo, UseScope } from '../models';
import {
  ProcessStartError,
  type ProcessListener,
  type ProcessOutput,
  type ProcessRunner,
  type RunningProcess,
} from '../process';
import { VfoxError, VfoxNotFoundError, vfoxCommand, type VfoxCommand } from './command';
import {
  PLUGIN_INFO_TEMPLATE,
  mergeInstalledSdks,
  parseAvailablePlugins,
  parseConfigList,
  parseCurrent,
  parseInstallPath,
  parseInstalledTree,
  parsePluginInfo,
  parseSearch,
  parseVfoxVersion,
  summarizeError,
} from './output';

/** The name of the vfox executable, looked up in `PATH` when no path is known. */
export const VFOX_EXECUTABLE_NAME = 'vfox';

/**
 * Variables that tie a process to an activated shell session. They are removed
 * so that a GUI started from an activated terminal does not act on that
 * terminal's session.
 */
export const SESSION_VARIABLES: ReadonlySet<string> = new Set([
  '__VFOX_SHELL',
  '__VFOX_PID',
  '__VFOX_CURTMPPATH',
  '__VFOX_INITIALIZED',
]);

/** How vfox processes are launched. */
export interface VfoxLaunchConfig {
  /** Path to the vfox executable, or a bare name to look up in `PATH`. */
  readonly executable: string;
  /** The environment inherited from the application. */
  readonly baseEnvironment: Readonly<Record<string, string>>;
  /**
   * Variables added by the user, for example download mirrors such as
   * `VFOX_NODEJS_MIRROR`. They override {@link baseEnvironment}.
   */
  readonly extraEnvironment: Readonly<Record<string, string>>;
  /**
   * Working directory for commands that are not tied to a project.
   *
   * vfox reads a project `.vfox.toml` (and legacy files such as `.nvmrc`) from
   * its working directory, so global queries must run somewhere that cannot
   * contain one. The filesystem root is used by default.
   */
  readonly neutralDirectory: string;
}

/** Creates a launch configuration; omitted fields take their defaults. */
export function launchConfig(fields: Partial<VfoxLaunchConfig> = {}): VfoxLaunchConfig {
  return {
    executable: VFOX_EXECUTABLE_NAME,
    baseEnvironment: {},
    extraEnvironment: {},
    neutralDirectory: '/',
    ...fields,
  };
}

/** The complete environment of a vfox child process. */
export function vfoxEnvironment(
  config: VfoxLaunchConfig,
  { hookShell = false }: { hookShell?: boolean } = {},
): Record<string, string> {
  const environment: Record<string, string> = {};
  for (const [key, value] of Object.entries({ ...config.baseEnvironment, ...config.extraEnvironment })) {
    if (!SESSION_VARIABLES.has(key)) {
      environment[key] = value;
    }
  }
  // Disable colours; vfox (through pterm) honours https://no-color.org.
  environment['NO_COLOR'] = '1';
  if (hookShell) {
    environment['__VFOX_SHELL'] = 'bash';
  }
  return environment;
}

const NAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._+-]*$/;
const CONFIG_KEY_PATTERN = /^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$/;

/** Whether `name` is accepted as a plugin name or alias. */
export function isValidPluginName(name: string): boolean {
  return NAME_PATTERN.test(name);
}

/**
 * Drives the vfox CLI.
 *
 * Read-only queries are executed directly and parsed into models.
 * State-changing operations are returned as {@link VfoxCommand} values
 * instead, so that the caller can queue them, stream their output and retry
 * them.
 *
 * Every command line built here is documented in `docs/vfox-cli-contract.md`.
 */
export class VfoxCli {
  /**
   * How vfox is launched.
   *
   * Can be replaced at any time, for example after the user picked a different
   * executable. Affects subsequent invocations only.
   */
  launchConfig: VfoxLaunchConfig;

  constructor(
    private readonly runner: ProcessRunner,
    config: VfoxLaunchConfig = launchConfig(),
  ) {
    this.launchConfig = config;
  }

  // -------------------------------------------------------------------------
  // Queries
  // -------------------------------------------------------------------------

  /** The version of vfox, for example `1.0.12`. */
  async version(): Promise<string> {
    const args = ['--version'];
    const output = await this.query(args);
    const version = parseVfoxVersion(`${output.stdout}\n${output.stderr}`);
    if (version === null) {
      throw new VfoxError('Unrecognised version output', {
        arguments: args,
        stdout: output.stdout,
        stderr: output.stderr,
      });
    }
    return version;
  }

  /**
   * All installed plugins with their installed and current versions.
   *
   * Runs `vfox list` and `vfox current` and merges their results.
   */
  async installedSdks(): Promise<InstalledSdk[]> {
    const [list, currentOutput] = await Promise.all([
      this.query(['list'], { allowFailure: true }),
      this.query(['current']),
    ]);
    const current = parseCurrent(currentOutput.stdout);
    if (list.exitCode !== 0) {
      // `vfox list` fails with "you don't have any sdk installed yet" when no
      // plugin is installed; `vfox current` then prints nothing.
      if (current.size === 0) {
        return [];
      }
      throw this.failure(['list'], list);
    }
    return mergeInstalledSdks(parseInstalledTree(list.stdout), current);
  }

  /** Metadata of the installed plugin `name`. */
  async pluginInfo(name: string): Promise<PluginInfo> {
    const args = ['info', checkName(name), '--format', PLUGIN_INFO_TEMPLATE];
    const output = await this.query(args);
    const info = parsePluginInfo(output.stdout);
    if (info === null) {
      throw new VfoxError('Unrecognised plugin information', {
        arguments: args,
        stdout: output.stdout,
        stderr: output.stderr,
      });
    }
    return info;
  }

  /** The directory of an installed version, or `null` if it is not installed. */
  async installPath(name: string, version: string): Promise<string | null> {
    const output = await this.query(['info', `${checkName(name)}@${checkVersion(version)}`]);
    return parseInstallPath(output.stdout);
  }

  /**
   * Versions of `name` that can be installed, newest first.
   *
   * May take a while: plugins fetch the list from the network unless vfox has
   * it cached (see the `cache.availableHookDuration` setting).
   */
  async searchVersions(name: string): Promise<AvailableVersion[]> {
    const output = await this.query(['search', checkName(name)]);
    return parseSearch(output.stdout);
  }

  /** Plugins listed in the vfox registry. */
  async availablePlugins(): Promise<AvailablePlugin[]> {
    const output = await this.query(['available']);
    return parseAvailablePlugins(output.stdout);
  }

  /** The vfox configuration as `key = value` entries. */
  async configEntries(): Promise<ConfigEntry[]> {
    const output = await this.query(['config', '--list']);
    return parseConfigList(output.stdout);
  }

  /** Sets the configuration `key` to `value`. */
  async setConfig(key: string, value: string): Promise<void> {
    await this.query(['config', checkConfigKey(key), value]);
  }

  /** Resets the configuration `key` to its default value. */
  async unsetConfig(key: string): Promise<void> {
    await this.query(['config', '--unset', checkConfigKey(key)]);
  }

  // -------------------------------------------------------------------------
  // Commands
  // -------------------------------------------------------------------------

  /** Installs `version` of `name`. */
  install(name: string, version: string): VfoxCommand {
    return vfoxCommand({
      operation: 'install',
      arguments: ['install', '--yes', `${checkName(name)}@${checkVersion(version)}`],
      sdk: name,
      version,
    });
  }

  /** Uninstalls `version` of `name`. */
  uninstall(name: string, version: string): VfoxCommand {
    return vfoxCommand({
      operation: 'uninstall',
      arguments: ['uninstall', `${checkName(name)}@${checkVersion(version)}`],
      sdk: name,
      version,
    });
  }

  /**
   * Makes `version` of `name` the current version in `scope`.
   *
   * `projectDirectory` is required for the project scope.
   */
  use(name: string, version: string, scope: UseScope = 'global', projectDirectory?: string): VfoxCommand {
    return vfoxCommand({
      operation: 'use',
      arguments: ['use', scopeFlag(scope), `${checkName(name)}@${checkVersion(version)}`],
      sdk: name,
      version,
      workingDirectory: scopeDirectory(scope, projectDirectory),
      requiresHookEnvironment: true,
    });
  }

  /** Removes the current version of `name` from `scope`. */
  unuse(name: string, scope: UseScope = 'global', projectDirectory?: string): VfoxCommand {
    return vfoxCommand({
      operation: 'unuse',
      arguments: ['unuse', scopeFlag(scope), checkName(name)],
      sdk: name,
      workingDirectory: scopeDirectory(scope, projectDirectory),
    });
  }

  /**
   * Adds the plugin `name` from the registry, or from `source` (a URL or a
   * local path) when given. `alias` installs the plugin under another name.
   */
  addPlugin(name: string, { source, alias }: { source?: string | null; alias?: string | null } = {}): VfoxCommand {
    const trimmedSource = blankToNull(source);
    const trimmedAlias = blankToNull(alias);
    const trimmedName = blankToNull(name);
    if (trimmedName === null && trimmedSource === null) {
      throw new RangeError('A plugin name or a source is required');
    }
    const args = ['add'];
    if (trimmedSource !== null) {
      args.push('--source', trimmedSource);
    }
    if (trimmedAlias !== null) {
      args.push('--alias', checkName(trimmedAlias));
    }
    if (trimmedName !== null) {
      args.push(checkName(trimmedName));
    }
    return vfoxCommand({ operation: 'addPlugin', arguments: args, sdk: trimmedAlias ?? trimmedName });
  }

  /** Removes the plugin `name` together with all of its installed versions. */
  removePlugin(name: string): VfoxCommand {
    return vfoxCommand({ operation: 'removePlugin', arguments: ['remove', '--yes', checkName(name)], sdk: name });
  }

  /** Updates the plugin `name` to its latest version. */
  updatePlugin(name: string): VfoxCommand {
    return vfoxCommand({ operation: 'updatePlugin', arguments: ['update', checkName(name)], sdk: name });
  }

  /** Updates every installed plugin. */
  updateAllPlugins(): VfoxCommand {
    return vfoxCommand({ operation: 'updateAllPlugins', arguments: ['update', '--all'] });
  }

  /**
   * Starts `command` and resolves with the running process.
   *
   * Rejects with a {@link VfoxNotFoundError} if vfox cannot be started.
   */
  async start(command: VfoxCommand, listener: ProcessListener): Promise<RunningProcess> {
    const config = this.launchConfig;
    try {
      return await this.runner.start(
        config.executable,
        command.arguments,
        {
          workingDirectory: command.workingDirectory ?? config.neutralDirectory,
          environment: vfoxEnvironment(config, { hookShell: command.requiresHookEnvironment }),
        },
        listener,
      );
    } catch (error) {
      if (error instanceof ProcessStartError) {
        throw new VfoxNotFoundError(config.executable, error.message);
      }
      throw error;
    }
  }

  // -------------------------------------------------------------------------
  // Internals
  // -------------------------------------------------------------------------

  private async query(args: string[], { allowFailure = false } = {}): Promise<ProcessOutput> {
    const config = this.launchConfig;
    let output: ProcessOutput;
    try {
      output = await this.runner.run(config.executable, args, {
        workingDirectory: config.neutralDirectory,
        environment: vfoxEnvironment(config),
      });
    } catch (error) {
      if (error instanceof ProcessStartError) {
        throw new VfoxNotFoundError(config.executable, error.message);
      }
      throw error;
    }
    if (output.exitCode !== 0 && !allowFailure) {
      throw this.failure(args, output);
    }
    return output;
  }

  private failure(args: string[], output: ProcessOutput): VfoxError {
    const summary = summarizeError(output.stdout, output.stderr);
    return new VfoxError(summary === '' ? `vfox exited with code ${output.exitCode}` : summary, {
      arguments: args,
      exitCode: output.exitCode,
      stdout: output.stdout,
      stderr: output.stderr,
    });
  }
}

function scopeFlag(scope: UseScope): string {
  return scope === 'global' ? '--global' : '--project';
}

function scopeDirectory(scope: UseScope, projectDirectory: string | undefined): string | null {
  if (scope === 'global') {
    return null;
  }
  const directory = blankToNull(projectDirectory);
  if (directory === null) {
    throw new RangeError('A project directory is required for the project scope');
  }
  return directory;
}

/**
 * Rejects names that vfox could mistake for flags or that contain separators.
 * Arguments never pass through a shell, so this only guards against option
 * injection and typos.
 */
function checkName(name: string): string {
  if (!NAME_PATTERN.test(name)) {
    throw new RangeError(`Not a valid plugin name: ${name}`);
  }
  return name;
}

function checkVersion(version: string): string {
  if (version === '' || version.startsWith('-') || /[\s@]/.test(version)) {
    throw new RangeError(`Not a valid version: ${version}`);
  }
  return version;
}

function checkConfigKey(key: string): string {
  if (!CONFIG_KEY_PATTERN.test(key)) {
    throw new RangeError(`Not a valid configuration key: ${key}`);
  }
  return key;
}

function blankToNull(value: string | null | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed === undefined || trimmed === '' ? null : trimmed;
}

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

// Test doubles shared by the unit tests.

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import type { FileSystem } from '../../src/core/host';
import {
  ProcessStartError,
  type ProcessListener,
  type ProcessOptions,
  type ProcessOutput,
  type ProcessRunner,
  type RunningProcess,
} from '../../src/core/process';
import { PLUGIN_INFO_TEMPLATE } from '../../src/core/vfox/output';

/** The shared resources of the repository. */
export const SHARED = fileURLToPath(new URL('../../../shared', import.meta.url));

/** The captured vfox outputs in `shared/fixtures` that the tests use by default. */
export const FIXTURES_VERSION = '1.0.13';

/**
 * Reads a capture of `shared/fixtures`, in which control characters are
 * written as `\xHH` and backslashes as `\\`.
 */
export function readCapture(path: string): string {
  return readFileSync(path, 'utf8').replace(/\\(?:x([0-9a-f]{2})|\\)/g, (_, hex: string | undefined) =>
    hex === undefined ? '\\' : String.fromCharCode(Number.parseInt(hex, 16)),
  );
}

/** The result of the fixture `name`, with `exitCode`. */
export function fixtureOutput(name: string, exitCode = 0): ProcessOutput {
  const base = `${SHARED}/fixtures/vfox-${FIXTURES_VERSION}/${name}`;
  return { exitCode, stdout: readCapture(`${base}.stdout`), stderr: readCapture(`${base}.stderr`) };
}

/** One recorded invocation of a {@link FakeProcessRunner}. */
export interface ProcessCall {
  readonly executable: string;
  readonly arguments: readonly string[];
  readonly workingDirectory: string;
  readonly environment: Readonly<Record<string, string>>;
  readonly streamed: boolean;
}

/** A {@link RunningProcess} whose output and exit are driven by the test. */
export class FakeRunningProcess implements RunningProcess {
  readonly exitCode: Promise<number>;
  /** Whether {@link kill} was called. */
  killed = false;
  private listener: ProcessListener | null = null;
  private resolveExit: (code: number) => void = () => undefined;
  private exited = false;
  private readonly encoder = new TextEncoder();

  constructor() {
    this.exitCode = new Promise((resolve) => {
      this.resolveExit = resolve;
    });
  }

  /** A process that prints `output` and exits right away. */
  static completed(output: ProcessOutput): FakeRunningProcess {
    const process = new FakeRunningProcess();
    queueMicrotask(() => {
      process.writeStdout(output.stdout);
      process.writeStderr(output.stderr);
      process.exit(output.exitCode);
    });
    return process;
  }

  /** @internal */
  attach(listener: ProcessListener): void {
    this.listener = listener;
  }

  writeStdout(text: string): void {
    if (text !== '') {
      this.listener?.stdout(this.encoder.encode(text));
    }
  }

  writeStderr(text: string): void {
    if (text !== '') {
      this.listener?.stderr(this.encoder.encode(text));
    }
  }

  /** Ends the process with `code`. */
  exit(code: number): void {
    if (!this.exited) {
      this.exited = true;
      this.resolveExit(code);
    }
  }

  kill(): void {
    this.killed = true;
    this.exit(-15);
  }
}

/**
 * A scripted {@link ProcessRunner}.
 *
 * Responses are matched by the joined argument list, so tests read like the
 * command lines they expect. Unmatched invocations fail with exit code 127.
 */
export class FakeProcessRunner implements ProcessRunner {
  /** Every invocation, in order. */
  readonly invocations: ProcessCall[] = [];
  /** When set, every invocation fails as if the executable did not exist. */
  missingExecutable = false;
  private readonly responses = new Map<string, () => ProcessOutput>();
  private readonly streams = new Map<string, () => FakeRunningProcess>();

  /** Responds to `vfox <args>` with `output`. */
  on(args: string, output: ProcessOutput): this {
    this.responses.set(args, () => output);
    return this;
  }

  /** Responds to `vfox <args>` with the result of `respond`. */
  onDynamic(args: string, respond: () => ProcessOutput): this {
    this.responses.set(args, respond);
    return this;
  }

  /** Responds to a started `vfox <args>` with the process from `create`. */
  onStart(args: string, create: () => FakeRunningProcess): this {
    this.streams.set(args, create);
    return this;
  }

  /** The calls whose arguments start with `prefix`. */
  calls(prefix: string): ProcessCall[] {
    return this.invocations.filter((call) => call.arguments.join(' ').startsWith(prefix));
  }

  run(executable: string, args: readonly string[], options: ProcessOptions): Promise<ProcessOutput> {
    this.record(executable, args, options, false);
    if (this.missingExecutable) {
      return Promise.reject(new ProcessStartError(executable, 'No such file or directory'));
    }
    const respond = this.responses.get(args.join(' '));
    return Promise.resolve(respond?.() ?? { exitCode: 127, stdout: '', stderr: 'unexpected command' });
  }

  start(
    executable: string,
    args: readonly string[],
    options: ProcessOptions,
    listener: ProcessListener,
  ): Promise<RunningProcess> {
    this.record(executable, args, options, true);
    if (this.missingExecutable) {
      return Promise.reject(new ProcessStartError(executable, 'No such file or directory'));
    }
    const create = this.streams.get(args.join(' '));
    const respond = this.responses.get(args.join(' '));
    const process = create?.() ?? FakeRunningProcess.completed(respond?.() ?? { exitCode: 0, stdout: '', stderr: '' });
    process.attach(listener);
    return Promise.resolve(process);
  }

  private record(executable: string, args: readonly string[], options: ProcessOptions, streamed: boolean): void {
    this.invocations.push({
      executable,
      arguments: [...args],
      workingDirectory: options.workingDirectory,
      environment: { ...options.environment },
      streamed,
    });
  }
}

/**
 * A runner that answers the read-only commands like the `list` and `current`
 * fixtures: maven 3.9.16 (current) and 3.8.9, mvn2 without versions and nodejs
 * 24.11.1 (current) and 22.21.0.
 */
export function fixtureRunner(): FakeProcessRunner {
  return new FakeProcessRunner()
    .on('--version', fixtureOutput('version'))
    .on('list', fixtureOutput('list'))
    .on('current', fixtureOutput('current'))
    .on('available', fixtureOutput('available'))
    .on('config --list', fixtureOutput('config-list'))
    .on('search maven', fixtureOutput('search-maven-installed'))
    .on('search nodejs', fixtureOutput('search-nodejs'))
    .on(`info nodejs --format ${PLUGIN_INFO_TEMPLATE}`, fixtureOutput('info'));
}

/** A {@link FileSystem} in memory. Directories are the keys ending in `/`. */
export class MemoryFileSystem implements FileSystem {
  readonly files = new Map<string, string>();
  readonly directories = new Set<string>(['/']);
  readonly executables = new Set<string>();
  /** When set, copying files fails. */
  failCopies = false;
  private counter = 0;

  exists(path: string): Promise<boolean> {
    return Promise.resolve(this.files.has(path) || this.directories.has(path));
  }

  isDirectory(path: string): Promise<boolean> {
    return Promise.resolve(this.directories.has(path));
  }

  isExecutable(path: string): Promise<boolean> {
    return Promise.resolve(this.executables.has(path));
  }

  readTextFile(path: string): Promise<string | null> {
    return Promise.resolve(this.files.get(path) ?? null);
  }

  async writeTextFile(path: string, text: string): Promise<void> {
    await this.createDirectory(path.slice(0, path.lastIndexOf('/')));
    this.files.set(path, text);
  }

  createDirectory(path: string): Promise<void> {
    const parts = path.split('/').filter((part) => part !== '');
    for (let i = 1; i <= parts.length; i++) {
      this.directories.add(`/${parts.slice(0, i).join('/')}`);
    }
    return Promise.resolve();
  }

  async createTemporaryDirectory(parent: string, prefix: string): Promise<string> {
    const path = `${parent}/${prefix}${++this.counter}`;
    await this.createDirectory(path);
    return path;
  }

  copyFile(from: string, to: string): Promise<void> {
    const content = this.files.get(from);
    if (content === undefined || this.failCopies) {
      return Promise.reject(new Error(`Cannot copy ${from}`));
    }
    this.files.set(to, content);
    return Promise.resolve();
  }

  remove(path: string): Promise<void> {
    for (const key of [...this.files.keys()]) {
      if (key === path || key.startsWith(`${path}/`)) {
        this.files.delete(key);
      }
    }
    for (const key of [...this.directories]) {
      if (key === path || key.startsWith(`${path}/`)) {
        this.directories.delete(key);
      }
    }
    return Promise.resolve();
  }

  systemTemporaryDirectory(): Promise<string> {
    return Promise.resolve('/tmp');
  }
}

/** Waits until the pending promise callbacks have run. */
export async function settle(rounds = 10): Promise<void> {
  for (let i = 0; i < rounds; i++) {
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
}

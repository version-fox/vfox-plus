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

/** A thin, testable abstraction over spawning child processes. */

/** Everything a finished process wrote, and how it exited. */
export interface ProcessOutput {
  /** The exit code. Zero means success. */
  readonly exitCode: number;
  /** Everything written to standard output, decoded as UTF-8. */
  readonly stdout: string;
  /** Everything written to standard error, decoded as UTF-8. */
  readonly stderr: string;
}

/** Where and how a child process runs. */
export interface ProcessOptions {
  /** The working directory of the child. */
  readonly workingDirectory: string;
  /**
   * The complete environment of the child: the environment of the
   * application is not inherited implicitly, so that callers can remove
   * variables as well as add them.
   */
  readonly environment: Readonly<Record<string, string>>;
}

/** Receives the output of a started process while it runs. */
export interface ProcessListener {
  /** A chunk of raw standard output. */
  stdout(chunk: Uint8Array): void;
  /** A chunk of raw standard error. */
  stderr(chunk: Uint8Array): void;
}

/** A child process whose output is consumed while it runs. */
export interface RunningProcess {
  /**
   * Resolves with the exit code when the process has ended, after every chunk
   * of its output was delivered to the listener.
   */
  readonly exitCode: Promise<number>;
  /** Asks the process to terminate with `SIGTERM`. */
  kill(): void;
}

/**
 * Spawns child processes.
 *
 * The application never spawns processes directly, which keeps the vfox client
 * testable with a fake runner and leaves room for a remote implementation (see
 * `docs/architecture.md`). Standard input is closed immediately: nothing is
 * ever written to the child, and an open stdin would make an unexpected
 * interactive prompt wait forever.
 */
export interface ProcessRunner {
  /**
   * Runs `executable` to completion and collects its output.
   *
   * Rejects with a {@link ProcessStartError} if the executable cannot be
   * started.
   */
  run(executable: string, args: readonly string[], options: ProcessOptions): Promise<ProcessOutput>;

  /**
   * Starts `executable` and resolves as soon as it runs.
   *
   * Rejects with a {@link ProcessStartError} if the executable cannot be
   * started.
   */
  start(
    executable: string,
    args: readonly string[],
    options: ProcessOptions,
    listener: ProcessListener,
  ): Promise<RunningProcess>;
}

/** An executable could not be started, for example because it is missing. */
export class ProcessStartError extends Error {
  constructor(
    readonly executable: string,
    message: string,
  ) {
    super(message);
    this.name = 'ProcessStartError';
  }
}

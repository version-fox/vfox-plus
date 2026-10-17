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

import type { RunningProcess } from '../core/process';
import { TerminalLineSplitter } from '../core/terminal-text';
import type { VfoxCli } from '../core/vfox/cli';
import { VfoxError, type VfoxCommand, type VfoxOperation } from '../core/vfox/command';
import { parseDownloadProgress, summarizeError } from '../core/vfox/output';
import { ChangeNotifier, Emitter } from './notifier';

/**
 * The life cycle of a {@link VfoxTask}:
 *
 * - `queued`: waiting for earlier tasks to finish;
 * - `running`: the vfox process is running;
 * - `succeeded`: vfox exited with code zero;
 * - `failed`: vfox could not be started or exited with a non-zero code;
 * - `cancelled`: the user cancelled the task.
 */
export type TaskStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';

/** Whether a task with `status` has not finished yet. */
export function isActive(status: TaskStatus): boolean {
  return status === 'queued' || status === 'running';
}

/** One execution of a state-changing {@link VfoxCommand}. */
export class VfoxTask extends ChangeNotifier {
  private readonly lines: string[] = [];
  private readonly stdoutLines: string[] = [];
  private readonly stderrLines: string[] = [];
  private dropped = 0;
  /** @internal The running process, while there is one. */
  process: RunningProcess | null = null;
  /** @internal */
  statusValue: TaskStatus = 'queued';
  /** @internal */
  progressValue: number | null = null;
  /** @internal */
  startedAtValue: number | null = null;
  /** @internal */
  finishedAtValue: number | null = null;
  /** @internal */
  exitCodeValue: number | null = null;
  /** @internal */
  errorMessageValue: string | null = null;

  /** @internal Created by {@link TaskQueue}. */
  constructor(
    /** Sequential identifier, unique within a {@link TaskQueue}. */
    readonly id: number,
    /** The command that is executed. */
    readonly command: VfoxCommand,
    /** When the task was queued, in milliseconds since the epoch. */
    readonly createdAt: number,
    private readonly maxLogLines: number,
  ) {
    super();
  }

  /** The current status. */
  get status(): TaskStatus {
    return this.statusValue;
  }

  /** Download progress between 0 and 1, if vfox reported any. */
  get progress(): number | null {
    return this.progressValue;
  }

  /** When the process started. */
  get startedAt(): number | null {
    return this.startedAtValue;
  }

  /** When the task finished. */
  get finishedAt(): number | null {
    return this.finishedAtValue;
  }

  /** The exit code of vfox, if it ran to completion. */
  get exitCode(): number | null {
    return this.exitCodeValue;
  }

  /** A short description of the failure, if the task failed. */
  get errorMessage(): string | null {
    return this.errorMessageValue;
  }

  /**
   * Output lines of the process (standard output and error interleaved),
   * without progress bar redraws. Only the most recent lines are kept.
   */
  get log(): readonly string[] {
    return this.lines;
  }

  /** How many old lines were dropped from {@link log}. */
  get droppedLines(): number {
    return this.dropped;
  }

  /** The text that summarises a successful run, usually its last line. */
  get lastLine(): string | null {
    return this.lines.at(-1) ?? null;
  }

  /** @internal */
  append(line: string, isError: boolean): void {
    const progress = parseDownloadProgress(line);
    if (progress !== null) {
      this.progressValue = progress;
    } else {
      (isError ? this.stderrLines : this.stdoutLines).push(line);
      this.lines.push(line);
      if (this.lines.length > this.maxLogLines) {
        this.lines.shift();
        this.dropped++;
      }
    }
    this.notifyListeners();
  }

  /** @internal The error summary of the collected output. */
  summary(): string {
    return summarizeError(this.stdoutLines.join('\n'), this.stderrLines.join('\n'));
  }

  /** @internal */
  setStatus(status: TaskStatus): void {
    this.statusValue = status;
    this.notifyListeners();
  }
}

/** The filter of {@link TaskQueue.activeTask}. */
export interface TaskFilter {
  readonly operation?: VfoxOperation;
  readonly sdk?: string;
  readonly version?: string;
}

/**
 * Runs state-changing vfox commands one after another.
 *
 * vfox does not lock its data directory, so concurrent installs, uses and
 * removals could corrupt `.vfox.toml` or the symlinks. The queue therefore runs
 * at most one command at a time, in the order they were enqueued.
 */
export class TaskQueue extends ChangeNotifier {
  private readonly all: VfoxTask[] = [];
  private readonly finished = new Emitter<VfoxTask>();
  private nextId = 1;
  private running: VfoxTask | null = null;

  constructor(
    private readonly cli: VfoxCli,
    /** How many log lines each task keeps. */
    readonly maxLogLines = 2000,
    /** How many finished tasks are kept before the oldest are forgotten. */
    readonly maxFinishedTasks = 50,
  ) {
    super();
  }

  /** All known tasks, newest first. */
  get tasks(): VfoxTask[] {
    return [...this.all].reverse();
  }

  /** The number of queued and running tasks. */
  get activeCount(): number {
    return this.all.filter((task) => isActive(task.status)).length;
  }

  /** Whether any finished task can be cleared. */
  get hasFinished(): boolean {
    return this.all.some((task) => !isActive(task.status));
  }

  /** Calls `listener` with every task once it has finished, failed or been cancelled. */
  onFinished(listener: (task: VfoxTask) => void): () => void {
    return this.finished.listen(listener);
  }

  /**
   * Queues `command` and returns its task.
   *
   * If an identical command is already queued or running, its task is
   * returned instead of queueing the command twice.
   */
  enqueue(command: VfoxCommand): VfoxTask {
    for (const task of this.all) {
      if (isActive(task.status) && sameCommand(task.command, command)) {
        return task;
      }
    }
    const task = new VfoxTask(this.nextId++, command, Date.now(), this.maxLogLines);
    this.all.push(task);
    this.trimFinished();
    this.notifyListeners();
    this.pump();
    return task;
  }

  /** Queues the command of `task` again. */
  retry(task: VfoxTask): VfoxTask {
    return this.enqueue(task.command);
  }

  /** Cancels `task`: a queued task is dropped, a running process is killed. */
  cancel(task: VfoxTask): void {
    switch (task.status) {
      case 'queued':
        this.finish(task, 'cancelled');
        break;
      case 'running':
        task.process?.kill();
        // The status is settled in execute() once the process has exited.
        task.errorMessageValue = null;
        task.setStatus('cancelled');
        break;
      default:
        break;
    }
  }

  /** Forgets every finished task. */
  clearFinished(): void {
    for (let i = this.all.length - 1; i >= 0; i--) {
      const task = this.all[i];
      if (task !== undefined && !isActive(task.status)) {
        this.all.splice(i, 1);
      }
    }
    this.notifyListeners();
  }

  /** The active task matching `filter`, if any. */
  activeTask(filter: TaskFilter = {}): VfoxTask | null {
    return (
      this.all.find(
        (task) =>
          isActive(task.status) &&
          (filter.operation === undefined || task.command.operation === filter.operation) &&
          (filter.sdk === undefined || task.command.sdk === filter.sdk) &&
          (filter.version === undefined || task.command.version === filter.version),
      ) ?? null
    );
  }

  private pump(): void {
    if (this.running !== null || this.isDisposed) {
      return;
    }
    const next = this.all.find((task) => task.status === 'queued');
    if (next === undefined) {
      return;
    }
    this.running = next;
    void this.execute(next);
  }

  private async execute(task: VfoxTask): Promise<void> {
    task.startedAtValue = Date.now();
    task.setStatus('running');
    this.notifyListeners();
    const stdout = new TerminalLineSplitter();
    const stderr = new TerminalLineSplitter();
    try {
      const process = await this.cli.start(task.command, {
        stdout: (chunk) => stdout.push(chunk).forEach((line) => task.append(line, false)),
        stderr: (chunk) => stderr.push(chunk).forEach((line) => task.append(line, true)),
      });
      task.process = process;
      if (task.status === 'cancelled') {
        // Cancelled while the process was being spawned.
        process.kill();
      }
      const exitCode = await process.exitCode;
      stdout.close().forEach((line) => task.append(line, false));
      stderr.close().forEach((line) => task.append(line, true));
      task.exitCodeValue = exitCode;
      if (task.status === 'cancelled') {
        this.finish(task, 'cancelled');
      } else if (exitCode === 0) {
        this.finish(task, 'succeeded');
      } else {
        const summary = task.summary();
        task.errorMessageValue = summary === '' ? `vfox exited with code ${exitCode}` : summary;
        this.finish(task, 'failed');
      }
    } catch (error) {
      task.errorMessageValue = error instanceof VfoxError ? error.message : String(error);
      this.finish(task, 'failed');
    } finally {
      task.process = null;
      this.running = null;
      this.pump();
    }
  }

  private finish(task: VfoxTask, status: TaskStatus): void {
    task.finishedAtValue = Date.now();
    task.setStatus(status);
    if (!this.isDisposed) {
      this.notifyListeners();
      this.finished.emit(task);
    }
  }

  private trimFinished(): void {
    const finished = this.all.filter((task) => !isActive(task.status));
    const excess = finished.length - this.maxFinishedTasks;
    if (excess > 0) {
      const forgotten = new Set(finished.slice(0, excess));
      for (let i = this.all.length - 1; i >= 0; i--) {
        const task = this.all[i];
        if (task !== undefined && forgotten.has(task)) {
          this.all.splice(i, 1);
        }
      }
    }
  }

  override dispose(): void {
    for (const task of this.all) {
      task.process?.kill();
    }
    this.finished.close();
    super.dispose();
  }
}

function sameCommand(a: VfoxCommand, b: VfoxCommand): boolean {
  return (
    a.workingDirectory === b.workingDirectory &&
    a.arguments.length === b.arguments.length &&
    a.arguments.every((argument, index) => argument === b.arguments[index])
  );
}

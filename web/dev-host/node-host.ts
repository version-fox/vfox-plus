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
 * The host operations of `src/hosts/protocol.ts`, implemented with Node.js.
 * Used by the development host and by the Electron main process of the
 * Capacitor implementation.
 */

import { spawn, type ChildProcess } from 'node:child_process';
import { access, copyFile, mkdir, mkdtemp, readFile, rename, rm, stat, writeFile } from 'node:fs/promises';
import { constants, tmpdir } from 'node:os';
import { dirname, join } from 'node:path';

import type {
  HostEvent,
  HostMethod,
  HostMethods,
  HostRequest,
  HostResponse,
  RunResult,
} from '../src/hosts/protocol.ts';

/** The operations that depend on the desktop, which every user of this module provides. */
export interface DesktopOperations {
  accentColor(): Promise<number | null>;
  openUrl(url: string): Promise<void>;
  pickDirectory(initialDirectory: string | null): Promise<string | null>;
  pickFile(initialDirectory: string | null, label: string | null, extensions: string[] | null): Promise<string | null>;
  copyText(text: string): Promise<void>;
  exit(code: number): Promise<void>;
  log(message: string): Promise<void>;
  /** The environment the front end sees; defaults to `process.env`. */
  environment?(): Record<string, string>;
}

/** The executable of `child` could not be started. */
class StartError extends Error {}

/**
 * Handles the requests of one front end. Processes it started are killed by
 * {@link NodeHost.close}.
 */
export class NodeHost {
  private nextProcess = 1;
  private readonly processes = new Map<number, ChildProcess>();

  constructor(
    private readonly desktop: DesktopOperations,
    private readonly emit: (event: HostEvent) => void,
  ) {}

  /** Executes `request` and returns its response. */
  async handle(request: HostRequest): Promise<HostResponse> {
    try {
      const result = await this.dispatch(request.method, request.params);
      return { id: request.id, result };
    } catch (error) {
      return {
        id: request.id,
        error: { kind: error instanceof StartError ? 'start' : 'error', message: describe(error) },
      };
    }
  }

  /** Kills every process that is still running. */
  close(): void {
    for (const child of this.processes.values()) {
      child.kill('SIGTERM');
    }
    this.processes.clear();
  }

  private async dispatch<M extends HostMethod>(method: M, params: HostMethods[M]['params']): Promise<unknown> {
    const p = params as unknown[];
    const text = (index: number): string => String(p[index]);
    switch (method) {
      case 'environment':
        return this.desktop.environment?.() ?? environmentOf(process.env);
      case 'run':
        return this.run(text(0), p[1] as string[], text(2), p[3] as Record<string, string>);
      case 'start':
        return this.start(text(0), p[1] as string[], text(2), p[3] as Record<string, string>);
      case 'kill':
        this.processes.get(Number(p[0]))?.kill('SIGTERM');
        return null;
      case 'exists':
        return exists(text(0));
      case 'isDirectory':
        return stat(text(0)).then(
          (info) => info.isDirectory(),
          () => false,
        );
      case 'isExecutable':
        return stat(text(0)).then(
          // Any of the user, group or other execute bits.
          (info) => info.isFile() && (info.mode & 0o111) !== 0,
          () => false,
        );
      case 'readTextFile':
        return readFile(text(0), 'utf8').catch((error: unknown) => {
          if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
            return null;
          }
          throw error;
        });
      case 'writeTextFile': {
        const path = text(0);
        await mkdir(dirname(path), { recursive: true });
        const temporary = `${path}.tmp`;
        await writeFile(temporary, text(1), { encoding: 'utf8', flush: true });
        await rename(temporary, path);
        return null;
      }
      case 'createDirectory':
        await mkdir(text(0), { recursive: true });
        return null;
      case 'createTemporaryDirectory':
        return mkdtemp(join(text(0), text(1)));
      case 'copyFile':
        await copyFile(text(0), text(1));
        return null;
      case 'remove':
        await rm(text(0), { recursive: true, force: true });
        return null;
      case 'systemTemporaryDirectory':
        return tmpdir();
      case 'accentColor':
        return this.desktop.accentColor();
      case 'openUrl':
        await this.desktop.openUrl(text(0));
        return null;
      case 'pickDirectory':
        return this.desktop.pickDirectory(p[0] as string | null);
      case 'pickFile':
        return this.desktop.pickFile(p[0] as string | null, p[1] as string | null, p[2] as string[] | null);
      case 'copyText':
        await this.desktop.copyText(text(0));
        return null;
      case 'exit':
        await this.desktop.exit(Number(p[0]));
        return null;
      case 'log':
        await this.desktop.log(text(0));
        return null;
      default:
        throw new Error(`Unknown method ${String(method)}`);
    }
  }

  private run(executable: string, args: string[], cwd: string, env: Record<string, string>): Promise<RunResult> {
    return new Promise((resolve, reject) => {
      const stdout: Buffer[] = [];
      const stderr: Buffer[] = [];
      const child = spawnChild(executable, args, cwd, env);
      child.once('error', (error) => {
        reject(new StartError(describe(error)));
      });
      child.stdout?.on('data', (chunk: Buffer) => stdout.push(chunk));
      child.stderr?.on('data', (chunk: Buffer) => stderr.push(chunk));
      child.once('close', (code, signal) => {
        resolve({
          exitCode: exitCodeOf(code, signal),
          stdout: Buffer.concat(stdout).toString('utf8'),
          stderr: Buffer.concat(stderr).toString('utf8'),
        });
      });
    });
  }

  private start(executable: string, args: string[], cwd: string, env: Record<string, string>): Promise<number> {
    return new Promise((resolve, reject) => {
      const id = this.nextProcess++;
      const child = spawnChild(executable, args, cwd, env);
      let started = false;
      child.once('error', (error) => {
        if (!started) {
          reject(new StartError(describe(error)));
        }
      });
      child.once('spawn', () => {
        started = true;
        this.processes.set(id, child);
        resolve(id);
      });
      const forward = (stream: 'stdout' | 'stderr') => (chunk: Buffer) => {
        this.emit({ event: 'output', process: id, stream, data: chunk.toString('base64') });
      };
      child.stdout?.on('data', forward('stdout'));
      child.stderr?.on('data', forward('stderr'));
      // `close` comes after the last output of both streams.
      child.once('close', (code, signal) => {
        if (started) {
          this.processes.delete(id);
          this.emit({ event: 'exit', process: id, code: exitCodeOf(code, signal) });
        }
      });
    });
  }
}

function spawnChild(executable: string, args: string[], cwd: string, env: Record<string, string>): ChildProcess {
  // stdin is closed: vfox then never prompts. No shell is involved.
  return spawn(executable, args, { cwd, env, stdio: ['ignore', 'pipe', 'pipe'], shell: false });
}

/** Signals count as the usual 128 + signal number of a shell. */
function exitCodeOf(code: number | null, signal: NodeJS.Signals | null): number {
  if (code !== null) {
    return code;
  }
  return signal === null ? -1 : 128 + constants.signals[signal];
}

async function exists(path: string): Promise<boolean> {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

/** The defined variables of `env`. */
export function environmentOf(env: NodeJS.ProcessEnv): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(env)) {
    if (value !== undefined) {
      result[key] = value;
    }
  }
  return result;
}

function describe(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

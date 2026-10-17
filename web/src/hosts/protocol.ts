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
 * The messages between the front end and a host that runs in another process
 * and is reached through a message channel: the development host of
 * `dev-host/` (over a WebSocket) and the Electron main process of the
 * Capacitor implementation (over IPC).
 *
 * Every request carries an `id` that its response repeats. A started process
 * has its own `process` id; its output and its exit are sent as events, in the
 * order they happened, the exit last.
 */

/** The operations of the host, by name, with their parameters and results. */
export interface HostMethods {
  environment: { params: []; result: Record<string, string> };
  run: { params: [executable: string, args: string[], cwd: string, env: Record<string, string>]; result: RunResult };
  start: { params: [executable: string, args: string[], cwd: string, env: Record<string, string>]; result: number };
  kill: { params: [process: number]; result: null };
  exists: { params: [path: string]; result: boolean };
  isDirectory: { params: [path: string]; result: boolean };
  isExecutable: { params: [path: string]; result: boolean };
  readTextFile: { params: [path: string]; result: string | null };
  writeTextFile: { params: [path: string, text: string]; result: null };
  createDirectory: { params: [path: string]; result: null };
  createTemporaryDirectory: { params: [parent: string, prefix: string]; result: string };
  copyFile: { params: [from: string, to: string]; result: null };
  remove: { params: [path: string]; result: null };
  systemTemporaryDirectory: { params: []; result: string };
  accentColor: { params: []; result: number | null };
  openUrl: { params: [url: string]; result: null };
  pickDirectory: { params: [initialDirectory: string | null]; result: string | null };
  pickFile: {
    params: [initialDirectory: string | null, label: string | null, extensions: string[] | null];
    result: string | null;
  };
  copyText: { params: [text: string]; result: null };
  exit: { params: [code: number]; result: null };
  log: { params: [message: string]; result: null };
}

/** The name of a host operation. */
export type HostMethod = keyof HostMethods;

/** The result of a finished process; the output is decoded as UTF-8. */
export interface RunResult {
  readonly exitCode: number;
  readonly stdout: string;
  readonly stderr: string;
}

/** A request from the front end. */
export interface HostRequest<M extends HostMethod = HostMethod> {
  readonly id: number;
  readonly method: M;
  readonly params: HostMethods[M]['params'];
}

/**
 * The response to a request. `start` errors mean that the executable could not
 * be started at all.
 */
export type HostResponse =
  | { readonly id: number; readonly result: unknown }
  | { readonly id: number; readonly error: { readonly kind: 'start' | 'error'; readonly message: string } };

/** Output or the exit of a started process. Output is base64-encoded. */
export type HostEvent =
  | { readonly event: 'output'; readonly process: number; readonly stream: 'stdout' | 'stderr'; readonly data: string }
  | { readonly event: 'exit'; readonly process: number; readonly code: number };

/** Decodes the base64 `data` of an output event. */
export function decodeBase64(data: string): Uint8Array {
  const binary = atob(data);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

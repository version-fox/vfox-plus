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

/** A {@link Host} that forwards every operation over a message channel. */

import type { FileFilter, FileSystem, Host } from '../core/host';
import {
  ProcessStartError,
  type ProcessListener,
  type ProcessOptions,
  type ProcessOutput,
  type RunningProcess,
} from '../core/process';
import {
  decodeBase64,
  type HostEvent,
  type HostMethod,
  type HostMethods,
  type HostRequest,
  type HostResponse,
} from './protocol';

/** Carries the messages of the host protocol. */
export interface Channel {
  /** Sends a request. */
  send(request: HostRequest): void;
  /** Calls `receive` with every response and event. */
  onMessage(receive: (message: HostResponse | HostEvent) => void): void;
}

/** A host error that is not about starting a process. */
export class HostError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'HostError';
  }
}

interface Pending {
  resolve(value: unknown): void;
  reject(error: Error): void;
  executable: string;
}

interface Started {
  listener: ProcessListener;
  exit(code: number): void;
}

/** Creates a host on top of `channel`. */
export function channelHost(name: string, channel: Channel): Host {
  let nextId = 1;
  const pending = new Map<number, Pending>();
  const processes = new Map<number, Started>();
  // Events of a process can arrive before the response to `start` is handled.
  const early = new Map<number, HostEvent[]>();

  const dispatch = (event: HostEvent): void => {
    const process = processes.get(event.process);
    if (process === undefined) {
      early.set(event.process, [...(early.get(event.process) ?? []), event]);
      return;
    }
    if (event.event === 'output') {
      const bytes = decodeBase64(event.data);
      if (event.stream === 'stdout') {
        process.listener.stdout(bytes);
      } else {
        process.listener.stderr(bytes);
      }
    } else {
      processes.delete(event.process);
      process.exit(event.code);
    }
  };

  channel.onMessage((message) => {
    if ('event' in message) {
      dispatch(message);
      return;
    }
    const request = pending.get(message.id);
    if (request === undefined) {
      return;
    }
    pending.delete(message.id);
    if ('error' in message) {
      request.reject(
        message.error.kind === 'start'
          ? new ProcessStartError(request.executable, message.error.message)
          : new HostError(message.error.message),
      );
    } else {
      request.resolve(message.result);
    }
  });

  const call = <M extends HostMethod>(
    method: M,
    ...params: HostMethods[M]['params']
  ): Promise<HostMethods[M]['result']> =>
    new Promise((resolve, reject) => {
      const id = nextId++;
      const executable = method === 'run' || method === 'start' ? String(params[0]) : '';
      pending.set(id, { resolve: resolve, reject, executable });
      channel.send({ id, method, params });
    });

  const files: FileSystem = {
    exists: (path) => call('exists', path),
    isDirectory: (path) => call('isDirectory', path),
    isExecutable: (path) => call('isExecutable', path),
    readTextFile: (path) => call('readTextFile', path),
    writeTextFile: async (path, text) => {
      await call('writeTextFile', path, text);
    },
    createDirectory: async (path) => {
      await call('createDirectory', path);
    },
    createTemporaryDirectory: (parent, prefix) => call('createTemporaryDirectory', parent, prefix),
    copyFile: async (from, to) => {
      await call('copyFile', from, to);
    },
    remove: async (path) => {
      await call('remove', path);
    },
    systemTemporaryDirectory: () => call('systemTemporaryDirectory'),
  };

  return {
    name,
    files,
    environment: () => call('environment'),
    run: async (executable: string, args: readonly string[], options: ProcessOptions): Promise<ProcessOutput> =>
      call('run', executable, [...args], options.workingDirectory, { ...options.environment }),
    start: async (
      executable: string,
      args: readonly string[],
      options: ProcessOptions,
      listener: ProcessListener,
    ): Promise<RunningProcess> => {
      const id = await call('start', executable, [...args], options.workingDirectory, { ...options.environment });
      let resolveExit: (code: number) => void = () => undefined;
      const exitCode = new Promise<number>((resolve) => {
        resolveExit = resolve;
      });
      processes.set(id, { listener, exit: resolveExit });
      for (const event of early.get(id) ?? []) {
        dispatch(event);
      }
      early.delete(id);
      return {
        exitCode,
        kill: () => {
          void call('kill', id);
        },
      };
    },
    accentColor: () => call('accentColor'),
    openUrl: async (url) => {
      await call('openUrl', url);
    },
    pickDirectory: (initialDirectory) => call('pickDirectory', initialDirectory ?? null),
    pickFile: (initialDirectory?: string, filter?: FileFilter) =>
      call(
        'pickFile',
        initialDirectory ?? null,
        filter?.label ?? null,
        filter === undefined ? null : [...filter.extensions],
      ),
    copyText: async (text) => {
      await call('copyText', text);
    },
    exit: async (code) => {
      await call('exit', code);
    },
    log: async (message) => {
      await call('log', message);
    },
    resizeWindow: async (width, height) => {
      await call('resizeWindow', width, height);
    },
  };
}

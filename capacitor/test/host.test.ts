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

// @vitest-environment happy-dom

// The VfoxHost plugin and the host of the front end on top of it, with the
// bridge of the Electron preload script replaced by a test double.

import { describe, expect, it } from 'vitest';

import {
  createCapacitorHost,
  VfoxHost,
  VfoxHostWeb,
  webImplementation,
  type HostMessage,
} from '../src/capacitor/vfox-host';
import { HOST_CHANNEL, type HostBridge } from '../electron/channel';
import type { HostRequest } from '@web/hosts/protocol';

/** A bridge that answers every request with `answer` and records them. */
function fakeBridge(answer: (request: HostRequest) => HostMessage[]): HostBridge & { requests: HostRequest[] } {
  let receive: (message: unknown) => void = () => undefined;
  const requests: HostRequest[] = [];
  return {
    requests,
    send(request) {
      requests.push(request as HostRequest);
      queueMicrotask(() => {
        answer(request as HostRequest).forEach((message) => {
          receive(message);
        });
      });
    },
    onMessage(listener) {
      receive = listener;
    },
  };
}

describe('VfoxHost', () => {
  it('carries the host protocol over the bridge', async () => {
    const bridge = fakeBridge((request) =>
      request.method === 'environment' ? [{ id: request.id, result: { HOME: '/home/me' } }] : [],
    );
    const host = await createCapacitorHost(new VfoxHostWeb(bridge));
    expect(host.name).toBe('Capacitor');
    expect(await host.environment()).toEqual({ HOME: '/home/me' });
    expect(bridge.requests).toEqual([{ id: 1, method: 'environment', params: [] }]);
  });

  it('streams the output of processes', async () => {
    const bridge = fakeBridge((request) =>
      request.method === 'start'
        ? [
            { id: request.id, result: 7 },
            { event: 'output', process: 7, stream: 'stdout', data: btoa('Install nodejs@24.11.0 success!') },
            { event: 'exit', process: 7, code: 0 },
          ]
        : [],
    );
    const host = await createCapacitorHost(new VfoxHostWeb(bridge));
    const output: string[] = [];
    const process = await host.start(
      'vfox',
      ['install', 'nodejs@24.11.0'],
      { workingDirectory: '/', environment: {} },
      {
        stdout: (bytes) => output.push(new TextDecoder().decode(bytes)),
        stderr: () => undefined,
      },
    );
    expect(await process.exitCode).toBe(0);
    expect(output).toEqual(['Install nodejs@24.11.0 success!']);
  });

  it('needs the Electron shell', async () => {
    await expect(VfoxHost.send({ request: { id: 1, method: 'environment', params: [] } })).rejects.toThrow(
      'needs its Electron shell',
    );
  });

  it('uses the bridge of the preload script', () => {
    const bridge = fakeBridge(() => []);
    Object.defineProperty(window, 'vfoxHost', { value: bridge, configurable: true });
    void webImplementation().send({ request: { id: 1, method: 'log', params: ['e2e: home'] } });
    expect(bridge.requests).toEqual([{ id: 1, method: 'log', params: ['e2e: home'] }]);
    expect(HOST_CHANNEL).toBe('vfox-host');
  });
});

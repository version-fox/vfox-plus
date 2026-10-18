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
 * The host of the Tauri implementation (`tauri/`). Every request of the host
 * protocol goes to the `host_call` command; `start` goes to `host_start`
 * together with a channel that carries the output and the exit of the
 * process, in order.
 */

import { Channel, invoke } from '@tauri-apps/api/core';

import type { Host } from '../core/host';
import { channelHost } from './channel-host';
import type { HostEvent, HostResponse } from './protocol';

/** Connects to the Rust side of the Tauri application. */
export function createTauriHost(): Promise<Host> {
  let receive: (message: HostResponse | HostEvent) => void = () => undefined;
  const host = channelHost('Tauri', {
    send: (request) => {
      const reply = (response: HostResponse): void => {
        receive(response);
      };
      const fail = (error: unknown): void => {
        receive({ id: request.id, error: { kind: 'error', message: String(error) } });
      };
      if (request.method === 'start') {
        const events = new Channel<HostEvent>();
        events.onmessage = (event) => {
          receive(event);
        };
        invoke<HostResponse>('host_start', { request, events }).then(reply, fail);
      } else {
        invoke<HostResponse>('host_call', { request }).then(reply, fail);
      }
    },
    onMessage: (handler) => {
      receive = handler;
    },
  });
  return Promise.resolve(host);
}

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
 * The host of the Wails implementation (`wails/`). Every request of the host
 * protocol goes to the `Call` method of the bound Go service `host.Service`;
 * the output and the exit of started processes arrive as `vfox-plus:process`
 * events, in order.
 */

import { Call, Events } from '@wailsio/runtime';

import type { Host } from '../core/host';
import { channelHost } from './channel-host';
import type { HostEvent, HostResponse } from './protocol';

/** The fully qualified name of the Go method that answers the host protocol. */
const CALL = 'github.com/version-fox/vfox-plus/wails/internal/host.Service.Call';

/** The event that carries process output and exits. */
const PROCESS_EVENT = 'vfox-plus:process';

/** Connects to the Go side of the Wails application. */
export function createWailsHost(): Promise<Host> {
  let receive: (message: HostResponse | HostEvent) => void = () => undefined;
  Events.On(PROCESS_EVENT, (event: { data: HostEvent }) => {
    receive(event.data);
  });
  const host = channelHost('Wails', {
    send: (request) => {
      (Call.ByName(CALL, request) as Promise<HostResponse>).then(
        (response) => {
          receive(response);
        },
        (error: unknown) => {
          receive({ id: request.id, error: { kind: 'error', message: String(error) } });
        },
      );
    },
    onMessage: (handler) => {
      receive = handler;
    },
  });
  return Promise.resolve(host);
}

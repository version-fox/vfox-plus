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
 * The native side of the front end as a Capacitor plugin, `VfoxHost`, which
 * carries the messages of the host protocol (../web/src/hosts/protocol.ts).
 *
 * Capacitor has no Linux platform, so the page runs in Electron, where
 * Capacitor sees the web platform: the web implementation of the plugin
 * forwards the messages to the Electron main process through the bridge of
 * electron/preload.ts. An Android or iOS app would implement the same plugin
 * natively and keep the rest of the front end.
 */

import { registerPlugin, WebPlugin, type PluginListenerHandle } from '@capacitor/core';

import type { Host } from '@web/core/host';
import { channelHost } from '@web/hosts/channel-host';
import type { HostEvent, HostRequest, HostResponse } from '@web/hosts/protocol';
import type { HostBridge } from '../../electron/channel';

declare global {
  interface Window {
    /** The bridge of electron/preload.ts; missing outside of Electron. */
    readonly vfoxHost?: HostBridge;
  }
}

/** A message from the host: a response or the event of a process. */
export type HostMessage = HostResponse | HostEvent;

/** The plugin: sends requests and reports what the host sends back as `message` events. */
export interface VfoxHostPlugin {
  send(options: { request: HostRequest }): Promise<void>;
  addListener(eventName: 'message', listener: (message: HostMessage) => void): Promise<PluginListenerHandle>;
}

/** The web implementation, on top of the bridge to the Electron main process. */
export class VfoxHostWeb extends WebPlugin implements VfoxHostPlugin {
  constructor(private readonly bridge: HostBridge) {
    super();
    bridge.onMessage((message) => {
      this.notifyListeners('message', message);
    });
  }

  send({ request }: { request: HostRequest }): Promise<void> {
    this.bridge.send(request);
    return Promise.resolve();
  }
}

/** The web implementation of the page, on the bridge of the Electron preload script. */
export function webImplementation(): VfoxHostWeb {
  if (window.vfoxHost === undefined) {
    throw new Error('vfox+ needs its Electron shell; run `npm start`.');
  }
  return new VfoxHostWeb(window.vfoxHost);
}

/** The plugin of this page. */
export const VfoxHost = registerPlugin<VfoxHostPlugin>('VfoxHost', { web: webImplementation });

/** A {@link Host} on top of `plugin`. */
export async function createCapacitorHost(plugin: VfoxHostPlugin = VfoxHost): Promise<Host> {
  const receivers: ((message: HostMessage) => void)[] = [];
  await plugin.addListener('message', (message) => {
    receivers.forEach((receive) => {
      receive(message);
    });
  });
  return channelHost('Capacitor', {
    send: (request) => {
      void plugin.send({ request });
    },
    onMessage: (receive) => {
      receivers.push(receive);
    },
  });
}

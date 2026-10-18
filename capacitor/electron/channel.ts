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
 * The IPC channel that carries the messages of the host protocol
 * (../web/src/hosts/protocol.ts) between the page and the main process:
 * requests from the page, responses and process events back to it.
 */
export const HOST_CHANNEL = 'vfox-host';

/** What the preload script gives the page, as `window.vfoxHost`. */
export interface HostBridge {
  /** Sends a request to the main process. */
  send(request: unknown): void;
  /** Calls `receive` with every response and event of the main process. */
  onMessage(receive: (message: unknown) => void): void;
}

/*
 * Copyright 2026 Han Li and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { Inventory, Task, TaskStatus, WebState, Pairing, Profile, PluginInfo, VersionEntry } from "./api";

// The mutable session state of the single page. One object on purpose: every
// tab reads the same inventory, tasks and detail, like the single page it is.
export type TabName = "home" | "sdks" | "plugins" | "tasks" | "settings";

export interface DetailState {
  name: string;
  tab: "installed" | "available" | "options";
  info: PluginInfo | null;
  profile: Profile | null;
  versions: VersionEntry[] | null;
  error: string | null;
  search: string;
  facets: Record<string, Record<string, boolean>>;
  query: string;
  limit: number;
  loading: boolean;
  token: number;
  installedKey?: string;
}

export const state = {
  tab: null as TabName | null,
  inventory: null as Inventory | null,
  tasks: [] as Task[],
  prevTasks: {} as Record<number, TaskStatus>,
  detail: null as DetailState | null,
  detailName: null as string | null,
  web: null as WebState | null,
  pairing: null as Pairing | null,
  live: false,
  langMode: "system" as string,
};

export function setLive(v: boolean): void {
  state.live = v;
  const live = document.getElementById("live");
  if (live) live.className = "live" + (v ? " on" : "");
  const text = document.getElementById("liveText");
  if (text) text.textContent = v ? "live" : "…";
}

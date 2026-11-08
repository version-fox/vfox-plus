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
  filter: string;
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

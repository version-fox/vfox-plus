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

import "./theme.css";
import { api, showError } from "./api";
import { el } from "./dom";
import { setLang, systemLang, t } from "./i18n";
import { currentRoute, switchTab } from "./router";
import { setLive, state, type TabName } from "./state";
import { renderHome, initHome } from "./home";
import { initPlugins, renderPlugins } from "./plugins";
import { initSdks, renderSdks, showDetail, showList, updateDetail } from "./sdks";
import {
  initSettings,
  loadWeb,
  renderSettings,
  wireLanguageSegs,
  wirePhoneCard,
} from "./settings";
import { initTasks, renderTasks, watchTasks } from "./tasks";

// The static labels of the chrome, in the current language.
export function renderChrome(): void {
  const names: Record<TabName, string> = {
    home: t("home"),
    sdks: t("sdks"),
    plugins: t("plugins"),
    tasks: t("tasks"),
    settings: t("settings"),
  };
  document.querySelectorAll(".dest").forEach((b) => {
    const s = b.querySelector(".lbl");
    const tab = (b as HTMLButtonElement).dataset.tab as TabName | undefined;
    if (s && tab && names[tab]) s.textContent = names[tab];
  });
  el("h-installed").textContent = t("installed");
  el("pickSdk").textContent = t("pick");
  el("h-add").textContent = t("add");
  el("addBtn").textContent = t("add");
  el("uploadBtn").textContent = t("uploadAdd");
  el("h-registry").textContent = t("registry");
  el("updateAllBtn").textContent = t("updateAll");
  (document.querySelector("#homeRefresh span:last-child") as HTMLElement).textContent = t("refresh");
  (document.getElementById("sdkRefresh") as HTMLElement).title = t("refresh");
  (document.getElementById("regRefresh") as HTMLElement).title = t("refresh");
  el("h-tasks").textContent = t("tasksH");
  el("clearBtn").textContent = t("clear");
  el("h-gui").textContent = t("app");
  el("h-config").textContent = t("config");
  el("l-lang").textContent = t("language");
  el("l-env").textContent = t("extraEnv");
  (document.querySelector('#langSeg [data-l="system"]') as HTMLElement).textContent = t("langSystem");
  el("l-pageLang").textContent = t("pageLang");
  (document.querySelector('#pageLangSeg [data-l="system"]') as HTMLElement).textContent = t("langSystem");
  document.querySelectorAll("#pageLangSeg button").forEach((b) =>
    b.setAttribute("aria-pressed", String((b as HTMLButtonElement).dataset.l === state.langMode)),
  );
  el("saveVfox").textContent = t("save");
  el("saveEnv").textContent = t("save");
  el("h-phone").textContent = t("phone");
  el("phoneLead").textContent = t("phoneLead");
  (document.querySelector("#phonePair span") as HTMLElement).textContent = t("phonePair");
  (document.querySelector("#phoneOut span") as HTMLElement).textContent = t("phoneOut");
  el("homeTitle").textContent = "vfox-plus";
  el("h-active").textContent = t("active");
  el("h-shell").textContent = t("shell");
  el("shellBody").textContent = t("shellBody");
  el("h-explore").textContent = t("explore");
  el("exploreBody").textContent = t("exploreBody");
  (document.querySelector("#exploreBtn span") as HTMLElement).textContent = t("plugins");
  el("h-docs").textContent = t("docsCard");
  document.querySelectorAll("#fabAdd span:last-child, #fabAddM span:last-child").forEach((e) => {
    e.textContent = t("add");
  });
  (document.getElementById("sdkSearch") as HTMLInputElement).placeholder = t("searchSdks");
  (document.getElementById("pluginSearch") as HTMLInputElement).placeholder = t("searchPlugins");
  el("liveText").textContent = state.live ? "live" : "…";
  const pageTitle = document.getElementById("pageTitle");
  if (pageTitle) pageTitle.textContent = t(state.tab === "sdks" ? "sdks" : state.tab || "home");
}

/* The data of the page: inventory, tasks, and the detail that is open. */
export async function refresh(full: boolean): Promise<void> {
  try {
    const inv = await api("/api/inventory");
    state.inventory = inv;
    const v = document.getElementById("vfoxVer");
    if (v) v.textContent = inv.vfox.version ? "vfox " + inv.vfox.version : inv.vfox.availability || "";
    renderHome(inv);
    renderSdks();
    renderPlugins();
    renderSettings();
    const sub = document.getElementById("homeSub");
    if (sub) sub.textContent = inv.vfox.version ? "vfox " + inv.vfox.version : t("vfoxMissing");
  } catch (e) {
    showError(e instanceof Error ? e.message : String(e));
  }
  try {
    const tasks = await api("/api/tasks");
    watchTasks(tasks);
    state.tasks = tasks;
    renderTasks();
  } catch (e) {
    showError(e instanceof Error ? e.message : String(e));
  }
  if (full && state.detail) showDetail(state.detail.name, true);
  else if (state.detail) updateDetail();
}

function connectEvents(): void {
  let es: EventSource;
  try {
    es = new EventSource("/api/events");
  } catch {
    setLive(false);
    return;
  }
  es.onopen = () => setLive(true);
  es.onerror = () => {
    setLive(false);
    es.close();
    window.setTimeout(connectEvents, 3000);
  };
  es.addEventListener("task", () => refresh(false));
  es.addEventListener("inventory", () => refresh(false));
  es.addEventListener("vfox", () => refresh(false));
  es.addEventListener("web", () => loadWeb());
  es.addEventListener("hello", () => refresh(true));
}

function route(): void {
  const r = currentRoute();
  if (r.sdk) {
    showDetail(decodeURIComponent(r.sdk), false);
    switchTab("sdks", false);
    return;
  }
  if (r.tab === "sdks") showList(false);
  switchTab(r.tab, false);
}

function applyTheme(): void {
  const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  document.documentElement.dataset.theme = dark ? "dark" : "light";
}

/* boot */
applyTheme();
try {
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", applyTheme);
} catch {
  /* older browsers */
}
{
  let stored: string | null = null;
  try {
    stored = localStorage.getItem("vfox-plus-lang");
  } catch {
    /* private mode */
  }
  const mode = stored || "system";
  state.langMode = mode;
  setLang(mode === "system" ? systemLang() : (mode as "en" | "zh"));
}
renderChrome();
wireLanguageSegs(renderChrome, refresh);
initHome();
initSdks();
initPlugins();
initTasks();
initSettings();
wirePhoneCard();
document.querySelectorAll(".dest").forEach((b) => {
  (b as HTMLButtonElement).onclick = () => {
    const tab = (b as HTMLButtonElement).dataset.tab;
    if (tab === "sdks") location.hash = "#/sdks";
    else location.hash = "#/" + tab;
  };
});
(["fabAdd", "fabAddM"] as const).forEach((id) => {
  const f = document.getElementById(id) as HTMLButtonElement | null;
  if (f)
    f.onclick = () => {
      location.hash = "#/plugins";
      window.setTimeout(() => (document.getElementById("addSource") as HTMLInputElement).focus(), 60);
    };
});
(document.getElementById("railToggle") as HTMLButtonElement).onclick = () =>
  document.getElementById("rail")?.classList.toggle("expanded");
addEventListener("hashchange", route);
addEventListener("popstate", route);
route();
refresh(true);
connectEvents();

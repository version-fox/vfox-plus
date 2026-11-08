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

import { state, type TabName } from "./state";
import { t } from "./i18n";

// The hash router: #/home, #/sdks, #/sdk/<name>, #/plugins, #/tasks,
// #/settings. The tabs are sections of the one page; switching fades the new
// one through on the slow effects spring.
const RM = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

const TABS: TabName[] = ["home", "sdks", "plugins", "tasks", "settings"];

export function switchTab(name: TabName, push?: boolean, onShow?: (tab: TabName) => void): void {
  if (state.tab === name) return;
  state.tab = name;
  document.body.dataset.tab = name;
  document.querySelectorAll(".dest").forEach((b) => {
    if (b instanceof HTMLElement) b.setAttribute("aria-current", String(b.dataset.tab === name));
  });
  document.querySelectorAll(".page").forEach((p) => p.classList.remove("active", "enter"));
  const page = document.getElementById("tab-" + name);
  if (page) {
    page.classList.add("active");
    if (!RM) page.classList.add("enter");
  }
  const title = document.getElementById("pageTitle");
  if (title) title.textContent = t(name === "sdks" ? "sdks" : name);
  if (push !== false && location.hash !== "#/" + name) history.pushState({ tab: name }, "", "#/" + name);
  if (onShow) onShow(name);
}

export function currentRoute(): { tab: TabName; sdk?: string } {
  const h = (location.hash || "#/home").slice(2);
  const parts = h.split("/");
  if (parts[0] === "sdk" && parts[1]) return { tab: "sdks", sdk: parts[1] };
  if ((TABS as string[]).includes(parts[0])) return { tab: parts[0] as TabName };
  return { tab: "home" };
}

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

import { copyText, openExternal } from "./api";
import type { Inventory } from "./api";
import { avatar, el, esc, icon } from "./dom";
import { t } from "./i18n";
import { refresh } from "./main";

// ---------------- home ----------------
export function renderHome(inv: Inventory): void {
  const installed = (inv.installed && inv.installed.value) || [];
  let nVersions = 0,
    nCurrent = 0;
  installed.forEach((s) => {
    nVersions += (s.Versions || []).length;
    if (s.HasCurrent) nCurrent++;
  });
  const stats: [number, string, string, string][] = [
    [installed.length, t("pluginsN"), "s1", "extension"],
    [nVersions, t("versionsN"), "s2", "deployed-code"],
    [nCurrent, t("inUse"), "s3", "check-circle"],
  ];
  el("homeStats").innerHTML = stats
    .map(
      (s) =>
        '<button class="stat ' +
        s[2] +
        '" data-go="sdks">' +
        '<span class="ic-row">' +
        icon(s[3]) +
        "</span>" +
        '<span class="n">' +
        s[0] +
        "</span>" +
        '<span class="l">' +
        esc(s[1]) +
        "</span>" +
        '<span class="shape"></span>' +
        "</button>",
    )
    .join("");
  el("homeStats")
    .querySelectorAll("[data-go]")
    .forEach((b) => ((b as HTMLButtonElement).onclick = () => (location.hash = "#/sdks")));
  const active = installed.filter((s) => s.HasCurrent);
  const box = el("homeActive");
  if (!active.length) box.innerHTML = '<p class="muted">' + esc(t("activeEmpty")) + "</p>";
  else
    box.innerHTML = active
      .map(
        (s) =>
          '<button class="item" data-sdk="' +
          esc(s.Name) +
          '">' +
          avatar(s.Name) +
          '<span class="grow"><b>' +
          esc(s.Name) +
          '</b><br><span class="muted t-label">' +
          esc(s.Current) +
          "</span></span>" +
          icon("chevron-right", "sm muted") +
          "</button>",
      )
      .join("");
  box.querySelectorAll("[data-sdk]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        (location.hash = "#/sdk/" + encodeURIComponent((b as HTMLButtonElement).dataset.sdk || ""))),
  );
}

export function initHome(): void {
  el("homeRefresh").onclick = () => refresh(true);
  el("shellCopy").onclick = async () => {
    const cmd = (document.getElementById("shellCmd") as HTMLElement).textContent || "";
    await copyText(cmd);
  };
  el("exploreBtn").onclick = () => (location.hash = "#/plugins");
  el("docsBtn").onclick = () => openExternal("https://vfox.dev/");
}

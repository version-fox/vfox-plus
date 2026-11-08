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

import { api, confirmDlg, openExternal, showError } from "./api";
import { avatar, el, esc, icon } from "./dom";
import { t } from "./i18n";
import { refresh } from "./main";
import { state } from "./state";
import { runTask } from "./tasks";
import { sectionStatus, stateBlock, wireStateRetry } from "./ui";

// ---------------- plugins ----------------
function pluginFilter(): string {
  return (document.querySelector("#pluginFilter [aria-pressed=true]") as HTMLElement)?.dataset?.f || "all";
}

export function renderPlugins(): void {
  const inv = state.inventory;
  if (!inv) return;
  const box = el("pluginList");
  const sec = inv.registry;
  const st = sectionStatus(sec);
  if (st) {
    box.innerHTML = "";
    const w = document.createElement("div");
    w.innerHTML = st;
    box.appendChild(w);
    wireStateRetry(w, () => refresh(true));
    return;
  }
  const q = ((document.getElementById("pluginSearch") as HTMLInputElement).value || "").toLowerCase();
  const f = pluginFilter();
  const installed = new Set(((sec.value && inv.installed.value) || []).map((s) => s.Name));
  const list = (sec.value || []).filter((p) => {
    if (f === "official" && !p.Official) return false;
    if (f === "community" && p.Official) return false;
    return !q || p.Name.toLowerCase().includes(q) || (p.Homepage || "").toLowerCase().includes(q);
  });
  if (!list.length) {
    box.innerHTML = stateBlock("empty", "search-off", t("registry"), "", null);
    return;
  }
  box.innerHTML = list
    .map(
      (p) =>
        '<div class="pcard">' +
        '<div class="row">' +
        avatar(p.Name) +
        "<span style='flex:1'><b><code>" +
        esc(p.Name) +
        "</code></b>" +
        (p.Official ? ' <span class="tag" title="' + esc(t("official")) + '">' + icon("verified", "sm") + "</span>" : "") +
        (installed.has(p.Name) ? ' <span class="tag cur">✓</span>' : "") +
        "</span>" +
        (p.Homepage
          ? '<button class="iconbtn sm" data-home="' +
            esc(p.Homepage) +
            '" title="' +
            esc(t("homepage")) +
            '">' +
            icon("open-in-new", "sm") +
            "</button>"
          : "") +
        "</div>" +
        (p.Homepage
          ? '<div class="muted t-label" style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">' +
            esc(p.Homepage) +
            "</div>"
          : "") +
        '<div class="row" style="margin-top:auto">' +
        (installed.has(p.Name)
          ? '<button class="btn text" data-open="' +
            esc(p.Name) +
            '">' +
            esc(t("added")) +
            "</button><span style='flex:1'></span><button class='btn text' data-update='" +
            esc(p.Name) +
            "'>" +
            esc(t("update")) +
            "</button><button class='btn text' data-remove='" +
            esc(p.Name) +
            "'>" +
            esc(t("remove")) +
            "</button>"
          : '<button class="btn primary" data-addname="' +
            esc(p.Name) +
            '">' +
            esc(t("install")) +
            "</button>") +
        "</div></div>",
    )
    .join("");
  box.querySelectorAll("[data-home]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => openExternal((b as HTMLButtonElement).dataset.home || "")),
  );
  box.querySelectorAll("[data-open]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => (location.hash = "#/sdk/" + encodeURIComponent((b as HTMLButtonElement).dataset.open || ""))),
  );
  box.querySelectorAll("[data-addname]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => runTask({ operation: "add", source: (b as HTMLButtonElement).dataset.addname || "" })),
  );
  box.querySelectorAll("[data-update]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => runTask({ operation: "update", sdk: (b as HTMLButtonElement).dataset.update || "" })),
  );
  box.querySelectorAll("[data-remove]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => removePlugin((b as HTMLButtonElement).dataset.remove || "")),
  );
}

export async function removePlugin(name: string): Promise<void> {
  if (await confirmDlg(t("remove"), name, t("remove"), true)) runTask({ operation: "remove", sdk: name });
}

export function initPlugins(): void {
  (document.getElementById("pluginSearch") as HTMLInputElement).oninput = () => renderPlugins();
  document.querySelectorAll("#pluginFilter button").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => {
        document.querySelectorAll("#pluginFilter button").forEach((x) => x.setAttribute("aria-pressed", String(x === b)));
        renderPlugins();
      }),
  );
  el("regRefresh").onclick = () => refresh(true);
  el("addBtn").onclick = () => {
    const source = ((document.getElementById("addSource") as HTMLInputElement).value || "").trim();
    const alias = ((document.getElementById("addAlias") as HTMLInputElement).value || "").trim();
    if (!source) return;
    api("/api/plugins", { method: "POST", body: JSON.stringify({ source, alias }) })
      .then(() => {
        location.hash = "#/tasks";
        refresh(false);
      })
      .catch((e: unknown) => showError(e instanceof Error ? e.message : String(e)));
  };
  el("uploadBtn").onclick = () => {
    const input = document.getElementById("addFile") as HTMLInputElement;
    const f = input.files && input.files[0];
    if (!f) return;
    const fd = new FormData();
    fd.append("file", f);
    fd.append("alias", ((document.getElementById("addAlias") as HTMLInputElement).value || "").trim());
    fetch("/api/plugins", { method: "POST", body: fd, credentials: "same-origin" })
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then(() => {
        location.hash = "#/tasks";
        refresh(false);
      })
      .catch((e: unknown) => showError(e instanceof Error ? e.message : String(e)));
  };
  el("updateAllBtn").onclick = () => runTask({ operation: "update" });
}

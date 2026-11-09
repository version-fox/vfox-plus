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
import { openSheet, sectionStatus, stateBlock, wireStateRetry } from "./ui";

// ---------------- plugins ----------------
function pluginFilter(): string {
  return (document.querySelector("#pluginFilter [aria-pressed=true]") as HTMLElement)?.dataset?.f || "all";
}

/** The repository as its owner names it: github.com/version-fox/vfox-java, not a URL. */
function shortSource(url: string): string {
  const m = /^https?:\/\/github\.com\/([^/?#]+)\/([^/?#]+)/i.exec(url || "");
  if (m) return m[1] + "/" + m[2].replace(/\.git$/i, "");
  return (url || "").replace(/^https?:\/\//i, "").replace(/\/$/, "");
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
    .map((p) => {
      const source = shortSource(p.Homepage);
      const home = p.Homepage
        ? '<button class="iconbtn sm" data-home="' +
          esc(p.Homepage) +
          '" title="' +
          esc(t("homepage")) +
          '">' +
          icon("open-in-new", "sm") +
          "</button>"
        : "";
      const actions = installed.has(p.Name)
        ? '<button class="btn tonal" data-open="' +
          esc(p.Name) +
          '">' +
          esc(t("added")) +
          "</button>"
        : '<button class="btn tonal" data-addname="' +
          esc(p.Name) +
          '">' +
          icon("add", "sm") +
          "<span>" +
          esc(t("install")) +
          "</span></button>";
      return (
        '<div class="pcard"><div class="lead">' +
        avatar(p.Name) +
        '<span class="names"><b class="nameline"><code>' +
        esc(p.Name) +
        "</code>" +
        (p.Official
          ? '<span class="seal" title="' + esc(t("official")) + '">' + icon("verified", "sm") + "</span>"
          : "") +
        "</b>" +
        (source ? '<span class="muted">' + esc(source) + "</span>" : "") +
        '</span></div><div class="actions">' +
        home +
        '<span class="grow"></span>' +
        actions +
        "</div></div>"
      );
    })
    .join("");
  box.querySelectorAll("[data-home]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => openExternal((b as HTMLButtonElement).dataset.home || "")),
  );
  box.querySelectorAll("[data-open]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        (location.hash = "#/sdk/" + encodeURIComponent((b as HTMLButtonElement).dataset.open || ""))),
  );
  box.querySelectorAll("[data-addname]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        runTask({ operation: "add", source: (b as HTMLButtonElement).dataset.addname || "" })),
  );
  box.querySelectorAll("[data-update]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        runTask({ operation: "update", sdk: (b as HTMLButtonElement).dataset.update || "" })),
  );
  box.querySelectorAll("[data-remove]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => removePlugin((b as HTMLButtonElement).dataset.remove || "")),
  );
}

export async function removePlugin(name: string): Promise<void> {
  if (await confirmDlg(t("remove"), name, t("remove"), true)) runTask({ operation: "remove", sdk: name });
}

/* The form of "Add from source", in the sheet that the header button opens. */
function openAddPlugin(): void {
  openSheet(t("addPluginTitle"), "add", (box, close) => {
    box.innerHTML =
      '<div class="field"><label for="addSource">' +
      esc(t("sourceUrl")) +
      '</label><input id="addSource" placeholder="https://…" autocomplete="off" spellcheck="false"></div>' +
      '<div class="field"><label for="addAlias">' +
      esc(t("aliasLabel")) +
      '</label><input id="addAlias" autocomplete="off" spellcheck="false"></div>' +
      '<div class="actions"><button class="btn text" id="addCancel">' +
      esc(t("cancel")) +
      '</button><button class="btn primary" id="addBtn">' +
      esc(t("add")) +
      "</button></div>" +
      '<div class="muted t-label" style="margin:16px 0 8px">' +
      esc(t("uploadAdd")) +
      '</div><div class="row"><input type="file" id="addFile" accept=".zip,.lua"></div>';
    (document.getElementById("addCancel") as HTMLButtonElement).onclick = close;
    (document.getElementById("addBtn") as HTMLButtonElement).onclick = () => {
      const source = ((document.getElementById("addSource") as HTMLInputElement).value || "").trim();
      const alias = ((document.getElementById("addAlias") as HTMLInputElement).value || "").trim();
      if (!source) return;
      close();
      api("/api/plugins", { method: "POST", body: JSON.stringify({ source, alias }) })
        .then(() => {
          location.hash = "#/tasks";
          refresh(false);
        })
        .catch((e: unknown) => showError(e instanceof Error ? e.message : String(e)));
    };
    const file = document.getElementById("addFile") as HTMLInputElement;
    file.onchange = () => {
      const f = file.files && file.files[0];
      if (!f) return;
      const fd = new FormData();
      fd.append("file", f);
      fd.append("alias", ((document.getElementById("addAlias") as HTMLInputElement).value || "").trim());
      close();
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
    window.setTimeout(() => (document.getElementById("addSource") as HTMLInputElement).focus(), 50);
  });
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
  el("addPluginBtn").onclick = () => openAddPlugin();
  el("updateAllBtn").onclick = () => runTask({ operation: "update" });
}

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

import { api, confirmDlg, openExternal, showError, snack } from "./api";
import type { VersionEntry } from "./api";
import { avatar, el, esc, icon, txt } from "./dom";
import { getLang, t } from "./i18n";
import { refresh } from "./main";
import { state, type DetailState } from "./state";
import { removePlugin } from "./plugins";
import { runTask } from "./tasks";
import { chip, openMenu, openSheet, partsLine, section, sectionStatus, spinner, stateBlock, tagList, wireStateRetry } from "./ui";

// ---------------- sdks ----------------
export function renderSdks(): void {
  const inv = state.inventory;
  if (!inv) return;
  const box = el("sdkList");
  const sec = inv.installed;
  const st = sectionStatus(sec.value ? { value: sec.value, loading: sec.loading, error: sec.error } : sec);
  if (st) {
    box.innerHTML = st;
    wireStateRetry(box, () => refresh(true));
    return;
  }
  const q = ((document.getElementById("sdkSearch") as HTMLInputElement).value || "").toLowerCase();
  const list = (sec.value || []).filter((s) => !q || s.Name.toLowerCase().includes(q));
  if (!list.length) {
    box.innerHTML = stateBlock("empty", "inventory-2", t("noSdks"), "", null);
    return;
  }
  // The version in use goes under the count, not beside the name: the list is narrow, and the
  // names of Java and Flutter versions (21.0.12.1+1-tem, 3.47.7-x64) are long.
  box.innerHTML = list
    .map(
      (sdk) =>
        '<button class="item" data-sdk="' +
        esc(sdk.Name) +
        '" aria-current="' +
        (state.detailName === sdk.Name) +
        '">' +
        avatar(sdk.Name) +
        "<span style='flex:1;min-width:0'><b><code>" +
        esc(sdk.Name) +
        "</code></b><br><span class='muted t-label'>" +
        (sdk.HasCurrent ? "" : esc(t("empty")) + " · ") +
        (sdk.Versions || []).length +
        " " +
        esc(t("versionsN")) +
        "</span>" +
        (sdk.HasCurrent ? '<br><span class="tag cur">' + esc(sdk.Current) + "</span>" : "") +
        "</span>" +
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

export function showList(push?: boolean): void {
  document.querySelector("#tab-sdks")?.classList.remove("detail-open");
  state.detail = null;
  state.detailName = null;
  renderSdks();
  if (push !== false && location.hash !== "#/sdks") history.pushState({ tab: "sdks" }, "", "#/sdks");
}

function installedOf(name: string) {
  const inv = state.inventory;
  return ((inv && inv.installed.value) || []).find((s) => s.Name === name);
}

function configValue(key: string): string {
  const inv = state.inventory;
  const sec = inv && inv.config;
  const c = ((sec && sec.value) || []).find((x) => x.Key === key);
  return c ? c.Value : "";
}

async function vfoxConfig(key: string, value: string | null): Promise<unknown> {
  if (value == null) return api("/api/config?key=" + encodeURIComponent(key), { method: "DELETE" });
  return api("/api/config", { method: "PUT", body: JSON.stringify({ key, value }) });
}

/* sdk details: Installed | Available | Options. A plugin that has a profile (java, nodejs, flutter) also
   gets the choice of what a search lists, filter chips, installation by name and its options. */
export async function showDetail(name: string, keep?: boolean): Promise<void> {
  const tab = keep && state.detail && state.detail.name === name ? state.detail.tab : "installed";
  state.detail = {
    name,
    tab,
    info: null,
    profile: null,
    versions: null,
    error: null,
    search: "",
    facets: {},
    query: "",
    limit: 200,
    loading: false,
    token: 0,
  };
  state.detailName = name;
  document.querySelector("#tab-sdks")?.classList.add("detail-open");
  renderSdks();
  const box = el("sdkDetail");
  box.innerHTML = spinner();
  const [info, profile] = await Promise.all([
    api("/api/sdks/" + encodeURIComponent(name) + "/info").catch(() => null),
    api("/api/sdks/" + encodeURIComponent(name) + "/profile").catch(() => null),
  ]);
  const d = state.detail;
  if (!d || d.name !== name) return;
  d.info = info;
  d.profile = profile;
  // What the user chose to list is kept by the application, so that the desktop and every browser agree on it.
  const s = profile && (profile as { search?: { current: string; default: string } }).search;
  d.search = s && s.current !== s.default ? s.current : "";
  renderDetail();
  renderSdks();
}

function renderDetail(): void {
  const d = state.detail;
  if (!d) return;
  const box = el("sdkDetail");
  const sdk = installedOf(d.name);
  const info = d.info as { Description?: string; Homepage?: string } | null;
  let h =
    '<div class="row"><button class="iconbtn sm" id="detailBack" title="' +
    esc(t("close")) +
    '">' +
    icon("arrow-back", "sm") +
    "</button>" +
    avatar(d.name, true) +
    "<span style='flex:1;min-width:0'><span class='t-title-l'><code>" +
    esc(d.name) +
    "</code></span>" +
    (sdk && sdk.HasCurrent ? ' <span class="tag cur">' + esc(sdk.Current) + "</span>" : "") +
    "</span>" +
    (info && info.Homepage
      ? '<button class="iconbtn sm" id="detailHome" title="' +
        esc(t("homepage")) +
        '">' +
        icon("open-in-new", "sm") +
        "</button>"
      : "") +
    '<button class="iconbtn sm" id="detailMenu" title="…">' +
    icon("more-vert", "sm") +
    "</button></div>";
  h += '<div class="seg" id="detailTabs" role="tablist"></div><div id="detailBody"></div>';
  box.innerHTML = h;
  el("detailBack").onclick = () => showList(true);
  const home = document.getElementById("detailHome");
  if (home && info && info.Homepage)
    (home as HTMLButtonElement).onclick = () => openExternal(info.Homepage as string);
  el("detailMenu").onclick = (ev) => {
    const items: { icon: string; label: string; danger?: boolean; onclick?: () => void }[] = [];
    if (info && info.Homepage)
      items.push({ icon: "open-in-new", label: t("homepageOpen"), onclick: () => openExternal(info.Homepage as string) });
    items.push({ icon: "delete", label: t("remove"), danger: true, onclick: () => removePlugin(d.name) });
    openMenu(ev.currentTarget as Element, items);
  };
  renderTabs();
  renderTab(false);
}

function renderTabs(): void {
  const d = state.detail;
  if (!d) return;
  const sdk = installedOf(d.name);
  const tabs: [string, string, string][] = [
    ["installed", t("installed") + " (" + ((sdk && sdk.Versions) || []).length + ")", "deployed-code"],
    ["available", t("availableTab"), "download"],
  ];
  if (d.profile) tabs.push(["options", t("optionsTab"), "tune"]);
  const seg = document.getElementById("detailTabs");
  if (!seg) return;
  seg.innerHTML = tabs
    .map(
      (x) =>
        '<button role="tab" data-tab="' +
        x[0] +
        '" aria-pressed="' +
        (d.tab === x[0]) +
        '">' +
        icon(x[2], "sm") +
        "<span>" +
        esc(x[1]) +
        "</span></button>",
    )
    .join("");
  seg.querySelectorAll("button").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => {
        const t2 = (b as HTMLButtonElement).dataset.tab as DetailState["tab"];
        if (d.tab === t2 || !state.detail) return;
        state.detail.tab = t2;
        renderTabs();
        renderTab(true);
      }),
  );
}

/* fade-through between the tabs */
function renderTab(animate: boolean): void {
  const body = document.getElementById("detailBody");
  if (!body) return;
  body.className = "tabpane" + (animate ? " enter" : "");
  const d = state.detail;
  if (!d) return;
  if (d.tab === "available") renderAvailable(body);
  else if (d.tab === "options") renderOptions(body);
  else renderInstalled(body);
}

/* ---- installed ---- */
function renderInstalled(body: HTMLElement): void {
  const d = state.detail;
  if (!d) return;
  const sdk = installedOf(d.name);
  const versions = (sdk && sdk.Versions) || [];
  if (!versions.length) {
    body.innerHTML = stateBlock("empty", "download", t("noVersions"), "", t("browse"));
    const b = body.querySelector("[data-act]");
    if (b)
      (b as HTMLButtonElement).onclick = () => {
        if (!state.detail) return;
        state.detail.tab = "available";
        renderTabs();
        renderTab(true);
      };
    return;
  }
  let h = "<div class='list'>";
  versions.forEach((v) => {
    const isCur = !!sdk && sdk.HasCurrent && sdk.Current === v;
    const parts = d.profile && d.profile.installed && d.profile.installed[v];
    const line = parts ? partsLine(parts, getLang()) : "";
    h +=
      "<div class='item'><span style='flex:1;min-width:0'><code>" +
      esc(v) +
      "</code>" +
      (isCur ? ' <span class="tag cur">' + esc(t("current")) + "</span>" : "") +
      (line ? "<br><span class='muted t-label'>" + esc(line) + "</span>" : "") +
      "</span>" +
      (isCur
        ? ""
        : "<button class='btn tonal' data-use='" + esc(v) + "'>" + icon("check", "sm") + "<span>" + esc(t("use")) + "</span></button>") +
      '<button class="iconbtn sm" data-more-version="' +
      esc(v) +
      '" title="' +
      esc(t("actions")) +
      '">' +
      icon("more-vert", "sm") +
      "</button></div>";
  });
  body.innerHTML = h + "</div>";
  body.querySelectorAll("[data-use]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        runTask({ operation: "use", sdk: d.name, version: (b as HTMLButtonElement).dataset.use || "", scope: "global" })),
  );
  body.querySelectorAll("[data-more-version]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = (ev) => {
        const v = (b as HTMLButtonElement).dataset.moreVersion || "";
        openMenu(ev.currentTarget as Element, [
          {
            icon: "delete",
            label: t("uninstall"),
            danger: true,
            onclick: async () => {
              if (await confirmDlg(t("uninstall"), v, t("uninstall"), true))
                runTask({ operation: "uninstall", sdk: d.name, version: v });
            },
          },
        ]);
      }),
  );
}


/* ---- available ---- */
function searchChoice() {
  const d = state.detail;
  const s = d && d.profile && d.profile.search;
  if (!s) return null;
  const value = (d && d.search) || s.default;
  return s.choices.find((c) => c.value === value) || null;
}

async function loadVersions(): Promise<void> {
  const d = state.detail;
  if (!d) return;
  d.token = (d.token || 0) + 1;
  const token = d.token;
  const name = d.name, search = d.search;
  d.versions = null;
  d.error = null;
  d.facets = {};
  d.limit = 200;
  d.loading = true;
  drawAvailable();
  // Always name the choice, the default too: the list is the one that the button says, whatever another interface chose meanwhile.
  const s = d.profile && d.profile.search;
  const query = s ? "?search=" + encodeURIComponent(search || s.default) : "";
  try {
    const versions = (await api("/api/sdks/" + encodeURIComponent(name) + "/versions" + query)) as VersionEntry[];
    if (state.detail !== d || d.token !== token) return;
    d.versions = versions || [];
  } catch (e) {
    if (state.detail !== d || d.token !== token) return;
    d.error = e instanceof Error ? e.message : String(e);
  }
  d.loading = false;
  drawAvailable();
}

function renderAvailable(body: HTMLElement): void {
  const d = state.detail;
  if (!d) return;
  const c = searchChoice();
  const s = d.profile && d.profile.search;
  const chooseBtn = s
    ? '<button class="btn tonal" id="chooseBtn">' +
      icon("hub", "sm") +
      "<span>" +
      esc(c ? txt(c.name, getLang()) : d.search) +
      "</span>" +
      icon("keyboard-arrow-down", "sm") +
      "</button>"
    : "";
  body.innerHTML =
    '<div class="row actions">' +
    chooseBtn +
    '<button class="btn outline" id="byNameBtn">' +
    icon("terminal", "sm") +
    "<span>" +
    esc(t("installByName")) +
    "</span></button>" +
    "</div>" +
    '<div id="availBody"></div>';
  const choose = document.getElementById("chooseBtn");
  if (choose) choose.onclick = () => openChooser();
  el("byNameBtn").onclick = () => openInstallByName("");
  if (d.versions === null && !d.error && !d.loading) loadVersions();
  else drawAvailable();
}

function drawAvailable(): void {
  const d = state.detail;
  const host = document.getElementById("availBody");
  if (!host || !d || d.tab !== "available") return;
  if (d.error) {
    host.innerHTML = stateBlock("err", "error", t("failed"), esc(d.error), t("retry"));
    const b = host.querySelector("[data-act]");
    if (b) (b as HTMLButtonElement).onclick = () => loadVersions();
    return;
  }
  if (d.versions === null) {
    host.innerHTML =
      spinner() +
      '<p class="muted t-body" style="text-align:center">' +
      esc(t("fetching")) +
      "</p>";
    return;
  }
  if (!document.getElementById("vSearch")) {
    host.innerHTML =
      "<div class='row'><div class='search' style='flex:1;margin:0;min-width:200px'>" +
      icon("search", "sm muted") +
      "<input id='vSearch' placeholder='" +
      esc(t("search")) +
      "'></div></div>" +
      "<div class='row chips' id='vChips'></div><div class='muted t-label' id='vCount' style='margin:4px 4px 8px'></div><div id='vList' class='list'></div>";
    const input = document.getElementById("vSearch") as HTMLInputElement;
    input.value = d.query;
    input.oninput = () => {
      if (!state.detail) return;
      state.detail.query = input.value;
      state.detail.limit = 200;
      drawVersions();
    };
  }
  drawChips();
  drawVersions();
}

interface FacetGroup {
  id: string;
  label: { en: string; zh: string };
  values: string[];
}

function facetGroups(): FacetGroup[] {
  const d = state.detail;
  const out: FacetGroup[] = [];
  if (!d) return out;
  (((d.profile && d.profile.facets) || []) as { id: string; label: { en: string; zh: string }; values: string[] }[]).forEach(
    (f) => {
      const counts: Record<string, number> = {};
      (d.versions || []).forEach((v) =>
        ((v.facets && v.facets[f.id]) || []).forEach((x) => {
          counts[x] = (counts[x] || 0) + 1;
        }),
      );
      const values = f.values.filter((x) => counts[x]);
      if (values.length) out.push({ id: f.id, label: f.label, values });
    },
  );
  return out;
}

function drawChips(): void {
  const d = state.detail;
  const box = document.getElementById("vChips");
  if (!box || !d) return;
  const groups = facetGroups();
  let h = "";
  groups.forEach((g) => {
    g.values.forEach((v) => {
      const on = !!(d.facets[g.id] && d.facets[g.id][v]);
      h += chip(g.id + ":" + v, v, on, 'data-g="' + esc(g.id) + '" data-v="' + esc(v) + '"');
    });
  });
  box.innerHTML = h;
  box.querySelectorAll(".chip").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => {
        if (!state.detail) return;
        const dd = state.detail;
        const g = (b as HTMLButtonElement).dataset.g || "";
        const v = (b as HTMLButtonElement).dataset.v || "";
        dd.facets[g] = dd.facets[g] || {};
        if (dd.facets[g][v]) delete dd.facets[g][v];
        else dd.facets[g][v] = true;
        dd.limit = 200;
        b.setAttribute("aria-pressed", String(!!dd.facets[g][v]));
        drawVersions();
      }),
  );
}

function drawVersions(): void {
  const d = state.detail;
  const list = document.getElementById("vList");
  const count = document.getElementById("vCount");
  if (!list || !d) return;
  const q = (d.query || "").toLowerCase();
  const c = searchChoice();
  const showSource = !!(c && c.everything);
  const inst = new Set(((installedOf(d.name) || {}).Versions) || []);
  const rows = (d.versions || []).filter((v) => {
    for (const g of Object.keys(d.facets)) {
      const chosen = Object.keys(d.facets[g]);
      if (chosen.length && !chosen.some((x) => ((v.facets && v.facets[g]) || []).includes(x))) return false;
    }
    return !q || v.Version.toLowerCase().includes(q) || (v.Note || "").toLowerCase().includes(q);
  });
  if (count) count.textContent = t("nVersions").replace("{n}", String(rows.length));
  if (!rows.length) {
    list.innerHTML = stateBlock("empty", "search-off", t("noMatch"), "", null);
    return;
  }
  const shown = rows.slice(0, d.limit);
  list.innerHTML =
    shown
      .map((v) => {
        const source = showSource && v.parts && v.parts.source ? txt(v.parts.source.name, getLang()) : "";
        return (
          "<div class='item vrow'><span style='flex:1;min-width:0'><code>" +
          esc(v.Version) +
          "</code>" +
          (source ? " " + tagList([source]) : "") +
          (v.HasNote ? ' <span class="tag">' + esc(v.Note) + "</span>" : "") +
          ((v.Additions || []).length
            ? ' <span class="muted t-label">' +
              esc(v.Additions.map((a) => (a.Version ? a.Name + " " + a.Version : a.Name)).join(", ")) +
              "</span>"
            : "") +
          (inst.has(v.Version) ? ' <span class="tag">✓</span>' : "") +
          "</span>" +
          (inst.has(v.Version)
            ? ""
            : "<button class='btn text' data-install='" + esc(v.Version) + "'>" + esc(t("install")) + "</button>") +
          "</div>"
        );
      })
      .join("") +
    (rows.length > shown.length
      ? "<button class='btn' id='showMore' style='margin-top:8px'>" + esc(t("showMore").replace("{n}", String(rows.length - shown.length))) + "</button>"
      : "");
  list.querySelectorAll("[data-install]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () =>
        runTask({ operation: "install", sdk: d.name, version: (b as HTMLButtonElement).dataset.install || "" })),
  );
  const more = document.getElementById("showMore");
  if (more)
    (more as HTMLButtonElement).onclick = () => {
      if (!state.detail) return;
      state.detail.limit += 400;
      drawVersions();
    };
}

/* ---- sheets: the dialog of the choices, the names and the variables ---- */
function openChooser(): void {
  const d = state.detail;
  if (!d || !d.profile || !d.profile.search) return;
  const s = d.profile.search;
  openSheet(txt(s.label, getLang()), "hub", (box, close) => {
    box.innerHTML =
      '<div class="search">' +
      icon("search", "sm muted") +
      '<input id="chQuery" placeholder="' +
      esc(t("chooseHint")) +
      '"></div><div class="list choices" id="chList"></div><div class="actions"><button class="btn text" id="chClose">' +
      esc(t("close")) +
      "</button></div>";
    (document.getElementById("chClose") as HTMLButtonElement).onclick = close;
    const input = document.getElementById("chQuery") as HTMLInputElement;
    const list = document.getElementById("chList") as HTMLElement;
    const current = d.search || s.default;
    const matches = (c: { value: string; name: { en: string; zh: string }; maker: { en: string; zh: string }; aliases?: string[] }, q: string) =>
      !q || [c.value, ...(c.aliases || []), c.name.en, c.name.zh, c.maker.en, c.maker.zh].some((x) => (x || "").toLowerCase().includes(q));
    const draw = () => {
      const q = input.value.trim().toLowerCase();
      const rows = s.choices.filter((c) => matches(c, q));
      list.innerHTML = rows.length
        ? rows
            .map(
              (c) =>
                "<button class='item' data-v='" +
                esc(c.value) +
                "' aria-current='" +
                (c.value === current) +
                "'><span class='lead'>" +
                (c.value === current ? icon("check") : "") +
                "</span><span style='flex:1;min-width:0'><b>" +
                esc(txt(c.name, getLang())) +
                "</b><br><span class='muted t-label'>" +
                esc([txt(c.maker, getLang()), c.value].filter(Boolean).join(" · ")) +
                "</span></span>" +
                (c.value === s.default ? '<span class="tag">' + esc(t("default")) + "</span>" : "") +
                "</button>",
            )
            .join("")
        : stateBlock("empty", "search-off", t("chooseEmpty"), "", null);
      list.querySelectorAll("[data-v]").forEach(
        (b) =>
          ((b as HTMLButtonElement).onclick = async () => {
            close();
            try {
              const r = (await api("/api/sdks/" + encodeURIComponent(d.name) + "/search", {
                method: "PUT",
                body: JSON.stringify({ value: (b as HTMLButtonElement).dataset.v }),
              })) as { value: string };
              s.current = r.value;
            } catch (e) {
              showError(e instanceof Error ? e.message : String(e));
              return;
            }
            if (state.detail !== d) return;
            d.search = s.current === s.default ? "" : s.current;
            renderTab(false);
            loadVersions();
          }),
      );
    };
    input.oninput = draw;
    draw();
    window.setTimeout(() => input.focus(), 50);
  });
}

function validSpec(v: string): boolean {
  return !!v && !/^[-\s@]/.test(v);
}

function openInstallByName(initial: string): void {
  const d = state.detail;
  if (!d) return;
  const p = d.profile;
  openSheet(t("installNameTitle").replace("{name}", d.name), "terminal", (box, close) => {
    box.innerHTML =
      '<div class="field"><label for="nameIn">' +
      esc(t("versionName")) +
      '</label><input id="nameIn" class="mono" autocomplete="off" autocapitalize="off" spellcheck="false"><div class="muted t-label" id="nameHelp"></div></div>' +
      (p && p.shorthands.length
        ? '<div class="muted t-label chiplabel">' +
          esc(t("namesToTry")) +
          '</div><div class="row chips" id="nameChips"></div><div class="t-body" id="nameMeaning" style="margin-top:8px"></div>'
        : "") +
      '<p class="muted t-label" style="margin:12px 0 0">' +
      esc(t("installNameNote")) +
      '</p><div class="actions"><button class="btn text" id="nameCancel">' +
      esc(t("cancel")) +
      '</button><button class="btn primary" id="nameOk">' +
      icon("download", "sm") +
      "<span>" +
      esc(t("install")) +
      "</span></button></div>";
    const input = document.getElementById("nameIn") as HTMLInputElement;
    const help = document.getElementById("nameHelp") as HTMLElement;
    help.textContent = t("installNameHelp").replace("{name}", d.name);
    input.value = initial || "";
    const sync = () => {
      const cur = input.value.trim();
      if (p) {
        const row = document.getElementById("nameChips");
        if (row) {
          row.innerHTML = p.shorthands.map((x) => chip(x.name, x.name, x.name === cur, 'data-name="' + esc(x.name) + '"')).join("");
          row.querySelectorAll(".chip").forEach(
            (b) =>
              ((b as HTMLButtonElement).onclick = () => {
                input.value = (b as HTMLButtonElement).dataset.name || "";
                help.textContent = t("installNameHelp").replace("{name}", d.name);
                help.classList.remove("bad");
                sync();
              }),
          );
        }
        const m = p.shorthands.find((x) => x.name === cur);
        (document.getElementById("nameMeaning") as HTMLElement).textContent = m ? txt(m.meaning, getLang()) : "";
      }
    };
    const submit = () => {
      const v = input.value.trim();
      if (!v) {
        help.textContent = t("nameRequired");
        help.classList.add("bad");
        return;
      }
      if (!validSpec(v)) {
        help.textContent = t("nameInvalid");
        help.classList.add("bad");
        return;
      }
      close();
      runTask({ operation: "install", sdk: d.name, version: v });
    };
    input.oninput = () => {
      help.textContent = t("installNameHelp").replace("{name}", d.name);
      help.classList.remove("bad");
      sync();
    };
    input.onkeydown = (ev) => {
      if (ev.key === "Enter") submit();
    };
    (document.getElementById("nameOk") as HTMLButtonElement).onclick = submit;
    (document.getElementById("nameCancel") as HTMLButtonElement).onclick = close;
    sync();
    window.setTimeout(() => input.focus(), 50);
  });
}

/* ---- options ---- */
function renderOptions(body: HTMLElement): void {
  const d = state.detail;
  if (!d) return;
  const p = d.profile;
  if (!p) {
    body.innerHTML = "";
    return;
  }
  let h = "";
  if (p.guide && (p.guide.en || p.guide.zh))
    h +=
      '<div class="list"><div class="item guide">' +
      icon("info") +
      "<span style='flex:1'><b>" +
      esc(t("guideTitle")) +
      "</b><br><span class='muted'>" +
      esc(txt(p.guide, getLang())) +
      "</span></span></div>";
  if (p.variables.length) {
    h += section(
      t("downloads"),
      t("downloadsHelp"),
      p.variables
        .map((v) => {
          const preset = (v.presets || []).find((x) => x.value === v.value);
          const sub = v.value ? (preset ? txt(preset.name, getLang()) + " · " : "") + v.value : t("unset");
          return (
            "<div class='item'>" +
            icon("download") +
            "<span style='flex:1;min-width:0'><b>" +
            esc(txt(v.label, getLang())) +
            "</b>" +
            (v.advanced ? " " + tagList([t("advanced")]) : "") +
            '<br><span class="muted t-label"><code>' +
            esc(v.key) +
            "</code><br>" +
            esc(sub) +
            "</span></span><button class='btn tonal' data-var='" +
            esc(v.key) +
            "'>" +
            esc(t("change")) +
            "</button></div>"
          );
        })
        .join(""),
    );
  }
  h += '<div class="t-title-m" style="margin-top:20px">' + esc(t("legacyTitle")) + '</div><div class="list">';
  if (!p.legacyFiles.length)
    h += "<div class='item'>" + icon("history") + "<span class='muted'>" + esc(t("legacyNone")) + "</span></div>";
  else {
    h +=
      "<div class='item'>" +
      icon("history") +
      "<span style='flex:1'><b>" +
      esc(t("legacyFiles")) +
      "</b><br><span class='muted t-label'>" +
      esc(t("legacyFilesHelp")) +
      "</span><br>" +
      tagList(p.legacyFiles) +
      "</span></div>";
    const on = configValue("legacyVersionFile.enable") === "true";
    h +=
      '<div class="item"><span class="lead"></span><span style="flex:1"><b>' +
      esc(t("legacyEnable")) +
      '</b><br><span class="muted t-label">' +
      esc(t("legacyEnableHelp")) +
      '</span></span><button class="switch" role="switch" id="legacySwitch" aria-checked="' +
      on +
      '" aria-label="' +
      esc(t("legacyEnable")) +
      '"></button></div>';
    if (p.legacyStrategy) {
      const strat = configValue("legacyVersionFile.strategy");
      h +=
        '<div class="item"><span class="lead"></span><span style="flex:1;min-width:0"><b>' +
        esc(t("legacyStrategy")) +
        '</b></span><div class="seg" id="legacyStrategy">' +
        (
          [
            ["specified", t("strategySpecified")],
            ["installed", t("strategyInstalled")],
            ["available", t("strategyAvailable")],
          ] as [string, string][]
        )
          .map(
            (x) =>
              "<button data-s='" + x[0] + "' aria-pressed='" + (strat === x[0]) + "'>" + esc(x[1]) + "</button>",
          )
          .join("") +
        "</div></div>";
    } else
      h +=
        '<div class="item"><span class="lead"></span><span class="muted">' +
        esc(t("legacyIgnored").replace("{setting}", t("legacyStrategy"))) +
        "</span></div>";
  }
  h += "</div>";
  if (p.shorthands.length) {
    h += section(
      t("shorthands"),
      t("shorthandsHelp").replace("{name}", d.name),
      p.shorthands
        .map(
          (s) =>
            "<div class='item'>" +
            icon("terminal") +
            "<span style='flex:1;min-width:0'><code>" +
            esc(s.name) +
            '</code><br><span class="muted t-label">' +
            esc(txt(s.meaning, getLang())) +
            "</span></span><button class='btn tonal' data-try='" +
            esc(s.name) +
            "'>" +
            icon("download", "sm") +
            "<span>" +
            esc(t("installDots")) +
            "</span></button></div>",
        )
        .join(""),
    );
  }
  if (p.tips.length)
    h += section(
      t("tips"),
      "",
      p.tips
        .map((x) => "<div class='item'>" + icon("info") + "<span style='flex:1'>" + esc(txt(x, getLang())) + "</span></div>")
        .join(""),
    );
  if (p.docs.length)
    h += section(
      t("docs"),
      "",
      p.docs
        .map(
          (x) =>
            "<button class='item' data-doc='" +
            esc(txt(x.url, getLang())) +
            "'>" +
            icon("menu-book") +
            "<span style='flex:1'>" +
            esc(txt(x.title, getLang())) +
            "</span>" +
            icon("open-in-new", "sm muted") +
            "</button>",
        )
        .join(""),
    );
  body.innerHTML = h;
  body.querySelectorAll("[data-var]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => {
        const v = p.variables.find((x) => x.key === (b as HTMLButtonElement).dataset.var);
        if (v) openVariable(v);
      }),
  );
  body.querySelectorAll("[data-try]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => openInstallByName((b as HTMLButtonElement).dataset.try || "")),
  );
  body.querySelectorAll("[data-doc]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => openExternal((b as HTMLButtonElement).dataset.doc || "")),
  );
  const sw = document.getElementById("legacySwitch");
  if (sw)
    (sw as HTMLButtonElement).onclick = async () => {
      try {
        await vfoxConfig("legacyVersionFile.enable", sw.getAttribute("aria-checked") === "true" ? "false" : "true");
        await refresh(false);
        renderTab(false);
      } catch (e) {
        showError(e instanceof Error ? e.message : String(e));
      }
    };
  body.querySelectorAll("#legacyStrategy button").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = async () => {
        try {
          await vfoxConfig("legacyVersionFile.strategy", (b as HTMLButtonElement).dataset.s || "");
          await refresh(false);
          renderTab(false);
        } catch (e) {
          showError(e instanceof Error ? e.message : String(e));
        }
      }),
  );
}

function openVariable(variable: {
  label: { en: string; zh: string };
  help: { en: string; zh: string };
  key: string;
  kind: string;
  example: string;
  value: string;
  presets: { value: string; name: { en: string; zh: string } }[];
}): void {
  const d = state.detail;
  if (!d) return;
  openSheet(txt(variable.label, getLang()), "download", (box, close) => {
    box.innerHTML =
      '<p class="muted t-body" style="margin:0">' +
      esc(txt(variable.help, getLang())) +
      '</p><p class="muted t-label"><code>' +
      esc(variable.key) +
      "</code></p>" +
      (variable.kind === "url"
        ? '<div class="field"><label for="varIn">' +
          esc(t("address")) +
          '</label><input id="varIn" class="mono" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="' +
          esc(variable.example || "") +
          '"><div class="muted t-label" id="varHelp"></div></div>'
        : '<div class="muted t-label" id="varHelp"></div>') +
      '<div class="muted t-label chiplabel">' +
      esc(t("quickPicks")) +
      '</div><div class="row chips" id="varChips"></div><div class="actions"><button class="btn text" id="varCancel">' +
      esc(t("cancel")) +
      '</button><button class="btn primary" id="varOk">' +
      esc(t("save")) +
      "</button></div>";
    let value = variable.value || "";
    const input = document.getElementById("varIn") as HTMLInputElement | null;
    const help = document.getElementById("varHelp") as HTMLElement;
    const sync = () => {
      if (input) input.value = value;
      const row = document.getElementById("varChips");
      if (!row) return;
      const cur = value.trim();
      row.innerHTML =
        chip("", t("defaultOption"), cur === "", 'data-value=""') +
        variable.presets.map((x) => chip(x.value, txt(x.name, getLang()), cur === x.value, 'data-value="' + esc(x.value) + '"')).join("");
      row.querySelectorAll(".chip").forEach(
        (b) =>
          ((b as HTMLButtonElement).onclick = () => {
            value = (b as HTMLButtonElement).dataset.value || "";
            help.textContent = "";
            help.classList.remove("bad");
            sync();
          }),
      );
    };
    if (input)
      input.oninput = () => {
        value = input.value;
        help.textContent = "";
        help.classList.remove("bad");
        drawPressed();
      };
    const drawPressed = () => {
      const cur = value.trim();
      document.querySelectorAll("#varChips .chip").forEach((b) => b.setAttribute("aria-pressed", String((b as HTMLButtonElement).dataset.value === cur)));
    };
    const submit = async () => {
      try {
        const r = (await api("/api/sdks/" + encodeURIComponent(d.name) + "/variables", {
          method: "PUT",
          body: JSON.stringify({ key: variable.key, value: value.trim() }),
        })) as { value: string };
        variable.value = r.value || "";
        close();
        snack(t("saved"));
        if (state.detail === d && d.tab === "options") renderTab(false);
      } catch (e) {
        help.textContent = e instanceof Error ? e.message : String(e);
        help.classList.add("bad");
      }
    };
    (document.getElementById("varOk") as HTMLButtonElement).onclick = submit;
    (document.getElementById("varCancel") as HTMLButtonElement).onclick = close;
    if (input)
      input.onkeydown = (ev) => {
        if (ev.key === "Enter") submit();
      };
    sync();
    if (input) window.setTimeout(() => input.focus(), 50);
  });
}

/* what a refresh changes in the details: the installed versions, and which listed versions are installed */
export function updateDetail(): void {
  const d = state.detail;
  if (!d || (!d.profile && !d.info && !document.getElementById("detailTabs"))) return;
  renderTabs();
  const sdk = installedOf(d.name);
  const key = ((sdk && sdk.Versions) || []).join("\n");
  if (d.installedKey !== undefined && d.installedKey !== key && d.profile) {
    api("/api/sdks/" + encodeURIComponent(d.name) + "/profile")
      .then((p) => {
        if (state.detail === d) {
          d.profile = p;
          d.installedKey = key;
          if (d.tab !== "available") renderTab(false);
        }
      })
      .catch(() => {});
  }
  d.installedKey = key;
  if (d.tab === "installed") {
    const body = document.getElementById("detailBody");
    if (body) renderInstalled(body);
  } else if (d.tab === "available") drawVersions();
}

export function initSdks(): void {
  (document.getElementById("sdkSearch") as HTMLInputElement).oninput = () => renderSdks();
  el("sdkRefresh").onclick = () => refresh(true);
}

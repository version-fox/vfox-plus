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

import { api, confirmDlg, copyText, showError, snack } from "./api";
import type { Pairing } from "./api";
import { el, esc, icon } from "./dom";
import { setLang, systemLang, t, type Lang } from "./i18n";
import { refresh } from "./main";
import { state } from "./state";
import { openMenu } from "./ui";

/* ---------------- settings ---------------- */
export function renderSettings(): void {
  loadWeb();
  api("/api/settings")
    .then((s: { vfoxExecutable?: string; language?: string; extraEnvironment?: Record<string, string> }) => {
      if (document.activeElement !== document.getElementById("setVfox"))
        (document.getElementById("setVfox") as HTMLInputElement).value = s.vfoxExecutable || "";
      document.querySelectorAll("#langSeg button").forEach((b) =>
        b.setAttribute("aria-pressed", String((b as HTMLButtonElement).dataset.l === (s.language || "system"))),
      );
      if (document.activeElement !== document.getElementById("setEnv"))
        (document.getElementById("setEnv") as HTMLTextAreaElement).value = Object.entries(s.extraEnvironment || {})
          .map(([k, v]) => k + "=" + v)
          .join("\n");
    })
    .catch(() => {});
  const inv = state.inventory;
  const box = el("configList");
  if (!inv) return;
  const sec = inv.config;
  if (!sec || (sec.loading && !sec.value && !sec.error)) {
    box.innerHTML = '<p class="muted">' + esc(t("loading")) + "</p>";
    return;
  }
  if (sec.error && (!sec.value || !sec.value.length)) {
    box.innerHTML = '<p class="bad">' + esc(sec.error) + "</p>";
    return;
  }
  box.innerHTML = (sec.value || [])
    .map(
      (c) =>
        '<div class="confrow"><span class="grow"><code>' +
        esc(c.Key) +
        "</code><span class=\"muted\">" +
        esc(c.Value || t("empty")) +
        '</span></span><button class="iconbtn sm" data-more="' +
        esc(c.Key) +
        '" title="' +
        esc(t("actions")) +
        '">' +
        icon("more-vert", "sm") +
        "</button></div>",
    )
    .join("");
  box.querySelectorAll("[data-more]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = (ev) => {
        const key = (b as HTMLButtonElement).dataset.more || "";
        const cur = (((sec.value || []).find((x) => x.Key === key) || {}).Value as string | undefined) || "";
        openMenu(ev.currentTarget as Element, [
          {
            icon: "tune",
            label: t("edit"),
            onclick: async () => {
              const v = prompt(key, cur);
              if (v == null) return;
              try {
                await vfoxConfig(key, v);
                refresh(false);
              } catch (e) {
                showError(e instanceof Error ? e.message : String(e));
              }
            },
          },
          {
            icon: "history",
            label: t("reset"),
            danger: true,
            onclick: async () => {
              try {
                await vfoxConfig(key, null);
                refresh(false);
              } catch (e) {
                showError(e instanceof Error ? e.message : String(e));
              }
            },
          },
        ]);
      }),
  );
}

async function vfoxConfig(key: string, value: string | null): Promise<unknown> {
  if (value == null) return api("/api/config?key=" + encodeURIComponent(key), { method: "DELETE" });
  return api("/api/config", { method: "PUT", body: JSON.stringify({ key, value }) });
}

/* ---------------- connections ---------------- */
function webRow(ic: string, title: string, control: string, extra?: string, help?: string): string {
  return (
    '<div class="rowitem"><span class="icon-tile">' +
    icon(ic) +
    '</span><span class="grow"><span class="t-title-m">' +
    esc(title) +
    "</span>" +
    (help ? '<span class="muted t-body">' + esc(help) + "</span>" : "") +
    (extra || "") +
    "</span>" +
    control +
    "</div>"
  );
}

function webSwitch(id: string, on: boolean, title: string): string {
  return (
    "<button class='switch' role='switch' id='" +
    id +
    "' aria-checked='" +
    (on ? "true" : "false") +
    "' aria-label='" +
    esc(title) +
    "'></button>"
  );
}

export function loadWeb(): void {
  api("/api/web")
    .then((web) => {
      state.web = web;
      renderWeb();
    })
    .catch(() => {
      /* the card keeps what it had */
    });
}

export async function putWeb(change: Record<string, unknown>): Promise<void> {
  try {
    state.web = await api("/api/web", { method: "PUT", body: JSON.stringify(change) });
  } catch (e) {
    showError(e instanceof Error ? e.message : String(e));
  }
  renderWeb();
}

export function renderWeb(): void {
  const web = state.web;
  if (!web) return;
  let h = webRow("devices", t("webRemote"), webSwitch("webRemote", web.remote, t("webRemote")));
  if (web.remote) {
    h += webRow(
      "tune",
      t("webPort"),
      "<input id='webPort' type='number' min='1024' max='65535' value='" +
        esc(web.port) +
        "' aria-label='" +
        esc(t("webPort")) +
        "'>",
      "",
      t("webPortHelp"),
    );
    if (web.error) {
      h += '<div class="rowitem"><span class="bad t-label">' + esc(t("webError")) + " " + esc(web.error) + "</span></div>";
    } else if (web.listening) {
      const reach = web.addresses.length
        ? "<br><span class='t-label'>" +
          esc(t("webReach")) +
          "</span> " +
          web.addresses.map((a) => "<code>" + esc("https://" + (a.includes(":") ? "[" + a + "]" : a) + ":" + web.port) + "</code>").join(" ")
        : "<br><span class='bad t-label'>" + esc(t("webNoAddress")) + "</span>";
      const print =
        "<br><span class='muted t-label'>" + esc(t("webFingerprint")) + "</span><br><code>" + esc(web.fingerprint) + "</code>";
      h += webRow(
        "hub",
        t("webPairBtn"),
        window.vfoxAndroid || !web.addresses.length
          ? ""
          : "<button class='btn tonal' id='webPair'>" + esc(t("webPairBtn")) + "</button>",
        reach + print,
      );
    }
  }
  h += webRow("eco", t("webLight"), webSwitch("webLight", web.lightweight, t("webLight")), "", t("webLightHelp"));
  h += webRow("power", t("webQuit"), "<button class='btn danger' id='webQuit'>" + esc(t("webQuitBtn")) + "</button>");
  el("webList").innerHTML = h;
  el("webRemote").onclick = async () => {
    if (!web.remote && !(await confirmDlg(t("webRemoteTitle"), t("webRemoteBody"), t("webRemoteOk"), false))) return;
    state.pairing = null;
    putWeb({ remote: !web.remote });
  };
  el("webLight").onclick = () => putWeb({ lightweight: !web.lightweight });
  el("webQuit").onclick = async () => {
    if (!(await confirmDlg(t("webQuitTitle"), t("webQuitBody"), t("webQuitBtn"), true))) return;
    try {
      await api("/api/quit", { method: "POST" });
      snack(t("webQuitting"));
    } catch (e) {
      showError(e instanceof Error ? e.message : String(e));
    }
  };
  const port = document.getElementById("webPort") as HTMLInputElement | null;
  if (port) port.onchange = () => putWeb({ port: parseInt(port.value, 10) || 0 });
  const pair = document.getElementById("webPair");
  if (pair) (pair as HTMLButtonElement).onclick = () => showPairing("");
  renderPairing();
}

export async function showPairing(host: string): Promise<void> {
  try {
    const web = state.web;
    const pairing = (await api("/api/web/pairing", { method: "POST", body: JSON.stringify({ host }) })) as Pairing;
    pairing.host = host || (web && web.addresses[0]) || "";
    state.pairing = pairing;
  } catch (e) {
    state.pairing = null;
    showError(e instanceof Error ? e.message : String(e));
  }
  renderPairing();
}

export function renderPairing(): void {
  const box = el("pairBox");
  const pairing = state.pairing;
  const web = state.web;
  if (!pairing || !web || !web.listening) {
    box.hidden = true;
    box.innerHTML = "";
    return;
  }
  const hosts =
    web.addresses.length > 1
      ? "<div class='seg' id='pairHosts'>" +
        web.addresses
          .map(
            (a) => "<button data-host='" + esc(a) + "' aria-pressed='" + (a === pairing.host) + "'>" + esc(a) + "</button>",
          )
          .join("") +
        "</div>"
      : "";
  box.innerHTML =
    "<div class='qr'>" +
    pairing.svg +
    '</div><div class="t-label muted">' +
    esc(t("webPairHelp")) +
    "</div>" +
    hosts +
    "<code>" +
    esc(pairing.url) +
    '</code><div class="row"><button class="btn" id="pairCopy">' +
    esc(t("webCopyLink")) +
    '</button><button class="btn text" id="pairClose">' +
    esc(t("close")) +
    "</button></div>";
  box.hidden = false;
  el("pairCopy").onclick = () => copyText(pairing.url);
  el("pairClose").onclick = () => {
    state.pairing = null;
    renderPairing();
  };
  box.querySelectorAll("[data-host]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => showPairing((b as HTMLButtonElement).dataset.host || "")),
  );
}

export function setPageLang(mode: string): void {
  state.langMode = mode;
  setLang(mode === "system" ? systemLang() : (mode as Lang));
  try {
    localStorage.setItem("vfox-plus-lang", mode);
  } catch {
    /* private mode */
  }
}

export function wireLanguageSegs(renderChrome: () => void, refreshFn: (full: boolean) => Promise<void>): void {
  document.querySelectorAll("#pageLangSeg button").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = async () => {
        setPageLang((b as HTMLButtonElement).dataset.l || "system");
        renderChrome();
        refreshFn(true);
      }),
  );
  document.querySelectorAll("#langSeg button").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = async () => {
        try {
          await api("/api/settings", { method: "PUT", body: JSON.stringify({ language: (b as HTMLButtonElement).dataset.l }) });
        } catch (err) {
          showError(err instanceof Error ? err.message : String(err));
        }
      }),
  );
}

export function initSettings(): void {
  el("saveVfox").onclick = async () => {
    try {
      await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ vfoxExecutable: (el("setVfox") as HTMLInputElement).value.trim() }),
      });
      refresh(false);
    } catch (e) {
      showError(e instanceof Error ? e.message : String(e));
    }
  };
  el("saveEnv").onclick = async () => {
    const env: Record<string, string> = {};
    ((el("setEnv") as HTMLTextAreaElement).value || "").split("\n").forEach((line) => {
      const i = line.indexOf("=");
      if (i > 0) env[line.slice(0, i).trim()] = line.slice(i + 1);
    });
    try {
      await api("/api/settings", { method: "PUT", body: JSON.stringify({ extraEnvironment: env }) });
      refresh(false);
    } catch (e) {
      showError(e instanceof Error ? e.message : String(e));
    }
  };
}

// In the Android app the settings offer what only the app can do.
export function wirePhoneCard(): void {
  if (!window.vfoxAndroid) return;
  const bridge = window.vfoxAndroid;
  (document.getElementById("phoneCard") as HTMLElement).hidden = false;
  (document.getElementById("phoneServer") as HTMLElement).textContent = String(bridge.server()).replace(
    /^[a-z]+:\/\//i,
    "",
  );
  el("phonePair").onclick = () => bridge.pairNew();
  el("phoneOut").onclick = async () => {
    if (await confirmDlg(t("phoneOutTitle"), t("phoneOutBody"), t("phoneOut"), true)) bridge.disconnect();
  };
}

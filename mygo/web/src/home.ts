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

import { api, copyText } from "./api";
import type { Inventory } from "./api";
import { avatar, el, esc, icon } from "./dom";
import { t } from "./i18n";
import { refresh } from "./main";
import { runTask } from "./tasks";

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
  renderDoctor();
}

async function renderDoctor(): Promise<void> {
  const box = document.getElementById("homeDoctor");
  if (!box) return;
  box.innerHTML = '<p class="muted">' + esc(t("loading")) + "</p>";
  try {
    const report = (await api("/api/doctor")) as { ok: boolean; findings: { id: string; severity: string; sdk: string; version: string; message: string }[] };
    if (!report.findings.length) {
      box.innerHTML = "<div class='item'>" + icon("verified") + "<span>" + esc(t("doctorNoProblems")) + "</span></div>";
      return;
    }
    box.innerHTML = report.findings
      .slice(0, 5)
      .map(
        (f) =>
          "<div class='item'><span style='flex:1'><b>" +
          esc(f.message || f.id) +
          "</b><br><span class='muted t-label'>" +
          esc([f.sdk, f.version].filter(Boolean).join("@") + " · " + f.severity) +
          "</span></span>" +
          (f.id === "sdk.no-receipt"
            ? "<button class='btn tonal' data-adopt-sdk='" + esc(f.sdk) + "' data-adopt-ver='" + esc(f.version) + "'>" + esc(t("doctorAdopt")) + "</button>"
            : f.id === "sdk.drift"
              ? "<button class='btn tonal' data-reinstall-sdk='" + esc(f.sdk) + "' data-reinstall-ver='" + esc(f.version) + "'>" + esc(t("versionReinstall")) + "</button>"
              : "") +
          "</div>",
      )
      .join("");
    box.querySelectorAll("[data-adopt-sdk]").forEach(
      (b) =>
        ((b as HTMLButtonElement).onclick = () =>
          runTask({
            operation: "doctorAdopt",
            sdk: (b as HTMLButtonElement).dataset.adoptSdk || "",
            version: (b as HTMLButtonElement).dataset.adoptVer || "",
          })),
    );
    box.querySelectorAll("[data-reinstall-sdk]").forEach(
      (b) =>
        ((b as HTMLButtonElement).onclick = () =>
          runTask({
            operation: "install",
            sdk: (b as HTMLButtonElement).dataset.reinstallSdk || "",
            version: (b as HTMLButtonElement).dataset.reinstallVer || "",
            reinstall: true,
          })),
    );
  } catch (e) {
    box.innerHTML = "<div class='item'><span class='bad'>" + esc(e instanceof Error ? e.message : String(e)) + "</span></div>";
  }
}

export function initHome(): void {
  el("homeRefresh").onclick = () => {
    const spin = el("homeRefresh").querySelector(".ic");
    spin?.classList.add("spin");
    setTimeout(() => spin?.classList.remove("spin"), 650);
    refresh(true);
  };
  el("shellCopy").onclick = async () => {
    const button = el("shellCopy");
    const cmd = (document.getElementById("shellCmd") as HTMLElement).textContent || "";
    await copyText(cmd);
    const use = button.querySelector("use");
    const was = use?.getAttribute("href") || "";
    use?.setAttribute("href", "#i-check");
    setTimeout(() => use?.setAttribute("href", was), 1200);
  };
}

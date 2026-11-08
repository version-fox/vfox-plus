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

import { esc } from "./dom";
import { t } from "./i18n";

// Typed shapes of the JSON the application serves. Only the fields the page
// reads are listed; the server may send more.
export interface InstalledSdk {
  Name: string;
  Versions: string[];
  HasCurrent: boolean;
  Current: string;
}

export interface RegistryEntry {
  Name: string;
  Homepage: string;
  Official: boolean;
}

export interface ConfigEntry {
  Key: string;
  Value: string;
}

export interface Section<T> {
  value: T[] | null;
  loading?: boolean;
  error?: string;
}

export interface Inventory {
  vfox: { version: string; availability: string };
  installed: Section<InstalledSdk>;
  registry: Section<RegistryEntry>;
  config: Section<ConfigEntry>;
}

export type TaskStatus = "queued" | "running" | "succeeded" | "failed" | "cancelled";

export interface Task {
  id: number;
  status: TaskStatus;
  operation: string;
  sdk: string;
  version: string;
  progress: number | null;
  errorMessage: string;
  commandLine: string;
  droppedLines: number;
  hasFullLog: boolean;
  log: string[];
}

export interface WebState {
  remote: boolean;
  port: number;
  error: string;
  listening: boolean;
  addresses: string[];
  fingerprint: string;
  lightweight: boolean;
}

export interface Pairing {
  url: string;
  svg: string;
  host: string;
}

export interface Localizable {
  en: string;
  zh: string;
}

export interface SearchChoiceOption {
  value: string;
  name: Localizable;
  maker: Localizable;
  aliases?: string[];
  everything?: boolean;
}

export interface SearchChoice {
  label: Localizable;
  value: string;
  current: string;
  default: string;
  choices: SearchChoiceOption[];
  everything?: boolean;
}

export interface Facet {
  id: string;
  label: { en: string; zh: string };
  values: string[];
}

export interface VersionEntry {
  Version: string;
  Note: string;
  HasNote: boolean;
  Additions: { Name: string; Version: string }[];
  facets: Record<string, string[]>;
  parts?: { source?: { name: { en: string; zh: string } } };
}

export interface Variable {
  key: string;
  label: { en: string; zh: string };
  help: { en: string; zh: string };
  value: string;
  kind: string;
  example: string;
  advanced: boolean;
  presets: { value: string; name: { en: string; zh: string } }[];
}

export interface Profile {
  search: SearchChoice | null;
  facets: { id: string; label: Localizable; values: string[] }[];
  variables: Variable[];
  legacyFiles: string[];
  legacyStrategy: boolean;
  shorthands: { name: string; meaning: { en: string; zh: string } }[];
  guide: { en: string; zh: string };
  tips: { en: string; zh: string }[];
  docs: { url: string; title: { en: string; zh: string } }[];
  installed: Record<string, { source?: { name: { en: string; zh: string } }; flavors?: string[] } | undefined> | null;
}

export interface PluginInfo {
  Description: string;
  Homepage: string;
  InstallPath: string;
}

export type RunRequest = Record<string, string>;

export async function api(path: string, opts?: RequestInit): Promise<any> {
  const init: RequestInit = {
    ...(opts || {}),
    headers: { "Content-Type": "application/json", ...((opts && opts.headers) || {}) },
    credentials: "same-origin",
  };
  const r = await fetch(path, init);
  if (r.status === 401) {
    location.href = "/";
    throw new Error(t("needSession"));
  }
  if (!r.ok) {
    const text = await r.text();
    throw new Error(text || t("failed"));
  }
  if (r.status === 204) return null;
  return r.json();
}

// The error toast of the page.
export function showError(msg: string): void {
  const e = document.getElementById("error");
  if (!e) return;
  e.textContent = msg;
  e.style.display = "block";
  window.clearTimeout((showError as unknown as { t?: number }).t);
  (showError as unknown as { t?: number }).t = window.setTimeout(() => {
    e.style.display = "none";
  }, 4000);
}

// Opens an address of another site in a new tab. A link is clicked rather
// than calling window.open, because the WebView of the Android app reports
// the navigation of a link to its shell, which hands the address to the
// browser of the phone.
export function openExternal(url: string): void {
  const a = document.createElement("a");
  a.href = url;
  a.target = "_blank";
  a.rel = "noopener noreferrer";
  document.body.appendChild(a);
  a.click();
  a.remove();
}

export async function copyText(s: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(s);
    snack(t("copied"));
  } catch {
    showError(t("failed"));
  }
}

// The snackbar of the page.
export function snack(msg: string, action?: string, onclick?: () => void): void {
  const box = document.getElementById("snacks");
  if (!box) return;
  const d = document.createElement("div");
  d.className = "snack";
  d.innerHTML = "<span>" + esc(msg) + "</span>" + (action ? "<button>" + esc(action) + "</button>" : "");
  if (action) {
    const b = d.querySelector("button");
    if (b) b.onclick = () => {
      dismiss();
      if (onclick) onclick();
    };
  }
  box.appendChild(d);
  let gone = false;
  const dismiss = () => {
    if (!gone) {
      gone = true;
      d.remove();
    }
  };
  window.setTimeout(dismiss, action ? 8000 : 4000);
}

// A confirm dialog; resolves false when it is dismissed.
export function confirmDlg(title: string, body: string, okLabel: string, danger: boolean): Promise<boolean> {
  return new Promise((resolve) => {
    const scrim = document.getElementById("scrim");
    if (!scrim) return resolve(false);
    (document.getElementById("dlgTitle") as HTMLElement).textContent = title;
    (document.getElementById("dlgBody") as HTMLElement).textContent = body;
    const ok = document.getElementById("dlgOk") as HTMLButtonElement;
    ok.textContent = okLabel;
    ok.className = "btn " + (danger ? "danger" : "primary");
    scrim.classList.add("open");
    const done = (v: boolean) => {
      scrim.classList.remove("open");
      ok.onclick = null;
      (document.getElementById("dlgCancel") as HTMLButtonElement).onclick = null;
      scrim.onclick = null;
      resolve(v);
    };
    ok.onclick = () => done(true);
    (document.getElementById("dlgCancel") as HTMLButtonElement).textContent = t("cancel");
    (document.getElementById("dlgCancel") as HTMLButtonElement).onclick = () => done(false);
    scrim.onclick = (ev) => {
      if (ev.target === scrim) done(false);
    };
  });
}

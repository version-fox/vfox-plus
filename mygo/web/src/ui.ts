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

import { esc, icon } from "./dom";
import { t } from "./i18n";

// Shared rendering pieces: menus, sheets, progress, states and chips.

export function wavy(progress: number | null, indeterminate: boolean): string {
  const wave =
    '<svg width="120" height="8" viewBox="0 0 120 8" preserveAspectRatio="none"><path d="M0 4 Q 10 0 20 4 T 40 4 T 60 4 T 80 4 T 100 4 T 120 4" fill="none" stroke="currentColor" stroke-width="3.2" stroke-linecap="round"/></svg>';
  if (indeterminate)
    return '<div class="wavy ind" role="progressbar"><div class="fill" style="color:var(--m3-primary)">' + wave + "</div></div>";
  const p = Math.max(0, Math.min(1, progress == null ? 0 : progress));
  return (
    '<div class="wavy" role="progressbar" aria-valuenow="' +
    Math.round(p * 100) +
    '"><div class="fill" style="--p:1;width:' +
    p * 100 +
    '%;color:var(--m3-primary)">' +
    wave +
    "</div>" +
    (p > 0.02 && p < 0.999 ? '<span class="stop"></span>' : "") +
    "</div>"
  );
}

export function spinner(): string {
  return '<div class="loader" role="progressbar" aria-label="' + esc(t("loading")) + '"><div class="blob"></div></div>';
}

export function stateBlock(kind: string, ic: string, title: string, body: string, actionLabel: string | null): string {
  return (
    '<div class="state' +
    (kind === "err" ? " err" : "") +
    '"><div class="shape">' +
    icon(ic) +
    "</div><h3>" +
    esc(title) +
    '</h3><div class="t-body">' +
    body +
    "</div>" +
    (actionLabel ? '<button class="btn" data-act="1">' + esc(actionLabel) + "</button>" : "") +
    "</div>"
  );
}

export interface SectionState<T> {
  value: T[] | null;
  loading?: boolean;
  error?: string;
}

export function sectionStatus<T>(sec: SectionState<T> | undefined): string {
  if (!sec) return '<p class="muted">' + esc(t("loading")) + "</p>";
  if (sec.loading && (!sec.value || !sec.value.length)) return spinner();
  if (sec.error && (!sec.value || !sec.value.length))
    return stateBlock("err", "error", t("failed"), esc(sec.error), t("refresh"));
  return "";
}

export function wireStateRetry(root: ParentNode, fn: () => void): void {
  const b = root.querySelector("[data-act]");
  if (b) (b as HTMLButtonElement).onclick = fn;
}

export function section(title: string, help: string, rows: string): string {
  return (
    '<div class="t-title-m" style="margin-top:20px">' +
    esc(title) +
    "</div>" +
    (help ? '<p class="muted t-label" style="margin:0 4px 8px">' + esc(help) + "</p>" : "") +
    '<div class="list">' +
    rows +
    "</div>"
  );
}

export function chip(id: string, label: string, on: boolean, attr?: string): string {
  return (
    '<button class="chip" ' +
    (attr || "") +
    ' data-id="' +
    esc(id) +
    '" aria-pressed="' +
    (on ? "true" : "false") +
    '"><span class="ck">' +
    icon("check", "sm") +
    "</span><span>" +
    esc(label) +
    "</span></button>"
  );
}

export function tagList(words: string[]): string {
  return words.filter(Boolean).map((w) => '<span class="tag">' + esc(w) + "</span>").join(" ");
}

export function partsLine(parts: { source?: { name: { en: string; zh: string } }; flavors?: string[] } | undefined, lang: string): string {
  if (!parts) return "";
  const names: string[] = [];
  if (parts.source) {
    const n = parts.source.name;
    names.push(lang === "zh" && n.zh ? n.zh : n.en);
  }
  for (const f of parts.flavors || []) names.push(f);
  return names.filter(Boolean).join(" · ");
}

export interface MenuItem {
  icon: string;
  label: string;
  danger?: boolean;
  onclick?: () => void;
}

export function openMenu(anchor: Element, items: MenuItem[]): void {
  const menu = document.getElementById("menu");
  if (!menu) return;
  menu.innerHTML = "";
  items.forEach((it) => {
    const b = document.createElement("button");
    if (it.danger) b.className = "danger";
    b.innerHTML = icon(it.icon) + "<span>" + esc(it.label) + "</span>";
    b.onclick = () => {
      closeMenu();
      if (it.onclick) it.onclick();
    };
    menu.appendChild(b);
  });
  const r = anchor.getBoundingClientRect();
  menu.classList.add("open");
  const mw = 220,
    mh = items.length * 48 + 16;
  menu.style.minWidth = mw + "px";
  let x = Math.min(r.left, window.innerWidth - mw - 8);
  let y = r.bottom + 6;
  if (y + mh > window.innerHeight - 8) y = Math.max(8, r.top - mh - 6);
  menu.style.left = Math.max(8, x) + "px";
  menu.style.top = y + "px";
  window.setTimeout(() => document.addEventListener("click", closeMenu, { once: true }), 0);
}

export function closeMenu(): void {
  document.getElementById("menu")?.classList.remove("open");
}

export function openSheet(title: string, ic: string, build: (box: HTMLElement, close: () => void) => void): () => void {
  const scrim = document.getElementById("sheet");
  const box = document.getElementById("sheetBody");
  if (!scrim || !box) return () => {};
  (document.getElementById("sheetIcon") as HTMLElement).innerHTML = ic ? icon(ic) : "";
  (document.getElementById("sheetTitle") as HTMLElement).textContent = title;
  box.innerHTML = "";
  const close = () => {
    scrim.classList.remove("open");
    scrim.onclick = null;
    document.removeEventListener("keydown", onKey);
  };
  const onKey = (ev: KeyboardEvent) => {
    if (ev.key === "Escape") close();
  };
  document.addEventListener("keydown", onKey);
  scrim.onclick = (ev) => {
    if (ev.target === scrim) close();
  };
  scrim.classList.add("open");
  build(box, close);
  return close;
}

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

// The small DOM kit of the page: escaping, icons and avatars. Everything the
// page writes into HTML goes through esc, in double- and single-quoted
// attributes alike (see esc.test.ts: both quotes must be escaped).
export function esc(s: unknown): string {
  return String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"}[c] as string));
}

export function icon(name: string, cls?: string): string {
  return '<svg class="ic' + (cls ? " " + cls : "") + '" aria-hidden="true"><use href="#i-' + name + '"/></svg>';
}

// stableHash is the hash of a name that picks the shape and the colours of its avatar, the same
// as on the desktop (kit.StableHash): FNV-1a over the UTF-16 code units.
export function stableHash(name: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < name.length; i++) {
    h ^= name.charCodeAt(i);
    h = Math.imul(h, 0x01000193) >>> 0;
  }
  return h;
}

export function avatar(name: string, lg?: boolean): string {
  const h = stableHash(name);
  const tones = ["av-p", "av-s", "av-t"];
  return (
    '<span class="avatar ' + tones[h % 3] + " av-" + (h % 12) + (lg ? " lg" : "") + '" aria-hidden="true"><span>' +
    esc((name || "?")[0].toUpperCase()) +
    "</span></span>"
  );
}

/** The localized text of a model value: {zh,en} or a plain string. */
export function txt(o: { zh?: string; en?: string } | string | undefined, lang: string): string {
  if (!o) return "";
  if (typeof o === "string") return o;
  return (lang === "zh" && o.zh ? o.zh : o.en) || "";
}

export function el<T extends HTMLElement>(id: string): T {
  const e = document.getElementById(id);
  if (!e) throw new Error("missing element #" + id);
  return e as T;
}

export function setPressed(button: Element | null, on: boolean): void {
  if (button) button.setAttribute("aria-pressed", String(on));
}

/** An alias of a version, as `vfox alias list` has it and the web API gives it. */
export interface VersionAlias {
  sdk: string;
  alias: string;
  target: string;
}

/** The aliases that name a version of a plugin, as `@j17, @lts`; vfox says the target as `plugin@version`. */
export function aliasesOf(aliases: VersionAlias[], sdk: string, version: string): string {
  return aliases
    .filter((a) => a.sdk === sdk && (a.target === sdk + "@" + version || a.target === version))
    .map((a) => "@" + a.alias)
    .join(", ");
}

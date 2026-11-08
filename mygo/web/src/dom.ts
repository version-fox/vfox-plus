// The small DOM kit of the page: escaping, icons and avatars. Everything the
// page writes into HTML goes through esc, in double- and single-quoted
// attributes alike (see esc.test.ts: both quotes must be escaped).
export function esc(s: unknown): string {
  return String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"}[c] as string));
}

export function icon(name: string, cls?: string): string {
  return '<svg class="ic' + (cls ? " " + cls : "") + '" aria-hidden="true"><use href="#i-' + name + '"/></svg>';
}

function hashName(name: string): number {
  let h = 0;
  for (const c of name) h = (h * 31 + c.codePointAt(0)!) >>> 0;
  return h;
}

export function avatar(name: string, lg?: boolean): string {
  const h = hashName(name);
  const tones = ["av-p", "av-s", "av-t"];
  return (
    '<span class="avatar ' + tones[h % 3] + " av-" + (h % 12) + (lg ? " lg" : "") + '" aria-hidden="true">' +
    esc((name || "?")[0].toUpperCase()) +
    "</span>"
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

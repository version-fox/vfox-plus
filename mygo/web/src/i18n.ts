// The message catalogs, one per language, with the key type shared: a
// missing key in either language is a compile error, not a blank label.
import { en } from "./i18n-en";
import { zh } from "./i18n-zh";

export type Key = keyof typeof en;

// The Chinese catalog must carry exactly the keys of the English one.
const _zhComplete: Record<Key, string> = zh;

export type Lang = "en" | "zh";

const catalogs: Record<Lang, Record<Key, string>> = { en, zh: _zhComplete };

let lang: Lang = "en";

export function setLang(l: Lang): void {
  lang = l;
}

export function getLang(): Lang {
  return lang;
}

export function systemLang(): Lang {
  try {
    return (navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en";
  } catch {
    return "en";
  }
}

/** The message for the key in the current language, English to fall back on. */
export function t(k: Key): string {
  return catalogs[lang][k] ?? en[k] ?? k;
}

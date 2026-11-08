import { state, type TabName } from "./state";
import { t } from "./i18n";

// The hash router: #/home, #/sdks, #/sdk/<name>, #/plugins, #/tasks,
// #/settings. The tabs are sections of the one page; switching fades the new
// one through on the slow effects spring.
const RM = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

const TABS: TabName[] = ["home", "sdks", "plugins", "tasks", "settings"];

export function switchTab(name: TabName, push?: boolean, onShow?: (tab: TabName) => void): void {
  if (state.tab === name) return;
  state.tab = name;
  document.body.dataset.tab = name;
  document.querySelectorAll(".dest").forEach((b) => {
    if (b instanceof HTMLElement) b.setAttribute("aria-current", String(b.dataset.tab === name));
  });
  document.querySelectorAll(".page").forEach((p) => p.classList.remove("active", "enter"));
  const page = document.getElementById("tab-" + name);
  if (page) {
    page.classList.add("active");
    if (!RM) page.classList.add("enter");
  }
  const title = document.getElementById("pageTitle");
  if (title) title.textContent = t(name === "sdks" ? "sdks" : name);
  if (push !== false && location.hash !== "#/" + name) history.pushState({ tab: name }, "", "#/" + name);
  if (onShow) onShow(name);
}

export function currentRoute(): { tab: TabName; sdk?: string } {
  const h = (location.hash || "#/home").slice(2);
  const parts = h.split("/");
  if (parts[0] === "sdk" && parts[1]) return { tab: "sdks", sdk: parts[1] };
  if ((TABS as string[]).includes(parts[0])) return { tab: parts[0] as TabName };
  return { tab: "home" };
}

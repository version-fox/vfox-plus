import { api, showError, snack, type Task } from "./api";
import { el, esc, icon } from "./dom";
import { t } from "./i18n";
import { refresh } from "./main";
import { state } from "./state";
import { stateBlock, wavy } from "./ui";

// ---------------- tasks ----------------
function statusIcon(st: string): string {
  return (
    { queued: "hourglass-empty", running: "schedule", succeeded: "task-alt", failed: "error", cancelled: "close" }[st] ||
    "info"
  );
}

export function renderTasks(): void {
  const box = el("taskList");
  (["taskCount", "taskCountM"] as const).forEach((id) => {
    const badge = document.getElementById(id);
    if (badge) {
      const n = state.tasks.filter((x) => ["queued", "running"].includes(x.status)).length;
      badge.hidden = n === 0;
      badge.textContent = String(n);
    }
  });
  if (!state.tasks.length) {
    box.innerHTML = stateBlock("empty", "pending-actions", t("noTasks"), "", null);
    return;
  }
  box.innerHTML = "";
  state.tasks.forEach((task) => {
    const d = document.createElement("div");
    d.className = "card";
    const pct = task.progress != null ? Math.round(task.progress * 100) + "%" : "";
    d.innerHTML =
      "<div class='row'>" +
      icon(statusIcon(task.status)) +
      "<b>" +
      esc(task.operation) +
      "</b> <code>" +
      esc(task.sdk + (task.version ? "@" + task.version : "")) +
      "</code> <span class='muted t-label'>" +
      esc(task.status + " " + pct) +
      "</span><span style='flex:1'></span><button class='btn text' data-open='" +
      task.id +
      "'>" +
      esc(t("log")) +
      "</button></div>" +
      (task.progress != null && ["queued", "running"].includes(task.status) ? wavy(task.progress, false) : "") +
      (["queued", "running"].includes(task.status) && task.progress == null ? wavy(0, true) : "") +
      (task.errorMessage ? "<p style='color:var(--m3-error)'>" + esc(task.errorMessage) + "</p>" : "") +
      "<div data-log='" +
      task.id +
      "' hidden></div>";
    box.appendChild(d);
  });
  box.querySelectorAll("[data-open]").forEach(
    (b) => ((b as HTMLButtonElement).onclick = () => openTask(parseInt((b as HTMLButtonElement).dataset.open || "0", 10))),
  );
}

export async function openTask(id: number): Promise<void> {
  const slot = document.querySelector("[data-log='" + id + "']");
  if (!slot) return;
  if (!(slot as HTMLElement).hidden) {
    (slot as HTMLElement).hidden = true;
    slot.innerHTML = "";
    return;
  }
  try {
    const task = (await api("/api/tasks/" + id)) as Task & {
      commandLine: string;
      droppedLines: number;
      hasFullLog: boolean;
      log: string[];
    };
    (slot as HTMLElement).hidden = false;
    const isWebView = (() => {
      try {
        return /;\s*wv\)/.test(navigator.userAgent);
      } catch {
        return false;
      }
    })();
    slot.innerHTML =
      "<p class='muted'><code>" +
      esc(task.commandLine) +
      "</code></p>" +
      (task.droppedLines ? "<p class='muted'>" + esc(t("droppedHead") + task.droppedLines + t("droppedTail")) + "</p>" : "") +
      (task.hasFullLog && !isWebView
        ? "<div class='row'><a class='btn text' download href='/api/tasks/" +
          task.id +
          "/log'>" +
          esc(t("downloadLog")) +
          "</a></div>"
        : "") +
      "<pre class='log'>" +
      esc(task.log.join("\n")) +
      "</pre><div class='row'>" +
      (["queued", "running"].includes(task.status)
        ? "<button class='btn danger' data-cancel='" + task.id + "'>" + esc(t("cancel")) + "</button>"
        : "") +
      (["failed", "cancelled"].includes(task.status)
        ? "<button class='btn' data-retry='" + task.id + "'>" + esc(t("retry")) + "</button>"
        : "") +
      "</div>";
    const c = slot.querySelector("[data-cancel]");
    if (c)
      (c as HTMLButtonElement).onclick = async () => {
        await api("/api/tasks/" + id + "/cancel", { method: "POST" });
        refresh(false);
      };
    const r = slot.querySelector("[data-retry]");
    if (r)
      (r as HTMLButtonElement).onclick = async () => {
        await api("/api/tasks/" + id + "/retry", { method: "POST" });
        refresh(false);
      };
  } catch (e) {
    showError(e instanceof Error ? e.message : String(e));
  }
}

export async function runTask(req: Record<string, string>): Promise<void> {
  try {
    await api("/api/tasks", { method: "POST", body: JSON.stringify(req) });
    location.hash = "#/tasks";
    await refresh(false);
  } catch (e) {
    showError(e instanceof Error ? e.message : String(e));
  }
}

export function watchTasks(tasks: Task[]): void {
  tasks.forEach((task) => {
    const prev = state.prevTasks[task.id];
    if (prev && ["queued", "running"].includes(prev) && ["succeeded"].includes(task.status))
      snack(t("taskSucceeded"), t("view"), () => (location.hash = "#/tasks"));
    if (prev && ["queued", "running"].includes(prev) && ["failed", "cancelled"].includes(task.status))
      snack(task.errorMessage || t("taskFailed"), t("view"), () => (location.hash = "#/tasks"));
  });
  state.prevTasks = {};
  tasks.forEach((task) => (state.prevTasks[task.id] = task.status));
}

export function initTasks(): void {
  el("clearBtn").onclick = async () => {
    try {
      await api("/api/tasks/finished", { method: "DELETE" });
      refresh(false);
    } catch (e) {
      showError(e instanceof Error ? e.message : String(e));
    }
  };
}

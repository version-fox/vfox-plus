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

import { api, copyText, showError, snack, type Task } from "./api";
import { el, esc, icon } from "./dom";
import { t, type Key } from "./i18n";
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

/** The status as the page says it, not as the API names it. */
function statusLabel(status: string): string {
  const key: Record<string, Key> = {
    queued: "statusQueued",
    running: "statusRunning",
    succeeded: "statusDone",
    failed: "statusFailed",
    cancelled: "statusCancelled",
  };
  return key[status] ? t(key[status]) : status;
}

/** "install nodejs@24.11.0" -> "Install nodejs@24.11.0" */
function taskTitle(task: Task): string {
  const op = task.operation || "";
  if (op === "aliasSet" || op === "aliasUnset") {
    const named = task.sdk + (task.version ? "@" + task.version : "");
    return t(op === "aliasSet" ? "taskAliasSet" : "taskAliasUnset").replace("{alias}", task.alias || "").replace("{subject}", named);
  }
  const known = ["install", "installUse", "uninstall", "update", "remove", "use", "unuse", "add", "link", "linkDetect", "aliasSet", "aliasUnset", "doctorAdopt"];
  const label = known.includes(op) ? t(op as Key) : op;
  const subject = task.sdk + (task.version ? "@" + task.version : "");
  return (label.charAt(0).toUpperCase() + label.slice(1) + (subject ? " " + subject : "")).trim();
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
  box.innerHTML = state.tasks
    .map((task) => {
      const pct = task.progress != null ? Math.round(task.progress * 100) + "%" : "";
      const running = ["queued", "running"].includes(task.status);
      return (
        '<button class="taskrow" data-open="' +
        task.id +
        '" aria-expanded="false">' +
        icon(statusIcon(task.status)) +
        '<span class="grow"><b>' +
        esc(taskTitle(task)) +
        '</b><span class="muted">' +
        esc(statusLabel(task.status) + (pct ? " · " + pct : "")) +
        "</span>" +
        (running && task.progress != null ? wavy(task.progress, false) : "") +
        (running && task.progress == null ? wavy(0, true) : "") +
        "</span>" +
        icon("keyboard-arrow-down", "sm muted") +
        "</button>" +
        '<div class="tasklog" data-log="' +
        task.id +
        '" hidden></div>'
      );
    })
    .join("");
  box.querySelectorAll("[data-open]").forEach(
    (b) =>
      ((b as HTMLButtonElement).onclick = () => {
        openTask(parseInt((b as HTMLButtonElement).dataset.open || "0", 10));
      }),
  );
}

export async function openTask(id: number): Promise<void> {
  const slot = document.querySelector("[data-log='" + id + "']") as HTMLElement | null;
  const row = document.querySelector("[data-open='" + id + "']") as HTMLElement | null;
  if (!slot) return;
  if (!slot.hidden) {
    slot.hidden = true;
    slot.innerHTML = "";
    if (row) row.setAttribute("aria-expanded", "false");
    return;
  }
  try {
    const task = (await api("/api/tasks/" + id)) as Task & {
      commandLine: string;
      droppedLines: number;
      hasFullLog: boolean;
      log: string[];
    };
    if (row) row.setAttribute("aria-expanded", "true");
    slot.hidden = false;
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
      (task.droppedLines
        ? "<p class='muted'>" + esc(t("droppedHead") + task.droppedLines + t("droppedTail")) + "</p>"
        : "") +
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
      '<span class="grow"></span><button class="iconbtn sm" data-copy="1" title="' +
      esc(t("log")) +
      '">' +
      icon("content-copy", "sm") +
      "</button></div>";
    const copy = slot.querySelector("[data-copy]");
    if (copy) (copy as HTMLButtonElement).onclick = () => copyText(task.log.join("\n"));
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

export async function runTask(req: Record<string, string | boolean>): Promise<void> {
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

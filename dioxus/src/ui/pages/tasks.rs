// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//! Running, queued and finished operations with their output.

use dioxus::prelude::*;

use super::sdk_detail::percent;
use crate::core::command::VfoxOperation;
use crate::state::tasks::{TaskStatus, VfoxTask};
use crate::ui::bridge;
use crate::ui::components::parts::{IconButton, StatusMessage, icon, page_header, unique_id};
use crate::ui::context::use_app;
use crate::ui::desktop;

/// The Tasks page.
#[component]
pub fn TasksPage() -> Element {
    let app = use_app();
    let model = app.model;
    let tasks = model.tasks.tasks();
    let count = tasks.len();
    let actions = rsx! {
        m3e-button { "variant": "text", "disabled": !model.tasks.has_finished(), onclick: move |_| model.tasks.clear_finished(),
            {icon("succeeded", false, Some("icon"))}
            {app.t("tasksClearFinished")}
        }
    };
    rsx! {
        div { class: "page-root tasks-page",
            div { class: "pane",
                {page_header(app.t("navTasks"), None, actions)}
                if tasks.is_empty() {
                    StatusMessage { icon: "tasks", shape: "sunny", heading: app.t("tasksEmptyTitle"), body: app.t("tasksEmptyBody") }
                } else {
                    div { class: "list",
                        for (index, task) in tasks.into_iter().enumerate() {
                            TaskItem { key: "{task.id}", task, first: index == 0, last: index + 1 == count }
                        }
                    }
                }
            }
        }
    }
}

/// One task of the list: status, title, progress, actions and its output.
#[component]
fn TaskItem(task: VfoxTask, first: bool, last: bool) -> Element {
    let app = use_app();
    let model = app.model;
    let mut expanded = use_signal(|| false);
    let output_id = use_hook(|| unique_id("task-output"));
    let state = task.state.read();
    let status = state.status;
    let progress = state.progress;
    let leading = match status {
        TaskStatus::Queued => icon("queued", false, None),
        TaskStatus::Running => rsx! { m3e-loading-indicator { "aria-label": app.t("taskStatusRunning") } },
        TaskStatus::Succeeded => icon("succeeded", true, None),
        TaskStatus::Failed => icon("error", true, None),
        TaskStatus::Cancelled => icon("cancel", false, None),
    };
    let detail = match status {
        TaskStatus::Queued => app.t("taskStatusQueued"),
        TaskStatus::Running => match progress {
            Some(progress) => app.tf("taskDownloading", &[("percent", percent(progress).into())]),
            None => app.t("taskStatusRunning"),
        },
        TaskStatus::Succeeded => app.t("taskStatusSucceeded"),
        TaskStatus::Failed => {
            format!("{} · {}", app.t("taskStatusFailed"), state.error_message.clone().unwrap_or_default())
        }
        TaskStatus::Cancelled => app.t("taskStatusCancelled"),
    };
    let command = format!("$ {}", task.command.command_line());
    let text = std::iter::once(command.clone()).chain(state.log.iter().cloned()).collect::<Vec<_>>().join("\n");
    let shown = if state.log.is_empty() { format!("{command}\n{}", app.t("taskNoOutput")) } else { text.clone() };
    let lines = state.log.len();
    drop(state);
    // Show the end of the output, where vfox writes.
    let scrolled = output_id.clone();
    use_effect(use_reactive((&lines,), move |(_lines,)| {
        if expanded() {
            bridge::scroll_to_end(&scrolled);
        }
    }));
    let class = format!(
        "task-item{}{}{}",
        if first { " first" } else { "" },
        if last { " last" } else { "" },
        if expanded() { " expanded" } else { "" }
    );
    let status_class = match status {
        TaskStatus::Succeeded => "status succeeded",
        TaskStatus::Failed => "status failed",
        _ => "status",
    };
    let (cancel, retry) = (task.clone(), task.clone());
    rsx! {
        div { class,
            div { class: "item", onclick: move |_| expanded.toggle(),
                div { class: status_class, {leading} }
                div { class: "text",
                    div { class: "title type-body-large-emphasized", {app.task_title(&task.command)} }
                    div { class: if status == TaskStatus::Failed { "detail type-body-medium failed" } else { "detail type-body-medium" }, "{detail}" }
                    if status == TaskStatus::Running && task.command.operation == VfoxOperation::Install {
                        m3e-linear-progress-indicator {
                            "variant": "wavy",
                            "mode": if progress.is_some() { "determinate" } else { "indeterminate" },
                            "value": "{percent(progress.unwrap_or(0.0))}",
                        }
                    }
                }
                div { class: "actions", onclick: move |event| event.stop_propagation(),
                    if status.is_active() {
                        IconButton { icon: "cancel", label: app.t("actionCancel"), onclick: move |_| model.tasks.cancel(&cancel) }
                    } else if matches!(status, TaskStatus::Failed | TaskStatus::Cancelled) {
                        IconButton {
                            icon: "retry",
                            label: app.t("actionRetry"),
                            onclick: move |_| {
                                model.tasks.retry(&retry);
                            },
                        }
                    }
                    IconButton {
                        class: "toggle",
                        icon: "expandMore",
                        label: if expanded() { app.t("taskHideOutput") } else { app.t("taskShowOutput") },
                        onclick: move |_| expanded.toggle(),
                    }
                }
            }
            if expanded() {
                div { class: "output-frame",
                    div {
                        div { class: "output",
                            pre { id: "{output_id}", class: "mono type-body-small", "{shown}" }
                            IconButton {
                                icon: "copy",
                                label: app.t("taskCopyOutput"),
                                onclick: move |_| {
                                    desktop::copy_text(&text);
                                    app.notify(app.t("actionCopied"));
                                },
                            }
                        }
                    }
                }
            }
        }
    }
}

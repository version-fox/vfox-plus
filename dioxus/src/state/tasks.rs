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

//! The queue of state-changing vfox commands.

use std::collections::VecDeque;
use std::rc::Rc;
use std::time::{SystemTime, UNIX_EPOCH};

use dioxus::prelude::*;

use crate::core::cli::VfoxCli;
use crate::core::command::{VfoxCommand, VfoxOperation};
use crate::core::output::{parse_download_progress, summarize_error};
use crate::core::process::{Killer, ProcessEvent};
use crate::core::terminal::TerminalLineSplitter;

/// The life cycle of a [`VfoxTask`].
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TaskStatus {
    /// Waiting for earlier tasks to finish.
    Queued,
    /// The vfox process runs.
    Running,
    /// vfox exited with code zero.
    Succeeded,
    /// vfox could not be started or exited with a non-zero code.
    Failed,
    /// The user cancelled the task.
    Cancelled,
}

impl TaskStatus {
    /// Whether the task has not finished yet.
    pub fn is_active(self) -> bool {
        matches!(self, Self::Queued | Self::Running)
    }
}

/// What a task did so far.
#[derive(Clone, Debug, PartialEq)]
pub struct TaskState {
    /// Where the task is in its life cycle.
    pub status: TaskStatus,
    /// Download progress between 0 and 1, if vfox reported any.
    pub progress: Option<f64>,
    /// When the process started, in milliseconds since the epoch.
    pub started_at: Option<u64>,
    /// When the task finished, in milliseconds since the epoch.
    pub finished_at: Option<u64>,
    /// The exit code of vfox, if it ran to completion.
    pub exit_code: Option<i32>,
    /// A short description of the failure, if the task failed.
    pub error_message: Option<String>,
    /// Output lines (standard output and error interleaved), without progress
    /// redraws; the most recent only.
    pub log: VecDeque<String>,
    /// How many old lines were dropped from the log.
    pub dropped_lines: usize,
    stdout: VecDeque<String>,
    stderr: VecDeque<String>,
}

/// How many lines of each stream are kept for the error summary.
const SUMMARY_LINES: usize = 200;

impl TaskState {
    fn append(&mut self, line: String, is_error: bool, max_log_lines: usize) {
        if let Some(progress) = parse_download_progress(&line) {
            self.progress = Some(progress);
            return;
        }
        let stream = if is_error { &mut self.stderr } else { &mut self.stdout };
        stream.push_back(line.clone());
        if stream.len() > SUMMARY_LINES {
            stream.pop_front();
        }
        self.log.push_back(line);
        if self.log.len() > max_log_lines {
            self.log.pop_front();
            self.dropped_lines += 1;
        }
    }

    fn summary(&self) -> String {
        let join = |lines: &VecDeque<String>| lines.iter().cloned().collect::<Vec<_>>().join("\n");
        summarize_error(&join(&self.stdout), &join(&self.stderr))
    }
}

/// One execution of a state-changing [`VfoxCommand`].
#[derive(Clone)]
pub struct VfoxTask {
    /// Sequential identifier, unique within a [`TaskQueue`].
    pub id: u32,
    /// The command that is executed.
    pub command: Rc<VfoxCommand>,
    /// When the task was queued, in milliseconds since the epoch.
    pub created_at: u64,
    /// What the task did so far.
    pub state: Signal<TaskState>,
    process: CopyValue<Option<Killer>>,
}

impl PartialEq for VfoxTask {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id && self.state == other.state
    }
}

impl VfoxTask {
    /// The current state.
    pub fn status(&self) -> TaskStatus {
        self.state.read().status
    }

    /// The download progress, if vfox reported any.
    pub fn progress(&self) -> Option<f64> {
        self.state.read().progress
    }
}

fn now() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |time| u64::try_from(time.as_millis()).unwrap_or(u64::MAX))
}

type Listener = Rc<dyn Fn(&VfoxTask)>;

/// Runs state-changing vfox commands one after another. vfox does not lock
/// its data directory, so concurrent installs, uses and removals could
/// corrupt `.vfox.toml` or the symlinks; the queue runs at most one command at
/// a time, in the order they were queued.
#[derive(Clone, Copy, PartialEq)]
pub struct TaskQueue {
    cli: CopyValue<VfoxCli>,
    all: Signal<Vec<VfoxTask>>,
    listeners: CopyValue<Vec<(u32, Listener)>>,
    counters: CopyValue<(u32, u32)>,
    running: CopyValue<bool>,
    max_log_lines: usize,
    max_finished_tasks: usize,
}

impl TaskQueue {
    /// A queue of the current scope that runs commands with `cli`.
    pub fn new(cli: VfoxCli) -> Self {
        Self::with_limits(cli, 2000, 50)
    }

    /// A queue that keeps `max_log_lines` lines per task and
    /// `max_finished_tasks` finished tasks.
    pub fn with_limits(cli: VfoxCli, max_log_lines: usize, max_finished_tasks: usize) -> Self {
        Self {
            cli: CopyValue::new(cli),
            all: Signal::new(Vec::new()),
            listeners: CopyValue::new(Vec::new()),
            counters: CopyValue::new((1, 1)),
            running: CopyValue::new(false),
            max_log_lines,
            max_finished_tasks,
        }
    }

    /// All known tasks, newest first.
    pub fn tasks(&self) -> Vec<VfoxTask> {
        self.all.read().iter().rev().cloned().collect()
    }

    /// The number of queued and running tasks.
    pub fn active_count(&self) -> usize {
        self.all.read().iter().filter(|task| task.status().is_active()).count()
    }

    /// Whether any finished task can be cleared.
    pub fn has_finished(&self) -> bool {
        self.all.read().iter().any(|task| !task.status().is_active())
    }

    /// Calls `listener` with every task once it has finished, failed or been
    /// cancelled; returns what removes the listener again.
    pub fn on_finished(&self, listener: impl Fn(&VfoxTask) + 'static) -> u32 {
        let mut counters = self.counters;
        let id = counters.peek().1;
        counters.write().1 += 1;
        let mut listeners = self.listeners;
        listeners.write().push((id, Rc::new(listener)));
        id
    }

    /// Removes the listener `id` of [`TaskQueue::on_finished`].
    pub fn remove_listener(&self, id: u32) {
        let mut listeners = self.listeners;
        if let Ok(mut listeners) = listeners.try_write() {
            listeners.retain(|(existing, _)| *existing != id);
        }
    }

    /// Queues `command` and returns its task. If an identical command is
    /// queued or running already, its task is returned instead.
    pub fn enqueue(&self, command: VfoxCommand) -> VfoxTask {
        let existing = self
            .all
            .peek()
            .iter()
            .find(|task| task.status().is_active() && same_command(&task.command, &command))
            .cloned();
        if let Some(task) = existing {
            return task;
        }
        let mut counters = self.counters;
        let id = counters.peek().0;
        counters.write().0 += 1;
        let task = VfoxTask {
            id,
            command: Rc::new(command),
            created_at: now(),
            state: Signal::new_in_scope(
                TaskState {
                    status: TaskStatus::Queued,
                    progress: None,
                    started_at: None,
                    finished_at: None,
                    exit_code: None,
                    error_message: None,
                    log: VecDeque::new(),
                    dropped_lines: 0,
                    stdout: VecDeque::new(),
                    stderr: VecDeque::new(),
                },
                self.all.origin_scope(),
            ),
            process: CopyValue::new_in_scope(None, self.all.origin_scope()),
        };
        let mut all = self.all;
        all.write().push(task.clone());
        self.trim_finished();
        self.pump();
        task
    }

    /// Queues the command of `task` again.
    pub fn retry(&self, task: &VfoxTask) -> VfoxTask {
        self.enqueue((*task.command).clone())
    }

    /// Cancels `task`: a queued task is dropped, a running process is killed.
    pub fn cancel(&self, task: &VfoxTask) {
        match task.status() {
            TaskStatus::Queued => self.finish(task, TaskStatus::Cancelled),
            TaskStatus::Running => {
                if let Some(kill) = task.process.peek().clone() {
                    kill();
                }
                // The task is settled once the process has exited.
                let mut state = task.state;
                let mut state = state.write();
                state.error_message = None;
                state.status = TaskStatus::Cancelled;
            }
            _ => {}
        }
    }

    /// Forgets every finished task.
    pub fn clear_finished(&self) {
        let mut all = self.all;
        all.write().retain(|task| task.status().is_active());
    }

    /// The active task matching the filter, if any.
    pub fn active_task(
        &self,
        operation: Option<VfoxOperation>,
        sdk: Option<&str>,
        version: Option<&str>,
    ) -> Option<VfoxTask> {
        self.all
            .read()
            .iter()
            .find(|task| {
                task.status().is_active()
                    && operation.is_none_or(|operation| task.command.operation == operation)
                    && sdk.is_none_or(|sdk| task.command.sdk.as_deref() == Some(sdk))
                    && version.is_none_or(|version| task.command.version.as_deref() == Some(version))
            })
            .cloned()
    }

    /// Kills the processes that still run, when the application quits.
    pub fn dispose(&self) {
        for task in self.all.peek().iter() {
            if let Some(kill) = task.process.peek().clone() {
                kill();
            }
        }
        let mut listeners = self.listeners;
        listeners.write().clear();
    }

    fn pump(&self) {
        if *self.running.peek() {
            return;
        }
        let Some(next) = self.all.peek().iter().find(|task| task.status() == TaskStatus::Queued).cloned() else {
            return;
        };
        let mut running = self.running;
        running.set(true);
        super::spawn_in(self.all.origin_scope(), self.execute(next));
    }

    async fn execute(self, task: VfoxTask) {
        let mut state = task.state;
        {
            let mut state = state.write();
            state.started_at = Some(now());
            state.status = TaskStatus::Running;
        }
        let cli = self.cli.peek().clone();
        match cli.start(&task.command) {
            Err(error) => {
                state.write().error_message = Some(error.message().to_owned());
                self.finish(&task, TaskStatus::Failed);
            }
            Ok(mut process) => {
                let mut killer = task.process;
                killer.set(Some(process.killer()));
                // Cancelled while the process was being spawned.
                if task.status() == TaskStatus::Cancelled {
                    process.kill();
                }
                let mut stdout = TerminalLineSplitter::new();
                let mut stderr = TerminalLineSplitter::new();
                let code = loop {
                    match process.next().await {
                        Some(ProcessEvent::Stdout(chunk)) => self.append(&task, stdout.push(&chunk), false),
                        Some(ProcessEvent::Stderr(chunk)) => self.append(&task, stderr.push(&chunk), true),
                        Some(ProcessEvent::Exit(code)) => break code,
                        None => break -1,
                    }
                };
                self.append(&task, stdout.close(), false);
                self.append(&task, stderr.close(), true);
                killer.set(None);
                self.settle(&task, code);
            }
        }
        let mut running = self.running;
        running.set(false);
        self.pump();
    }

    fn append(&self, task: &VfoxTask, lines: Vec<String>, is_error: bool) {
        if lines.is_empty() {
            return;
        }
        let mut state = task.state;
        let mut state = state.write();
        for line in lines {
            state.append(line, is_error, self.max_log_lines);
        }
    }

    fn settle(&self, task: &VfoxTask, exit_code: i32) {
        let mut state = task.state;
        state.write().exit_code = Some(exit_code);
        if task.status() == TaskStatus::Cancelled {
            self.finish(task, TaskStatus::Cancelled);
        } else if exit_code == 0 {
            self.finish(task, TaskStatus::Succeeded);
        } else {
            let summary = state.peek().summary();
            state.write().error_message =
                Some(if summary.is_empty() { format!("vfox exited with code {exit_code}") } else { summary });
            self.finish(task, TaskStatus::Failed);
        }
    }

    fn finish(&self, task: &VfoxTask, status: TaskStatus) {
        let mut state = task.state;
        {
            let mut state = state.write();
            state.finished_at = Some(now());
            state.status = status;
        }
        let listeners: Vec<Listener> = self.listeners.peek().iter().map(|(_, listener)| listener.clone()).collect();
        for listener in listeners {
            listener(task);
        }
    }

    fn trim_finished(&self) {
        let finished: Vec<u32> =
            self.all.peek().iter().filter(|task| !task.status().is_active()).map(|task| task.id).collect();
        let excess = finished.len().saturating_sub(self.max_finished_tasks);
        if excess > 0 {
            let forgotten = &finished[..excess];
            let mut all = self.all;
            all.write().retain(|task| !forgotten.contains(&task.id));
        }
    }
}

fn same_command(a: &VfoxCommand, b: &VfoxCommand) -> bool {
    a.working_directory == b.working_directory && a.arguments == b.arguments
}

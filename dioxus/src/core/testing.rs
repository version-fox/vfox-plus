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

//! Test support: the captured output of `shared/fixtures` and a scripted
//! process runner.

use std::cell::{Cell, RefCell};
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::rc::Rc;

use tokio::sync::mpsc::{UnboundedSender, unbounded_channel};

use super::output::PLUGIN_INFO_TEMPLATE;
use super::process::{
    BoxFuture, ProcessEvent, ProcessOptions, ProcessOutput, ProcessRunner, ProcessStartError, RunningProcess,
};

/// The captured vfox outputs that the tests use by default.
pub const FIXTURES_VERSION: &str = "1.0.13";

/// The shared resources of the repository.
pub fn shared() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../shared")
}

/// Reads a capture of `shared/fixtures`, in which control characters are
/// `\xHH` and backslashes `\\`.
///
/// # Panics
///
/// When the capture cannot be read.
pub fn read_capture(path: &Path) -> String {
    let text = std::fs::read_to_string(path).unwrap_or_else(|error| panic!("{}: {error}", path.display()));
    let mut result = String::with_capacity(text.len());
    let mut chars = text.chars().peekable();
    while let Some(c) = chars.next() {
        if c != '\\' {
            result.push(c);
            continue;
        }
        match chars.peek() {
            Some('\\') => {
                chars.next();
                result.push('\\');
            }
            Some('x') => {
                chars.next();
                let hex: String = chars.by_ref().take(2).collect();
                result.push(char::from(u8::from_str_radix(&hex, 16).expect("a \\xHH escape")));
            }
            _ => result.push('\\'),
        }
    }
    result
}

/// The output of the fixture `name`, with `exit_code`.
pub fn fixture_output(name: &str, exit_code: i32) -> ProcessOutput {
    let base = shared().join(format!("fixtures/vfox-{FIXTURES_VERSION}"));
    ProcessOutput {
        exit_code,
        stdout: read_capture(&base.join(format!("{name}.stdout"))),
        stderr: read_capture(&base.join(format!("{name}.stderr"))),
    }
}

/// A plain output.
pub fn output(exit_code: i32, stdout: &str, stderr: &str) -> ProcessOutput {
    ProcessOutput { exit_code, stdout: stdout.into(), stderr: stderr.into() }
}

/// One recorded invocation of a [`FakeProcessRunner`].
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProcessCall {
    /// The executable.
    pub executable: String,
    /// The arguments.
    pub arguments: Vec<String>,
    /// How the process was started.
    pub options: ProcessOptions,
    /// Whether the output was streamed.
    pub streamed: bool,
}

/// A started process whose output and exit the test drives.
#[derive(Clone)]
pub struct FakeProcess {
    sender: UnboundedSender<ProcessEvent>,
    killed: Rc<Cell<bool>>,
}

impl FakeProcess {
    /// Writes `text` to standard output.
    pub fn stdout(&self, text: &str) {
        let _ = self.sender.send(ProcessEvent::Stdout(text.as_bytes().to_vec()));
    }

    /// Writes `text` to standard error.
    pub fn stderr(&self, text: &str) {
        let _ = self.sender.send(ProcessEvent::Stderr(text.as_bytes().to_vec()));
    }

    /// Exits with `code`.
    pub fn exit(&self, code: i32) {
        let _ = self.sender.send(ProcessEvent::Exit(code));
    }

    /// Whether the process was killed.
    pub fn killed(&self) -> bool {
        self.killed.get()
    }
}

type Response = Rc<dyn Fn() -> ProcessOutput>;
type Processes = Rc<RefCell<Vec<FakeProcess>>>;

/// A scripted [`ProcessRunner`]: responses are matched by the joined argument
/// list, so tests read like the command lines they expect. Unmatched queries
/// fail with exit code 127, unmatched streamed commands succeed silently.
#[derive(Clone, Default)]
pub struct FakeProcessRunner {
    calls: Rc<RefCell<Vec<ProcessCall>>>,
    responses: Rc<RefCell<HashMap<String, Response>>>,
    streams: Rc<RefCell<HashMap<String, Processes>>>,
    held: Rc<RefCell<std::collections::HashSet<String>>>,
    missing: Rc<Cell<bool>>,
}

impl FakeProcessRunner {
    /// A runner without responses.
    pub fn new() -> Self {
        Self::default()
    }

    /// Answers the command line `arguments` with `output`.
    pub fn on(&self, arguments: &str, output: ProcessOutput) -> &Self {
        self.responses.borrow_mut().insert(arguments.to_owned(), Rc::new(move || output.clone()));
        self
    }

    /// Makes streamed starts whose command line starts with `prefix` wait
    /// for the test, which drives them through [`FakeProcessRunner::processes`].
    pub fn hold(&self, prefix: &str) -> &Self {
        self.held.borrow_mut().insert(prefix.to_owned());
        self
    }

    /// The held processes of [`FakeProcessRunner::hold`] with `prefix`, in order.
    pub fn processes(&self, prefix: &str) -> Vec<FakeProcess> {
        self.streams.borrow().get(prefix).map(|processes| processes.borrow().clone()).unwrap_or_default()
    }

    /// When set, every invocation fails as if the executable did not exist.
    pub fn set_missing_executable(&self, missing: bool) {
        self.missing.set(missing);
    }

    /// Every invocation so far.
    pub fn calls(&self) -> Vec<ProcessCall> {
        self.calls.borrow().clone()
    }

    /// The invocations whose arguments start with `prefix`.
    pub fn calls_starting(&self, prefix: &str) -> Vec<ProcessCall> {
        self.calls().into_iter().filter(|call| call.arguments.join(" ").starts_with(prefix)).collect()
    }

    fn record(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
        streamed: bool,
    ) -> Result<(), ProcessStartError> {
        self.calls.borrow_mut().push(ProcessCall {
            executable: executable.into(),
            arguments: arguments.to_vec(),
            options: options.clone(),
            streamed,
        });
        if self.missing.get() {
            return Err(ProcessStartError {
                executable: executable.into(),
                message: "No such file or directory".into(),
            });
        }
        Ok(())
    }

    fn response(&self, key: &str) -> Option<ProcessOutput> {
        self.responses.borrow().get(key).map(|response| response())
    }
}

impl ProcessRunner for FakeProcessRunner {
    fn run(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> BoxFuture<Result<ProcessOutput, ProcessStartError>> {
        let result = self
            .record(executable, arguments, options, false)
            .map(|()| self.response(&arguments.join(" ")).unwrap_or_else(|| output(127, "", "unexpected command")));
        Box::pin(async move { result })
    }

    fn start(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> Result<RunningProcess, ProcessStartError> {
        self.record(executable, arguments, options, true)?;
        let key = arguments.join(" ");
        let (sender, receiver) = unbounded_channel();
        let process = FakeProcess { sender, killed: Rc::new(Cell::new(false)) };
        let held = self.held.borrow().iter().find(|prefix| key.starts_with(prefix.as_str())).cloned();
        if let Some(prefix) = held {
            self.streams.borrow_mut().entry(prefix).or_default().borrow_mut().push(process.clone());
        } else {
            let response = self.response(&key).unwrap_or_else(|| output(0, "", ""));
            process.stdout(&response.stdout);
            process.stderr(&response.stderr);
            process.exit(response.exit_code);
        }
        let killer = process.clone();
        Ok(RunningProcess::new(receiver, move || {
            killer.killed.set(true);
            killer.exit(143);
        }))
    }
}

/// A runner that answers the read-only commands like the fixtures: maven
/// 3.9.16 (current) and 3.8.9, mvn2 without versions and nodejs 24.11.1
/// (current) and 22.21.0.
pub fn fixture_runner() -> FakeProcessRunner {
    let runner = FakeProcessRunner::new();
    runner
        .on("--version", fixture_output("version", 0))
        .on("list", fixture_output("list", 0))
        .on("current", fixture_output("current", 0))
        .on("available", fixture_output("available", 0))
        .on("config --list", fixture_output("config-list", 0))
        .on("search maven", fixture_output("search-maven-installed", 0))
        .on("search nodejs", fixture_output("search-nodejs", 0))
        .on(&format!("info nodejs --format {PLUGIN_INFO_TEMPLATE}"), fixture_output("info", 0));
    runner
}

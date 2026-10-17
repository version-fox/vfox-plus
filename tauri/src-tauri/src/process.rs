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

//! Spawns vfox as `docs/vfox-cli-contract.md` asks: directly, with an argument
//! vector, a complete environment, a working directory and a closed standard
//! input.

use std::collections::HashMap;
use std::io::{self, Read};
use std::os::unix::process::ExitStatusExt;
use std::process::{Child, Command, ExitStatus, Stdio};
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{Arc, Mutex, PoisonError};
use std::thread;

use base64::Engine;
use nix::sys::signal::{Signal, kill};
use nix::unistd::Pid;
use serde::Serialize;

/// How to start a process.
pub struct Launch {
    /// The executable, a path or a name looked up in the `PATH` of `environment`.
    pub executable: String,
    /// The arguments.
    pub arguments: Vec<String>,
    /// The working directory.
    pub working_directory: String,
    /// The complete environment: nothing is inherited.
    pub environment: HashMap<String, String>,
}

impl Launch {
    fn command(&self) -> Command {
        let mut command = Command::new(&self.executable);
        command
            .args(&self.arguments)
            .current_dir(&self.working_directory)
            .env_clear()
            .envs(&self.environment)
            // vfox detects that it does not run in a terminal and never
            // prompts; an open stdin would make an unexpected prompt hang.
            .stdin(Stdio::null());
        command
    }
}

/// The result of a finished process, decoded as UTF-8 with malformed bytes
/// replaced.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct RunResult {
    exit_code: i32,
    stdout: String,
    stderr: String,
}

/// Output or the exit of a started process, as the front end expects it.
#[derive(Clone, Serialize)]
#[serde(tag = "event", rename_all = "lowercase")]
pub enum ProcessEvent {
    /// A chunk of output, base64-encoded.
    Output {
        /// The identifier returned by [`Processes::start`].
        process: u32,
        /// `stdout` or `stderr`.
        stream: &'static str,
        /// The bytes, base64-encoded.
        data: String,
    },
    /// The process ended; always the last event of a process.
    Exit {
        /// The identifier returned by [`Processes::start`].
        process: u32,
        /// The exit code, or 128 plus the number of the signal that ended it.
        code: i32,
    },
}

/// The exit code of `status`, or 128 plus the signal, as a shell reports it.
fn exit_code(status: ExitStatus) -> i32 {
    status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0))
}

/// Runs a process to completion. Fails if it cannot be started.
pub fn run(launch: &Launch) -> io::Result<RunResult> {
    let output = launch.command().output()?;
    Ok(RunResult {
        exit_code: exit_code(output.status),
        stdout: String::from_utf8_lossy(&output.stdout).into_owned(),
        stderr: String::from_utf8_lossy(&output.stderr).into_owned(),
    })
}

/// The processes started by the front end that are still running.
#[derive(Default)]
pub struct Processes {
    next: AtomicU32,
    running: Arc<Mutex<HashMap<u32, Pid>>>,
}

impl Processes {
    /// Starts a process and forwards its output and its exit to `send`, in
    /// order, from background threads. Returns the identifier of the process.
    pub fn start(&self, launch: &Launch, send: impl Fn(ProcessEvent) + Clone + Send + 'static) -> io::Result<u32> {
        let mut child = launch.command().stdout(Stdio::piped()).stderr(Stdio::piped()).spawn()?;
        let id = self.next.fetch_add(1, Ordering::Relaxed) + 1;
        let pid = Pid::from_raw(i32::try_from(child.id()).unwrap_or(i32::MAX));
        self.running.lock().unwrap_or_else(PoisonError::into_inner).insert(id, pid);
        let running = Arc::clone(&self.running);
        let readers = [
            forward(child.stdout.take(), id, "stdout", send.clone()),
            forward(child.stderr.take(), id, "stderr", send.clone()),
        ];
        thread::spawn(move || {
            for reader in readers.into_iter().flatten() {
                let _ = reader.join();
            }
            let code = wait(&mut child);
            running.lock().unwrap_or_else(PoisonError::into_inner).remove(&id);
            send(ProcessEvent::Exit { process: id, code });
        });
        Ok(id)
    }

    /// Asks the process `id` to terminate with `SIGTERM`, as the CLI contract
    /// asks for cancelling vfox.
    pub fn kill(&self, id: u32) {
        let running = self.running.lock().unwrap_or_else(PoisonError::into_inner);
        if let Some(pid) = running.get(&id) {
            let _ = kill(*pid, Signal::SIGTERM);
        }
    }
}

fn wait(child: &mut Child) -> i32 {
    child.wait().map_or(-1, exit_code)
}

/// Forwards everything `stream` produces as output events.
fn forward(
    stream: Option<impl Read + Send + 'static>,
    process: u32,
    name: &'static str,
    send: impl Fn(ProcessEvent) + Send + 'static,
) -> Option<thread::JoinHandle<()>> {
    let mut stream = stream?;
    Some(thread::spawn(move || {
        let mut buffer = [0_u8; 16 * 1024];
        loop {
            match stream.read(&mut buffer) {
                Ok(0) => break,
                Ok(read) => send(ProcessEvent::Output {
                    process,
                    stream: name,
                    data: base64::engine::general_purpose::STANDARD.encode(&buffer[..read]),
                }),
                Err(error) if error.kind() == io::ErrorKind::Interrupted => {}
                Err(_) => break,
            }
        }
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::mpsc;

    fn launch(script: &str) -> Launch {
        Launch {
            executable: "/bin/sh".into(),
            arguments: vec!["-c".into(), script.into()],
            working_directory: "/".into(),
            environment: HashMap::from([("ONLY".into(), "this".into())]),
        }
    }

    #[test]
    fn runs_with_exactly_the_given_environment_and_directory() {
        let result = run(&launch("echo \"$ONLY:$HOME:$(pwd)\"; echo err >&2; exit 3")).unwrap();
        assert_eq!(result.exit_code, 3);
        assert_eq!(result.stdout, "this::/\n");
        assert_eq!(result.stderr, "err\n");
    }

    #[test]
    fn closes_standard_input() {
        let result = run(&launch("cat; echo done")).unwrap();
        assert_eq!(result.stdout, "done\n");
    }

    #[test]
    fn reports_a_missing_executable() {
        let mut missing = launch("");
        missing.executable = "/nonexistent/vfox".into();
        assert!(run(&missing).is_err());
        assert!(Processes::default().start(&missing, |_| {}).is_err());
    }

    #[test]
    fn replaces_malformed_output() {
        let result = run(&launch("printf 'a\\377b'")).unwrap();
        assert_eq!(result.stdout, "a\u{FFFD}b");
    }

    #[test]
    fn streams_output_and_ends_with_the_exit() {
        let (sender, receiver) = mpsc::channel();
        let processes = Processes::default();
        let id = processes
            .start(&launch("echo out; echo err >&2; exit 4"), move |event| {
                sender.send(event).unwrap();
            })
            .unwrap();
        let events: Vec<ProcessEvent> = receiver.iter().collect();
        let mut output = HashMap::new();
        for event in &events[..events.len() - 1] {
            if let ProcessEvent::Output { process, stream, data } = event {
                assert_eq!(*process, id);
                output.insert(*stream, data.clone());
            }
        }
        assert_eq!(output["stdout"], "b3V0Cg==");
        assert_eq!(output["stderr"], "ZXJyCg==");
        assert!(matches!(events.last(), Some(ProcessEvent::Exit { code: 4, .. })));
    }

    #[test]
    fn kills_with_sigterm() {
        let (sender, receiver) = mpsc::channel();
        let processes = Processes::default();
        let id = processes
            .start(&launch("trap 'exit 42' TERM; echo ready; while :; do sleep 0.05; done"), move |event| {
                sender.send(event).unwrap();
            })
            .unwrap();
        // Wait until the trap is installed.
        assert!(matches!(receiver.recv().unwrap(), ProcessEvent::Output { .. }));
        processes.kill(id);
        let last = receiver.iter().last().unwrap();
        assert!(matches!(last, ProcessEvent::Exit { code: 42, .. }));
    }
}

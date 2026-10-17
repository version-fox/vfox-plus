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

//! Child processes, as `docs/vfox-cli-contract.md` asks: started directly,
//! with an argument vector, a complete environment, a working directory and a
//! closed standard input.

use std::collections::BTreeMap;
use std::future::Future;
use std::os::unix::process::ExitStatusExt;
use std::path::Path;
use std::pin::Pin;
use std::process::{ExitStatus, Stdio};
use std::rc::Rc;

use tokio::io::{AsyncRead, AsyncReadExt};
use tokio::sync::mpsc::{UnboundedReceiver, UnboundedSender, unbounded_channel};

/// A future that the process runner returns.
pub type BoxFuture<T> = Pin<Box<dyn Future<Output = T>>>;

/// Everything a finished process wrote, decoded as UTF-8, and how it exited.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProcessOutput {
    /// The exit code, or 128 plus the number of the signal that ended it.
    pub exit_code: i32,
    /// Standard output, with malformed bytes replaced.
    pub stdout: String,
    /// Standard error, with malformed bytes replaced.
    pub stderr: String,
}

/// Where and how a child process runs.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProcessOptions {
    /// The directory the process runs in.
    pub working_directory: String,
    /// The complete environment of the child: the environment of the
    /// application is not inherited, so that callers can remove variables as
    /// well as add them.
    pub environment: BTreeMap<String, String>,
}

/// The output or the exit of a started process.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ProcessEvent {
    /// A chunk of raw standard output.
    Stdout(Vec<u8>),
    /// A chunk of raw standard error.
    Stderr(Vec<u8>),
    /// The exit code, always the last event, after all of the output.
    Exit(i32),
}

/// Terminates a started process.
pub type Killer = Rc<dyn Fn()>;

/// A child process whose output is consumed while it runs.
pub struct RunningProcess {
    events: UnboundedReceiver<ProcessEvent>,
    killer: Killer,
}

impl RunningProcess {
    /// A process whose events arrive on `events` and which `killer` terminates.
    pub fn new(events: UnboundedReceiver<ProcessEvent>, killer: impl Fn() + 'static) -> Self {
        Self { events, killer: Rc::new(killer) }
    }

    /// What terminates the process, to keep while its output is read.
    pub fn killer(&self) -> Killer {
        self.killer.clone()
    }

    /// The next output or the exit; `None` once the exit was delivered.
    pub async fn next(&mut self) -> Option<ProcessEvent> {
        self.events.recv().await
    }

    /// Asks the process to terminate with `SIGTERM`.
    pub fn kill(&self) {
        (self.killer)();
    }
}

/// An executable could not be started, for example because it is missing.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProcessStartError {
    /// The path or name that could not be executed.
    pub executable: String,
    /// Why it could not be started.
    pub message: String,
}

/// Spawns child processes. The application never spawns processes directly,
/// which keeps the vfox client testable with a fake runner. Standard input is
/// closed immediately: an open stdin would make an unexpected prompt wait
/// forever.
pub trait ProcessRunner {
    /// Runs `executable` to completion.
    fn run(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> BoxFuture<Result<ProcessOutput, ProcessStartError>>;

    /// Starts `executable` and streams its output.
    fn start(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> Result<RunningProcess, ProcessStartError>;
}

/// Whether `path` is an existing regular file that may be executed.
pub fn is_executable_file(path: &str) -> bool {
    use std::os::unix::fs::PermissionsExt;
    std::fs::metadata(path).is_ok_and(|metadata| metadata.is_file() && metadata.permissions().mode() & 0o111 != 0)
}

/// Finds `name` in the directories of `path`, the PATH of the child, as a
/// shell would; names that contain a slash are returned as they are.
fn look_path(name: &str, path: &str) -> Result<String, ProcessStartError> {
    if name.contains('/') {
        return Ok(name.to_owned());
    }
    path.split(':')
        .filter(|directory| !directory.is_empty())
        .map(|directory| Path::new(directory).join(name).to_string_lossy().into_owned())
        .find(|candidate| is_executable_file(candidate))
        .ok_or_else(|| ProcessStartError {
            executable: name.to_owned(),
            message: format!("{name}: executable file not found in PATH"),
        })
}

/// The exit code of a finished process, or 128 plus the number of the signal
/// that ended it, as a shell reports it.
fn exit_code(status: ExitStatus) -> i32 {
    status.code().unwrap_or_else(|| status.signal().map_or(-1, |signal| 128 + signal))
}

/// Runs processes with tokio.
#[derive(Clone, Copy, Debug, Default)]
pub struct TokioProcessRunner;

impl TokioProcessRunner {
    fn command(
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> Result<tokio::process::Command, ProcessStartError> {
        let path = look_path(executable, options.environment.get("PATH").map_or("", String::as_str))?;
        let mut command = tokio::process::Command::new(path);
        command
            .args(arguments)
            .current_dir(&options.working_directory)
            .env_clear()
            .envs(&options.environment)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        Ok(command)
    }
}

fn start_error(executable: &str, error: &std::io::Error) -> ProcessStartError {
    ProcessStartError { executable: executable.to_owned(), message: error.to_string() }
}

/// Sends what `stream` writes as events made by `event`, until it ends.
async fn forward(
    mut stream: impl AsyncRead + Unpin,
    sender: UnboundedSender<ProcessEvent>,
    event: fn(Vec<u8>) -> ProcessEvent,
) {
    let mut buffer = vec![0; 16 * 1024];
    loop {
        match stream.read(&mut buffer).await {
            Ok(0) | Err(_) => break,
            Ok(read) => {
                if sender.send(event(buffer[..read].to_vec())).is_err() {
                    break;
                }
            }
        }
    }
}

impl ProcessRunner for TokioProcessRunner {
    fn run(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> BoxFuture<Result<ProcessOutput, ProcessStartError>> {
        let executable = executable.to_owned();
        let command = Self::command(&executable, arguments, options);
        Box::pin(async move {
            let output = command?.output().await.map_err(|error| start_error(&executable, &error))?;
            Ok(ProcessOutput {
                exit_code: exit_code(output.status),
                stdout: String::from_utf8_lossy(&output.stdout).into_owned(),
                stderr: String::from_utf8_lossy(&output.stderr).into_owned(),
            })
        })
    }

    fn start(
        &self,
        executable: &str,
        arguments: &[String],
        options: &ProcessOptions,
    ) -> Result<RunningProcess, ProcessStartError> {
        let mut child =
            Self::command(executable, arguments, options)?.spawn().map_err(|error| start_error(executable, &error))?;
        let (sender, receiver) = unbounded_channel();
        let stdout =
            child.stdout.take().map(|stream| tokio::spawn(forward(stream, sender.clone(), ProcessEvent::Stdout)));
        let stderr =
            child.stderr.take().map(|stream| tokio::spawn(forward(stream, sender.clone(), ProcessEvent::Stderr)));
        let pid = child.id().and_then(|id| i32::try_from(id).ok());
        tokio::spawn(async move {
            // The exit is only reported once both streams are drained.
            for reader in [stdout, stderr].into_iter().flatten() {
                let _ = reader.await;
            }
            let code = child.wait().await.map_or(-1, exit_code);
            let _ = sender.send(ProcessEvent::Exit(code));
        });
        Ok(RunningProcess::new(receiver, move || {
            if let Some(pid) = pid {
                // A process that has exited already cannot be signalled, which is fine.
                let _ = nix::sys::signal::kill(nix::unistd::Pid::from_raw(pid), nix::sys::signal::Signal::SIGTERM);
            }
        }))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn options(path: &str) -> ProcessOptions {
        ProcessOptions {
            working_directory: "/".into(),
            environment: BTreeMap::from([("ONLY".into(), "this".into()), ("PATH".into(), path.into())]),
        }
    }

    fn sh(script: &str) -> Vec<String> {
        vec!["-c".into(), script.into()]
    }

    async fn collect(mut process: RunningProcess) -> (String, String, i32) {
        let (mut out, mut err) = (Vec::new(), Vec::new());
        loop {
            match process.next().await {
                Some(ProcessEvent::Stdout(chunk)) => out.extend(chunk),
                Some(ProcessEvent::Stderr(chunk)) => err.extend(chunk),
                Some(ProcessEvent::Exit(code)) => {
                    assert_eq!(process.next().await, None, "the exit is the last event");
                    return (String::from_utf8(out).unwrap(), String::from_utf8(err).unwrap(), code);
                }
                None => panic!("the process ended without its exit"),
            }
        }
    }

    #[tokio::test]
    async fn runs_with_exactly_the_given_environment_and_directory() {
        let output = TokioProcessRunner
            .run("sh", &sh("echo \"$ONLY:$HOME:$(pwd)\"; echo err >&2; exit 3"), &options("/usr/bin:/bin"))
            .await
            .unwrap();
        assert_eq!(output, ProcessOutput { exit_code: 3, stdout: "this::/\n".into(), stderr: "err\n".into() });
    }

    #[tokio::test]
    async fn closes_standard_input_and_replaces_malformed_output() {
        let runner = TokioProcessRunner;
        assert_eq!(runner.run("sh", &sh("cat; echo done"), &options("/bin")).await.unwrap().stdout, "done\n");
        assert_eq!(runner.run("sh", &sh("printf 'a\\377b'"), &options("/bin")).await.unwrap().stdout, "a\u{fffd}b");
    }

    #[tokio::test]
    async fn looks_executables_up_in_the_path_of_the_child() {
        let runner = TokioProcessRunner;
        assert!(runner.run("sh", &sh("true"), &options("/nonexistent")).await.is_err());
        assert!(runner.run("/nonexistent/vfox", &[], &options("/bin")).await.is_err());
        assert!(runner.start("/nonexistent/vfox", &[], &options("/bin")).is_err());
        let mut bad_directory = options("/bin");
        bad_directory.working_directory = "/nonexistent".into();
        assert!(runner.start("sh", &sh("true"), &bad_directory).is_err());
    }

    #[tokio::test]
    async fn streams_the_output_and_exits_after_it() {
        let process = TokioProcessRunner.start("sh", &sh("echo out; echo err >&2; exit 4"), &options("/bin")).unwrap();
        assert_eq!(collect(process).await, ("out\n".into(), "err\n".into(), 4));
    }

    #[tokio::test]
    async fn kill_sends_sigterm_and_signals_become_128_plus_the_signal() {
        let directory = tempfile::tempdir().unwrap();
        let ready = directory.path().join("ready");
        let script = format!("trap 'exit 42' TERM; touch '{}'; while :; do sleep 0.05; done", ready.display());
        let process = TokioProcessRunner.start("sh", &sh(&script), &options("/bin:/usr/bin")).unwrap();
        while !ready.exists() {
            tokio::time::sleep(std::time::Duration::from_millis(10)).await;
        }
        process.kill();
        assert_eq!(collect(process).await.2, 42);
        let killed = TokioProcessRunner.start("sh", &sh("kill -KILL $$"), &options("/bin")).unwrap();
        assert_eq!(collect(killed).await.2, 128 + 9);
        assert!(is_executable_file("/bin/sh"));
        assert!(!is_executable_file("/"));
    }
}

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

//! The host protocol of the web front end (`web/src/hosts/protocol.ts`) as
//! two Tauri commands: `host_call` for every request, and `host_start`, which
//! also takes the channel that carries the output of the process.

use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use tauri::ipc::Channel;
use tauri::{AppHandle, State};

use crate::process::{Launch, ProcessEvent, Processes};
use crate::{desktop, files};

/// A request of the front end.
#[derive(Deserialize)]
pub struct HostRequest {
    id: u64,
    method: String,
    params: Vec<Value>,
}

/// Why a request failed: `start` when an executable could not be started.
#[derive(Serialize)]
pub struct HostErrorBody {
    kind: &'static str,
    message: String,
}

/// The answer to a request.
#[derive(Serialize)]
#[serde(untagged)]
pub enum HostResponse {
    /// The request succeeded.
    Success {
        /// The identifier of the request.
        id: u64,
        /// What the method returned.
        result: Value,
    },
    /// The request failed.
    Failure {
        /// The identifier of the request.
        id: u64,
        /// What went wrong.
        error: HostErrorBody,
    },
}

/// A failed request.
enum Failure {
    /// The executable could not be started.
    Start(String),
    /// Anything else.
    Other(String),
}

impl<E: std::fmt::Display> From<E> for Failure {
    fn from(error: E) -> Self {
        Self::Other(error.to_string())
    }
}

fn respond(id: u64, result: Result<Value, Failure>) -> HostResponse {
    match result {
        Ok(result) => HostResponse::Success { id, result },
        Err(Failure::Start(message)) => HostResponse::Failure { id, error: HostErrorBody { kind: "start", message } },
        Err(Failure::Other(message)) => HostResponse::Failure { id, error: HostErrorBody { kind: "error", message } },
    }
}

/// The parameters of a request, read by position.
struct Params(Vec<Value>);

impl Params {
    fn get(&self, index: usize) -> &Value {
        self.0.get(index).unwrap_or(&Value::Null)
    }

    fn string(&self, index: usize) -> Result<String, Failure> {
        self.get(index)
            .as_str()
            .map(str::to_owned)
            .ok_or_else(|| Failure::Other(format!("parameter {index} must be a string")))
    }

    fn optional_string(&self, index: usize) -> Option<String> {
        self.get(index).as_str().map(str::to_owned)
    }

    fn strings(&self, index: usize) -> Option<Vec<String>> {
        serde_json::from_value(self.get(index).clone()).ok()
    }

    fn launch(&self) -> Result<Launch, Failure> {
        Ok(Launch {
            executable: self.string(0)?,
            arguments: self.strings(1).unwrap_or_default(),
            working_directory: self.string(2)?,
            environment: serde_json::from_value::<HashMap<String, String>>(self.get(3).clone())?,
        })
    }
}

/// Runs `work` on a thread that may block.
async fn blocking<T: Send + 'static>(work: impl FnOnce() -> T + Send + 'static) -> Result<T, Failure> {
    Ok(tauri::async_runtime::spawn_blocking(work).await?)
}

async fn dispatch(app: AppHandle, processes: &Processes, method: &str, params: Params) -> Result<Value, Failure> {
    Ok(match method {
        "environment" => json!(std::env::vars().collect::<HashMap<_, _>>()),
        "run" => {
            let launch = params.launch()?;
            let result = blocking(move || crate::process::run(&launch))
                .await?
                .map_err(|error| Failure::Start(error.to_string()))?;
            serde_json::to_value(result)?
        }
        "kill" => {
            if let Some(id) = params.get(0).as_u64().and_then(|id| u32::try_from(id).ok()) {
                processes.kill(id);
            }
            Value::Null
        }
        "exists" => json!(files::exists(&params.string(0)?)),
        "isDirectory" => json!(files::is_directory(&params.string(0)?)),
        "isExecutable" => json!(files::is_executable(&params.string(0)?)),
        "readTextFile" => json!(files::read_text_file(&params.string(0)?)?),
        "writeTextFile" => {
            files::write_text_file(&params.string(0)?, &params.string(1)?)?;
            Value::Null
        }
        "createDirectory" => {
            files::create_directory(&params.string(0)?)?;
            Value::Null
        }
        "createTemporaryDirectory" => json!(files::create_temporary_directory(&params.string(0)?, &params.string(1)?)?),
        "copyFile" => {
            files::copy_file(&params.string(0)?, &params.string(1)?)?;
            Value::Null
        }
        "remove" => {
            files::remove(&params.string(0)?)?;
            Value::Null
        }
        "systemTemporaryDirectory" => json!(files::system_temporary_directory()),
        "accentColor" => json!(blocking(desktop::accent_color).await?),
        "openUrl" => {
            desktop::open_url(&app, &params.string(0)?)?;
            Value::Null
        }
        "pickDirectory" => {
            let initial = params.optional_string(0);
            json!(blocking(move || desktop::pick_directory(&app, initial)).await?)
        }
        "pickFile" => {
            let (initial, label, extensions) =
                (params.optional_string(0), params.optional_string(1), params.strings(2));
            json!(blocking(move || desktop::pick_file(&app, initial, label, extensions)).await?)
        }
        "copyText" => {
            desktop::copy_text(&app, params.string(0)?)?;
            Value::Null
        }
        "exit" => {
            let code = params.get(0).as_i64().and_then(|code| i32::try_from(code).ok()).unwrap_or(1);
            app.exit(code);
            Value::Null
        }
        "log" => {
            println!("{}", params.string(0)?);
            Value::Null
        }
        _ => return Err(Failure::Other(format!("unknown method {method}"))),
    })
}

/// Answers a request of the host protocol other than `start`.
#[tauri::command]
pub async fn host_call(
    app: AppHandle,
    processes: State<'_, Processes>,
    request: HostRequest,
) -> Result<HostResponse, ()> {
    let id = request.id;
    Ok(respond(id, dispatch(app, &processes, &request.method, Params(request.params)).await))
}

/// Answers a `start` request: starts the process and sends its output and its
/// exit through `events`.
#[tauri::command]
pub fn host_start(
    processes: State<'_, Processes>,
    request: HostRequest,
    events: Channel<ProcessEvent>,
) -> HostResponse {
    let params = Params(request.params);
    let result = params.launch().and_then(|launch| {
        processes
            .start(&launch, move |event| {
                // The front end may be gone when the application quits.
                let _ = events.send(event);
            })
            .map(|id| json!(id))
            .map_err(|error| Failure::Start(error.to_string()))
    });
    respond(request.id, result)
}

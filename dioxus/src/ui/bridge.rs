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

//! What the user interface asks of the page, through the helpers of
//! `window.vfoxPlus` (`components/index.js`): snack bars, the end of dialogs,
//! the theme of the document, media queries, the visible part of long lists
//! and values that Dioxus does not read from custom elements.

use dioxus::prelude::*;
use serde::Serialize;
use serde::de::DeserializeOwned;

/// `value` as a JavaScript literal.
fn js(value: &impl Serialize) -> String {
    serde_json::to_string(value).unwrap_or_else(|_| "null".to_owned())
}

/// Runs the body of an async JavaScript function, `script`, and returns what it
/// returns, or the message of what it throws.
///
/// The value travels through the channel of the evaluation, and the script
/// waits for Rust to acknowledge it. dioxus-desktop frees an evaluation once
/// the page has collected its channel, which can happen before Rust has read
/// the return value; a channel that the script still uses is not collected.
pub async fn query<T: DeserializeOwned>(script: &str) -> Result<T, String> {
    let mut eval = document::eval(&format!(
        "let reply;
         try {{
           reply = {{ value: await (async () => {{ {script} }})() ?? null }};
         }} catch (error) {{
           reply = {{ error: String(error?.message ?? error) }};
         }}
         dioxus.send(reply);
         await dioxus.recv();"
    ));
    let reply = eval.recv::<serde_json::Value>().await.map_err(|error| format!("{error:?}"));
    let _ = eval.send(true);
    let mut reply = reply?;
    if let Some(error) = reply.get("error").and_then(serde_json::Value::as_str) {
        return Err(error.to_owned());
    }
    serde_json::from_value(reply["value"].take()).map_err(|error| error.to_string())
}

/// Calls `window.vfoxPlus.<function>` with `arguments` and returns its result.
async fn call<T: DeserializeOwned>(function: &str, arguments: &[String]) -> Option<T> {
    query(&format!("return await window.vfoxPlus.{function}({});", arguments.join(", "))).await.ok()
}

/// Shows `message` at the bottom of the window, replacing the current snack
/// bar; a persistent one stays until it is dismissed. Resolves with whether
/// the user chose `action`.
pub async fn snackbar(message: &str, action: Option<&str>, persistent: bool, close_label: &str) -> bool {
    let options = serde_json::json!({ "action": action, "persistent": persistent, "closeLabel": close_label });
    call("snackbar", &[js(&message), js(&options)]).await.unwrap_or(false)
}

/// Shows a snack bar without an action and without waiting for it.
pub fn notify(message: String, close_label: String) {
    spawn(async move {
        snackbar(&message, None, false, &close_label).await;
    });
}

/// Waits until the dialog with the identifier `id` has closed.
pub async fn dialog_closed(id: &str) {
    let _: Option<bool> = call("closed", &[js(&id)]).await;
}

/// Writes the colour and motion tokens to the root element of the document,
/// with its colour scheme, language and title.
pub fn apply_theme(tokens: &[(String, String)], dark: bool, language: &str, title: &str) {
    let properties: serde_json::Map<String, serde_json::Value> =
        tokens.iter().map(|(name, value)| (name.clone(), value.clone().into())).collect();
    document::eval(&format!(
        "window.vfoxPlus.applyTheme({}, {dark}, {}, {});",
        js(&properties),
        js(&language),
        js(&title)
    ));
}

/// Calls `on_change` with whether the media `query` matches, now and
/// whenever that changes, for as long as the task that awaits it runs.
pub async fn watch_media(query: &str, mut on_change: impl FnMut(bool)) {
    let mut eval = document::eval(
        "const query = await dioxus.recv();
         await new Promise(() => window.vfoxPlus.watchMedia(query, (matches) => dioxus.send(matches)));",
    );
    if eval.send(query).is_err() {
        return;
    }
    while let Ok(matches) = eval.recv::<bool>().await {
        on_change(matches);
    }
}

/// Calls `on_change` with the range of the rows of the list with the
/// identifier `id` that are in view, or nearly: the rows are `pitch` pixels
/// apart, and `overscan` rows on either side count as in view.
pub async fn watch_visible_rows(id: &str, pitch: f64, overscan: usize, mut on_change: impl FnMut(usize, usize)) {
    let mut eval = document::eval(
        "const [id, pitch, overscan] = await dioxus.recv();
         await new Promise(() => window.vfoxPlus.visibleRows(id, pitch, overscan, (range) => dioxus.send(range)));",
    );
    if eval.send((id, pitch, overscan)).is_err() {
        return;
    }
    while let Ok((start, end)) = eval.recv::<(usize, usize)>().await {
        on_change(start, end);
    }
}

/// Scrolls the element with the identifier `id` to its end.
pub fn scroll_to_end(id: &str) {
    document::eval(&format!("window.vfoxPlus.scrollToEnd({});", js(&id)));
}

/// The `value` of the element with the identifier `id`, such as an
/// `m3e-select`, whose events do not carry it.
pub async fn value_of(id: &str) -> String {
    call("valueOf", &[js(&id)]).await.unwrap_or_default()
}

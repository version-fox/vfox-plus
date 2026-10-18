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

//! Desktop integration: the accent colour, file choosers, links and the
//! clipboard. The functions block and run on worker threads.

use tauri::AppHandle;
use tauri_plugin_clipboard_manager::ClipboardExt;
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_opener::OpenerExt;
use zbus::zvariant::Value;

/// The accent colour of the desktop as ARGB, from the XDG desktop portal
/// (`org.freedesktop.appearance` `accent-color`), if the desktop sets one.
pub fn accent_color() -> Option<u32> {
    let connection = zbus::blocking::Connection::session().ok()?;
    let reply = connection
        .call_method(
            Some("org.freedesktop.portal.Desktop"),
            "/org/freedesktop/portal/desktop",
            Some("org.freedesktop.portal.Settings"),
            "ReadOne",
            &("org.freedesktop.appearance", "accent-color"),
        )
        .ok()?;
    let body = reply.body();
    let value: Value<'_> = body.deserialize().ok()?;
    argb(&value)
}

/// The colour of a `(ddd)` structure, possibly wrapped in variants. The portal
/// reports components outside of 0–1 when no accent colour is set.
fn argb(value: &Value<'_>) -> Option<u32> {
    match value {
        Value::Value(inner) => argb(inner),
        Value::Structure(structure) => {
            let mut channels = [0_u32; 3];
            if structure.fields().len() != 3 {
                return None;
            }
            for (channel, field) in channels.iter_mut().zip(structure.fields()) {
                let Value::F64(component) = field else {
                    return None;
                };
                if !(0.0..=1.0).contains(component) {
                    return None;
                }
                // In range, so the conversion is exact enough and cannot overflow.
                #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
                {
                    *channel = (component * 255.0).round() as u32;
                }
            }
            Some(0xFF00_0000 | channels[0] << 16 | channels[1] << 8 | channels[2])
        }
        _ => None,
    }
}

/// Lets the user choose a directory.
pub fn pick_directory(app: &AppHandle, initial: Option<String>) -> Option<String> {
    let mut chooser = app.dialog().file();
    if let Some(directory) = initial {
        chooser = chooser.set_directory(directory);
    }
    chooser
        .blocking_pick_folder()
        .and_then(|path| path.into_path().ok())
        .map(|path| path.to_string_lossy().into_owned())
}

/// Lets the user choose a file, optionally with one of `extensions`.
pub fn pick_file(
    app: &AppHandle,
    initial: Option<String>,
    label: Option<String>,
    extensions: Option<Vec<String>>,
) -> Option<String> {
    let mut chooser = app.dialog().file();
    if let Some(directory) = initial {
        chooser = chooser.set_directory(directory);
    }
    if let Some(extensions) = extensions {
        let extensions: Vec<&str> = extensions.iter().map(String::as_str).collect();
        chooser = chooser.add_filter(label.unwrap_or_default(), &extensions);
    }
    chooser.blocking_pick_file().and_then(|path| path.into_path().ok()).map(|path| path.to_string_lossy().into_owned())
}

/// Opens `url` in the default browser.
pub fn open_url(app: &AppHandle, url: &str) -> Result<(), String> {
    app.opener().open_url(url, None::<&str>).map_err(|error| error.to_string())
}

/// Copies `text` to the clipboard.
pub fn copy_text(app: &AppHandle, text: String) -> Result<(), String> {
    app.clipboard().write_text(text).map_err(|error| error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use zbus::zvariant::StructureBuilder;

    fn rgb(r: f64, g: f64, b: f64) -> Value<'static> {
        Value::Structure(StructureBuilder::new().add_field(r).add_field(g).add_field(b).build().unwrap())
    }

    #[test]
    fn converts_portal_colours_to_argb() {
        assert_eq!(argb(&rgb(1.0, 0.5, 0.0)), Some(0xFFFF_8000));
        assert_eq!(argb(&Value::Value(Box::new(rgb(0.0, 0.0, 1.0)))), Some(0xFF00_00FF));
    }

    #[test]
    fn rejects_unset_colours() {
        assert_eq!(argb(&rgb(-1.0, -1.0, -1.0)), None);
        assert_eq!(argb(&Value::from("blue")), None);
    }
}

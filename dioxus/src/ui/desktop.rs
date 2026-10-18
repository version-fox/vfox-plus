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

//! Desktop integration: the appearance of the desktop from the XDG desktop
//! portal, links, the clipboard and the file choosers of GTK, which use the
//! portal where there is one. The user interface runs on the main thread of
//! GTK, so GTK is called directly.

use std::cell::RefCell;
use std::path::Path;
use std::process::Stdio;
use std::rc::Rc;
use std::time::Duration;

use gtk::prelude::*;

/// Reads a setting of `org.freedesktop.appearance` from the XDG desktop
/// portal with `gdbus`, which every `GLib` installation has.
async fn appearance(key: &str) -> Option<String> {
    let command = tokio::process::Command::new("gdbus")
        .args([
            "call",
            "--session",
            "--timeout",
            "2",
            "--dest",
            "org.freedesktop.portal.Desktop",
            "--object-path",
            "/org/freedesktop/portal/desktop",
            "--method",
            "org.freedesktop.portal.Settings.ReadOne",
            "org.freedesktop.appearance",
            key,
        ])
        .stdin(Stdio::null())
        .stderr(Stdio::null())
        .kill_on_drop(true)
        .output();
    let output = tokio::time::timeout(Duration::from_secs(5), command).await.ok()?.ok()?;
    output.status.success().then(|| String::from_utf8_lossy(&output.stdout).into_owned())
}

/// The accent colour of the desktop as ARGB, if it sets one.
pub async fn accent_color() -> Option<u32> {
    parse_portal_color(&appearance("accent-color").await?)
}

/// Whether the desktop prefers a dark appearance, or `None` if it has no
/// preference.
pub async fn prefers_dark() -> Option<bool> {
    parse_color_scheme(&appearance("color-scheme").await?)
}

/// The colour in the output of `gdbus` for the portal setting, such as
/// `(<<(0.2, 0.4, 0.8)>>,)`. The portal reports components outside of 0–1
/// when no accent colour is set.
pub fn parse_portal_color(output: &str) -> Option<u32> {
    let start = output.rfind('(')?;
    let end = start + output[start..].find(')')?;
    let components: Vec<f64> =
        output[start + 1..end].split(',').map(|part| part.trim().parse().ok()).collect::<Option<_>>()?;
    let [red, green, blue] = components.as_slice() else { return None };
    let channel = |value: f64| {
        // In range, so the conversion is exact enough and cannot overflow.
        #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
        (0.0..=1.0).contains(&value).then(|| (value * 255.0).round() as u32)
    };
    Some(0xFF00_0000 | channel(*red)? << 16 | channel(*green)? << 8 | channel(*blue)?)
}

/// The colour scheme in the output of `gdbus`, such as `(<<uint32 1>>,)`:
/// 1 prefers dark, 2 prefers light and 0 has no preference.
pub fn parse_color_scheme(output: &str) -> Option<bool> {
    let value = output.split("uint32 ").nth(1)?.chars().next()?;
    match value {
        '1' => Some(true),
        '2' => Some(false),
        _ => None,
    }
}

/// Opens `url` in the default browser.
pub fn open_url(url: &str) {
    // The opener detaches from the application; tokio reaps it.
    let _ = tokio::process::Command::new("xdg-open").arg(url).stdin(Stdio::null()).spawn();
}

/// Copies `text` to the clipboard.
pub fn copy_text(text: &str) {
    gtk::Clipboard::get(&gtk::gdk::SELECTION_CLIPBOARD).set_text(text);
}

/// What a file chooser chooses.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Choose {
    /// An existing file.
    File,
    /// An existing directory.
    Directory,
}

/// Lets the user choose a file or a directory, starting in `initial`; a file
/// may be limited to `extensions`, described by `label`. Resolves with `None`
/// when the user cancels.
pub async fn choose(
    choose: Choose,
    title: &str,
    initial: Option<&str>,
    filter: Option<(&str, &[&str])>,
) -> Option<String> {
    let action = match choose {
        Choose::File => gtk::FileChooserAction::Open,
        Choose::Directory => gtk::FileChooserAction::SelectFolder,
    };
    let parent = dioxus::desktop::window();
    let dialog = gtk::FileChooserNative::new(Some(title), Some(parent.window.gtk_window()), action, None, None);
    dialog.set_modal(true);
    if let Some(directory) = initial.filter(|directory| Path::new(directory).is_dir()) {
        dialog.set_current_folder(directory);
    }
    if let Some((label, extensions)) = filter {
        let files = gtk::FileFilter::new();
        files.set_name(Some(label));
        for extension in extensions {
            files.add_pattern(&format!("*.{extension}"));
        }
        dialog.add_filter(files);
    }
    let (sender, receiver) = tokio::sync::oneshot::channel();
    // The dialog lives until it has answered.
    let pending = Rc::new(RefCell::new(Some((dialog.clone(), sender))));
    dialog.connect_response(move |dialog, response| {
        if let Some((_, sender)) = pending.borrow_mut().take() {
            let path = if response == gtk::ResponseType::Accept { dialog.filename() } else { None };
            let _ = sender.send(path.map(|path| path.to_string_lossy().into_owned()));
        }
    });
    dialog.show();
    receiver.await.ok().flatten()
}

/// Lets `tao` expose the GTK window of the application.
use dioxus::desktop::tao::platform::unix::WindowExtUnix as _;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn portal_colours_and_schemes() {
        assert_eq!(parse_portal_color("(<<(0.2, 0.4, 0.8)>>,)\n"), Some(0xFF33_66CC));
        assert_eq!(parse_portal_color("(<<(-1.0, -1.0, -1.0)>>,)"), None);
        assert_eq!(parse_portal_color("(<<(1, 0, 0.5)>>,)"), Some(0xFFFF_0080));
        assert_eq!(parse_portal_color("Error: no such interface"), None);
        assert_eq!(parse_portal_color("(<<(0.2, 0.4)>>,)"), None);
        assert_eq!(parse_color_scheme("(<<uint32 1>>,)\n"), Some(true));
        assert_eq!(parse_color_scheme("(<<uint32 2>>,)"), Some(false));
        assert_eq!(parse_color_scheme("(<<uint32 0>>,)"), None);
        assert_eq!(parse_color_scheme("Error"), None);
    }
}

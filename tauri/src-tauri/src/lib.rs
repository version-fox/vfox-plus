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

//! The native side of vfox+ on Tauri.
//!
//! The user interface, the vfox client and the task queue are the web front
//! end of `../../web`. This crate answers its host protocol
//! (`web/src/hosts/protocol.ts`): it spawns vfox, reads and writes the few
//! files the application needs and talks to the desktop.

mod desktop;
mod files;
mod host;
mod process;

/// The application ID, also the name of the desktop entry.
pub const APP_ID: &str = "dev.vfox.VfoxPlus";

/// Starts the application and blocks until it exits.
///
/// # Panics
///
/// Panics if the window or the web view cannot be created.
pub fn run() {
    // Before GTK starts: X11 takes WM_CLASS and Wayland the app ID from it.
    #[cfg(target_os = "linux")]
    glib::set_prgname(Some(APP_ID));
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        .manage(process::Processes::default())
        .invoke_handler(tauri::generate_handler![host::host_call, host::host_start])
        .run(tauri::generate_context!())
        .expect("vfox+ could not start");
}

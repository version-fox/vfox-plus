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

//! Sources passed to `vfox add --source`.

use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// Whether `source` is a URL rather than a local path.
pub fn is_remote_plugin_source(source: &str) -> bool {
    source.starts_with("http://") || source.starts_with("https://")
}

/// Whether vfox would move `source` instead of reading it. vfox (still in
/// 1.0.13) renames a local `.lua` plugin into its temporary directory, which
/// deletes the user's file and fails when the file is on another filesystem;
/// local `.zip` archives and URLs are only read. See
/// `docs/vfox-cli-contract.md`.
pub fn vfox_moves_plugin_source(source: &str) -> bool {
    !is_remote_plugin_source(source) && source.to_lowercase().ends_with(".lua")
}

/// The temporary directory of vfox when it runs with `environment`:
/// `~/.version-fox/tmp` if the legacy `~/.version-fox` exists, `~/.vfox/tmp`
/// otherwise. Without `HOME`, where vfox does not run at all, the system
/// temporary directory.
pub fn vfox_temporary_directory(environment: &BTreeMap<String, String>) -> PathBuf {
    match environment.get("HOME").filter(|home| !home.is_empty()) {
        Some(home) => {
            let legacy = Path::new(home).join(".version-fox");
            if legacy.exists() { legacy.join("tmp") } else { Path::new(home).join(".vfox").join("tmp") }
        }
        None => std::env::temp_dir(),
    }
}

/// A plugin source that is safe to hand to vfox. A private copy is deleted
/// when it is dropped.
#[derive(Debug)]
pub struct PreparedPluginSource {
    /// The path or URL to pass to `vfox add --source`.
    pub path: String,
    copy: Option<tempfile::TempDir>,
}

impl PreparedPluginSource {
    /// Whether [`PreparedPluginSource::path`] is a private copy of the original source.
    pub fn is_copy(&self) -> bool {
        self.copy.is_some()
    }
}

/// Prepares `source` for `vfox add --source`. A local `.lua` file is copied
/// into a new directory under `temporary_root`, so that vfox moves the copy
/// and the user's file stays; pass the [`vfox_temporary_directory`], because
/// vfox moves the file with a rename, which only works on the same
/// filesystem. Other sources are returned as they are. If the copy cannot be
/// made, the original path is returned and vfox reports the problem.
pub fn prepare_plugin_source(source: &str, temporary_root: &Path) -> PreparedPluginSource {
    let original = PreparedPluginSource { path: source.to_owned(), copy: None };
    if !vfox_moves_plugin_source(source) {
        return original;
    }
    let copy = || -> std::io::Result<PreparedPluginSource> {
        std::fs::create_dir_all(temporary_root)?;
        let directory = tempfile::Builder::new().prefix("vfox-plus-plugin-").tempdir_in(temporary_root)?;
        let name = Path::new(source).file_name().ok_or_else(|| std::io::Error::other("no file name"))?;
        let target = directory.path().join(name);
        std::fs::copy(source, &target)?;
        Ok(PreparedPluginSource { path: target.to_string_lossy().into_owned(), copy: Some(directory) })
    };
    copy().unwrap_or(original)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn local_lua_files_are_copied_and_other_sources_kept() {
        assert!(vfox_moves_plugin_source("/home/u/plugin.lua"));
        assert!(vfox_moves_plugin_source("/home/u/PLUGIN.LUA"));
        assert!(!vfox_moves_plugin_source("/home/u/plugin.zip"));
        assert!(!vfox_moves_plugin_source("https://example.com/plugin.lua"));
        assert!(is_remote_plugin_source("http://127.0.0.1:8080/p.zip"));

        let home = tempfile::tempdir().unwrap();
        let environment = BTreeMap::from([("HOME".to_owned(), home.path().to_string_lossy().into_owned())]);
        assert_eq!(vfox_temporary_directory(&environment), home.path().join(".vfox/tmp"));
        std::fs::create_dir(home.path().join(".version-fox")).unwrap();
        assert_eq!(vfox_temporary_directory(&environment), home.path().join(".version-fox/tmp"));
        assert_eq!(vfox_temporary_directory(&BTreeMap::new()), std::env::temp_dir());

        let root = home.path().join(".vfox/tmp");
        for source in ["https://example.com/p.lua".to_owned(), format!("{}/p.zip", home.path().display())] {
            let prepared = prepare_plugin_source(&source, &root);
            assert_eq!(prepared.path, source);
            assert!(!prepared.is_copy());
        }

        let original = home.path().join("mine.lua");
        std::fs::write(&original, "PLUGIN = {}").unwrap();
        let prepared = prepare_plugin_source(&original.to_string_lossy(), &root);
        assert!(prepared.is_copy());
        assert!(prepared.path.starts_with(&format!("{}/vfox-plus-plugin-", root.display())));
        assert!(prepared.path.ends_with("/mine.lua"));
        assert_eq!(std::fs::read_to_string(&prepared.path).unwrap(), "PLUGIN = {}");
        let copy = PathBuf::from(&prepared.path);
        drop(prepared);
        assert!(!copy.exists());
        assert!(original.exists());
        assert_eq!(std::fs::read_dir(&root).unwrap().count(), 0);

        let missing = home.path().join("missing.lua").to_string_lossy().into_owned();
        let prepared = prepare_plugin_source(&missing, &home.path().join("tmp"));
        assert_eq!(prepared.path, missing);
        assert!(!prepared.is_copy());
        assert_eq!(std::fs::read_dir(home.path().join("tmp")).unwrap().count(), 0);
    }
}

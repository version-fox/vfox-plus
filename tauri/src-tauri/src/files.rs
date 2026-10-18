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

//! The file system operations of the host protocol: the settings file, the
//! locator of vfox and the copies of local plugins.

use std::fs;
use std::io::{self, Write};
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

/// Whether something exists at `path`.
pub fn exists(path: &str) -> bool {
    Path::new(path).exists()
}

/// Whether `path` is an existing directory.
pub fn is_directory(path: &str) -> bool {
    Path::new(path).is_dir()
}

/// Whether `path` is an existing regular file with any execute bit.
pub fn is_executable(path: &str) -> bool {
    fs::metadata(path).is_ok_and(|metadata| metadata.is_file() && metadata.permissions().mode() & 0o111 != 0)
}

/// The content of a UTF-8 text file, or `None` if it does not exist.
pub fn read_text_file(path: &str) -> io::Result<Option<String>> {
    match fs::read_to_string(path) {
        Ok(text) => Ok(Some(text)),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(error) => Err(error),
    }
}

/// Replaces the content of `path` atomically, creating missing parents.
pub fn write_text_file(path: &str, text: &str) -> io::Result<()> {
    let path = Path::new(path);
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }
    let mut temporary = path.as_os_str().to_owned();
    temporary.push(".tmp");
    let mut file = fs::File::create(&temporary)?;
    file.write_all(text.as_bytes())?;
    file.sync_all()?;
    fs::rename(&temporary, path)
}

/// Creates `path` and its missing parents.
pub fn create_directory(path: &str) -> io::Result<()> {
    fs::create_dir_all(path)
}

/// Creates a new, empty directory in `parent` whose name starts with `prefix`.
pub fn create_temporary_directory(parent: &str, prefix: &str) -> io::Result<String> {
    let directory = tempfile::Builder::new().prefix(prefix).tempdir_in(parent)?.keep();
    Ok(directory.to_string_lossy().into_owned())
}

/// Copies a file.
pub fn copy_file(from: &str, to: &str) -> io::Result<()> {
    fs::copy(from, to).map(|_| ())
}

/// Deletes `path` recursively, if it exists.
pub fn remove(path: &str) -> io::Result<()> {
    let result = match fs::symlink_metadata(path) {
        Ok(metadata) if metadata.is_dir() => fs::remove_dir_all(path),
        Ok(_) => fs::remove_file(path),
        Err(error) => Err(error),
    };
    match result {
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(()),
        other => other,
    }
}

/// The temporary directory of the system.
pub fn system_temporary_directory() -> String {
    std::env::temp_dir().to_string_lossy().into_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn scratch() -> tempfile::TempDir {
        tempfile::tempdir().unwrap()
    }

    #[test]
    fn writes_atomically_and_reads_back() {
        let directory = scratch();
        let path = directory.path().join("nested/settings.json");
        let path = path.to_str().unwrap();
        assert_eq!(read_text_file(path).unwrap(), None);
        write_text_file(path, "{}\n").unwrap();
        assert_eq!(read_text_file(path).unwrap().as_deref(), Some("{}\n"));
        assert!(!exists(&format!("{path}.tmp")));
    }

    #[test]
    fn tells_files_directories_and_executables_apart() {
        let directory = scratch();
        let base = directory.path().to_str().unwrap();
        let script = format!("{base}/vfox");
        fs::write(&script, "#!/bin/sh\n").unwrap();
        assert!(!is_executable(&script));
        fs::set_permissions(&script, fs::Permissions::from_mode(0o755)).unwrap();
        assert!(is_executable(&script));
        assert!(!is_executable(base));
        assert!(is_directory(base));
        assert!(!is_directory(&script));
        assert!(!exists(&format!("{base}/missing")));
    }

    #[test]
    fn copies_into_private_directories_and_removes_them() {
        let directory = scratch();
        let base = directory.path().to_str().unwrap();
        fs::write(format!("{base}/mine.lua"), "PLUGIN = {}").unwrap();
        create_directory(&format!("{base}/.vfox/tmp")).unwrap();
        let private = create_temporary_directory(&format!("{base}/.vfox/tmp"), "vfox-plus-plugin-").unwrap();
        assert!(private.contains("/.vfox/tmp/vfox-plus-plugin-"));
        copy_file(&format!("{base}/mine.lua"), &format!("{private}/mine.lua")).unwrap();
        assert_eq!(fs::read_to_string(format!("{private}/mine.lua")).unwrap(), "PLUGIN = {}");
        remove(&private).unwrap();
        assert!(!exists(&private));
        remove(&private).unwrap();
        remove(&format!("{base}/mine.lua")).unwrap();
        assert!(!exists(&format!("{base}/mine.lua")));
    }
}

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

//! Finds the vfox executable.

use std::collections::BTreeMap;
use std::rc::Rc;

use super::cli::VFOX_EXECUTABLE_NAME;
use super::process::is_executable_file;

/// Looks for the vfox executable in `PATH` and in common install locations.
/// Applications started from a desktop launcher do not source the user's
/// shell profile, so directories such as `~/.local/bin` are often missing from
/// their `PATH`; the install locations of the official installers and of
/// package managers are therefore searched as well.
#[derive(Clone)]
pub struct VfoxLocator {
    environment: BTreeMap<String, String>,
    is_executable: Rc<dyn Fn(&str) -> bool>,
}

impl VfoxLocator {
    /// A locator for `environment` that checks files on disk.
    pub fn new(environment: BTreeMap<String, String>) -> Self {
        Self::with_check(environment, is_executable_file)
    }

    /// A locator for `environment` that asks `is_executable` about candidates.
    pub fn with_check(environment: BTreeMap<String, String>, is_executable: impl Fn(&str) -> bool + 'static) -> Self {
        Self { environment, is_executable: Rc::new(is_executable) }
    }

    /// Directories searched in addition to `PATH`, in order.
    pub fn fallback_directories(&self) -> Vec<String> {
        let mut directories = vec!["/usr/local/bin".to_owned(), "/usr/bin".to_owned()];
        if let Some(home) = self.environment.get("HOME") {
            directories.extend([format!("{home}/.local/bin"), format!("{home}/bin"), format!("{home}/.linuxbrew/bin")]);
        }
        directories.extend(["/home/linuxbrew/.linuxbrew/bin".to_owned(), "/snap/bin".to_owned()]);
        directories
    }

    /// All directories that are searched, without duplicates.
    pub fn search_directories(&self) -> Vec<String> {
        let mut directories: Vec<String> = Vec::new();
        let path = self.environment.get("PATH").map_or("", String::as_str);
        for directory in path
            .split(':')
            .filter(|directory| !directory.is_empty())
            .map(str::to_owned)
            .chain(self.fallback_directories())
        {
            if !directories.contains(&directory) {
                directories.push(directory);
            }
        }
        directories
    }

    /// The path of the first vfox executable found.
    pub fn locate(&self) -> Option<String> {
        self.search_directories()
            .into_iter()
            .map(|directory| format!("{directory}/{VFOX_EXECUTABLE_NAME}"))
            .find(|candidate| (self.is_executable)(candidate))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn searches_path_first_then_the_usual_places() {
        let environment =
            BTreeMap::from([("PATH".into(), "/opt/bin::/usr/bin".into()), ("HOME".into(), "/home/me".into())]);
        assert_eq!(
            VfoxLocator::with_check(environment, |_| false).search_directories(),
            vec![
                "/opt/bin",
                "/usr/bin",
                "/usr/local/bin",
                "/home/me/.local/bin",
                "/home/me/bin",
                "/home/me/.linuxbrew/bin",
                "/home/linuxbrew/.linuxbrew/bin",
                "/snap/bin",
            ]
        );
        let home = BTreeMap::from([("HOME".into(), "/home/me".into())]);
        let locator =
            VfoxLocator::with_check(home, |path| path == "/home/me/.local/bin/vfox" || path == "/snap/bin/vfox");
        assert_eq!(locator.locate().as_deref(), Some("/home/me/.local/bin/vfox"));
        assert_eq!(VfoxLocator::with_check(BTreeMap::new(), |_| false).locate(), None);
    }
}

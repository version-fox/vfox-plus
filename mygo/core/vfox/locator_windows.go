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

//go:build windows

package vfox

import (
	"os"
	"path/filepath"
)

// VfoxExecutableName is the name of the vfox executable, looked up in `PATH`
// when no path is known.
const VfoxExecutableName = "vfox.exe"

// pathListSeparator separates entries of PATH.
const pathListSeparator = ';'

func isExecutableMode(mode os.FileMode) bool {
	return true
}

// fallbackDirectories are the directories searched in addition to PATH: the
// install locations of winget, Scoop and a manual install that a packaged
// application's PATH often lacks.
func fallbackDirectories(environment map[string]string) []string {
	var directories []string
	home := environment["USERPROFILE"]
	if home == "" {
		home = environment["HOME"]
	}
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	localAppData := environment["LOCALAPPDATA"]
	programFiles := environment["ProgramFiles"]
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	directories = append(directories, filepath.Join(programFiles, "vfox"))
	if localAppData != "" {
		directories = append(directories,
			filepath.Join(localAppData, "Programs", "vfox"),
			filepath.Join(localAppData, "Microsoft", "WindowsApps"))
	}
	if home != "" {
		directories = append(directories,
			filepath.Join(home, "scoop", "shims"),
			filepath.Join(home, ".vfox", "bin"))
	}
	return directories
}

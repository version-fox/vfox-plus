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

package vfox

import (
	"os"
	"strings"
)

// IsExecutableFile reports whether path is an existing regular file that
// anybody may execute.
func IsExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// VfoxLocator looks for the vfox executable in `PATH` and in common install
// locations.
//
// Applications started from a desktop launcher do not source the user's shell
// profile, so directories such as `~/.local/bin` are often missing from their
// `PATH`. The well-known install locations of the official installers and
// package managers are therefore searched as well.
type VfoxLocator struct {
	environment   map[string]string
	isExecutable  func(path string) bool
	fallbackCache []string
}

// NewVfoxLocator creates a locator that searches environment and, for
// executability, isExecutable (IsExecutableFile by default, a stub in
// tests).
func NewVfoxLocator(environment map[string]string, isExecutable func(path string) bool) *VfoxLocator {
	if isExecutable == nil {
		isExecutable = IsExecutableFile
	}
	return &VfoxLocator{environment: environment, isExecutable: isExecutable}
}

// FallbackDirectories are the directories searched in addition to `PATH`, in
// order.
func (l *VfoxLocator) FallbackDirectories() []string {
	if l.fallbackCache == nil {
		directories := []string{"/usr/local/bin", "/usr/bin"}
		if home, ok := l.environment["HOME"]; ok && home != "" {
			directories = append(directories,
				home+"/.local/bin", home+"/bin", home+"/.linuxbrew/bin")
		}
		directories = append(directories, "/home/linuxbrew/.linuxbrew/bin", "/snap/bin")
		l.fallbackCache = directories
	}
	return l.fallbackCache
}

// SearchDirectories returns all directories that are searched, without
// duplicates.
func (l *VfoxLocator) SearchDirectories() []string {
	var path []string
	for _, entry := range strings.Split(l.environment["PATH"], ":") {
		if entry != "" {
			path = append(path, entry)
		}
	}
	seen := make(map[string]bool)
	var result []string
	for _, directory := range append(path, l.FallbackDirectories()...) {
		if !seen[directory] {
			seen[directory] = true
			result = append(result, directory)
		}
	}
	return result
}

// Locate returns the path of the first vfox executable found, or "".
func (l *VfoxLocator) Locate() string {
	for _, directory := range l.SearchDirectories() {
		if candidate := directory + "/" + VfoxExecutableName; l.isExecutable(candidate) {
			return candidate
		}
	}
	return ""
}

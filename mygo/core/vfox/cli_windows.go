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

// neutralDirectory is the working directory for commands that are not tied to
// a project: the root of the system drive, which cannot contain a project
// configuration (see docs/en/dev/vfox-cli-contract.md#working-directory).
func neutralDirectory() string {
	if systemDrive := os.Getenv("SystemDrive"); systemDrive != "" {
		return systemDrive + string(filepath.Separator)
	}
	if temp := os.TempDir(); temp != "" {
		return temp
	}
	return `C:\`
}

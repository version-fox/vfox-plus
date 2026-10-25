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

package measure

import "os"

// procAlive reports whether pid is still running. Windows has no signal-zero
// probe; callers only use this before cleanup, where attempting the kill is
// harmless, so conservatively report the process as alive.
func procAlive(pid int) bool {
	_ = pid
	return true
}

// killProcess terminates pid.
func killProcess(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

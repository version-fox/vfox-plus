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

package core

import "os"

// ProcessLock serializes state-changing vfox commands across processes.
//
// The task queue runs one command at a time, but only within its own process:
// the desktop application, `vfox-plus serve` and a background service can run
// side by side on one machine, and vfox itself does not lock its data
// directory. Every task therefore holds this lock on `tasks.lock` next to the
// settings while its process runs, so concurrent installs, uses and removals
// in different processes still cannot corrupt `.vfox.toml` or the symlinks.
// The lock is advisory and released when the holder dies, so a crashed
// process never wedges the others; pass an empty path to disable it.
type ProcessLock struct {
	path string
	file *os.File
}

// NewProcessLock creates a lock on path. The file is created on first use.
func NewProcessLock(path string) *ProcessLock {
	return &ProcessLock{path: path}
}

// Unlock releases the lock. Unlocking an unlocked lock does nothing.
func (l *ProcessLock) Unlock() {
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

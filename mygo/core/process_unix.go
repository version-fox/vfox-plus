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

//go:build !windows

package core

import (
	"os/exec"
	"syscall"
)

// killProcess asks the process to terminate with SIGTERM, so that vfox can
// clean up interrupted installs (see docs/vfox-cli-contract.md#cancelling).
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}

// platformExitCode maps an ExitError to an exit code. A process killed by a
// signal has a negative status, which the task queue treats as a failure.
func platformExitCode(exit *exec.ExitError) int {
	if status, ok := exit.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return -int(status.Signal())
		}
		return status.ExitStatus()
	}
	return 1
}

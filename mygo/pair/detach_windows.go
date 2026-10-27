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

package pair

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// detachProcess detaches command so that it survives the parent: the service
// daemon keeps running after `service start` exits. Like the file lock, the
// standard syscall package has no detached flag on Windows, so this uses
// golang.org/x/sys.
func detachProcess(command *exec.Cmd) {
	command.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS}
}

// terminateProcess stops pid.
func terminateProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

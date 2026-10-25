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

package core_test

import (
	"testing"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

func TestRunCollectsOutputWindows(t *testing.T) {
	runner := core.OsProcessRunner{}
	output, err := runner.Run("cmd", []string{"/c", "echo hello"},
		core.ProcessOptions{WorkingDirectory: "C:\\", Environment: map[string]string{"SystemRoot": "C:\\Windows"}})
	if err != nil {
		t.Fatal(err)
	}
	if output.ExitCode != 0 {
		t.Errorf("got %#v", output)
	}
}

func TestKillTerminatesWindows(t *testing.T) {
	collected := &collector{}
	process, err := core.OsProcessRunner{}.Start("cmd",
		[]string{"/c", "ping -n 30 127.0.0.1 >NUL"}, testOptionsWindows, collected)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	process.Kill()
	done := make(chan int, 1)
	go func() { done <- process.Wait() }()
	select {
	case code := <-done:
		if code == 0 {
			t.Errorf("killed processes fail, got %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Error("the process did not end")
	}
}

var testOptionsWindows = core.ProcessOptions{WorkingDirectory: "C:\\", Environment: map[string]string{}}

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

package core_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

var testOptions = core.ProcessOptions{WorkingDirectory: "/", Environment: map[string]string{"PATH": "/usr/bin:/bin", "GREETING": "hello"}}

type collector struct {
	mu  sync.Mutex
	out []byte
	err []byte
}

func (c *collector) Stdout(chunk []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.out = append(c.out, chunk...)
}

func (c *collector) Stderr(chunk []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = append(c.err, chunk...)
}

func (c *collector) output() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return string(c.out), string(c.err)
}

func TestRunCollectsOutput(t *testing.T) {
	runner := core.OsProcessRunner{}
	output, err := runner.Run("/bin/sh", []string{"-c", `echo "$GREETING $HOME"; echo oops >&2; pwd; read line || echo "no input"; exit 3`},
		core.ProcessOptions{WorkingDirectory: "/tmp", Environment: map[string]string{"GREETING": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if output.ExitCode != 3 || output.Stdout != "hello \n/tmp\nno input\n" || output.Stderr != "oops\n" {
		t.Errorf("got %#v", output)
	}
}

func TestRunDecodesMalformedOutput(t *testing.T) {
	output, err := core.OsProcessRunner{}.Run("/bin/sh", []string{"-c", `printf 'a\377b'`}, testOptions)
	if err != nil || output.Stdout != "a�b" {
		t.Errorf("got %#v %v", output, err)
	}
}

func TestRunReportsMissingExecutable(t *testing.T) {
	_, err := core.OsProcessRunner{}.Run("/nonexistent/vfox", nil, testOptions)
	var startErr *core.ProcessStartError
	if !errors.As(err, &startErr) || startErr.Executable != "/nonexistent/vfox" {
		t.Errorf("got %#v", err)
	}
}

func TestStartStreamsOutput(t *testing.T) {
	collected := &collector{}
	process, err := core.OsProcessRunner{}.Start("/bin/sh",
		[]string{"-c", "echo one; echo two >&2; echo three"}, testOptions, collected)
	if err != nil {
		t.Fatal(err)
	}
	if code := process.Wait(); code != 0 {
		t.Errorf("got %d", code)
	}
	if out, errOut := collected.output(); out != "one\nthree\n" || errOut != "two\n" {
		t.Errorf("got %q %q", out, errOut)
	}
	process.Kill()
}

func TestKillTerminates(t *testing.T) {
	collected := &collector{}
	process, err := core.OsProcessRunner{}.Start("/bin/sh", []string{"-c", "echo started; exec sleep 30"}, testOptions, collected)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if out, _ := collected.output(); out != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	process.Kill()
	done := make(chan int, 1)
	go func() { done <- process.Wait() }()
	select {
	case code := <-done:
		if code >= 0 {
			t.Errorf("killed processes fail, got %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Error("the process did not end")
	}
}

func TestStartReportsMissingExecutable(t *testing.T) {
	_, err := core.OsProcessRunner{}.Start("/nonexistent/vfox", nil, testOptions, &collector{})
	var startErr *core.ProcessStartError
	if !errors.As(err, &startErr) {
		t.Errorf("got %#v", err)
	}
}

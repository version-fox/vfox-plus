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

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// asProgramEnv makes the test binary run main, so that the tests start the program the way its users do and
// look at how it ends.
const asProgramEnv = "VFOX_PLUS_GATEWAY_AS_PROGRAM"

func TestMain(m *testing.M) {
	if os.Getenv(asProgramEnv) != "" {
		main()
		return
	}
	os.Exit(m.Run())
}

// program runs the program with the arguments and the environment, and says how it ended.
func program(t *testing.T, env []string, args ...string) (output string, exitCode int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], args...) //nolint:gosec // The test binary itself.
	command.Env = append(env, asProgramEnv+"=1")
	out, err := command.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the program did not run: %v", err)
	}
	return string(out), exit.ExitCode()
}

// environment is the environment of the tests without the variables that say where a program remembers.
func environment() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "HOME=") && !strings.HasPrefix(kv, "XDG_CONFIG_HOME=") && !strings.HasPrefix(kv, "AppData=") && !strings.HasPrefix(kv, asProgramEnv+"=") {
			env = append(env, kv)
		}
	}
	return env
}

// The gateway is a client of vfox+ and not a command: it takes no arguments, and the program that is
// told some says so and ends, where a program with a flag would listen on port 8080.
func TestTheProgramTakesNoArguments(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"serve"}, {"--port", "9000"}, {""}} {
		output, code := program(t, append(environment(), "HOME="+t.TempDir()), args...)
		if code != 2 || !strings.Contains(output, "takes no arguments") {
			t.Errorf("%q: ended with %d and said %q, want 2 and that it takes no arguments", args, code, output)
		}
	}
}

func TestTheProgramSaysWhenItHasNowhereToRemember(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "plan9" {
		t.Skip("the configuration directory of this system does not come from HOME alone")
	}
	output, code := program(t, environment())
	if code != 1 || !strings.Contains(output, "no directory to remember in") {
		t.Errorf("ended with %d and said %q, want 1 and why", code, output)
	}
}

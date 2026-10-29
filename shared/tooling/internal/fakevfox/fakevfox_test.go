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

package fakevfox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodePercentB(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain", "vfox version 1.0.13\n", "vfox version 1.0.13\n"},
		{"double backslash", `a\\b`, `a\b`},
		{"escaped backslash before n", `\\n`, "\\n"},
		{"newline", `\n`, "\n"},
		{"tab carriage return", `\t\r`, "\t\r"},
		{"bell backspace", `\a\b`, "\a\b"},
		{"escape", `\e\E`, "\x1b\x1b"},
		{"form feed vertical tab", `\f\v`, "\f\v"},
		{"quotes", `\'\"`, "'\""},
		{"hex field separator", `\x1f`, "\x1f"},
		{"hex carriage return", `\x0d`, "\r"},
		{"hex one digit", `\x4`, "\x04"},
		{"hex uppercase", `\x1F`, "\x1f"},
		{"hex no digits kept", `\xYZ`, `\xYZ`},
		{"octal", `\101\12\7`, "A\n\x07"},
		{"octal wraps", `\777`, "\xff"},
		{"unicode", `M\u00e9x`, "Méx"},
		{"terminate", `ab\ccd`, "ab"},
		{"trailing backslash", `a\`, `a\`},
		{"unknown kept", `\q`, `\q`},
		{"round trip", `path\\to\x00dir`, "path\\to\x00dir"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(DecodePercentB(tt.input)); got != tt.want {
				t.Errorf("DecodePercentB(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// fixtureEnv points the fake vfox at a temporary fixture directory and sets
// the caller contract environment the dispatch requires.
func fixtureEnv(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("VFOX_PLUS_FIXTURES", dir)
	t.Setenv("NO_COLOR", "1")
	t.Setenv("PWD", "/")
	t.Setenv("__VFOX_PID", "")
	t.Setenv("__VFOX_SHELL", "")
	t.Setenv("VFOX_PLUS_FAKE_DELAY", "0")
}

func writeFixture(t *testing.T, dir, name, stdout, stderr string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".stdout"), []byte(stdout), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".stderr"), []byte(stderr), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunCommandVersionAndList(t *testing.T) {
	dir := t.TempDir()
	// Captures escape backslashes and control characters; the fake vfox must
	// decode them back to the bytes vfox printed.
	writeFixture(t, dir, "version", "vfox version 1.0.13\n", "")
	writeFixture(t, dir, "list", `nodejs\\x1f24.11.1\n`, `\x1b[31merr\x1b[0m`+"\n")
	fixtureEnv(t, dir)

	var out, errOut strings.Builder
	if code := runCommand([]string{"--version"}, &out, &errOut); code != 0 {
		t.Fatalf("--version exit code = %d, want 0", code)
	}
	if got, want := out.String(), "vfox version 1.0.13\n"; got != want {
		t.Errorf("--version stdout = %q, want %q", got, want)
	}
	if errOut.String() != "" {
		t.Errorf("--version stderr = %q, want empty", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := runCommand([]string{"list"}, &out, &errOut); code != 0 {
		t.Fatalf("list exit code = %d, want 0", code)
	}
	if got, want := out.String(), "nodejs\\x1f24.11.1\n"; got != want {
		t.Errorf("list stdout = %q, want %q", got, want)
	}
	if got, want := errOut.String(), "\x1b[31merr\x1b[0m\n"; got != want {
		t.Errorf("list stderr = %q, want %q", got, want)
	}
}

func TestRunCommandInstallProgress(t *testing.T) {
	dir := t.TempDir()
	fixtureEnv(t, dir)

	var out, errOut strings.Builder
	code := runCommand([]string{"install", "--yes", "maven@3.9.16"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("install exit code = %d, want 0", code)
	}
	if got, want := out.String(), "Preinstalling maven@3.9.16...\nInstall maven@3.9.16 success!\n"; got != want {
		t.Errorf("install stdout = %q, want %q", got, want)
	}
	var wantErr strings.Builder
	for _, percent := range []int{10, 45, 80, 100} {
		wantErr.WriteString(fmt.Sprintf("\rDownloading... %3d%% [=====>    ]\r", percent))
	}
	if got := errOut.String(); got != wantErr.String() {
		t.Errorf("install stderr = %q, want %q", got, wantErr.String())
	}
}

func TestRunCommandContract(t *testing.T) {
	dir := t.TempDir()
	fixtureEnv(t, dir)

	t.Run("no color", func(t *testing.T) {
		t.Setenv("NO_COLOR", "0")
		var out, errOut strings.Builder
		if code := runCommand([]string{"--version"}, &out, &errOut); code != exitNoColor {
			t.Errorf("exit code = %d, want %d", code, exitNoColor)
		}
		if got := out.String(); got != "NO_COLOR is not set\n" {
			t.Errorf("stdout = %q", got)
		}
	})

	t.Run("pid leaked", func(t *testing.T) {
		t.Setenv("__VFOX_PID", "123")
		var out, errOut strings.Builder
		if code := runCommand([]string{"--version"}, &out, &errOut); code != exitPIDLeaked {
			t.Errorf("exit code = %d, want %d", code, exitPIDLeaked)
		}
	})

	t.Run("not at root", func(t *testing.T) {
		t.Setenv("PWD", "/somewhere")
		if err := os.Chdir(t.TempDir()); err != nil {
			t.Skipf("cannot change directory: %v", err)
		}
		var out, errOut strings.Builder
		if code := runCommand([]string{"list"}, &out, &errOut); code != exitNotAtRoot {
			t.Errorf("exit code = %d, want %d", code, exitNotAtRoot)
		}
	})

	t.Run("leaked PWD is ignored for the real root", func(t *testing.T) {
		// The environment PWD leaks in through every launcher layer; only
		// the working directory decides.
		t.Setenv("PWD", "/somewhere")
		if err := os.Chdir("/"); err != nil {
			t.Skipf("cannot change to /: %v", err)
		}
		var out, errOut strings.Builder
		if code := runCommand([]string{"config", "--list"}, &out, &errOut); code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
	})

	t.Run("empty PWD falls back to working directory", func(t *testing.T) {
		// Like bash, which sets PWD on startup: the application spawns
		// vfox with a clean environment that has no PWD.
		t.Setenv("PWD", "")
		previous, err := os.Getwd()
		if err != nil {
			t.Skipf("cannot read working directory: %v", err)
		}
		if err := os.Chdir("/"); err != nil {
			t.Skipf("cannot change to /: %v", err)
		}
		defer func() { _ = os.Chdir(previous) }()
		var out, errOut strings.Builder
		if code := runCommand([]string{"config", "--list"}, &out, &errOut); code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
	})

	t.Run("use allowed off root", func(t *testing.T) {
		t.Setenv("PWD", "/somewhere")
		t.Setenv("__VFOX_SHELL", "bash")
		var out, errOut strings.Builder
		if code := runCommand([]string{"use", "--global", "nodejs@24.11.1"}, &out, &errOut); code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		if got := out.String(); got != "Now using nodejs@24.11.1.\n" {
			t.Errorf("stdout = %q", got)
		}
	})

	t.Run("unexpected", func(t *testing.T) {
		var out, errOut strings.Builder
		if code := runCommand([]string{"frobnicate"}, &out, &errOut); code != exitUnexpected {
			t.Errorf("exit code = %d, want %d", code, exitUnexpected)
		}
		if got := errOut.String(); got != "unexpected command: frobnicate\n" {
			t.Errorf("stderr = %q", got)
		}
	})
}

func TestRunCommandReplaysTheCapturesOfJavaAndFlutter(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"search-java", "search-java-tem", "search-java-all", "search-flutter", "info-java", "info-flutter"} {
		writeFixture(t, dir, name, name+"\n", "")
	}
	fixtureEnv(t, dir)
	infoJava := "info java --format {{.Name}}"
	for _, c := range []struct{ command, want string }{
		{"search java", "search-java\n"},
		{"search java tem", "search-java-tem\n"},
		{"search java all", "search-java-all\n"},
		{"search flutter", "search-flutter\n"},
		{infoJava, "info-java\n"},
		{"info flutter --format {{.Name}}", "info-flutter\n"},
	} {
		var out, errOut strings.Builder
		if code := runCommand(strings.Fields(c.command), &out, &errOut); code != 0 || out.String() != c.want {
			t.Errorf("%s: exit %d, stdout %q, want %q", c.command, code, out.String(), c.want)
		}
	}
	// Another distribution is not a capture: the fake answers like it does for any other plugin.
	var out, errOut strings.Builder
	if code := runCommand([]string{"search", "java", "zulu"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "2.1.0 (LTS)") {
		t.Errorf("search java zulu: exit %d, stdout %q", code, out.String())
	}
}

func TestRunCommandListsTheMachineWithJavaAndFlutterWhenAsked(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "list", "plain list\n", "")
	writeFixture(t, dir, "current", "plain current\n", "")
	writeFixture(t, dir, "list-profiles", "java list\n", "")
	writeFixture(t, dir, "current-profiles", "java current\n", "")
	fixtureEnv(t, dir)
	for _, c := range []struct{ inventory, command, want string }{
		{"", "list", "plain list\n"},
		{"", "current", "plain current\n"},
		{"profiles", "list", "java list\n"},
		{"profiles", "current", "java current\n"},
		{"other", "list", "plain list\n"},
	} {
		t.Setenv("VFOX_PLUS_FAKE_INVENTORY", c.inventory)
		var out, errOut strings.Builder
		if code := runCommand([]string{c.command}, &out, &errOut); code != 0 || out.String() != c.want {
			t.Errorf("inventory %q, %s: exit %d, stdout %q, want %q", c.inventory, c.command, code, out.String(), c.want)
		}
	}
}

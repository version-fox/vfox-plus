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

// Package capture recreates the vfox CLI output fixtures in
// shared/fixtures/vfox-<version>/, replacing shared/fixtures/capture.sh.
//
// Run it as a regular user: one scenario relies on file permissions.
//
// Every command runs against a throw-away HOME under a fixed sandbox path,
// so the real vfox installation is never touched and the captured paths stay
// stable. It needs network access: it adds the nodejs and maven plugins,
// searches their versions, installs Maven, and adds the java and flutter
// plugins to search them. After a run, compare the new
// outputs with cases.json and update the expected values there by hand. See
// shared/fixtures/README.md.
package capture

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// infoFormat is the --format template used for the info captures.
const infoFormat = `{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}`

// versionRe matches the version number in `vfox --version` output,
// the same pattern capture.sh greps for.
var versionRe = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+[^ ]*`)

// unsetEnv lists the variables capture.sh removes so vfox uses (and changes)
// the throw-away HOME instead of a shared vfox directory.
var unsetEnv = []string{
	"VFOX_HOME",
	"__VFOX_SHELL",
	"__VFOX_PID",
	"__VFOX_CURTMPPATH",
	"__VFOX_INITIALIZED",
	"CI",
	"GITHUB_ACTIONS",
}

// Run captures the vfox CLI outputs into shared/fixtures/vfox-<version>/.
// It accepts the same selection of settings capture.sh derives from its
// environment: --vfox PATH (default "vfox", or the first positional argument)
// and --maven-version VER (default $MAVEN_VERSION or 3.9.16).
func Run(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	vfoxPath := fs.String("vfox", "", "path to the vfox binary")
	mavenVersion := fs.String("maven-version", "", "Maven version to install")
	if err := fs.Parse(args); err != nil {
		return err
	}
	vfox := *vfoxPath
	if vfox == "" {
		if fs.NArg() > 0 {
			vfox = fs.Arg(0)
		} else {
			vfox = "vfox"
		}
	}
	maven := *mavenVersion
	if maven == "" {
		maven = envOr("MAVEN_VERSION", "3.9.16")
	}
	return run(vfox, maven, os.Stdout, os.Stderr)
}

// EscapeControls rewrites data the way capture.sh escape_controls does, so
// every capture stays plain text: a backslash becomes two backslashes, and a
// control character other than tab and line feed becomes \xHH with lowercase
// hexadecimal digits. All other bytes pass through unchanged.
func EscapeControls(data []byte) []byte {
	const hex = "0123456789abcdef"
	out := make([]byte, 0, len(data))
	for _, b := range data {
		switch {
		case b == '\\':
			out = append(out, '\\', '\\')
		case b == '\t' || b == '\n' || (b >= 0x20 && b != 0x7f):
			out = append(out, b)
		default:
			out = append(out, '\\', 'x', hex[b>>4], hex[b&0x0f])
		}
	}
	return out
}

// escapeFiles applies EscapeControls to each named file in place.
func escapeFiles(paths ...string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, EscapeControls(data), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// fixturesDir finds the shared/fixtures directory the captures belong in: the
// one above the current directory when running from the repository,
// otherwise next to the running executable, mirroring how capture.sh writes
// next to itself.
func fixturesDir() string {
	dir, err := os.Getwd()
	if err == nil {
		for {
			candidate := filepath.Join(dir, "shared", "fixtures")
			if st, err := os.Stat(candidate); err == nil && st.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("shared", "fixtures")
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// runner holds the state every captured command shares.
type runner struct {
	vfox string
	out  string
	home string
	env  []string // base environment for vfox, without caller overrides
	null *os.File
	// print and errOut receive the "name ... exit ..." progress lines and the
	// retry notices, i.e. what capture.sh writes to its own stdout and stderr.
	print  io.Writer
	errOut io.Writer
}

// environ returns the base environment plus extra VAR=value overrides.
func (r *runner) environ(extra []string) []string {
	return append(append([]string{}, r.env...), extra...)
}

// baseEnv builds the environment every vfox invocation shares: the current
// environment with the vfox session variables removed, HOME pointed at the
// throw-away sandbox and colors disabled.
func baseEnv(home string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+2)
skip:
	for _, kv := range env {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		for _, drop := range unsetEnv {
			if key == drop {
				continue skip
			}
		}
		out = append(out, kv)
	}
	return append(out, "HOME="+home, "NO_COLOR=1")
}

// capture runs vfox with extra VAR=value overrides and vfoxArgs, saving its
// standard output, standard error and exit code the way capture.sh does:
// <name>.stdout, <name>.stderr and a "<name> exit <code>   vfox <args>" line.
// A non-zero exit is recorded, not reported as an error.
func (r *runner) capture(name string, extraEnv []string, vfoxArgs ...string) error {
	stdoutPath := filepath.Join(r.out, name+".stdout")
	stderrPath := filepath.Join(r.out, name+".stderr")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return err
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		stdoutFile.Close()
		return err
	}
	cmd := exec.Command(r.vfox, vfoxArgs...)
	cmd.Dir = "/"
	cmd.Env = r.environ(extraEnv)
	cmd.Stdin = r.null
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	runErr := cmd.Run()
	closeErr := stdoutFile.Close()
	if cerr := stderrFile.Close(); cerr != nil && closeErr == nil {
		closeErr = cerr
	}
	status := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			status = exitErr.ExitCode()
		} else {
			return fmt.Errorf("capture %s: %w", name, runErr)
		}
	}
	if closeErr != nil {
		return fmt.Errorf("capture %s: %w", name, closeErr)
	}
	if err := escapeFiles(stdoutPath, stderrPath); err != nil {
		return err
	}
	fmt.Fprintf(r.print, "%-30s exit %d   vfox %s\n", name, status, strings.Join(vfoxArgs, " "))
	return nil
}

// retry runs a setup step, retrying on network errors up to five times.
func (r *runner) retry(vfoxArgs ...string) error {
	joined := strings.Join(vfoxArgs, " ")
	for attempt := 1; attempt <= 5; attempt++ {
		cmd := exec.Command(r.vfox, vfoxArgs...)
		cmd.Dir = "/"
		cmd.Env = r.environ(nil)
		cmd.Stdin = r.null
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err == nil {
			return nil
		}
		fmt.Fprintf(r.errOut, "  vfox %s failed, retrying (%d)\n", joined, attempt)
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("vfox %s keeps failing", joined)
}

// runCmd runs vfox once, wiring its output to stdout and stderr. Unlike
// capture it propagates a non-zero exit as an error.
func (r *runner) runCmd(extraEnv []string, stdout, stderr io.Writer, vfoxArgs ...string) error {
	cmd := exec.Command(r.vfox, vfoxArgs...)
	cmd.Dir = "/"
	cmd.Env = r.environ(extraEnv)
	cmd.Stdin = r.null
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vfox %s: %w", strings.Join(vfoxArgs, " "), err)
	}
	return nil
}

// fakeInstall fakes an installed version, so that the tree output contains
// several versions without downloading them.
func (r *runner) fakeInstall(sdk, version string) error {
	dir := filepath.Join(r.home, ".vfox", "cache", sdk, "v-"+version, sdk+"-"+version, "bin")
	return os.MkdirAll(dir, 0o755)
}

// existsDir reports whether path exists and is a directory.
func existsDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func run(vfox, mavenVersion string, print, errOut io.Writer) error {
	if strings.ContainsRune(vfox, filepath.Separator) {
		st, err := os.Stat(vfox)
		if err != nil || st.IsDir() || st.Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("vfox not found: %s", vfox)
		}
	} else if _, err := exec.LookPath(vfox); err != nil {
		return fmt.Errorf("vfox not found: %s", vfox)
	}

	// The version decides the output directory, like capture.sh.
	versionCmd := exec.Command(vfox, "--version")
	versionOut, err := versionCmd.Output()
	if err != nil {
		return fmt.Errorf("vfox --version: %w", err)
	}
	version := versionRe.FindString(string(versionOut))
	if version == "" {
		return fmt.Errorf("cannot determine vfox version from %q", strings.TrimSpace(string(versionOut)))
	}

	out := filepath.Join(fixturesDir(), "vfox-"+version)
	sandbox := envOr("VFOX_FIXTURE_SANDBOX", "/tmp/vfox-sb")
	home := filepath.Join(sandbox, "home")

	// A fixed sandbox path keeps the captured paths stable between runs.
	if err := os.RemoveAll(sandbox); err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	null, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer null.Close()

	r := &runner{vfox: vfox, out: out, home: home, env: baseEnv(home), null: null, print: print, errOut: errOut}
	pluginDir := func(name string) string { return filepath.Join(home, ".vfox", "plugin", name) }
	cacheDir := func(sdk, version string) string {
		return filepath.Join(home, ".vfox", "cache", sdk, "v-"+version)
	}

	if err := r.capture("version", nil, "--version"); err != nil {
		return err
	}
	if err := r.capture("current-empty", nil, "current"); err != nil {
		return err
	}
	if err := r.capture("list-empty", nil, "list"); err != nil {
		return err
	}

	for attempt := 1; attempt <= 5; attempt++ {
		if err := r.capture("add", nil, "add", "nodejs"); err != nil {
			return err
		}
		if existsDir(pluginDir("nodejs")) {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err := r.retry("add", "maven"); err != nil {
		return err
	}
	if err := r.capture("add-existing", nil, "add", "maven"); err != nil {
		return err
	}
	if err := r.capture("add-unknown", nil, "add", "nosuchplugin"); err != nil {
		return err
	}
	for attempt := 1; attempt <= 5; attempt++ {
		if err := r.capture("add-alias", nil, "add", "--alias", "mvn2", "maven"); err != nil {
			return err
		}
		if existsDir(pluginDir("mvn2")) {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if err := r.capture("current-unset", nil, "current"); err != nil {
		return err
	}
	if err := r.capture("current-sdk-unset", nil, "current", "maven"); err != nil {
		return err
	}
	if err := r.capture("list-sdk-without-versions", nil, "list", "maven"); err != nil {
		return err
	}

	if err := r.capture("search-maven", nil, "search", "maven"); err != nil {
		return err
	}
	if err := r.capture("search-nodejs", nil, "search", "nodejs"); err != nil {
		return err
	}
	if err := r.capture("search-unknown", nil, "search", "nosuchsdk"); err != nil {
		return err
	}
	if err := r.capture("available", nil, "available"); err != nil {
		return err
	}
	if err := r.capture("info", nil, "info", "nodejs", "--format", infoFormat); err != nil {
		return err
	}
	if err := r.capture("info-unknown", nil, "info", "nosuchsdk"); err != nil {
		return err
	}

	if err := r.capture("config-list", nil, "config", "--list"); err != nil {
		return err
	}
	if err := r.capture("config-set", nil, "config", "proxy.url", "http://127.0.0.1:7890"); err != nil {
		return err
	}
	if err := r.capture("config-get", nil, "config", "proxy.url"); err != nil {
		return err
	}
	if err := r.runCmd(nil, print, errOut, "config", "--unset", "proxy.url"); err != nil {
		return err
	}
	if err := r.capture("config-set-invalid-duration", nil, "config", "cache.availableHookDuration", "notaduration"); err != nil {
		return err
	}

	if err := r.capture("install-without-version", nil, "install", "--yes", "maven"); err != nil {
		return err
	}
	for attempt := 1; attempt <= 5; attempt++ {
		if err := r.capture("install", nil, "install", "--yes", "maven@"+mavenVersion); err != nil {
			return err
		}
		if existsDir(cacheDir("maven", mavenVersion)) {
			break
		}
		fmt.Fprintf(print, "  download failed, retrying (%d)\n", attempt)
		time.Sleep(3 * time.Second)
	}
	if err := r.capture("search-maven-installed", nil, "search", "maven"); err != nil {
		return err
	}

	if err := r.capture("use-without-hook", nil, "use", "--global", "maven@"+mavenVersion); err != nil {
		return err
	}
	if err := r.capture("use-global", []string{"__VFOX_SHELL=bash"}, "use", "--global", "maven@"+mavenVersion); err != nil {
		return err
	}

	if err := r.fakeInstall("maven", "3.8.9"); err != nil {
		return err
	}
	if err := r.fakeInstall("nodejs", "24.11.1"); err != nil {
		return err
	}
	if err := r.fakeInstall("nodejs", "22.21.0"); err != nil {
		return err
	}
	if err := r.runCmd([]string{"__VFOX_SHELL=bash"}, io.Discard, errOut, "use", "--global", "nodejs@24.11.1"); err != nil {
		return err
	}
	if err := r.capture("list", nil, "list"); err != nil {
		return err
	}
	if err := r.capture("current", nil, "current"); err != nil {
		return err
	}
	if err := r.capture("current-sdk", nil, "current", "nodejs"); err != nil {
		return err
	}
	if err := r.capture("list-sdk", nil, "list", "nodejs"); err != nil {
		return err
	}
	if err := r.capture("info-version-path", nil, "info", "nodejs@24.11.1"); err != nil {
		return err
	}
	if err := r.capture("info-version-not-installed", nil, "info", "nodejs@1.0.0"); err != nil {
		return err
	}

	if err := r.capture("remove-without-yes", nil, "remove", "mvn2"); err != nil {
		return err
	}
	if err := r.capture("remove", nil, "remove", "--yes", "mvn2"); err != nil {
		return err
	}

	if err := r.capture("unuse-global", nil, "unuse", "--global", "maven"); err != nil {
		return err
	}
	if err := r.capture("uninstall-not-installed", nil, "uninstall", "maven@1.2.3"); err != nil {
		return err
	}
	if err := r.capture("uninstall", nil, "uninstall", "maven@3.8.9"); err != nil {
		return err
	}

	if err := r.capture("update", nil, "update", "maven"); err != nil {
		return err
	}
	if err := r.capture("update-all", nil, "update", "--all"); err != nil {
		return err
	}
	if err := r.capture("update-unknown", nil, "update", "nosuch"); err != nil {
		return err
	}

	// A read-only install directory makes the installation fail
	// deterministically. (Do not run this command as root, which ignores file
	// permissions.)
	mavenCache := filepath.Join(home, ".vfox", "cache", "maven")
	if err := os.RemoveAll(cacheDir("maven", mavenVersion)); err != nil {
		return err
	}
	st, err := os.Stat(mavenCache)
	if err != nil {
		return err
	}
	if err := os.Chmod(mavenCache, st.Mode().Perm()&^0o222); err != nil {
		return err
	}
	if err := r.capture("install-failure", nil, "install", "--yes", "maven@"+mavenVersion); err != nil {
		return err
	}
	if st, err := os.Stat(mavenCache); err != nil {
		return err
	} else if err := os.Chmod(mavenCache, st.Mode().Perm()|0o200); err != nil {
		return err
	}

	// The plugins of the version-fox organisation whose lists have notes and
	// names of their own: vfox-java takes an argument (`vfox search java tem`
	// lists one distribution, `all` every one), vfox-flutter's notes are the
	// channels. They are added last, so that the outputs above do not list
	// them.
	for _, plugin := range []string{"java", "flutter"} {
		if err := r.retry("add", plugin); err != nil {
			return err
		}
	}
	for _, search := range []struct {
		name string
		args []string
	}{
		{"search-java", []string{"search", "java"}},
		{"search-java-tem", []string{"search", "java", "tem"}},
		{"search-java-all", []string{"search", "java", "all"}},
		{"search-flutter", []string{"search", "flutter"}},
	} {
		if err := r.capture(search.name, nil, search.args...); err != nil {
			return err
		}
	}

	fmt.Fprintf(print, "Fixtures written to %s\n", out)
	return nil
}

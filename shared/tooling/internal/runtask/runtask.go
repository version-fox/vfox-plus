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

// Package runtask runs one task inside a sandbox container of the mygo
// implementation, replacing mygo/sandbox/run-task.sh and
// shared/sandbox/task-lib.sh.
//
// The steps mirror the jobs of .github/workflows/mygo.yml; keep them in sync.
//
// The repository is mounted read-only at /src and the results directory at
// /out. A task works on a private copy of the repository as the
// unprivileged user tester, so the only thing that reaches the host is what
// it copies to /out.
package runtask

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/display"
)

const usage = "Usage: runtask <task>"

// Runtime dependencies of the release binary on Ubuntu 26.04; the
// installation test checks that they are complete. The executable is linked
// to the C library alone: mygo loads GTK 3, which brings GLib, Pango, Cairo
// and the rest, when it starts (see docs/en/dev/architecture.md#dependencies).
// Fonts: Ubuntu Sans, or any other sans-serif font of the desktop.
const Depends = "libc6, libgtk-3-0t64, libglib2.0-0t64, libpango-1.0-0, libpangoft2-1.0-0, libcairo2, libgdk-pixbuf-2.0-0, fontconfig, fonts-ubuntu | fonts-noto-core"

// Chinese text needs a CJK font, the file choosers go through the desktop
// portal where there is one, and links open in the browser.
const Recommends = "fonts-noto-cjk, xdg-desktop-portal, xdg-utils"

// tasks are the valid first arguments, mirroring the case of the old
// run-task.sh.
var tasks = []string{
	"deps",
	"check",
	"build",
	"e2e",
	"measure",
	"screenshots",
	"package",
	"install-test",
	"shell",
	"print-depends",
	"print-recommends",
}

// ValidTask reports whether name is a task runtask knows.
func ValidTask(name string) bool {
	for _, task := range tasks {
		if name == task {
			return true
		}
	}
	return false
}

// env holds the container paths of task-lib.sh: the read-only repository,
// the results directory and the working copy. Each may be overridden by the
// environment of the same name.
type env struct {
	src  string
	out  string
	work string
}

func loadEnv() env {
	return env{
		orDefault(os.Getenv("SRC"), "/src"),
		orDefault(os.Getenv("OUT"), "/out"),
		orDefault(os.Getenv("WORK"), "/home/tester/work"),
	}
}

func orDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

// goOffline is the prefix of the Go commands of every task but deps: with
// GOPROXY=off a missing module is an error rather than an attempt to
// download it.
var goOffline = []string{"env", "GOPROXY=off", "GOFLAGS=-mod=readonly"}

// copyExcludes are what a task neither sees nor spends time copying: what
// the host may have built or downloaded in the copied directories, and what
// the build generates.
var copyExcludes = []string{
	"--exclude-vcs",
	"--exclude=.idea",
	"--exclude=*.iml",
	"--exclude=./mygo/build",
	"--exclude=./mygo/coverage.out",
	"--exclude=./mygo/assets/fonts",
	"--exclude=./mygo/assets/i18n",
	"--exclude=./mygo/assets/icons.json",
	"--exclude=./mygo/assets/licenses.txt",
}

// offlineEnv carries the offline Go settings to inner tool calls: with
// GOPROXY=off a missing module is an error rather than an attempt to
// download it.
var offlineEnv = []string{"GOPROXY=off", "GOFLAGS=-mod=readonly"}

// tool runs a vfxtool subcommand as tester in a directory, with HOME set
// like asTester. The binary re-executes itself: under `go run` os.Executable
// is the compiled copy, which accepts the same subcommands.
// alreadyTester reports whether the process already runs as tester. Only
// root may use runuser, so when the container entrypoint has switched user
// itself, commands run directly instead of through runuser again.
func alreadyTester() bool {
	u, err := user.Lookup("tester")
	if err != nil {
		return false
	}
	return strconv.Itoa(os.Getuid()) == u.Uid
}

// testerCommand builds a command that runs as tester with HOME set, through
// runuser when the process runs as root and directly when it already runs
// as tester.
func testerCommand(args ...string) *exec.Cmd {
	full := append([]string{"env", "HOME=/home/tester"}, args...)
	if !alreadyTester() {
		full = append([]string{"runuser", "-u", "tester", "--"}, full...)
	}
	return exec.Command(full[0], full[1:]...)
}

func tool(directory string, env []string, subcommand string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	argv := append([]string{}, env...)
	argv = append(argv, exe, subcommand)
	argv = append(argv, args...)
	cmd := testerCommand(argv...)
	cmd.Dir = directory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run runs one task inside a sandbox container. args are the command line
// without the program name: runtask <task>.
func Run(args []string) error {
	task := ""
	if len(args) > 0 {
		task = args[0]
	}
	switch task {
	case "print-depends":
		fmt.Println(Depends)
		return nil
	case "print-recommends":
		fmt.Println(Recommends)
		return nil
	}
	if !ValidTask(task) {
		return fmt.Errorf("Unknown task: %s. Use vfxtool sandbox.", task)
	}
	e := loadEnv()
	app := filepath.Join(e.work, "mygo")
	binary := "/home/tester/build/vfox-plus"

	prepare := func() error {
		if err := e.copyRepository("mygo"); err != nil {
			return err
		}
		return tool(app, offlineEnv, "prepare", "--root", app)
	}
	prepareLight := func() error {
		if err := e.copyRepository("mygo"); err != nil {
			return err
		}
		return tool(app, append(append([]string{}, offlineEnv...), "MYGO_SKIP_LICENSES=1"), "prepare", "--root", app)
	}
	build := func(kind, output string) error {
		if err := run("install", "-d", "-o", "tester", "-g", "tester", filepath.Dir(output)); err != nil {
			return err
		}
		buildArgs := []string{"build", "--root", app}
		if kind == "e2e" {
			buildArgs = []string{"build", "--e2e", "--root", app}
		}
		return tool(app, offlineEnv, buildArgs[0], append(buildArgs[1:], output)...)
	}

	switch task {
	case "deps":
		if err := e.copyRepository("mygo"); err != nil {
			return err
		}
		download := []string{"env"}
		if proxy := os.Getenv("SANDBOX_GOPROXY"); proxy != "" {
			download = append(download, "GOPROXY="+proxy)
		}
		if err := asTester(app, append(download, "go", "mod", "download")...); err != nil {
			return err
		}
		// The container runs every task through `go run` of the vfxtool,
		// so its modules belong in the shared module cache too.
		return asTester(filepath.Join(e.work, "shared", "tooling"), append(download, "go", "mod", "download")...)

	case "check":
		if err := prepareLight(); err != nil {
			return err
		}
		offline := func(args ...string) []string {
			return append(append([]string{}, goOffline...), args...)
		}
		for _, step := range [][]string{
			offline("go", "mod", "verify"),
			offline("golangci-lint", "fmt", "--diff"),
			offline("golangci-lint", "run"),
			offline("golangci-lint", "run", "--build-tags", "e2e"),
			offline("go", "vet", "./..."),
			// The race detector needs the C compiler; the application is
			// built without.
			offline("env", "CGO_ENABLED=1", "go", "test", "-race", "-coverprofile=coverage.out", "./..."),
		} {
			if err := asTester(app, step...); err != nil {
				return err
			}
		}
		if err := tool(app, offlineEnv, "coverage", "coverage.out", "85"); err != nil {
			return err
		}
		// The vfxtool sources the task runs.
		tooling := filepath.Join(e.work, "shared", "tooling")
		if out, err := asTesterOutput(tooling, append(goOffline, "gofmt", "-l", ".")...); err != nil {
			return err
		} else if strings.TrimSpace(out) != "" {
			return fmt.Errorf("files need gofmt:\n%s", out)
		}
		return asTester(tooling, append(goOffline, "go", "test", "./...")...)

	case "build":
		if err := prepare(); err != nil {
			return err
		}
		return build("release", binary)

	case "e2e":
		if err := prepareLight(); err != nil {
			return err
		}
		if err := build("e2e", binary); err != nil {
			return err
		}
		d, err := startDisplay("1280x800x24")
		if err != nil {
			return err
		}
		defer d.Stop()
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := inDisplay("VFOX_PLUS_E2E_TIMEOUT=300", exe, "e2e-run", "fake", binary); err != nil {
			return err
		}
		return inDisplay("VFOX_PLUS_E2E_TIMEOUT=300", "VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox", exe, "e2e-run", "real", binary)

	case "measure":
		if err := prepare(); err != nil {
			return err
		}
		if err := build("e2e", binary); err != nil {
			return err
		}
		d, err := startDisplay("1280x800x24")
		if err != nil {
			return err
		}
		defer d.Stop()
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		out, err := inDisplayOutput(exe, "measure", binary)
		if err != nil {
			return err
		}
		// Like `tee /tmp/measure.log`: the output goes to the log and to
		// the console.
		fmt.Print(out)
		if err := os.WriteFile("/tmp/measure.log", []byte(out), 0o644); err != nil {
			return err
		}
		var memory strings.Builder
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "memory") {
				memory.WriteString(line + "\n")
			}
		}
		if err := os.WriteFile("/tmp/memory.txt", []byte(memory.String()), 0o644); err != nil {
			return err
		}
		return e.publish("/tmp/memory.txt")

	case "screenshots":
		if err := prepare(); err != nil {
			return err
		}
		if err := build("e2e", binary); err != nil {
			return err
		}
		// Taller than the window, so that the captures show the whole
		// window with its title bar. The window manager draws it with the
		// Greybird theme: a dark bar over the light content, the way modern
		// GNOME applications look (this window has no custom title bar, so
		// GTK leaves the decorations to the window manager). The default
		// wallpaper of Ubuntu fills the root window behind it, instead of
		// the black of a bare X server.
		d, err := startDisplay("1280x860x24")
		if err != nil {
			return err
		}
		defer d.Stop()
		// The settings daemon the window manager reads its theme from; it
		// lives outside PATH.
		if err := inDisplayStart("/usr/lib/x86_64-linux-gnu/xfce4/xfconfd"); err != nil {
			return err
		}
		time.Sleep(time.Second)
		if err := inDisplay("xfconf-query", "-c", "xfwm4", "-p", "/general/theme", "-n", "-t", "string", "-s", "Greybird"); err != nil {
			return err
		}
		if err := inDisplayStart("xfwm4"); err != nil {
			return err
		}
		// Wait until the window manager has taken over and settled, so
		// that the first window it frames already has its decorations.
		for range 60 {
			if err := inDisplay("xprop", "-root", "_NET_SUPPORTING_WM_CHECK"); err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(3 * time.Second)
		if err := setWallpaper(); err != nil {
			return err
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := inDisplay(exe, "screenshots", "/home/tester/screenshots", binary); err != nil {
			return err
		}
		matches, err := filepath.Glob("/home/tester/screenshots/*.png")
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("no screenshots in /home/tester/screenshots")
		}
		return e.publish(matches...)

	case "package":
		if err := prepare(); err != nil {
			return err
		}
		if err := build("release", binary); err != nil {
			return err
		}
		if err := smokeTest(binary); err != nil {
			return err
		}
		// The Windows executables cross-compile on Linux: mygo needs no C
		// compiler. They are packaged as unsigned MSIX below.
		for _, arch := range []string{"amd64", "arm64"} {
			output := "/home/tester/stage-windows-" + arch + "/vfox-plus.exe"
			if err := run("install", "-d", "-o", "tester", "-g", "tester", filepath.Dir(output)); err != nil {
				return err
			}
			if err := tool(app, offlineEnv, "build", "--root", app, "--goos", "windows", "--goarch", arch, output); err != nil {
				return err
			}
		}
		if err := run("install", "-d", "-o", "tester", "-g", "tester", "/home/tester/stage", "/home/tester/dist"); err != nil {
			return err
		}
		if err := run("install", "-o", "tester", "-m", "755", binary, "/home/tester/stage/vfox-plus"); err != nil {
			return err
		}
		// A static vfxtool for the installation test, which runs in a
		// clean container without a Go toolchain. It is published next
		// to the .deb and mounted as /pkg.
		tooling := filepath.Join(e.work, "shared", "tooling")
		if err := asTester(tooling, append(append([]string{}, goOffline...),
			"env", "CGO_ENABLED=0", "go", "build", "-o", "/home/tester/dist/vfxtool", "./cmd/vfxtool")...); err != nil {
			return err
		}
		versionData, err := os.ReadFile(filepath.Join(app, "VERSION"))
		if err != nil {
			return err
		}
		version := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, string(versionData))
		epoch := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH"))
		if epoch == "" {
			epoch = fmt.Sprint(time.Now().Unix())
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := asTester(app,
			"SOURCE_DATE_EPOCH="+epoch,
			exe, "build-deb",
			"--package", "vfox-plus",
			"--version", version,
			"--stage", "/home/tester/stage",
			"--executable", "vfox-plus",
			"--technology", "mygo",
			"--depends", Depends,
			"--recommends", Recommends,
			"--notice", filepath.Join(app, "assets", "licenses.txt"),
			"/home/tester/dist"); err != nil {
			return err
		}
		if err := run("desktop-file-validate", filepath.Join(e.work, "shared", "packaging", "linux", "dev.vfox.VfoxPlus.desktop")); err != nil {
			return err
		}
		debs, err := filepath.Glob("/home/tester/dist/*.deb")
		if err != nil {
			return err
		}
		if len(debs) == 0 {
			return fmt.Errorf("no .deb in /home/tester/dist")
		}
		if err := asTester(app, append([]string{
			"lintian", "--fail-on", "error",
			"--suppress-tags", "malformed-contact,bogus-mail-host-in-debian-changelog",
		}, debs...)...); err != nil {
			return err
		}
		if err := asTester(app,
			"SOURCE_DATE_EPOCH="+epoch,
			exe, "build-msix",
			"--package", "vfox-plus",
			"--version", version,
			"--stage-x64", "/home/tester/stage-windows-amd64",
			"--stage-arm64", "/home/tester/stage-windows-arm64",
			"--executable", "vfox-plus.exe",
			"/home/tester/dist"); err != nil {
			return err
		}
		if err := asTester(app, exe, "validate-msix", "/home/tester/dist"); err != nil {
			return err
		}
		// The inputs of `vfxtool sign-msix`, which signs on the user's Windows
		// machine: the loose executables, and a Windows vfxtool, which
		// rebuilds the packages with the self-signed publisher.
		if err := run("install", "-d", "-o", "tester", "-g", "tester",
			"/home/tester/dist/stage-x64", "/home/tester/dist/stage-arm64"); err != nil {
			return err
		}
		if err := run("install", "-o", "tester", "-m", "644",
			"/home/tester/stage-windows-amd64/vfox-plus.exe",
			"/home/tester/dist/stage-x64/vfox-plus.exe"); err != nil {
			return err
		}
		if err := run("install", "-o", "tester", "-m", "644",
			"/home/tester/stage-windows-arm64/vfox-plus.exe",
			"/home/tester/dist/stage-arm64/vfox-plus.exe"); err != nil {
			return err
		}
		if err := asTester(tooling, append(append([]string{}, goOffline...),
			"env", "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64",
			"go", "build", "-o", "/home/tester/dist/vfxtool.exe", "./cmd/vfxtool")...); err != nil {
			return err
		}
		files, err := filepath.Glob("/home/tester/dist/*")
		if err != nil {
			return err
		}
		return e.publish(files...)

	case "install-test":
		// Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
		return installDebTest("vfox-plus", "/usr/lib/vfox-plus/vfox-plus")

	case "shell":
		if err := prepare(); err != nil {
			return err
		}
		shell := testerCommand("bash")
		shell.Dir = app
		shell.Stdin = os.Stdin
		shell.Stdout = os.Stdout
		shell.Stderr = os.Stderr
		return shell.Run()

	default:
		return fmt.Errorf("Unknown task: %s. Use vfxtool sandbox.", task)
	}
}

// asTester runs a command as tester in a directory, with HOME set like
// as_tester in task-lib.sh.
func asTester(directory string, args ...string) error {
	cmd := testerCommand(args...)
	cmd.Dir = directory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// asTesterOutput is asTester with stdout captured instead of attached.
func asTesterOutput(directory string, args ...string) (string, error) {
	cmd := testerCommand(args...)
	cmd.Dir = directory
	cmd.Stdin = os.Stdin
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

// copyRepository copies the shared resources, the license, the docs, the
// resources of the Android app and the given top-level directories of the
// repository to the working directory, without what the host has built in
// them. The extraction runs as tester, like the piped tar of copy_repository
// in task-lib.sh.
func (e env) copyRepository(directories ...string) error {
	if err := run("install", "-d", "-o", "tester", "-g", "tester", e.work); err != nil {
		return err
	}
	// The tests of the icon in shared/tooling compare its drawing with the
	// vector drawables of the Android app, so they need those resources. Only
	// they are copied: the rest of android/ holds Gradle outputs on a host
	// that has built the app.
	members := []string{"./shared", "./LICENSE", "./docs", "./android/app/src/main/res"}
	for _, directory := range directories {
		members = append(members, "./"+directory)
	}
	createArgs := append([]string{"-C", e.src, "--create"}, copyExcludes...)
	createArgs = append(createArgs, members...)
	create := exec.Command("tar", createArgs...)
	create.Stderr = os.Stderr
	extract := testerCommand("tar", "-C", e.work, "--extract")
	extract.Stderr = os.Stderr
	pipe, err := create.StdoutPipe()
	if err != nil {
		return err
	}
	extract.Stdin = pipe
	if err := extract.Start(); err != nil {
		return err
	}
	if err := create.Run(); err != nil {
		_ = extract.Wait()
		return err
	}
	return extract.Wait()
}

// publish copies results to /out for the user on the host.
func (e env) publish(paths ...string) error {
	if err := run("cp", append([]string{"-r"}, append(paths, e.out+"/")...)...); err != nil {
		return err
	}
	// Only set with a rootful Docker daemon; under rootless Docker, root in
	// the container already is the user on the host.
	if owner := os.Getenv("OUT_OWNER"); owner != "" {
		return run("chown", "-R", owner, e.out)
	}
	return nil
}

// startDisplay starts a virtual display and a session bus, and exports
// DISPLAY and DBUS_SESSION_BUS_ADDRESS for the applications started
// afterwards. screen is the -screen argument of Xvfb, e.g. 1280x800x24.
// The bus daemon runs as tester, so that the applications of the tasks,
// which run as tester, can connect to it.
func startDisplay(screen string) (*display.Display, error) {
	d, err := display.Start(screen, func() ([]byte, error) {
		return testerCommand("dbus-launch", "--sh-syntax").Output()
	})
	if err != nil {
		return nil, err
	}
	if err := os.Setenv("DISPLAY", d.Name); err != nil {
		d.Stop()
		return nil, err
	}
	if err := os.Setenv("DBUS_SESSION_BUS_ADDRESS", d.BusAddress); err != nil {
		d.Stop()
		return nil, err
	}
	return d, nil
}

// inDisplay runs a command as tester on the virtual display.
func inDisplay(args ...string) error {
	prefix := []string{"env", "DISPLAY=" + os.Getenv("DISPLAY"), "DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS")}
	return asTester(loadEnv().work, append(prefix, args...)...)
}

// inDisplayOutput is inDisplay with stdout captured instead of attached.
func inDisplayOutput(args ...string) (string, error) {
	prefix := []string{"env", "DISPLAY=" + os.Getenv("DISPLAY"), "DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS")}
	return asTesterOutput(loadEnv().work, append(prefix, args...)...)
}

// inDisplayStart starts a command as tester on the virtual display without
// waiting for it.
func inDisplayStart(args ...string) error {
	prefix := []string{"env", "HOME=/home/tester",
		"DISPLAY=" + os.Getenv("DISPLAY"), "DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS")}
	cmd := testerCommand(append(prefix, args...)...)
	cmd.Dir = loadEnv().work
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// defaultWallpaper is the default wallpaper of Ubuntu, shipped by the
// ubuntu-wallpapers package. The file name has stayed the same across
// releases; only the picture changes.
const defaultWallpaper = "/usr/share/backgrounds/warty-final-ubuntu.png"

// setWallpaper fills the root window with the default wallpaper of Ubuntu,
// instead of the black of a bare X server. feh paints it directly, without
// a desktop environment.
func setWallpaper() error {
	path, err := findWallpaper("/usr/share/backgrounds")
	if err != nil {
		return err
	}
	return inDisplay("feh", "--bg-fill", path)
}

// findWallpaper resolves the default wallpaper in dir.
func findWallpaper(dir string) (string, error) {
	path := filepath.Join(dir, filepath.Base(defaultWallpaper))
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	var names []string
	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				names = append(names, entry.Name())
			}
		}
	}
	return "", fmt.Errorf("the default wallpaper is not at %s (found: %s): install ubuntu-wallpapers",
		path, strings.Join(names, ", "))
}

// smokeTest starts the application on a virtual display and checks that it
// still runs 15 seconds later.
func smokeTest(executable string) error {
	return display.SmokeTest(executable, display.Options{
		LogPath:   "/tmp/vfox-plus.log",
		Prefix:    testerHead(),
		LaunchBus: dbusLauncher(),
	})
}

// testerHead is the command prefix that runs what follows as tester.
func testerHead() []string {
	if alreadyTester() {
		return []string{"env", "HOME=/home/tester"}
	}
	return []string{"runuser", "-u", "tester", "--", "env", "HOME=/home/tester"}
}

// dbusLauncher starts a session bus with the rights the applications need:
// as tester, like the settings daemon above.
func dbusLauncher() func() ([]byte, error) {
	return func() ([]byte, error) {
		return testerCommand("dbus-launch", "--sh-syntax").Output()
	}
}

// installDebTest runs in a clean Ubuntu container with the .deb mounted at
// /pkg: the declared dependencies must be all the application needs. It
// checks the shared libraries of the executables, which may be patterns
// that are expanded after the installation, the desktop integration, starts
// the application and removes the package again.
func installDebTest(pkg string, patterns ...string) error {
	if mirror := os.Getenv("UBUNTU_MIRROR"); mirror != "" {
		if err := run("sed", "-i",
			"s#http://archive.ubuntu.com/ubuntu/#"+mirror+"#g; s#http://security.ubuntu.com/ubuntu/#"+mirror+"#g",
			"/etc/apt/sources.list.d/ubuntu.sources"); err != nil {
			return err
		}
	}
	aptEnv := os.Environ()
	aptEnv = append(aptEnv, "DEBIAN_FRONTEND=noninteractive")
	apt := func(args ...string) error {
		cmd := exec.Command("apt-get", args...)
		cmd.Env = aptEnv
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	if err := apt("update", "-qq"); err != nil {
		return err
	}
	debs, err := filepath.Glob("/pkg/" + pkg + "_*.deb")
	if err != nil {
		return err
	}
	if len(debs) == 0 {
		return fmt.Errorf("no .deb for %s in /pkg", pkg)
	}
	installArgs := append([]string{"install", "-y", "-qq", "--no-install-recommends"}, debs...)
	if err := apt(installArgs...); err != nil {
		return err
	}
	// The patterns are expanded now that the package is installed.
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			matches = []string{pattern}
		}
		files = append(files, matches...)
	}
	ldd := exec.Command("ldd", files...)
	var lddOut bytes.Buffer
	ldd.Stdout = &lddOut
	ldd.Stderr = os.Stderr
	_ = ldd.Run()
	if strings.Contains(lddOut.String(), "not found") {
		fmt.Fprint(os.Stderr, lddOut.String())
		return fmt.Errorf("The dependencies of the package miss the libraries above.")
	}
	for _, path := range []string{"/usr/bin/vfox-plus"} {
		st, err := os.Stat(path)
		if err != nil || st.Mode()&0o111 == 0 {
			return fmt.Errorf("%s is not executable", path)
		}
	}
	for _, path := range []string{
		"/usr/share/applications/dev.vfox.VfoxPlus.desktop",
		"/usr/share/icons/hicolor/scalable/apps/dev.vfox.VfoxPlus.svg",
		"/usr/share/metainfo/dev.vfox.VfoxPlus.metainfo.xml",
	} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("%s is missing: %w", path, err)
		}
	}
	if err := apt("install", "-y", "-qq", "--no-install-recommends", "xvfb", "xauth", "dbus-x11"); err != nil {
		return err
	}
	// Like `|| true`: tester may already exist in the image.
	_ = run("useradd", "--create-home", "tester")
	if err := smokeTest("/usr/bin/vfox-plus"); err != nil {
		return err
	}
	if err := apt("remove", "-y", "-qq", pkg); err != nil {
		return err
	}
	if _, err := os.Stat("/usr/bin/vfox-plus"); !os.IsNotExist(err) {
		return fmt.Errorf("/usr/bin/vfox-plus still exists after removing %s", pkg)
	}
	fmt.Println("The package installs, starts and uninstalls cleanly.")
	return nil
}

// run executes a command with the standard streams attached.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

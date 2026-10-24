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

// Package build implements `vfxtool build`, a port of mygo/tools/build.sh.
//
// It builds the mygo application. Run `vfxtool prepare`
// first: the application embeds what it generates.
package build

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// config holds the parsed command line.
type config struct {
	e2e    bool
	output string
	root   string
}

// Run builds the mygo application.
//
// Usage: build [--e2e] [--root DIR] <output>
//
// With --e2e the build includes the end-to-end scenarios, which run in the
// application when VFOX_PLUS_E2E_SCENARIO is set; release builds must not have
// them.
func Run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	root, err := resolveRoot(cfg.root)
	if err != nil {
		return err
	}
	output, err := resolveOutput(cfg.output)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build",
		"-trimpath",
		"-buildmode=pie",
		"-tags", cfg.tags(),
		"-ldflags", "-s -w -X github.com/egoist/mygo.production=1",
		"-o", output,
		".",
	)
	cmd.Dir = root
	cmd.Env = withEnv(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build in %s: %w", root, err)
	}
	return nil
}

// parseArgs parses [--e2e] [--root DIR] <output>, keeping positional
// compatibility with build.sh [--e2e] <output>.
func parseArgs(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("vfxtool build", flag.ContinueOnError)
	fs.BoolVar(&cfg.e2e, "e2e", false, "include the end-to-end scenarios")
	fs.StringVar(&cfg.root, "root", "", "mygo module directory (default: located from the current directory)")
	// Keep the usage compatible with build.sh.
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: build [--e2e] [--root DIR] <output>\n")
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	rest := fs.Args()
	if len(rest) != 1 || rest[0] == "" {
		return config{}, errors.New("Usage: build [--e2e] <output>")
	}
	cfg.output = rest[0]
	return cfg, nil
}

// tags returns the -tags value: mygo_noinspector, plus e2e with --e2e.
func (c config) tags() string {
	if c.e2e {
		return "mygo_noinspector,e2e"
	}
	return "mygo_noinspector"
}

// resolveOutput creates the parent directory of output and returns its
// absolute path. The output is relative to where the command was started,
// not to the module, mirroring build.sh.
func resolveOutput(output string) (string, error) {
	if output == "" {
		return "", errors.New("Usage: build [--e2e] <output>")
	}
	dir := filepath.Dir(output)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return "", fmt.Errorf("create output directory %s: %w", dir, err)
		}
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("resolve output %s: %w", output, err)
	}
	return abs, nil
}

// resolveRoot returns the mygo module directory: flagRoot when given,
// otherwise located by walking up from the current directory, the vfxtool
// executable, or the source tree (see locateMygoRoot).
func resolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		abs, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", fmt.Errorf("resolve --root %s: %w", flagRoot, err)
		}
		if !isMygoRoot(abs) {
			return "", fmt.Errorf("--root %s holds no mygo module (no go.mod)", abs)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	if root, ok := locateMygoRoot(cwd); ok {
		return root, nil
	}
	if exe, err := os.Executable(); err == nil {
		if root, ok := locateMygoRoot(filepath.Dir(exe)); ok {
			return root, nil
		}
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		// This source file lives at shared/tooling/internal/build, so the
		// repository root is three levels up and mygo is below it.
		candidate := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "mygo")
		if abs, err := filepath.Abs(candidate); err == nil && isMygoRoot(abs) {
			return abs, nil
		}
	}
	return "", errors.New("cannot locate mygo module (no mygo/go.mod above the working directory): pass --root DIR")
}

// locateMygoRoot walks start and its parents, returning the first directory
// that is a mygo module root: either the directory itself holds go.mod for
// the mygo module, or its mygo child does (a repository checkout).
func locateMygoRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if isMygoRoot(dir) {
			return dir, true
		}
		if isMygoRoot(filepath.Join(dir, "mygo")) {
			return filepath.Join(dir, "mygo"), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// isMygoRoot reports whether dir is the mygo module root: it holds a go.mod
// that declares the mygo module.
func isMygoRoot(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module") {
			fields := strings.Fields(line)
			return len(fields) == 2 && fields[1] == "github.com/version-fox/vfox-plus/mygo"
		}
	}
	return false
}

// withEnv returns env with key=value set, replacing any existing key= entry.
func withEnv(env []string, keyValue string) []string {
	key := keyValue
	if i := strings.IndexByte(keyValue, '='); i >= 0 {
		key = keyValue[:i]
	}
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if kv == key || strings.HasPrefix(kv, prefix) {
			continue
		}
		out = append(out, kv)
	}
	return append(out, keyValue)
}

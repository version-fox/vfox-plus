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

// Package prepare implements `vfxtool prepare`, a port of
// mygo/tools/prepare.sh.
//
// It collects the inputs of the mygo application that are not committed in
// its module, into assets/, where the application embeds them:
//
//   - icons/: the SVGs of shared/icons;
//   - icons.json: the symbols of the icons, by meaning (shared/icons.json);
//   - i18n/*.arb: the catalogues of shared/i18n, as they are;
//   - licenses.txt: the licenses of what the application is built from.
package prepare

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/golicenses"
)

// config holds the parsed command line.
type config struct {
	assets string
	root   string
}

// Run collects the embedded assets of the mygo application.
//
// Usage: prepare [--assets DIR] [--root DIR]
func Run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	root, err := resolveRoot(cfg.root)
	if err != nil {
		return err
	}
	shared := filepath.Join(filepath.Dir(root), "shared")
	assets := resolveAssetsDir(cfg.assets, root)

	// rm -rf assets/icons assets/i18n; mkdir -p assets/icons assets/i18n.
	for _, name := range []string{"icons", "i18n"} {
		dir := filepath.Join(assets, name)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	if err := copyFiles(filepath.Join(shared, "icons"), "*.svg", filepath.Join(assets, "icons")); err != nil {
		return err
	}
	if err := copyFiles(shared, "icons.json", assets); err != nil {
		return err
	}
	if err := copyFiles(filepath.Join(shared, "i18n"), "*.arb", filepath.Join(assets, "i18n")); err != nil {
		return err
	}
	if err := writeLicenses(filepath.Join(root, "NOTICE.txt"), filepath.Join(shared, "icons", "LICENSE"), filepath.Join(assets, "licenses.txt")); err != nil {
		return err
	}

	// The licenses of the Go modules need them in the module cache.
	if os.Getenv("MYGO_SKIP_LICENSES") == "" {
		dl := exec.Command("go", "mod", "download")
		dl.Dir = root
		dl.Stdout = os.Stdout
		dl.Stderr = os.Stderr
		if err := dl.Run(); err != nil {
			return fmt.Errorf("go mod download in %s: %w", root, err)
		}
		if err := golicenses.Run([]string{
			"--module", root,
			"--output", filepath.Join(assets, "licenses.txt"),
			"--append",
		}); err != nil {
			return fmt.Errorf("collect Go module licenses: %w", err)
		}
	}

	fmt.Printf("Prepared %s\n", assets)
	return nil
}

// parseArgs parses [--assets DIR] [--root DIR].
func parseArgs(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("vfxtool prepare", flag.ContinueOnError)
	fs.StringVar(&cfg.assets, "assets", "", "output assets directory (default: <mygo>/assets)")
	fs.StringVar(&cfg.root, "root", "", "mygo module directory (default: located from the current directory)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: prepare [--assets DIR] [--root DIR]\n")
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if rest := fs.Args(); len(rest) != 0 {
		return config{}, errors.New("Usage: prepare [--assets DIR] [--root DIR]")
	}
	return cfg, nil
}

// resolveAssetsDir returns the --assets flag when given, else <root>/assets.
func resolveAssetsDir(flagVal, root string) string {
	if flagVal != "" {
		return flagVal
	}
	return filepath.Join(root, "assets")
}

// copyFiles copies the files of srcDir that match pattern to dstDir, and fails
// when there are none.
func copyFiles(srcDir, pattern, dstDir string) error {
	matches, err := filepath.Glob(filepath.Join(srcDir, pattern))
	if err != nil {
		return fmt.Errorf("list %s: %w", filepath.Join(srcDir, pattern), err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no %s in %s", pattern, srcDir)
	}
	for _, src := range matches {
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read %s: %w", src, err)
		}
		dst := filepath.Join(dstDir, filepath.Base(src))
		if err := os.WriteFile(dst, data, 0o666); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	return nil
}

// licensesHeader separates the notices of the ports from the license of the
// icons: two newlines, a rule, the heading, a rule.
var licensesHeader = "\n\n" + strings.Repeat("=", 80) + "\n" +
	"Material Symbols Rounded (the icons)\n" +
	strings.Repeat("=", 80) + "\n\n"

// writeLicenses builds licenses.txt from NOTICE.txt and the license of the
// icons (before --append adds the Go modules).
func writeLicenses(noticePath, iconsLicense, output string) error {
	notice, err := os.ReadFile(noticePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", noticePath, err)
	}
	license, err := os.ReadFile(iconsLicense)
	if err != nil {
		return fmt.Errorf("read %s: %w", iconsLicense, err)
	}
	var out []byte
	out = append(out, notice...)
	out = append(out, licensesHeader...)
	out = append(out, license...)
	if !strings.HasSuffix(string(license), "\n") {
		out = append(out, '\n')
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o777); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(output), err)
	}
	if err := os.WriteFile(output, out, 0o666); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	return nil
}

// resolveRoot returns the mygo module directory, like build.resolveRoot.
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
		candidate := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "mygo")
		if abs, err := filepath.Abs(candidate); err == nil && isMygoRoot(abs) {
			return abs, nil
		}
	}
	return "", errors.New("cannot locate mygo module (no mygo/go.mod above the working directory): pass --root DIR")
}

// locateMygoRoot walks start and its parents for the mygo module root.
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

// isMygoRoot reports whether dir declares the mygo module.
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

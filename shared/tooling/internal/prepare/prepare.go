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
// It generates the inputs of the mygo application that are not committed,
// into assets/, where the application embeds them:
//
//   - fonts/MaterialSymbolsRounded.ttf: the outlined icon font;
//   - fonts/MaterialSymbolsRoundedFilled.ttf: the filled icon font;
//   - icons.json: the code points of the icons, by meaning;
//   - i18n/*.arb: the catalogues of shared/i18n, as they are;
//   - licenses.txt: the licenses of what the application is built from.
//
// The iconfont, renamefont and golicenses steps are other vfxtool subcommands
// (other workers implement them); this package shells out to them through the
// vfxtool binary so the pieces compose like the shell did.
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
)

// config holds the parsed command line.
type config struct {
	symbols string
	assets  string
	root    string
}

// Run generates the embedded assets of the mygo application.
//
// Usage: prepare [--symbols DIR] [--assets DIR] [--root DIR]
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
	symbols := resolveSymbolsDir(cfg.symbols, os.Getenv)

	if !hasValidSymbols(symbols) {
		if err := runVfxtool("iconfont", "download", symbols); err != nil {
			return fmt.Errorf("symbols directory %q holds no Material Symbols (no LICENSE/font files): download them with `vfxtool iconfont download %q`: %w", symbols, symbols, err)
		}
		if !hasValidSymbols(symbols) {
			return fmt.Errorf("symbols directory %q holds no Material Symbols (no LICENSE/font files) even after download: run `vfxtool iconfont download %q`", symbols, symbols)
		}
	}

	// rm -rf assets/i18n; mkdir -p assets/fonts assets/i18n.
	if err := os.RemoveAll(filepath.Join(assets, "i18n")); err != nil {
		return fmt.Errorf("remove %s: %w", filepath.Join(assets, "i18n"), err)
	}
	for _, dir := range []string{filepath.Join(assets, "fonts"), filepath.Join(assets, "i18n")} {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	// The user interface draws every icon at weight 400, grade 0 and optical
	// size 24, outlined or filled.
	if err := runVfxtool("iconfont", "subset",
		"--fonts", symbols,
		"--output", filepath.Join(assets, "fonts", "MaterialSymbolsRounded.ttf"),
		"--json", filepath.Join(assets, "icons.json"),
		"--pin", "wght=400", "--pin", "GRAD=0", "--pin", "opsz=24", "--pin", "FILL=0",
	); err != nil {
		return fmt.Errorf("subset outlined icon font: %w", err)
	}
	filled := filepath.Join(assets, "fonts", "MaterialSymbolsRoundedFilled.ttf")
	if err := runVfxtool("iconfont", "subset",
		"--fonts", symbols,
		"--output", filled,
		"--pin", "wght=400", "--pin", "GRAD=0", "--pin", "opsz=24", "--pin", "FILL=1",
	); err != nil {
		return fmt.Errorf("subset filled icon font: %w", err)
	}
	if err := runVfxtool("renamefont", filled, "Material Symbols Rounded Filled"); err != nil {
		return fmt.Errorf("rename filled icon font: %w", err)
	}

	if err := copyCatalogues(filepath.Join(shared, "i18n"), filepath.Join(assets, "i18n")); err != nil {
		return err
	}
	if err := writeLicenses(filepath.Join(root, "NOTICE.txt"), filepath.Join(symbols, "LICENSE"), filepath.Join(assets, "licenses.txt")); err != nil {
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
		if err := runVfxtool("golicenses",
			"--module", root,
			"--output", filepath.Join(assets, "licenses.txt"),
			"--append",
		); err != nil {
			return fmt.Errorf("collect Go module licenses: %w", err)
		}
	}

	fmt.Printf("Prepared %s\n", assets)
	return nil
}

// parseArgs parses [--symbols DIR] [--assets DIR] [--root DIR].
func parseArgs(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("vfxtool prepare", flag.ContinueOnError)
	fs.StringVar(&cfg.symbols, "symbols", "", "directory of the Material Symbols variable font (default: $VFOX_PLUS_SYMBOLS or the cache)")
	fs.StringVar(&cfg.assets, "assets", "", "output assets directory (default: <mygo>/assets)")
	fs.StringVar(&cfg.root, "root", "", "mygo module directory (default: located from the current directory)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: prepare [--symbols DIR] [--assets DIR] [--root DIR]\n")
	}
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if rest := fs.Args(); len(rest) != 0 {
		return config{}, errors.New("Usage: prepare [--symbols DIR] [--assets DIR] [--root DIR]")
	}
	return cfg, nil
}

// resolveSymbolsDir returns the --symbols flag when given, else
// $VFOX_PLUS_SYMBOLS, else the cache directory.
func resolveSymbolsDir(flagVal string, lookup func(string) string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := lookup("VFOX_PLUS_SYMBOLS"); v != "" {
		return v
	}
	return defaultSymbolsDir(lookup)
}

// defaultSymbolsDir returns
// $XDG_CACHE_HOME/vfox-plus/material-symbols, or
// $HOME/.cache/vfox-plus/material-symbols, mirroring prepare.sh.
func defaultSymbolsDir(lookup func(string) string) string {
	if xdg := lookup("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "vfox-plus", "material-symbols")
	}
	return filepath.Join(lookup("HOME"), ".cache", "vfox-plus", "material-symbols")
}

// resolveAssetsDir returns the --assets flag when given, else <root>/assets.
func resolveAssetsDir(flagVal, root string) string {
	if flagVal != "" {
		return flagVal
	}
	return filepath.Join(root, "assets")
}

// hasValidSymbols reports whether dir holds a downloaded Material Symbols
// tree: the LICENSE plus the variable font and its codepoints. The exact
// font file name is pinned in internal/iconfont; accept any .ttf and
// any .codepoints file so the check does not rot with the pin.
func hasValidSymbols(dir string) bool {
	if fi, err := os.Stat(filepath.Join(dir, "LICENSE")); err != nil || fi.IsDir() {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	var ttf, codepoints bool
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch {
		case strings.HasSuffix(e.Name(), ".ttf"):
			ttf = true
		case strings.HasSuffix(e.Name(), ".codepoints"):
			codepoints = true
		}
	}
	return ttf && codepoints
}

// copyCatalogues copies shared/i18n/*.arb to the assets i18n directory.
func copyCatalogues(srcDir, dstDir string) error {
	matches, err := filepath.Glob(filepath.Join(srcDir, "*.arb"))
	if err != nil {
		return fmt.Errorf("list %s: %w", filepath.Join(srcDir, "*.arb"), err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no .arb catalogues in %s", srcDir)
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

// licensesSeparator mirrors the shell: NOTICE.txt, two newlines, a rule, the
// Material Symbols heading, a rule, the symbols LICENSE, one newline.
var licensesHeader = "\n\n" + strings.Repeat("=", 80) + "\n" +
	"Material Symbols Rounded (the icon fonts)\n" +
	strings.Repeat("=", 80) + "\n\n"

// writeLicenses builds licenses.txt from NOTICE.txt and the symbols LICENSE,
// exactly as prepare.sh does (before --append adds the Go modules).
func writeLicenses(noticePath, symbolsLicense, output string) error {
	notice, err := os.ReadFile(noticePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", noticePath, err)
	}
	license, err := os.ReadFile(symbolsLicense)
	if err != nil {
		return fmt.Errorf("read %s: %w", symbolsLicense, err)
	}
	var out []byte
	out = append(out, notice...)
	out = append(out, licensesHeader...)
	out = append(out, license...)
	out = append(out, '\n')
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

// runVfxtool runs `vfxtool <sub> <args...>` using the current binary, or
// `go run ./cmd/vfxtool` from the tooling module when VFXTOOL_DEV=1 (for
// development before vfxtool is installed).
func runVfxtool(sub string, args ...string) error {
	bin, prefix, dir, err := vfxtoolCommand()
	if err != nil {
		return err
	}
	full := append(append([]string{}, prefix...), sub)
	full = append(full, args...)
	cmd := exec.Command(bin, full...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vfxtool %s: %w", sub, err)
	}
	return nil
}

// vfxtoolCommand locates the vfxtool binary to shell out to.
func vfxtoolCommand() (bin string, prefix []string, dir string, err error) {
	if os.Getenv("VFXTOOL_DEV") == "1" {
		root, ok := locateToolingRoot()
		if !ok {
			return "", nil, "", errors.New("VFXTOOL_DEV=1 but cannot locate the tooling module (no shared/tooling/go.mod above the working directory)")
		}
		return "go", []string{"run", "./cmd/vfxtool"}, root, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", nil, "", fmt.Errorf("locate vfxtool binary: %w", err)
	}
	return exe, nil, "", nil
}

// locateToolingRoot walks up from the working directory, the executable, or
// this source file for the shared/tooling module root.
func locateToolingRoot() (string, bool) {
	var starts []string
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		starts = append(starts, filepath.Dir(file))
	}
	for _, start := range starts {
		if root, ok := searchToolingRoot(start); ok {
			return root, true
		}
	}
	return "", false
}

func searchToolingRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if isToolingRoot(dir) {
			return dir, true
		}
		if isToolingRoot(filepath.Join(dir, "shared", "tooling")) {
			return filepath.Join(dir, "shared", "tooling"), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// isToolingRoot reports whether dir declares the tooling module.
func isToolingRoot(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module") {
			fields := strings.Fields(line)
			return len(fields) == 2 && fields[1] == "github.com/version-fox/vfox-plus/shared/tooling"
		}
	}
	return false
}

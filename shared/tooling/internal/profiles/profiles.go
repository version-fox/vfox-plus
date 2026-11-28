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

// Package profiles implements `vfxtool profile`, for the plugin profiles of
// shared/profile (docs/en/dev/profiles.md):
//
//   - lint checks profile files with the reader of the application
//     (mygo/core/profile/cmd/profilelint), by default those of the repository,
//     or the one a plugin ships against the plugin;
//   - index writes shared/profile/index.json, the list of the online directory.
package profiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/prepare"
)

// usage is the command line.
const usage = "Usage: profile lint [--root DIR] [--plugin DIR] [--strict] [FILE...] | profile index [--root DIR]"

// indexSchema is the format of index.json, as the application reads it.
const indexSchema = 2

// Run runs `vfxtool profile`.
func Run(args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "lint":
		return lint(args[1:])
	case "index":
		return index(args[1:])
	default:
		return errors.New(usage)
	}
}

// directory is shared/profile of the repository whose mygo module is root.
func directory(root string) string {
	return filepath.Join(filepath.Dir(root), "shared", "profile")
}

// lint checks profiles with the reader of the application, which runs from the
// mygo module.
func lint(args []string) error {
	fs := flag.NewFlagSet("vfxtool profile lint", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "mygo module directory (default: located from the current directory)")
	plugin := fs.String("plugin", "", "the directory of the plugin that the profile describes; without a FILE, its lib/vfox-plus.json is checked")
	strict := fs.Bool("strict", false, "fail for warnings too")
	fs.Usage = func() { fmt.Fprintln(fs.Output(), usage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := prepare.ResolveRoot(*rootFlag)
	if err != nil {
		return err
	}
	command, err := lintArguments(root, *plugin, *strict, fs.Args())
	if err != nil {
		return err
	}
	run := exec.Command("go", command...) //nolint:gosec // The arguments are file names for the linter.
	run.Dir = root
	run.Stdout, run.Stderr = os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		return fmt.Errorf("profile lint: %w", err)
	}
	return nil
}

// lintArguments is the command line of go that lints: the files, with their
// paths made absolute because the linter runs in the mygo module, or the
// profiles of the repository when there is neither a file nor a plugin.
func lintArguments(root, plugin string, strict bool, files []string) ([]string, error) {
	command := []string{"run", "./core/profile/cmd/profilelint"}
	if strict {
		command = append(command, "--strict")
	}
	if plugin != "" {
		absolute, err := filepath.Abs(plugin)
		if err != nil {
			return nil, err
		}
		command = append(command, "--plugin", absolute)
	}
	if len(files) == 0 && plugin == "" {
		command = append(command, "--directory")
		found, err := profileFiles(root)
		if err != nil {
			return nil, err
		}
		files = found
	}
	for _, file := range files {
		absolute, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		command = append(command, absolute)
	}
	return command, nil
}

// profileFiles are the profiles of the repository.
func profileFiles(root string) ([]string, error) {
	pattern := filepath.Join(directory(root), "plugins", "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no profile matches %s", pattern)
	}
	return files, nil
}

// index writes shared/profile/index.json: the SHA-256 of every profile.
func index(args []string) error {
	fs := flag.NewFlagSet("vfxtool profile index", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "mygo module directory (default: located from the current directory)")
	fs.Usage = func() { fmt.Fprintln(fs.Output(), usage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New(usage)
	}
	root, err := prepare.ResolveRoot(*rootFlag)
	if err != nil {
		return err
	}
	content, err := buildIndex(root)
	if err != nil {
		return err
	}
	output := filepath.Join(directory(root), "index.json")
	if err := os.WriteFile(output, content, 0o666); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	fmt.Printf("Wrote %s\n", output)
	return nil
}

// buildIndex is the content of index.json for the profiles of the repository.
func buildIndex(root string) ([]byte, error) {
	files, err := profileFiles(root)
	if err != nil {
		return nil, err
	}
	plugins := make(map[string]string, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		sum := sha256.Sum256(content)
		plugins[strings.TrimSuffix(filepath.Base(file), ".json")] = hex.EncodeToString(sum[:])
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(struct {
		Schema  int               `json:"schema"`
		Plugins map[string]string `json:"plugins"`
	}{indexSchema, plugins}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

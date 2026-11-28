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

// Command profilelint checks vfox plus profile files (docs/en/dev/profiles.md),
// for `vfxtool profile lint`:
//
//	profilelint [--plugin DIR] [--directory] [--strict] [FILE...]
//
// With --plugin, the profile is checked against the plugin in DIR, whose
// lib/vfox-plus.json is checked when no file is given. With --directory, the
// files are those of the directory of profiles, which are named after their
// plugin. It fails for an error, and with --strict for a warning too.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "profilelint: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, output io.Writer) error {
	flags := flag.NewFlagSet("profilelint", flag.ContinueOnError)
	plugin := flags.String("plugin", "", "the directory of the plugin that the profile describes")
	directory := flags.Bool("directory", false, "the files are those of the directory of profiles, named after their plugin")
	strict := flags.Bool("strict", false, "fail for warnings too")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	files := flags.Args()
	if len(files) == 0 && *plugin != "" {
		files = []string{filepath.Join(*plugin, filepath.FromSlash(profile.PluginProfile))}
	}
	if len(files) == 0 {
		return errors.New("usage: profilelint [--plugin DIR] [--directory] [--strict] [FILE...]")
	}
	failed := 0
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // The files to check are the arguments.
		if err != nil {
			return err
		}
		var options profile.LintOptions
		if *directory {
			options.Name = strings.TrimSuffix(filepath.Base(file), ".json")
		}
		if *plugin != "" {
			options.Plugin = os.DirFS(*plugin)
		}
		problems := profile.Lint(content, options)
		bad := false
		for _, problem := range problems {
			if _, err := fmt.Fprintf(output, "%s: %s\n", file, problem); err != nil {
				return err
			}
			bad = bad || !problem.Warning || *strict
		}
		if bad {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d files have problems", failed, len(files))
	}
	_, err := fmt.Fprintf(output, "%d files checked\n", len(files))
	return err
}

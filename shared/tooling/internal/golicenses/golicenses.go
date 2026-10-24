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

// Package golicenses checks the licenses of the Go modules that an
// implementation compiles into its executable, and of the Go standard
// library, and collects their texts into one file, which the application
// shows and the packages install.
//
// It replaces shared/tools/go_licenses.py. Run `go mod download` first.
package golicenses

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-licenses/v2/licenses"
)

// licenseFile matches the names of files that may hold a license text.
var licenseFile = regexp.MustCompile(`(?i)^(LICEN[CS]E|COPYING|NOTICE)`)

// shippable lists the license names, as reported by the go-licenses
// classifier, under which a module may be shipped with an Apache-2.0 work.
// It is the same set that the previous hand-rolled recogniser accepted.
var shippable = map[string]bool{
	"Apache-2.0":   true,
	"MIT":          true,
	"BSD-3-Clause": true,
	"BSD-2-Clause": true,
	"ISC":          true,
}

// newClassifier builds the go-licenses classifier. It is a variable so that
// tests can substitute a stub.
var newClassifier = licenses.NewClassifier

// classifyFile names the shippable licenses that cover path, using the
// go-licenses classifier. Names the classifier reports that are not
// shippable, such as copyleft licenses, are dropped; a file covered only by
// those, or by nothing the classifier recognises, yields no names.
func classifyFile(classifier licenses.Classifier, path string) ([]string, error) {
	found, err := classifier.Identify(path)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, license := range found {
		if shippable[license.Name] {
			names[license.Name] = true
		}
	}
	var out []string
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// goModule is the Module section of one `go list -json` package.
type goModule struct {
	Path    string
	Version string
	Dir     string
	Main    bool
}

// listModules returns the non-main modules of the packages that the main
// package of dir builds with under tags, and whether it uses the standard
// library.
func listModules(dir, tags string) ([]goModule, bool, error) {
	args := []string{"list", "-deps", "-json"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, false, fmt.Errorf("go list: %w", err)
	}
	var pkg struct {
		Standard bool
		Module   *goModule
	}
	found := map[string]goModule{}
	standard := false
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		// Reset pkg so that a missing Module stays nil.
		pkg.Standard = false
		pkg.Module = nil
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, false, fmt.Errorf("go list: %w", err)
		}
		if pkg.Standard {
			standard = true
		}
		if pkg.Module != nil && !pkg.Module.Main {
			found[pkg.Module.Path] = *pkg.Module
		}
	}
	modules := make([]goModule, 0, len(found))
	for _, module := range found {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	return modules, standard, nil
}

// goEnv runs `go env NAME` and returns its stripped output.
func goEnv(name string) (string, error) {
	out, err := exec.Command("go", "env", name).Output()
	if err != nil {
		return "", fmt.Errorf("go env %s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// entry is one section of the output: the license files of one module or of
// the standard library.
type entry struct {
	title string
	files []string
}

// collectEntries lists the license files of the standard library, when used,
// and of every dependency module.
func collectEntries(dir, tags string) ([]entry, error) {
	modules, standard, err := listModules(dir, tags)
	if err != nil {
		return nil, err
	}
	var entries []entry
	if standard {
		root, err := goEnv("GOROOT")
		if err != nil {
			return nil, err
		}
		version, err := goEnv("GOVERSION")
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry{
			title: fmt.Sprintf("The Go standard library and runtime, %s", version),
			files: []string{filepath.Join(root, "LICENSE")},
		})
	}
	for _, module := range modules {
		// FindCandidates lists the files of the module directory that may
		// hold a license text; keep the same top-level set as before.
		candidates, err := licenses.FindCandidates(module.Dir, module.Dir)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, candidate := range candidates {
			if licenseFile.MatchString(filepath.Base(candidate)) {
				files = append(files, candidate)
			}
		}
		entries = append(entries, entry{
			title: fmt.Sprintf("%s@%s", module.Path, module.Version),
			files: files,
		})
	}
	return entries, nil
}

// separator joins the sections of the output file.
const separator = "\n\n" + "------------------------------------------------------------------------" + "\n\n"

// renderSections classifies the license of every entry with the go-licenses
// classifier and renders the sections, returning the errors for entries with
// no shippable license.
func renderSections(entries []entry) ([]string, []string, error) {
	classifier, err := newClassifier()
	if err != nil {
		return nil, nil, err
	}
	var errs, sections []string
	for _, e := range entries {
		licenses := map[string]bool{}
		var texts []string
		for _, file := range e.files {
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, nil, err
			}
			texts = append(texts, strings.TrimSpace(string(data)))
			names, err := classifyFile(classifier, file)
			if err != nil {
				return nil, nil, err
			}
			for _, name := range names {
				licenses[name] = true
			}
		}
		if len(e.files) == 0 || len(licenses) == 0 {
			errs = append(errs, fmt.Sprintf("%s: no license file with a license compatible with the Apache License 2.0", e.title))
			continue
		}
		names := make([]string, 0, len(licenses))
		for name := range licenses {
			names = append(names, name)
		}
		sort.Strings(names)
		sections = append(sections, fmt.Sprintf("%s (%s)\n\n%s", e.title, strings.Join(names, " AND "), strings.Join(texts, "\n\n")))
	}
	return sections, errs, nil
}

// Run collects the dependency licenses of the module in dir and writes them
// to the output file. It accepts --module DIR --tags TAGS --output FILE
// --append.
func Run(args []string) error {
	fs := flag.NewFlagSet("golicenses", flag.ContinueOnError)
	moduleDir := fs.String("module", "", "the directory of the main package")
	tags := fs.String("tags", "", "the build tags")
	output := fs.String("output", "", "the output file")
	appendMode := fs.Bool("append", false, "add to the end of the output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *moduleDir == "" || *output == "" {
		return fmt.Errorf("usage: golicenses --module DIR [--tags TAGS] --output FILE [--append]")
	}

	entries, err := collectEntries(*moduleDir, *tags)
	if err != nil {
		return err
	}
	sections, errs, err := renderSections(entries)
	if err != nil {
		return err
	}
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Modules with licenses that cannot be shipped:\n  %s\n", strings.Join(errs, "\n  "))
		return fmt.Errorf("%d module(s) with licenses that cannot be shipped", len(errs))
	}

	rendered := strings.Join(sections, separator) + "\n"
	if *appendMode {
		if prev, err := os.ReadFile(*output); err == nil {
			rendered = strings.TrimRight(string(prev), "\n") + separator + rendered
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o777); err != nil {
		return err
	}
	if err := os.WriteFile(*output, []byte(rendered), 0o666); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%d licenses of Go modules and the standard library)\n", *output, len(sections))
	return nil
}

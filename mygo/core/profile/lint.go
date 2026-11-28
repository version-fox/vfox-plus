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

// What `vfox-plus profile lint` checks: what the application cannot read, and
// what would surprise the author of a profile.

package profile

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// LintOptions say which file Lint checks.
type LintOptions struct {
	// Name is the plugin of a file of the directory, which its name gives; it is
	// empty for the file that a plugin ships.
	Name string
	// Plugin, when set, are the files of the plugin that the profile
	// describes: its code has to name the variables.
	Plugin fs.FS
}

// lintPlatforms are the platforms that Lint looks at.
var lintPlatforms = func() []Platform {
	var platforms []Platform
	for _, system := range []string{"linux", "darwin", "windows", "freebsd"} {
		for _, arch := range []string{"amd64", "arm64", "386", "arm", "loong64", "riscv64"} {
			platforms = append(platforms, Platform{OS: system, Arch: arch})
		}
	}
	return platforms
}()

// Lint checks a profile file: everything that ParseDocument cannot read is a
// problem, and so is what no platform or the plugin would use.
func Lint(content []byte, options LintOptions) []Problem {
	d, problems, err := ParseDocument(content)
	if err != nil {
		return []Problem{{Message: err.Error()}}
	}
	report := func(warning bool, path, format string, arguments ...any) {
		problems = append(problems, Problem{Path: path, Message: fmt.Sprintf(format, arguments...), Warning: warning})
	}
	if options.Name != "" && d.Plugin != options.Name {
		report(false, "plugin", "the file of %s names %q", options.Name, d.Plugin)
	}
	if d.sections == 0 {
		report(true, "", "the file carries no section")
	}
	duplicates(d.shorthands, func(s Shorthand) string { return s.Name }, func(i int, name string) {
		report(false, "shorthands", "%q is listed twice for one platform", name)
	})
	duplicates(d.variables, func(v Variable) string { return v.Key }, func(i int, key string) {
		report(false, "variables", "%q is listed twice for one platform", key)
	})
	duplicates(d.facets, func(f Facet) string { return f.ID }, func(i int, id string) {
		report(false, "facets", "%q is listed twice for one platform", id)
	})
	for i, variant := range d.search {
		if !slices.ContainsFunc(lintPlatforms, func(platform Platform) bool {
			first := slices.IndexFunc(d.search, func(v conditional[*Search]) bool { return v.when.matches(platform) })
			return first == i
		}) {
			report(true, fmt.Sprintf("search[%d]", i), "no platform uses this variant: an earlier one is for the same platforms")
		}
		translated(variant.value.Label, fmt.Sprintf("search[%d].label", i), report)
	}
	for i, s := range d.shorthands {
		translated(s.value.Meaning, fmt.Sprintf("shorthands[%d].meaning", i), report)
	}
	for i, v := range d.variables {
		// A label that the file leaves to the name of the variable is missing, not
		// untranslated.
		if v.value.Label != (Text{EN: humanizeEnvKey(v.value.Key)}) {
			translated(v.value.Label, fmt.Sprintf("variables[%d].label", i), report)
		}
		translated(v.value.Help, fmt.Sprintf("variables[%d].help", i), report)
	}
	for i, f := range d.facets {
		translated(f.value.Label, fmt.Sprintf("facets[%d].label", i), report)
	}
	translated(d.guide, "guide", report)
	for i, tip := range d.tips {
		translated(tip.value, fmt.Sprintf("tips[%d]", i), report)
	}
	if options.Plugin != nil {
		lintPlugin(d, options.Plugin, report)
	}
	return problems
}

// translated warns about a sentence without a Chinese text. A text that is a
// name of a product, which both languages share, is fine.
func translated(text Text, path string, report func(bool, string, string, ...any)) {
	if text.EN != "" && text.ZH == "" && strings.ContainsRune(strings.TrimSpace(text.EN), ' ') {
		report(true, path, "has no Chinese text")
	}
}

// duplicates reports the parts with the same key that are for the same
// platform, of which only the first counts.
func duplicates[T any](parts []conditional[T], key func(T) string, report func(int, string)) {
	reported := map[string]bool{}
	for _, platform := range lintPlatforms {
		seen := map[string]bool{}
		for i, part := range parts {
			if !part.when.matches(platform) {
				continue
			}
			id := key(part.value)
			if seen[id] && !reported[id] {
				reported[id] = true
				report(i, id)
			}
			seen[id] = true
		}
	}
}

// lintPlugin checks the profile against the files of the plugin.
func lintPlugin(d Document, files fs.FS, report func(bool, string, string, ...any)) {
	lua := false
	var code [][]byte
	_ = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !isCode(name) {
			if err == nil && entry.IsDir() && name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if content, err := readFSFile(files, name, maxLuaFile); err == nil {
			code = append(code, content)
			lua = lua || strings.HasSuffix(strings.ToLower(name), ".lua")
		}
		return nil
	})
	if !lua {
		report(false, "", "the plugin has no Lua file")
		return
	}
	for i, v := range d.variables {
		// A variable is read by the Lua files, or by a script that they run.
		word := regexp.MustCompile(`\b` + regexp.QuoteMeta(v.value.Key) + `\b`)
		if !slices.ContainsFunc(code, word.Match) {
			report(false, fmt.Sprintf("variables[%d].key", i), "no file of the plugin names %q", v.value.Key)
		}
	}
	plugin := ReadPlugin(files)
	if d.sections.Has(SectionLegacyFiles) {
		for _, name := range plugin.LegacyFiles {
			if !slices.Contains(d.legacyFiles, name) {
				report(true, "legacyFiles", "metadata.lua names %q too", name)
			}
		}
	}
}

// isCode reports whether a file of a plugin may read the environment: not its
// profile, its documents or its license.
func isCode(name string) bool {
	base := strings.ToLower(path.Base(name))
	return name != PluginProfile && !strings.HasSuffix(base, ".md") && !strings.HasSuffix(base, ".json") &&
		!strings.HasPrefix(base, "license")
}

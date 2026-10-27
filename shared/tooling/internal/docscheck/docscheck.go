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

// Package docscheck verifies the documentation language mapping described in
// docs/en/dev/contributing.md: every Markdown document has an English
// canonical version and a Simplified Chinese translation. Documents live
// under docs/, first by language (en, zh-CN) then by audience (user, dev);
// the repository root and the source directories keep only thin entry points
// (README.md, CONTRIBUTING.md) that link into docs/.
package docscheck

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Run checks the mapping. args are the command line without the program name:
//
//	docs-check [--root DIR]
//
// It reports every Markdown document without a counterpart and every document
// without a link to its counterpart.
func Run(args []string) error {
	root := ""
	rest := args
	for len(rest) >= 2 && rest[0] == "--root" {
		root = rest[1]
		rest = rest[2:]
	}
	if len(rest) != 0 {
		return fmt.Errorf("Usage: docs-check [--root DIR]")
	}
	if root == "" {
		var err error
		root, err = findRepoRoot()
		if err != nil {
			return err
		}
	}
	files, err := markdownFiles(root)
	if err != nil {
		return err
	}
	var problems []string
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		counterpart := counterpartPath(root, rel)
		if counterpart == "" {
			continue
		}
		if _, err := os.Stat(counterpart); err != nil {
			problems = append(problems, fmt.Sprintf("%s has no counterpart %s", rel, relPath(root, counterpart)))
			continue
		}
		content, err := os.ReadFile(file) //nolint:gosec // The repository's own docs.
		if err != nil {
			return err
		}
		want := filepath.Base(counterpart)
		if !strings.Contains(string(content), want) {
			problems = append(problems, fmt.Sprintf("%s does not link to %s", rel, want))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("documentation language problems:\n  %s", strings.Join(problems, "\n  "))
	}
	fmt.Printf("docs-check: %d documents, all with counterparts\n", len(files))
	return nil
}

// counterpartPath returns the absolute path of the counterpart of rel (relative
// to root), or "" when rel is not a translated document.
func counterpartPath(root, rel string) string {
	rel = filepath.ToSlash(rel)
	if strings.HasSuffix(rel, ".zh-CN.md") {
		english := strings.TrimSuffix(rel, ".zh-CN.md") + ".md"
		return filepath.Join(root, filepath.FromSlash(english))
	}
	if strings.HasPrefix(rel, "docs/zh-CN/") {
		english := "docs/en/" + strings.TrimPrefix(rel, "docs/zh-CN/")
		return filepath.Join(root, filepath.FromSlash(english))
	}
	if strings.HasPrefix(rel, "docs/en/") {
		chinese := "docs/zh-CN/" + strings.TrimPrefix(rel, "docs/en/")
		return filepath.Join(root, filepath.FromSlash(chinese))
	}
	dir, base := pathDirBase(rel)
	if strings.HasSuffix(base, ".md") && (base == "README.md" || base == "CONTRIBUTING.md") {
		_ = dir
		stem := strings.TrimSuffix(base, ".md")
		return filepath.Join(root, filepath.FromSlash(dir+"/"+stem+".zh-CN.md"))
	}
	return ""
}

func pathDirBase(p string) (string, string) {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i], p[i+1:]
	}
	return ".", p
}

func relPath(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return rel
}

// markdownFiles lists the Markdown documents the mapping covers, skipping
// build outputs and vendored directories.
func markdownFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			switch name {
			case ".git", ".idea", "dist", "build", "node_modules", "screenshots":
				return filepath.SkipDir
			}
			if name == "assets" && strings.Contains(path, "mygo") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !covered(rel) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// covered reports whether rel is a document the language mapping covers.
func covered(rel string) bool {
	if strings.HasSuffix(rel, ".zh-CN.md") {
		return true
	}
	if strings.HasPrefix(rel, "docs/en/") || strings.HasPrefix(rel, "docs/zh-CN/") {
		return strings.HasSuffix(rel, ".md")
	}
	if strings.HasPrefix(rel, "docs/") {
		return false
	}
	base := rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		base = rel[i+1:]
	}
	return base == "README.md" || base == "CONTRIBUTING.md"
}

// findRepoRoot locates the repository root by walking up to a directory with
// README.md and docs/.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "docs")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("cannot locate repository root: pass --root DIR")
}

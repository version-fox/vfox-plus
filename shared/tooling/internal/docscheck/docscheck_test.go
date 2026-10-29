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

package docscheck

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestUnclosedLinks(t *testing.T) {
	cases := map[string]struct {
		content string
		want    []int
	}{
		"a link with its bracket":       {"See [the guide](guide.md) and [`x`](../x.md#y).\n", nil},
		"a code label lacks it":         {"See [`shared/i18n`(../../../shared/i18n) here.\n", []int{1}},
		"a text label lacks it":         {"ok\n\nSee [docs/i18n.md(i18n.md), and more.\n", []int{3}},
		"an anchor lacks it":            {"See [mygo README(implementation-mygo.md#memory).\n", []int{1}},
		"a URL lacks it":                {"[vfox(https://vfox.dev/) is a version manager.\n", []int{1}},
		"a table cell lacks it":         {"| [`base`(../base.Dockerfile) | The base image |\n", []int{1}},
		"a fenced block":                {"```go\nx := a[i](b.c)\n```\n", nil},
		"a fence is closed by its kind": {"````md\n```\n[x(y.md)\n````\n\n[z(w.md)\n", []int{6}},
		"a tilde fence":                 {"~~~\n[x(y.md)\n~~~\n", nil},
		"a code span":                   {"Write `[x(y.md)` to see it.\n", nil},
		"a parenthesis in a label":      {"A [note (see below)] and (x.md) apart.\n", nil},
		"no destination":                {"The array [a](b) and f(x) are fine.\n", nil},
		"an autolink":                   {"<https://example.com/a(b)> and [c](d.md).\n", nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := unclosedLinks(c.content); !slices.Equal(got, c.want) {
				t.Errorf("unclosedLinks(%q) = %v, want %v", c.content, got, c.want)
			}
		})
	}
}

// write creates a file of a repository under test.
func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	repository := func(t *testing.T, english string) string {
		t.Helper()
		root := t.TempDir()
		write(t, root, "README.md", "# R\n\n[简体中文](README.zh-CN.md)\n")
		write(t, root, "README.zh-CN.md", "# R\n\n[English](README.md)\n")
		write(t, root, "docs/en/dev/a.md", english)
		write(t, root, "docs/zh-CN/dev/a.md", "[English](../../en/dev/a.md)\n")
		return root
	}
	t.Run("a repository in order", func(t *testing.T) {
		root := repository(t, "[简体中文](../../zh-CN/dev/a.md) and [a](b.md)\n")
		if err := Run([]string{"--root", root}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("names the document and the line of an unclosed link", func(t *testing.T) {
		root := repository(t, "[简体中文](../../zh-CN/dev/a.md)\n\nSee [`b`(b.md).\n")
		err := Run([]string{"--root", root})
		if err == nil || !strings.Contains(err.Error(), `docs/en/dev/a.md:3 has a link without its "]"`) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a document without a counterpart", func(t *testing.T) {
		root := repository(t, "[简体中文](../../zh-CN/dev/a.md)\n")
		write(t, root, "docs/en/dev/lonely.md", "x\n")
		err := Run([]string{"--root", root})
		if err == nil || !strings.Contains(err.Error(), "docs/en/dev/lonely.md has no counterpart") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a document that does not link to its counterpart", func(t *testing.T) {
		root := repository(t, "no link\n")
		err := Run([]string{"--root", root})
		if err == nil || !strings.Contains(err.Error(), "does not link to a.md") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("usage", func(t *testing.T) {
		if err := Run([]string{"stray"}); err == nil {
			t.Fatal("want an error")
		}
	})
}

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

package golicenses

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-licenses/v2/licenses"
)

// stubClassifier stands in for the go-licenses classifier. It reports the
// names listed for each path, so the rendering below is fully deterministic.
type stubClassifier struct {
	names map[string][]string
}

func (s stubClassifier) Identify(path string) ([]licenses.License, error) {
	var out []licenses.License
	for _, name := range s.names[path] {
		out = append(out, licenses.License{Name: name, Type: licenses.LicenseType(name)})
	}
	return out, nil
}

func writeFile(t *testing.T, dir, name, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClassifyFileKeepsOnlyShippable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "LICENSE")
	stub := stubClassifier{names: map[string][]string{
		path: {"MIT", "GPL-3.0", "MIT", "ISC"},
	}}
	got, err := classifyFile(stub, path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ISC", "MIT"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("classifyFile = %q, want %q (copyleft dropped, sorted, deduplicated)", got, want)
	}
}

func TestClassifyFileNoneShippable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "LICENSE")
	stub := stubClassifier{names: map[string][]string{
		path: {"GPL-3.0"},
	}}
	got, err := classifyFile(stub, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("classifyFile = %q, want none", got)
	}
}

// The texts below are the canonical license texts; the real classifier must
// identify them, which wires the library into the gate.
const mitText = `Copyright 2020 Google Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`

const iscText = `ISC License

Copyright (c) 2012-2016 Dave Collins <dave@davec.name>

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.`

func TestClassifyFileRealLicenses(t *testing.T) {
	classifier, err := licenses.NewClassifier()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cases := []struct {
		name string
		text string
		want string
	}{
		{name: "LICENSE-MIT", text: mitText, want: "MIT"},
		{name: "LICENSE-ISC", text: iscText, want: "ISC"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, dir, tc.name, tc.text)
			got, err := classifyFile(classifier, path)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, name := range got {
				if name == tc.want {
					found = true
				}
			}
			if !found {
				t.Errorf("classifyFile(%s) = %q, want it to contain %q", tc.name, got, tc.want)
			}
		})
	}
}

func withStub(t *testing.T, stub licenses.Classifier) {
	t.Helper()
	old := newClassifier
	newClassifier = func() (licenses.Classifier, error) { return stub, nil }
	t.Cleanup(func() { newClassifier = old })
}

func TestRenderSectionsGolden(t *testing.T) {
	dir := t.TempDir()
	mit := writeFile(t, dir, "LICENSE-MIT", "  the MIT text  \n")
	isc := writeFile(t, dir, "LICENSE-ISC", "the ISC text")
	goText := writeFile(t, dir, "LICENSE-GO", "the Go text")
	withStub(t, stubClassifier{names: map[string][]string{
		mit:    {"MIT"},
		isc:    {"ISC", "MIT"},
		goText: {"BSD-3-Clause"},
	}})
	entries := []entry{
		{title: "example.com/mod@v1.2.3", files: []string{mit, isc}},
		{title: "The Go standard library and runtime, go1.27.1", files: []string{goText}},
	}
	sections, errs, err := renderSections(entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 0 {
		t.Fatalf("errs = %q", errs)
	}
	dashes := strings.Repeat("-", 72)
	want := "example.com/mod@v1.2.3 (ISC AND MIT)\n\n" +
		"the MIT text\n\nthe ISC text\n\n" +
		dashes + "\n\n" +
		"The Go standard library and runtime, go1.27.1 (BSD-3-Clause)\n\nthe Go text"
	if got := strings.Join(sections, separator) + "\n"; got != want+"\n" {
		t.Errorf("rendered licenses.txt differs:\ngot:\n%s\nwant:\n%s", got, want+"\n")
	}
}

func TestRenderSectionsUnknownGate(t *testing.T) {
	dir := t.TempDir()
	// The real classifier reports no license for gibberish.
	classifier, err := licenses.NewClassifier()
	if err != nil {
		t.Fatal(err)
	}
	old := newClassifier
	newClassifier = func() (licenses.Classifier, error) { return classifier, nil }
	t.Cleanup(func() { newClassifier = old })

	entries := []entry{
		{title: "example.com/mystery@v0.0.1", files: []string{writeFile(t, dir, "LICENSE", "All rights reserved. Do whatever your local laws allow, maybe.")}},
		{title: "example.com/bare@v0.0.1", files: nil},
	}
	_, errs, err := renderSections(entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 2 {
		t.Fatalf("errs = %q, want one per entry", errs)
	}
	for i, title := range []string{"example.com/mystery@v0.0.1", "example.com/bare@v0.0.1"} {
		if !strings.Contains(errs[i], title) || !strings.Contains(errs[i], "no license file with a license compatible with the Apache License 2.0") {
			t.Errorf("errs[%d] = %q, want the title and the gate message", i, errs[i])
		}
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "usage: golicenses --module") {
		t.Fatalf("err = %v, want the usage", err)
	}
}

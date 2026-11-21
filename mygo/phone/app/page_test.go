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

package app

import "testing"

func TestPagePaths(t *testing.T) {
	tests := []struct {
		page Page
		path string
	}{
		{Page{Kind: PageSDK, Name: "go"}, "/sdk/go"},
		{Page{Kind: PageTask, Name: "42"}, "/tasks/42"},
		{Page{Kind: PageLog, Name: "42"}, "/tasks/42/log"},
		{Page{Kind: PagePair}, "/pair"},
		{Page{Kind: PageSDK, Name: "a b/c"}, "/sdk/a%20b%2Fc"},
	}
	for _, tt := range tests {
		if got := tt.page.Path(); got != tt.path {
			t.Errorf("Path(%+v) = %q, want %q", tt.page, got, tt.path)
		}
	}
}

func TestParsePageReadsWhatPathWrites(t *testing.T) {
	pages := []Page{
		{Kind: PageSDK, Name: "go"},
		{Kind: PageSDK, Name: "a b/c"},
		{Kind: PageSDK, Name: "100%"},
		{Kind: PageTask, Name: "task-7"},
		{Kind: PageLog, Name: "task-7"},
		{Kind: PagePair},
	}
	for _, page := range pages {
		got, ok := ParsePage(page.Path())
		if !ok || got != page {
			t.Errorf("ParsePage(%q) = %+v, %v; want %+v", page.Path(), got, ok, page)
		}
	}
}

func TestParsePageRejectsWhatIsNoPage(t *testing.T) {
	for _, path := range []string{
		"", RootPath, "sdk/go", "/sdk", "/sdk/", "/sdk/go/other", "/sdk/go/versions", "/sdk/go/versions/x",
		"/tasks", "/tasks/", "/tasks/42/logs", "/pair/x", "/nope", "/sdk/%zz",
	} {
		if page, ok := ParsePage(path); ok {
			t.Errorf("ParsePage(%q) = %+v, want no page", path, page)
		}
	}
}

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

import (
	"net/url"
	"strings"
)

// RootPath is the path of the tabs, where the history of the window starts.
const RootPath = "/"

// PageKind tells the pages above the tabs apart.
type PageKind int

// The kinds of pages. The installable versions of an SDK are a tab of its page,
// not a page of their own.
const (
	// PageSDK is the detail of an SDK, which is named by its plugin.
	PageSDK PageKind = iota + 1
	// PageTask is a task and its output.
	PageTask
	// PageLog is the full log of a task.
	PageLog
	// PagePair is the pairing with a new link.
	PagePair
)

// Page is a page above the tabs.
type Page struct {
	// Kind is the kind of the page.
	Kind PageKind
	// Name is the SDK of PageSDK, and the ID of the task of PageTask and PageLog.
	// It is empty for PagePair.
	Name string
}

// valid reports whether p has a kind, and the name that the kind needs.
func (p Page) valid() bool {
	switch p.Kind {
	case PageSDK, PageTask, PageLog:
		return p.Name != ""
	case PagePair:
		return p.Name == ""
	}
	return false
}

// Path returns the path of the page, which the history of the window shows:
// /sdk/<name>, /tasks/<id>, /tasks/<id>/log or /pair. The name is escaped as one
// segment of the path.
func (p Page) Path() string {
	switch p.Kind {
	case PageSDK:
		return "/sdk/" + url.PathEscape(p.Name)
	case PageTask:
		return "/tasks/" + url.PathEscape(p.Name)
	case PageLog:
		return "/tasks/" + url.PathEscape(p.Name) + "/log"
	case PagePair:
		return "/pair"
	}
	return RootPath
}

// ParsePage returns the page that a path shows, which Path makes. It returns
// false for the root, and for a path that is no page.
func ParsePage(path string) (Page, bool) {
	if !strings.HasPrefix(path, "/") {
		return Page{}, false
	}
	if path == "/pair" {
		return Page{Kind: PagePair}, true
	}
	segments := strings.Split(path[1:], "/")
	switch {
	case len(segments) == 2 && segments[0] == "sdk":
		return named(PageSDK, segments[1])
	case len(segments) == 2 && segments[0] == "tasks":
		return named(PageTask, segments[1])
	case len(segments) == 3 && segments[0] == "tasks" && segments[2] == "log":
		return named(PageLog, segments[1])
	}
	return Page{}, false
}

// named returns the page of a kind that is named by an escaped path segment.
func named(kind PageKind, escaped string) (Page, bool) {
	name, err := url.PathUnescape(escaped)
	if err != nil || name == "" {
		return Page{}, false
	}
	return Page{Kind: kind, Name: name}, true
}

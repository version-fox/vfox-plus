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

package automation

import (
	"strings"
	"testing"
)

// sdkPage is what the SDKs page of the application shows, as its elements: two rows
// of versions with their texts and buttons, and a setting with its field.
func sdkPage() Tree {
	press := []string{"press", "focus"}
	return Tree{Window: 1, Nodes: []Node{
		{ID: "1", Role: "button", Key: float64(1), Label: "SDKs", Actions: press},
		{ID: "2", Role: "list"},
		{ID: "3", Parent: "2", Role: "listItem", Key: "av:2.0.0", Label: "2.0.0 LTS Install"},
		{ID: "4", Parent: "3", Role: "text", Label: "2.0.0"},
		{ID: "5", Parent: "3", Role: "text", Label: "LTS"},
		{ID: "6", Parent: "3", Role: "button", Key: "install", Label: "Install", Actions: press},
		{ID: "7", Parent: "2", Role: "listItem", Key: "av:1.0.0", Label: "1.0.0 Install"},
		{ID: "8", Parent: "7", Role: "text", Label: "1.0.0"},
		{ID: "9", Parent: "7", Role: "button", Key: "install", Label: "Install", Actions: press},
		{ID: "10", Parent: "7", Role: "text", Label: "Downloading… 40%"},
		{ID: "11", Role: "scroll"},
		{ID: "12", Parent: "11", Role: "text", Label: "Plugin registry"},
		{ID: "13", Parent: "11", Role: "textField", Label: "Official registry", Actions: []string{"focus", "setValue"}},
		{ID: "14", Parent: "11", Role: "heading", Label: "2 versions"},
		{ID: "15", Role: "dialog", Label: "Remove?"},
		{ID: "16", Parent: "15", Role: "button", Key: "confirm", Label: "Remove", Actions: press},
	}}
}

func TestFindFilterAndWithin(t *testing.T) {
	tree := sdkPage()
	install, ok := tree.Find(func(n Node) bool { return n.Label == "Install" })
	if !ok || install.ID != "6" {
		t.Fatalf("Find = %+v, %v, want the first", install, ok)
	}
	if _, ok := tree.Find(func(n Node) bool { return n.Label == "nothing" }); ok {
		t.Fatal("Find found what is not there")
	}
	if got := tree.Filter(func(n Node) bool { return n.Label == "Install" }); len(got) != 2 || got[1].ID != "9" {
		t.Fatalf("Filter = %+v", got)
	}
	remove, _ := tree.Find(func(n Node) bool { return n.ID == "16" })
	if !tree.Within(remove, func(n Node) bool { return n.Role == "dialog" }) || tree.Within(install, func(n Node) bool { return n.Role == "dialog" }) {
		t.Fatal("Within does not follow the parents")
	}
	// A tree whose parents lead round in a circle does not hold the reading.
	loop := Tree{Nodes: []Node{{ID: "a", Parent: "b"}, {ID: "b", Parent: "a"}}}
	if loop.Within(loop.Nodes[0], func(Node) bool { return false }) {
		t.Fatal("Within found an ancestor in a loop")
	}
}

func TestTextsAndWhatTheyShow(t *testing.T) {
	tree := sdkPage()
	if !tree.Shows("2 versions") || tree.Shows("3 versions") || !tree.Shows("Official registry") {
		t.Fatal("Shows")
	}
	if tree.Count("Install") != 2 || tree.Count("LTS") != 1 {
		t.Fatalf("Count: Install %d, LTS %d", tree.Count("Install"), tree.Count("LTS"))
	}
	if !tree.ShowsContaining("registry") || tree.ShowsContaining("REGISTRY") {
		t.Fatal("ShowsContaining")
	}
	texts := tree.Texts()
	if strings.Join(texts, "|") != "2.0.0|LTS|1.0.0|Downloading… 40%|Plugin registry|2 versions" {
		t.Fatalf("Texts = %q", texts)
	}
}

func TestRowsAreTitledByTheirTexts(t *testing.T) {
	tree := sdkPage()
	row, _ := tree.Find(func(n Node) bool { return n.Key == "av:1.0.0" })
	if !tree.HasTitle(row, "1.0.0") || tree.HasTitle(row, "2.0.0") {
		t.Fatal("HasTitle")
	}
	// Only a row of a list has a title: the scroll that holds the setting does not.
	scroll, _ := tree.Find(func(n Node) bool { return n.Role == "scroll" })
	if tree.HasTitle(scroll, "Plugin registry") {
		t.Fatal("a container that is no row was titled by its text")
	}
	if !tree.RowSays("2.0.0", "LTS") || tree.RowSays("1.0.0", "LTS") || !tree.RowSays("1.0.0", "Install") || tree.RowSays("3.0.0", "Install") {
		t.Fatal("RowSays")
	}
}

func TestAFieldFollowsTheTitleOfItsSetting(t *testing.T) {
	tree := sdkPage()
	field, ok := tree.FieldAfter("Plugin registry")
	if !ok || field.ID != "13" {
		t.Fatalf("FieldAfter = %+v, %v", field, ok)
	}
	if _, ok := tree.FieldAfter("2 versions"); ok {
		t.Fatal("a heading with no field after it was given one")
	}
	if _, ok := tree.FieldAfter("Nowhere"); ok {
		t.Fatal("a title that is not there was given a field")
	}
	// The field of another container is not the one: only its siblings are.
	other := Tree{Nodes: []Node{
		{ID: "1", Parent: "a", Role: "text", Label: "Title"},
		{ID: "2", Parent: "b", Role: "textField", Label: "elsewhere"},
	}}
	if _, ok := other.FieldAfter("Title"); ok {
		t.Fatal("a field of another container was taken")
	}
}

func TestDownloadingSaysThePercent(t *testing.T) {
	if percent, ok := sdkPage().Downloading(); !ok || percent != 40 {
		t.Fatalf("Downloading = %d, %v", percent, ok)
	}
	idle := Tree{Nodes: []Node{{Role: "text", Label: "Install"}, {Role: "text", Label: "Downloading… many%"}}}
	if percent, ok := idle.Downloading(); ok {
		t.Fatalf("Downloading = %d, true for no percent", percent)
	}
}

func TestDescribeListsWhatAPersonSees(t *testing.T) {
	tree := Tree{Nodes: []Node{
		{Role: "heading", Label: "Tasks"},
		{Role: "button", Key: "clear", Label: "Clear finished"},
		{Role: "button", Key: "x", Label: "Off", States: []string{"disabled"}},
		{Role: "textField", Value: "abc"},
		{Role: "list"},
		{Role: "text", Label: strings.Repeat("long ", 40)},
	}}
	lines := strings.Split(tree.Describe(), "\n")
	if len(lines) != 5 {
		t.Fatalf("Describe has %d lines, want the 5 that say something:\n%s", len(lines), tree.Describe())
	}
	if lines[0] != `heading "Tasks"` || lines[1] != `button [clear] "Clear finished"` || lines[2] != `button [x] "Off" (disabled)` || lines[3] != `textField "abc"` {
		t.Fatalf("Describe =\n%s", tree.Describe())
	}
	if !strings.HasSuffix(lines[4], `…"`) || len([]rune(lines[4])) > 110 {
		t.Fatalf("a long text is not cut: %q", lines[4])
	}
}

func TestKeyTextOfEveryKindOfKey(t *testing.T) {
	cases := map[string]Node{
		"":        {},
		"sdk:x":   {Key: "sdk:x"},
		"3":       {Key: float64(3)},
		"1.5":     {Key: 1.5},
		"true":    {Key: true},
		"[1 2]":   {Key: []int{1, 2}},
		"menu":    {Key: "menu"},
		"-7":      {Key: float64(-7)},
		"1000000": {Key: float64(1000000)},
	}
	for want, node := range cases {
		if got := node.KeyText(); got != want {
			t.Errorf("KeyText(%v) = %q, want %q", node.Key, got, want)
		}
	}
}

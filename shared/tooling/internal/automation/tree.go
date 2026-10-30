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
	"strconv"
	"strings"
)

// Find returns the first node for which match is true.
func (t Tree) Find(match func(Node) bool) (Node, bool) {
	for _, node := range t.Nodes {
		if match(node) {
			return node, true
		}
	}
	return Node{}, false
}

// Filter returns the nodes for which match is true, in tree order.
func (t Tree) Filter(match func(Node) bool) []Node {
	var found []Node
	for _, node := range t.Nodes {
		if match(node) {
			found = append(found, node)
		}
	}
	return found
}

// parentOf returns the node that holds node.
func (t Tree) parentOf(node Node) (Node, bool) {
	if node.Parent == "" {
		return Node{}, false
	}
	for _, other := range t.Nodes {
		if other.ID == node.Parent {
			return other, true
		}
	}
	return Node{}, false
}

// Within reports whether node is inside an element (at any depth) for which
// ancestor is true.
func (t Tree) Within(node Node, ancestor func(Node) bool) bool {
	for range len(t.Nodes) {
		parent, ok := t.parentOf(node)
		if !ok {
			return false
		}
		if ancestor(parent) {
			return true
		}
		node = parent
	}
	return false
}

// Shows reports whether a node says the text, which is its label or its value:
// text, a heading, the label of a button.
func (t Tree) Shows(text string) bool { return t.Count(text) > 0 }

// Count returns how many nodes say the text.
func (t Tree) Count(text string) int {
	count := 0
	for _, node := range t.Nodes {
		if node.Label == text || node.Value == text {
			count++
		}
	}
	return count
}

// Texts returns what the nodes of text and headings say.
func (t Tree) Texts() []string {
	var texts []string
	for _, node := range t.Nodes {
		if (node.Role == "text" || node.Role == "heading") && node.Label != "" {
			texts = append(texts, node.Label)
		}
	}
	return texts
}

// Describe lists the elements that a person sees, one line each, to tell what
// was on the screen when a scenario failed.
func (t Tree) Describe() string {
	var lines []string
	for _, node := range t.Nodes {
		label := node.Label
		if label == "" && node.Value != "" {
			label = node.Value
		}
		if label == "" && node.KeyText() == "" {
			continue
		}
		line := node.Role
		if key := node.KeyText(); key != "" {
			line += " [" + key + "]"
		}
		if label != "" {
			line += " " + quote(label)
		}
		if node.Has("disabled") {
			line += " (disabled)"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func quote(text string) string {
	const limit = 90
	if runes := []rune(text); len(runes) > limit {
		text = string(runes[:limit]) + "…"
	}
	return "\"" + strings.ReplaceAll(text, "\"", "'") + "\""
}

// HasTitle reports whether the node is an item of a list that is titled by a text
// or a heading that says the title: a row, with its title among its texts.
func (t Tree) HasTitle(item Node, title string) bool {
	if item.Role != "listItem" {
		return false
	}
	for _, node := range t.Nodes {
		if node.Parent == item.ID && (node.Role == "text" || node.Role == "heading") && node.Label == title {
			return true
		}
	}
	return false
}

// ShowsContaining reports whether a node says something with the text in it.
func (t Tree) ShowsContaining(text string) bool {
	for _, node := range t.Nodes {
		if strings.Contains(node.Label, text) || strings.Contains(node.Value, text) {
			return true
		}
	}
	return false
}

// RowSays reports whether the list item titled row has a text or a button that says
// the text.
func (t Tree) RowSays(row, text string) bool {
	for _, item := range t.Nodes {
		if !t.HasTitle(item, row) {
			continue
		}
		for _, node := range t.Nodes {
			if node.Parent == item.ID && node.Label == text {
				return true
			}
		}
	}
	return false
}

// Downloading returns how many percent of a download the window says it has, as the
// row of a version being installed does: "Downloading… 40%".
func (t Tree) Downloading() (percent int, ok bool) {
	for _, node := range t.Nodes {
		if text, found := strings.CutPrefix(node.Label, "Downloading… "); found {
			if number, err := strconv.Atoi(strings.TrimSuffix(text, "%")); err == nil {
				return number, true
			}
		}
	}
	return 0, false
}

// FieldAfter returns the text field that follows the text or the heading that says
// the title, among the same elements: a setting is a title and the field that holds
// its value, and the field is named after its hint, not after the title.
func (t Tree) FieldAfter(title string) (Node, bool) {
	parent, titled := "", false
	for _, node := range t.Nodes {
		if !titled {
			if (node.Role == "text" || node.Role == "heading") && node.Label == title {
				parent, titled = node.Parent, true
			}
			continue
		}
		if node.Role == "textField" && node.Parent == parent {
			return node, true
		}
	}
	return Node{}, false
}

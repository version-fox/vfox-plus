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

package measureapk

import (
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// element is one element of the accessibility tree of a screen: its label, the text or else the
// description, whether a tap on it does something, and its bounds on the screen, in pixels.
type element struct {
	label                    string
	clickable                bool
	left, top, right, bottom int
}

// center is the middle of the element, where a tap goes.
func (e element) center() (int, int) {
	return (e.left + e.right) / 2, (e.top + e.bottom) / 2
}

// screen is one dump of the accessibility tree of the screen: its elements, and the dump as the
// device wrote it, which a failing run keeps for the diagnosis.
type screen struct {
	elements []element
	raw      string
}

// hierarchy is the root of the dump that uiautomator writes.
type hierarchy struct {
	Nodes []xmlNode `xml:"node"`
}

// xmlNode is one node of the dump: its attributes, and the nodes inside it.
type xmlNode struct {
	Text      string    `xml:"text,attr"`
	Desc      string    `xml:"content-desc,attr"`
	Clickable bool      `xml:"clickable,attr"`
	Bounds    string    `xml:"bounds,attr"`
	Children  []xmlNode `xml:"node"`
}

// boundsPattern matches the bounds of a node: "[left,top][right,bottom]".
var boundsPattern = regexp.MustCompile(`^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$`)

// parseScreen reads a dump of the accessibility tree. A dump without any element is an error: the
// screen is not drawn yet, or the app is not in the foreground.
func parseScreen(data []byte) (screen, error) {
	var root hierarchy
	if err := xml.Unmarshal(data, &root); err != nil {
		return screen{}, fmt.Errorf("the screen dump is not XML: %w", err)
	}
	var elements []element
	if err := flatten(root.Nodes, &elements); err != nil {
		return screen{}, err
	}
	if len(elements) == 0 {
		return screen{}, errors.New("the screen dump has no element")
	}
	return screen{elements: elements, raw: string(data)}, nil
}

// flatten appends the elements of the nodes, and of the nodes in them, in the order of the dump.
func flatten(nodes []xmlNode, out *[]element) error {
	for _, n := range nodes {
		e := element{label: n.Text, clickable: n.Clickable}
		if e.label == "" {
			e.label = n.Desc
		}
		if n.Bounds != "" {
			match := boundsPattern.FindStringSubmatch(n.Bounds)
			if match == nil {
				return fmt.Errorf("the bounds %q of an element are not [left,top][right,bottom]", n.Bounds)
			}
			values := make([]int, 4)
			for i := range values {
				values[i], _ = strconv.Atoi(match[i+1])
			}
			e.left, e.top, e.right, e.bottom = values[0], values[1], values[2], values[3]
		}
		*out = append(*out, e)
		if err := flatten(n.Children, out); err != nil {
			return err
		}
	}
	return nil
}

// lowestClickable returns the clickable element whose label matches pattern and which is lowest on
// the screen, and whether there is one. A tab bar is at the bottom of the screen, so its tab is the
// lowest element with the label of the tab.
func lowestClickable(elements []element, pattern *regexp.Regexp) (element, bool) {
	var best element
	found := false
	for _, e := range elements {
		if !e.clickable || !pattern.MatchString(strings.TrimSpace(e.label)) {
			continue
		}
		if _, y := e.center(); !found {
			best, found = e, true
		} else if _, by := best.center(); y > by {
			best = e
		}
	}
	return best, found
}

// hasLabel reports whether an element of the screen has a label that matches pattern, clickable or not.
func hasLabel(elements []element, pattern *regexp.Regexp) bool {
	for _, e := range elements {
		if pattern.MatchString(strings.TrimSpace(e.label)) {
			return true
		}
	}
	return false
}

// labelsOf lists the labels of the elements, each once, for the errors that say what a screen shows.
func labelsOf(elements []element) string {
	seen := map[string]bool{}
	var labels []string
	for _, e := range elements {
		label := strings.TrimSpace(e.label)
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		labels = append(labels, strconv.Quote(label))
		if len(labels) == 30 {
			break
		}
	}
	if len(labels) == 0 {
		return "no label"
	}
	return strings.Join(labels, ", ")
}

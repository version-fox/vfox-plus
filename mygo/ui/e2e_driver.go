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

//go:build e2e

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// scenario is what a scenario can use.
type scenario struct {
	app *App
}

func newScenario(a *App) *scenario { return &scenario{app: a} }

// step reports progress on the output of the application.
func (s *scenario) step(message string) { report("%s", message) }

// check fails the scenario unless the condition holds.
func check(condition bool, description string) {
	if !condition {
		panic("Check failed: " + description)
	}
}

// pause waits for the user interface to settle, as a person would.
func pause(milliseconds int) { time.Sleep(time.Duration(milliseconds) * time.Millisecond) }

// onMain runs f on the main thread, where the registry and the widgets live, and
// waits for it.
func (s *scenario) onMain(f func()) {
	done := make(chan struct{})
	s.app.win.Update(func() {
		defer close(done)
		f()
	})
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		panic("the window does not run calls")
	}
}

// onWindow runs f on the main thread, like onMain, and waits for the frame that
// shows what it did.
func (s *scenario) onWindow(f func()) {
	var before uint64
	s.onMain(func() {
		before = s.app.e2e.builds.Load()
		f()
	})
	s.waitUntil("the window to draw a frame", func() bool { return s.app.e2e.builds.Load() > before }, 30*time.Second)
}

// nodes returns what the user can reach, like the topmost layer: the nodes of the
// dialog or the menu that is open, or else all the others.
func (s *scenario) nodes() []kit.Node {
	var reachable []kit.Node
	s.onMain(func() {
		all := s.app.env.Reg.Nodes()
		modal := false
		for _, node := range all {
			modal = modal || node.Modal
		}
		for _, node := range all {
			if node.Modal == modal {
				reachable = append(reachable, *node)
			}
		}
	})
	return reachable
}

// waitUntil polls until the condition holds.
func (s *scenario) waitUntil(description string, condition func() bool, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			panic(fmt.Sprintf("Timed out waiting for %s", description))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

const defaultWait = 30 * time.Second

// texts are the texts on the screen.
func (s *scenario) texts() []string {
	var result []string
	for _, node := range s.nodes() {
		if node.Role == kit.RoleText && node.Text != "" {
			result = append(result, node.Text)
		}
	}
	return result
}

func (s *scenario) waitForText(text string) {
	s.waitUntil(fmt.Sprintf("the text %q", text), func() bool {
		for _, shown := range s.texts() {
			if shown == text {
				return true
			}
		}
		return false
	}, defaultWait)
}

// find returns the first node that matches.
func (s *scenario) find(match func(kit.Node, []kit.Node) bool) (kit.Node, bool) {
	all := s.nodes()
	for _, node := range all {
		if match(node, all) {
			return node, true
		}
	}
	return kit.Node{}, false
}

// await waits until a node matches and returns it.
func (s *scenario) await(description string, match func(kit.Node, []kit.Node) bool) kit.Node {
	var found kit.Node
	s.waitUntil(description, func() bool {
		node, ok := s.find(match)
		found = node
		return ok
	}, defaultWait)
	return found
}

// settle waits for the frames in which what the user did takes effect: the click
// is handled by the frame that follows it, and what it changed is shown by the
// one after that.
func (s *scenario) settle() {
	for range 3 {
		s.onWindow(func() {})
	}
}

// click does what a click on the node does.
func (s *scenario) click(node kit.Node) {
	check(node.Click != nil, fmt.Sprintf("%q can be clicked", node.Name))
	s.onWindow(node.Click)
	s.settle()
}

func isButton(node kit.Node, name string) bool {
	return (node.Role == kit.RoleButton || node.Role == kit.RoleMenuItem) && node.Name == name && node.Enabled
}

// clickButton clicks the button that is called name: its label or its tooltip.
func (s *scenario) clickButton(name string) {
	s.click(s.await(fmt.Sprintf("the button %q", name), func(node kit.Node, _ []kit.Node) bool { return isButton(node, name) }))
}

// clickItem clicks the list item whose title starts with title.
func (s *scenario) clickItem(title string) {
	s.click(s.await(fmt.Sprintf("the item %q", title), func(node kit.Node, _ []kit.Node) bool {
		return node.Role == kit.RoleListItem && node.Click != nil && strings.HasPrefix(node.Name, title)
	}))
}

// clickButtonNear clicks the button name in the list item that is called row.
func (s *scenario) clickButtonNear(row, name string) {
	s.click(s.await(fmt.Sprintf("the button %q next to %q", name, row), func(node kit.Node, _ []kit.Node) bool {
		return isButton(node, name) && node.Scope == row
	}))
}

// fill types value into the text field that is labelled or hinted label, or into
// the text field of the list item titled label.
func (s *scenario) fill(label, value string, submit bool) {
	node := s.await(fmt.Sprintf("the field %q", label), func(node kit.Node, _ []kit.Node) bool {
		return node.Role == kit.RoleTextField && node.Fill != nil && (node.Name == label || node.Scope == label)
	})
	s.onWindow(func() { node.Fill(value, submit) })
	s.settle()
}

// navigate chooses the destination label in the navigation rail or bar.
func (s *scenario) navigate(label string) {
	s.click(s.await(fmt.Sprintf("the destination %q", label), func(node kit.Node, _ []kit.Node) bool {
		return node.Role == kit.RoleButton && node.Name == label && node.Enabled
	}))
}

// chooseMenuItem opens the menu of the button that is called menuTooltip and
// chooses item.
func (s *scenario) chooseMenuItem(menuTooltip, item string) {
	s.clickButton(menuTooltip)
	s.click(s.await(fmt.Sprintf("the menu item %q", item), func(node kit.Node, _ []kit.Node) bool {
		return node.Role == kit.RoleMenuItem && node.Name == item && node.Enabled
	}))
}

// tapDialog presses the last (the confirm) or the one before it (the cancel) of
// the buttons of a dialog, which are the actions at its bottom.
func (s *scenario) tapDialog(confirm bool) {
	s.waitUntil("a dialog", func() bool {
		_, ok := s.find(func(node kit.Node, _ []kit.Node) bool { return node.Role == kit.RoleDialog })
		return ok
	}, defaultWait)
	var chosen []kit.Node
	for _, node := range s.nodes() {
		if node.Role == kit.RoleButton && node.Enabled && node.Scope == "dialog" {
			chosen = append(chosen, node)
		}
	}
	check(len(chosen) > 0, "the dialog has buttons")
	last := len(chosen) - 1
	if confirm || last == 0 {
		s.click(chosen[last])
	} else {
		s.click(chosen[last-1])
	}
}

// dump reports what is on the screen: the texts and the buttons.
func (s *scenario) dump() {
	var texts, buttons []string
	for _, node := range s.nodes() {
		switch {
		case node.Role == kit.RoleText && node.Text != "":
			texts = append(texts, fmt.Sprintf("%q", node.Text))
		case node.Name != "":
			state := ""
			if !node.Enabled {
				state = " (disabled)"
			}
			buttons = append(buttons, fmt.Sprintf("%s %q%s", node.Role, node.Name, state))
		}
	}
	report("on screen, texts: %s", strings.Join(texts, ", "))
	report("on screen, widgets: %s", strings.Join(buttons, ", "))
}

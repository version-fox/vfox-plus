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

package appsession

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

// The pages of the application, by the key of their button in the navigation,
// which does not depend on the language.
const (
	Home     = "0"
	SDKs     = "1"
	Plugins  = "2"
	Tasks    = "3"
	Settings = "4"
)

// failure is what a scenario that does not hold ends with.
type failure struct{ message string }

// Fail ends the scenario with a failure.
func (s *Session) Fail(format string, args ...any) { panic(failure{fmt.Sprintf(format, args...)}) }

// Check ends the scenario unless the condition holds.
func (s *Session) Check(condition bool, description string) {
	if !condition {
		s.Fail("Check failed: %s", description)
	}
}

// Step reports the progress of the scenario.
func (s *Session) Step(message string) { Report("%s", message) }

// Run runs the scenario against the application. A failed check, a wait that
// times out and the end of the application end it with an error, after what the
// window showed is reported.
func (s *Session) Run(name string, scenario func(*Session)) (err error) {
	Report("scenario %s started", name)
	started := time.Now()
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		var failed failure
		if f, ok := recovered.(failure); ok {
			failed = f
		} else {
			failed = failure{fmt.Sprint(recovered)}
		}
		err = errors.New(failed.message)
		Report("scenario %s FAILED: %v", name, err)
		s.dump()
	}()
	scenario(s)
	Report("scenario %s passed in %d s", name, int(time.Since(started).Seconds()+0.5))
	return nil
}

// dump reports what the window shows, to find out why a scenario failed.
func (s *Session) dump() {
	if !s.Alive() {
		data, _ := os.ReadFile(s.Log)
		Report("the application has ended; it wrote:\n%s", strings.TrimSpace(string(data)))
		return
	}
	tree, err := s.client.Tree()
	if err != nil {
		Report("the window cannot be read: %v", err)
		return
	}
	lines := strings.Split(tree.Describe(), "\n")
	const limit = 150
	if len(lines) > limit {
		lines = append(lines[:limit], fmt.Sprintf("… and %d more", len(lines)-limit))
	}
	Report("on screen:\n%s", strings.Join(lines, "\n"))
}

// Tree reads the elements of the window. A scenario fails when it cannot.
func (s *Session) Tree() automation.Tree {
	tree, err := s.client.Tree()
	if err != nil {
		if !s.Alive() {
			s.Fail("the application has ended: %v", s.explain(err))
		}
		s.Fail("%v", err)
	}
	return tree
}

// Settle waits for the window to draw the frames that it asked for, as a person
// waits for the screen to calm down. A window that never does, because
// something in it keeps moving, is not waited for longer than a moment.
func (s *Session) Settle() {
	err := s.client.Idle(1500 * time.Millisecond)
	if err != nil && !automation.IsCode(err, "timeout") {
		if !s.Alive() {
			s.Fail("the application has ended: %v", s.explain(err))
		}
		s.Fail("%v", err)
	}
}

// Pause waits for the given time, as a person looking at the screen does.
func (s *Session) Pause(duration time.Duration) { time.Sleep(duration) }

// defaultWait is how long a scenario waits for something to show, and taskWait how
// long for a task to finish. They are variables so that tests need not wait.
var (
	defaultWait = 30 * time.Second
	taskWait    = 60 * time.Second
)

// WaitUntil polls the elements of the window until the condition holds.
func (s *Session) WaitUntil(description string, timeout time.Duration, condition func(automation.Tree) bool) automation.Tree {
	deadline := time.Now().Add(timeout)
	for {
		tree := s.Tree()
		if condition(tree) {
			return tree
		}
		if time.Now().After(deadline) {
			s.Fail("Timed out waiting for %s", description)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// WaitFor polls a condition that is about something else than the window, such as
// a file or what vfox says.
func (s *Session) WaitFor(description string, timeout time.Duration, condition func() bool) {
	deadline := time.Now().Add(timeout)
	for !condition() {
		if time.Now().After(deadline) {
			s.Fail("Timed out waiting for %s", description)
		}
		if !s.Alive() {
			s.Fail("the application has ended while waiting for %s", description)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// WaitForText waits until something in the window says the text.
func (s *Session) WaitForText(text string) {
	s.WaitUntil(fmt.Sprintf("the text %q", text), defaultWait, func(t automation.Tree) bool { return t.Shows(text) })
}

// WaitForTextContaining waits until something in the window says something with
// the text in it, as the output of a task does.
func (s *Session) WaitForTextContaining(text string) {
	s.WaitUntil(fmt.Sprintf("a text with %q", text), defaultWait, func(t automation.Tree) bool { return t.ShowsContaining(text) })
}

// WaitForTextGone waits until nothing in the window says the text, such as the
// count of a list that a filter narrows.
func (s *Session) WaitForTextGone(text string) {
	s.WaitUntil(fmt.Sprintf("the text %q to go away", text), defaultWait, func(t automation.Tree) bool { return !t.Shows(text) })
}

// WaitForTask waits until the task with the title (in English) has succeeded. The
// application tells it in a snack bar, "<title>: done", except on the Tasks page,
// where the row of the task says "<title>, Done". It fails when the task failed.
func (s *Session) WaitForTask(title string) {
	snackDone, snackFailed := title+": done", title+": failed"
	rowDone, rowFailed := title+", Done", title+", Failed"
	s.WaitUntil(fmt.Sprintf("the task %q to finish", title), taskWait, func(t automation.Tree) bool {
		for _, node := range t.Nodes {
			switch {
			case node.Role == "status" && strings.HasPrefix(node.Label, snackFailed), node.Role == "button" && strings.HasPrefix(node.Label, rowFailed):
				s.Fail("%s", strings.ReplaceAll(node.Label, "\n", ": "))
			case node.Role == "status" && strings.HasPrefix(node.Label, snackDone), node.Role == "button" && strings.HasPrefix(node.Label, rowDone):
				return true
			}
		}
		return false
	})
}

// pressWhere presses the first element that matches, once there is one.
func (s *Session) pressWhere(description string, match func(automation.Tree, automation.Node) bool) {
	var lastErr error
	for range 3 {
		var found automation.Node
		s.WaitUntil(description, defaultWait, func(t automation.Tree) bool {
			node, ok := t.Find(func(n automation.Node) bool { return match(t, n) })
			found = node
			return ok
		})
		if found.Has("offscreen") {
			_ = s.client.Act(found.ID, automation.ScrollIntoView, "")
		}
		lastErr = s.client.Act(found.ID, automation.Press, "")
		if lastErr == nil {
			s.Settle()
			return
		}
		// The frame between the reading and the action may have changed the
		// element's ID: read it again.
		if !automation.IsCode(lastErr, "no_element") {
			break
		}
	}
	s.Fail("press %s: %v", description, lastErr)
}

func isButton(n automation.Node, name string) bool {
	return n.Label == name && n.CanDo(automation.Press) && (n.Role == "button" || n.Role == "menuItem")
}

// PressKey presses the first element with the key that the application gave it.
func (s *Session) PressKey(key string) {
	s.pressWhere(fmt.Sprintf("the element with the key %q", key), func(_ automation.Tree, n automation.Node) bool {
		return n.KeyText() == key && n.CanDo(automation.Press)
	})
}

// ClickButton presses the button that is called name: its label, or its tooltip
// for a button with an icon only.
func (s *Session) ClickButton(name string) {
	s.pressWhere(fmt.Sprintf("the button %q", name), func(_ automation.Tree, n automation.Node) bool { return isButton(n, name) })
}

// ClickItem presses the list item whose title starts with title.
func (s *Session) ClickItem(title string) {
	s.pressWhere(fmt.Sprintf("the item %q", title), func(_ automation.Tree, n automation.Node) bool {
		return n.CanDo(automation.Press) && strings.HasPrefix(n.Label, title) && (n.Role == "button" || n.Role == "listItem")
	})
}

// ClickButtonNear presses the button name in the list item that is titled row.
func (s *Session) ClickButtonNear(row, name string) {
	s.pressWhere(fmt.Sprintf("the button %q next to %q", name, row), func(t automation.Tree, n automation.Node) bool {
		return isButton(n, name) && t.Within(n, func(parent automation.Node) bool { return t.HasTitle(parent, row) })
	})
}

// Navigate chooses a page (Home, SDKs, ...) in the navigation.
func (s *Session) Navigate(page string) {
	s.pressWhere("the page with the key "+page, func(_ automation.Tree, n automation.Node) bool {
		return n.Role == "button" && n.Parent == "" && n.KeyText() == page && n.CanDo(automation.Press)
	})
}

// Fill puts the value into the text field that is labelled or hinted label, that is
// in the list item titled label, or that follows the text label (a setting), and
// with submit presses Enter.
func (s *Session) Fill(label, value string, submit bool) {
	var field automation.Node
	s.WaitUntil(fmt.Sprintf("the field %q", label), defaultWait, func(t automation.Tree) bool {
		node, ok := t.Find(func(n automation.Node) bool {
			return n.Role == "textField" && n.CanDo(automation.SetValue) &&
				(n.Label == label || n.Placeholder == label || t.Within(n, func(parent automation.Node) bool { return t.HasTitle(parent, label) }))
		})
		if !ok {
			node, ok = t.FieldAfter(label)
		}
		field = node
		return ok
	})
	if err := s.client.Act(field.ID, automation.SetValue, value); err != nil {
		s.Fail("fill %q: %v", label, err)
	}
	s.Settle()
	if submit {
		if err := s.client.Input(automation.Event{Type: "key", Key: "Enter"}); err != nil {
			s.Fail("submit %q: %v", label, err)
		}
		s.Settle()
	}
}

// isDialog says what the dialogs are.
func isDialog(n automation.Node) bool { return n.Role == "dialog" || n.Role == "alertDialog" }

// TapDialog presses the confirm or the cancel button of the dialog that is open.
func (s *Session) TapDialog(confirm bool) {
	s.WaitUntil("a dialog", defaultWait, func(t automation.Tree) bool {
		_, ok := t.Find(isDialog)
		return ok
	})
	key := "cancel"
	if confirm {
		key = "confirm"
	}
	s.pressWhere(fmt.Sprintf("the %s button of the dialog", key), func(t automation.Tree, n automation.Node) bool {
		return n.KeyText() == key && n.CanDo(automation.Press) && t.Within(n, isDialog)
	})
}

// ChooseMenuItem opens the menu of the button that is called menuButton and
// chooses the item.
func (s *Session) ChooseMenuItem(menuButton, item string) {
	s.ClickButton(menuButton)
	s.pressWhere(fmt.Sprintf("the menu item %q", item), func(t automation.Tree, n automation.Node) bool {
		return isButton(n, item) && t.Within(n, func(parent automation.Node) bool { return parent.Role == "popup" || parent.Role == "menu" })
	})
}

// CheckNoTaskFailed fails the scenario when a task is announced as failed: the snack
// bar of a failure stays until it is closed, and the Tasks page shows the row.
func (s *Session) CheckNoTaskFailed() {
	for _, node := range s.Tree().Nodes {
		if (node.Role == "status" && strings.Contains(node.Label, ": failed")) || (node.Role == "button" && strings.Contains(node.Label, ", Failed")) {
			s.Fail("%s", strings.ReplaceAll(node.Label, "\n", ": "))
		}
	}
}

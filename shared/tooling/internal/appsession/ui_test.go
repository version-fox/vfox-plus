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
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

var press = []string{"press", "focus"}

// failsWith runs f and returns the message of the failure that ends the scenario.
func failsWith(t *testing.T, f func()) (message string, failed bool) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			failure, ok := recovered.(failure)
			if !ok {
				panic(recovered)
			}
			message, failed = failure.message, true
		}
	}()
	f()
	return "", false
}

// shortWaits makes the waits of a scenario so short that a test of a wait that does
// not end need not wait.
func shortWaits(t *testing.T) {
	t.Helper()
	defaultBefore, taskBefore := defaultWait, taskWait
	defaultWait, taskWait = 300*time.Millisecond, 300*time.Millisecond
	t.Cleanup(func() { defaultWait, taskWait = defaultBefore, taskBefore })
}

// captureReports collects what the scenarios report.
func captureReports(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	before := reportTo
	reportTo = &buffer
	t.Cleanup(func() { reportTo = before })
	return &buffer
}

// pressed returns the IDs of the elements that were pressed, in order.
func pressed(app *fakeApp) []string {
	var ids []string
	for _, a := range app.actions() {
		if a.Action == "press" {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

func expectPressed(t *testing.T, app *fakeApp, want ...string) {
	t.Helper()
	if got := pressed(app); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("pressed %v, want %v (all actions: %+v)", got, want, app.actions())
	}
}

func TestNavigatePressesTheButtonOfTheNavigation(t *testing.T) {
	s, app := newSession(t,
		node("20", "", "list", ""),
		// The same key deeper in the page is no page of the navigation.
		keyed(node("21", "20", "button", "SDKs", press...), float64(1)),
		keyed(node("10", "", "button", "SDKs", press...), float64(1)),
		keyed(node("11", "", "button", "Plugins", press...), float64(2)),
	)
	s.Navigate(SDKs)
	s.Navigate(Plugins)
	expectPressed(t, app, "10", "11")
}

func TestPressKeyPressesTheFirstElementWithTheKey(t *testing.T) {
	s, app := newSession(t,
		keyed(node("1", "", "text", "Dark"), "dark"),
		keyed(node("2", "", "button", "Dark", press...), "dark"),
		keyed(node("3", "", "button", "Light", press...), "light"),
	)
	s.PressKey("dark")
	expectPressed(t, app, "2")
}

func TestClickButtonSkipsWhatIsNoButtonOrCannotBePressed(t *testing.T) {
	s, app := newSession(t,
		node("1", "", "text", "Install"),
		node("2", "", "button", "Install", "focus"),
		node("3", "", "heading", "Install", "press"),
		node("4", "", "button", "Install now", press...),
		node("5", "", "button", "Install", press...),
	)
	s.ClickButton("Install")
	expectPressed(t, app, "5")
}

func TestClickItemTakesTheItemThatStartsWithTheTitle(t *testing.T) {
	s, app := newSession(t,
		node("1", "", "listItem", "Search installed plugins"),
		node("2", "", "button", "Show nodejs", press...),
		keyed(node("3", "", "button", "nodejs", press...), "sdk:nodejs"),
	)
	s.ClickItem("nodejs")
	expectPressed(t, app, "3")

	// The row of a task is called after its title and its status.
	s, app = newSession(t, keyed(node("1", "", "button", "Install nodejs@24.11.0, Done", press...), "item"))
	s.ClickItem("Install nodejs@24.11.0")
	expectPressed(t, app, "1")

	// A row that is a list item takes the click, too.
	s, app = newSession(t, node("1", "", "listItem", "maven 3.9", "press"))
	s.ClickItem("maven")
	expectPressed(t, app, "1")
}

func TestClickButtonNearPicksTheButtonOfTheRow(t *testing.T) {
	s, app := newSession(t,
		keyed(node("1", "", "listItem", "2.0.0 LTS Install"), "av:2.0.0"),
		node("2", "1", "text", "2.0.0"),
		node("3", "1", "button", "Install", press...),
		keyed(node("4", "", "listItem", "1.0.0 Install"), "av:1.0.0"),
		node("5", "4", "text", "1.0.0"),
		node("6", "4", "button", "Install", press...),
	)
	s.ClickButtonNear("1.0.0", "Install")
	s.ClickButtonNear("2.0.0", "Install")
	expectPressed(t, app, "6", "3")
}

func field(id, parent, label string) automation.Node {
	return node(id, parent, "textField", label, "focus", "setValue")
}

// hinted is a field that has a hint and no label.
func hinted(id, hint string) automation.Node {
	n := field(id, "", "")
	n.Placeholder = hint
	return n
}

func TestFillFindsTheField(t *testing.T) {
	cases := map[string]struct {
		label string
		nodes []automation.Node
		want  string
	}{
		"by its label": {"Filter versions", []automation.Node{
			field("1", "", "Other"), field("2", "", "Filter versions"),
		}, "2"},
		"by its hint": {"Source", []automation.Node{hinted("1", "Name"), hinted("2", "Source")}, "2"},
		"by the title of its row": {"Version name", []automation.Node{
			node("1", "", "listItem", "Version name"),
			node("2", "1", "text", "Version name"),
			field("3", "1", "1.0.0"),
		}, "3"},
		"by the title of its setting": {"Plugin registry", []automation.Node{
			node("1", "", "scroll", ""),
			node("2", "1", "text", "Plugin registry"),
			field("3", "1", "Official registry"),
		}, "3"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s, app := newSession(t, c.nodes...)
			s.Fill(c.label, "the value", false)
			got := app.actions()
			if len(got) != 1 || got[0] != (act{ID: c.want, Action: "setValue", Text: "the value"}) {
				t.Fatalf("actions = %+v, want setValue of the element %s", got, c.want)
			}
			if events := app.events(); len(events) != 0 {
				t.Fatalf("a fill that does not submit sent %+v", events)
			}
		})
	}
}

func TestFillSubmitsWithEnter(t *testing.T) {
	s, app := newSession(t, field("1", "", "Plugin registry"))
	s.Fill("Plugin registry", "http://127.0.0.1/registry", true)
	if events := app.events(); len(events) != 1 || events[0].Type != "key" || events[0].Key != "Enter" {
		t.Fatalf("events = %+v, want Enter", events)
	}
}

func TestFillFailsWithoutTheFieldAndWhenTheApplicationRefuses(t *testing.T) {
	shortWaits(t)
	s, _ := newSession(t, field("1", "", "Other"))
	if message, failed := failsWith(t, func() { s.Fill("Missing", "x", false) }); !failed || !strings.Contains(message, `the field "Missing"`) {
		t.Fatalf("a missing field = %q, %v", message, failed)
	}

	s, app := newSession(t, field("1", "", "Name"))
	app.onAct = func(*fakeApp, act) *automation.Error {
		return &automation.Error{Code: "failed", Message: "not editable"}
	}
	if message, failed := failsWith(t, func() { s.Fill("Name", "x", false) }); !failed || !strings.Contains(message, "not editable") {
		t.Fatalf("a refused fill = %q, %v", message, failed)
	}
}

func TestTapDialogPressesTheButtonsOfTheDialog(t *testing.T) {
	for _, role := range []string{"dialog", "alertDialog"} {
		t.Run(role, func(t *testing.T) {
			s, app := newSession(t,
				keyed(node("1", "", "button", "Add plugin", press...), "confirm"),
				node("2", "", role, "Remove?", "press"),
				keyed(node("3", "2", "button", "Cancel", press...), "cancel"),
				keyed(node("4", "2", "button", "Remove", press...), "confirm"),
			)
			s.TapDialog(true)
			s.TapDialog(false)
			expectPressed(t, app, "4", "3")
		})
	}
}

func TestTapDialogFailsWithoutADialog(t *testing.T) {
	shortWaits(t)
	s, _ := newSession(t, keyed(node("1", "", "button", "Add", press...), "confirm"))
	if message, failed := failsWith(t, func() { s.TapDialog(true) }); !failed || !strings.Contains(message, "a dialog") {
		t.Fatalf("got %q, %v", message, failed)
	}
}

func TestChooseMenuItemOpensTheMenuAndPressesTheItem(t *testing.T) {
	for _, role := range []string{"button", "menuItem"} {
		t.Run(role, func(t *testing.T) {
			s, app := newSession(t,
				keyed(node("1", "", "button", "More actions for 1.0.0", press...), "more"),
				node("2", "", "button", "Uninstall", press...),
			)
			app.onAct = func(app *fakeApp, a act) *automation.Error {
				if a.ID == "1" {
					app.set(
						keyed(node("1", "", "button", "More actions for 1.0.0", press...), "more"),
						// The same label outside the menu is not the item.
						node("2", "", "button", "Uninstall", press...),
						node("5", "", "popup", ""),
						node("6", "5", role, "Uninstall", press...),
					)
				}
				return nil
			}
			s.ChooseMenuItem("More actions for 1.0.0", "Uninstall")
			expectPressed(t, app, "1", "6")
		})
	}
}

func TestPressScrollsAnElementOutOfViewFirst(t *testing.T) {
	offscreen := node("1", "", "button", "Go", "press", "scrollIntoView")
	offscreen.States = []string{"offscreen"}
	s, app := newSession(t, offscreen)
	s.ClickButton("Go")
	got := app.actions()
	if len(got) != 2 || got[0].Action != "scrollIntoView" || got[1].Action != "press" || got[0].ID != "1" || got[1].ID != "1" {
		t.Fatalf("actions = %+v, want the element scrolled into view and pressed", got)
	}
}

func TestPressReadsTheElementAgainWhenAFrameChangedIt(t *testing.T) {
	s, app := newSession(t, node("1", "", "button", "Go", press...))
	calls := 0
	app.onAct = func(app *fakeApp, a act) *automation.Error {
		calls++
		if calls == 1 {
			// The frame that was drawn between the reading and the action gave the
			// button another ID.
			app.set(node("2", "", "button", "Go", press...))
			return &automation.Error{Code: "no_element", Message: "no element 1"}
		}
		return nil
	}
	s.ClickButton("Go")
	expectPressed(t, app, "1", "2")
}

func TestPressGivesUpAfterThreeAttemptsOrAtOtherFailures(t *testing.T) {
	s, app := newSession(t, node("1", "", "button", "Go", press...))
	app.onAct = func(*fakeApp, act) *automation.Error {
		return &automation.Error{Code: "no_element", Message: "no element"}
	}
	if message, failed := failsWith(t, func() { s.ClickButton("Go") }); !failed || !strings.Contains(message, `press the button "Go"`) || !strings.Contains(message, "no_element") {
		t.Fatalf("got %q, %v", message, failed)
	}
	expectPressed(t, app, "1", "1", "1")

	s, app = newSession(t, node("1", "", "button", "Go", press...))
	app.onAct = func(*fakeApp, act) *automation.Error {
		return &automation.Error{Code: "failed", Message: "boom"}
	}
	if message, failed := failsWith(t, func() { s.ClickButton("Go") }); !failed || !strings.Contains(message, "boom") {
		t.Fatalf("got %q, %v", message, failed)
	}
	expectPressed(t, app, "1")
}

func TestWaitForTaskSeesTheSnackBarAndTheRowOfTheTask(t *testing.T) {
	cases := map[string]automation.Node{
		"the snack bar":             node("1", "", "status", "Install nodejs@24.11.0: done"),
		"the row on the Tasks page": node("1", "", "button", "Install nodejs@24.11.0, Done", press...),
	}
	for name, done := range cases {
		t.Run(name, func(t *testing.T) {
			s, app := newSession(t, node("1", "", "button", "Install nodejs@24.11.0, Running", press...))
			time.AfterFunc(250*time.Millisecond, func() { app.set(done) })
			s.WaitForTask("Install nodejs@24.11.0")
		})
	}
}

func TestWaitForTaskFailsWhenTheTaskFailsOrDoesNotFinish(t *testing.T) {
	shortWaits(t)
	cases := map[string]struct {
		node automation.Node
		want string
	}{
		"the snack bar":             {node("1", "", "status", "Install nodejs@24.11.0: failed\nvfox exited with 1"), "Install nodejs@24.11.0: failed: vfox exited with 1"},
		"the row on the Tasks page": {node("1", "", "button", "Install nodejs@24.11.0, Failed", press...), "Install nodejs@24.11.0, Failed"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s, _ := newSession(t, c.node)
			if message, failed := failsWith(t, func() { s.WaitForTask("Install nodejs@24.11.0") }); !failed || message != c.want {
				t.Fatalf("got %q, %v, want %q", message, failed, c.want)
			}
		})
	}
	// Another task finishing is not this one finishing.
	s, _ := newSession(t, node("1", "", "status", "Use nodejs@22.21.0: done"))
	if message, failed := failsWith(t, func() { s.WaitForTask("Install nodejs@24.11.0") }); !failed || !strings.Contains(message, `the task "Install nodejs@24.11.0" to finish`) {
		t.Fatalf("got %q, %v", message, failed)
	}
}

func TestCheckNoTaskFailed(t *testing.T) {
	s, _ := newSession(t, node("1", "", "status", "Add plugin x: done"), node("2", "", "button", "Add plugin x, Done", press...))
	s.CheckNoTaskFailed()

	for _, failed := range []automation.Node{
		node("1", "", "status", "Install x: failed\nboom"),
		node("1", "", "button", "Install x, Failed", press...),
	} {
		s, _ := newSession(t, node("0", "", "status", "Use y: done"), failed)
		if message, ok := failsWith(t, s.CheckNoTaskFailed); !ok || !strings.HasPrefix(message, "Install x") {
			t.Fatalf("got %q, %v", message, ok)
		}
	}
}

func TestWaitsSeeWhatTheWindowShowsLater(t *testing.T) {
	s, app := newSession(t, node("1", "", "text", "Loading"))
	time.AfterFunc(250*time.Millisecond, func() {
		app.set(node("1", "", "text", "872 versions"), node("2", "", "text", "Install nodejs@24.11.0 success!"))
	})
	s.WaitForText("872 versions")
	s.WaitForTextContaining("nodejs@24.11.0 success")
	time.AfterFunc(250*time.Millisecond, func() { app.set(node("1", "", "text", "1 version")) })
	s.WaitForTextGone("872 versions")
	if tree := s.WaitUntil("anything", time.Second, func(automation.Tree) bool { return true }); len(tree.Nodes) != 1 {
		t.Fatalf("WaitUntil returned %+v, want the tree that satisfied it", tree)
	}
}

func TestWaitsFailAfterTheirTime(t *testing.T) {
	shortWaits(t)
	s, _ := newSession(t, node("1", "", "text", "Loading"))
	for want, wait := range map[string]func(){
		`the text "872 versions"`:       func() { s.WaitForText("872 versions") },
		`a text with "success"`:         func() { s.WaitForTextContaining("success") },
		`the text "Loading" to go away`: func() { s.WaitForTextGone("Loading") },
		"Timed out waiting for the last thing": func() {
			s.WaitUntil("the last thing", 150*time.Millisecond, func(automation.Tree) bool { return false })
		},
	} {
		if message, failed := failsWith(t, wait); !failed || !strings.Contains(message, want) {
			t.Errorf("got %q, %v, want a timeout waiting for %s", message, failed, want)
		}
	}
}

func TestWaitForPollsAConditionAboutSomethingElse(t *testing.T) {
	s, _ := newSession(t)
	polls := 0
	s.WaitFor("the third poll", 5*time.Second, func() bool { polls++; return polls == 3 })
	if polls != 3 {
		t.Fatalf("polled %d times", polls)
	}
	if message, failed := failsWith(t, func() { s.WaitFor("never", 150*time.Millisecond, func() bool { return false }) }); !failed || message != "Timed out waiting for never" {
		t.Fatalf("got %q, %v", message, failed)
	}
	close(s.exited)
	if message, failed := failsWith(t, func() { s.WaitFor("never", time.Hour, func() bool { return false }) }); !failed || !strings.Contains(message, "has ended while waiting for never") {
		t.Fatalf("got %q, %v", message, failed)
	}
}

func TestSettleWaitsForTheFramesAndIgnoresAWindowThatNeverRests(t *testing.T) {
	s, app := newSession(t)
	s.Settle()
	app.idle = &automation.Error{Code: "timeout", Message: "window 1 still draws"}
	s.Settle()
	app.idle = &automation.Error{Code: "failed", Message: "boom"}
	if message, failed := failsWith(t, s.Settle); !failed || message != "idle: failed: boom" {
		t.Fatalf("got %q, %v", message, failed)
	}
	if err := os.WriteFile(s.Log, []byte("fatal: no display\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	close(s.exited)
	message, failed := failsWith(t, s.Settle)
	if !failed || !strings.Contains(message, "the application has ended") || !strings.Contains(message, "The application wrote:\nfatal: no display") {
		t.Fatalf("got %q, %v, want the end of the application with what it wrote", message, failed)
	}
}

func TestTreeExplainsAnApplicationThatEnded(t *testing.T) {
	s, app := newSession(t)
	app.treeFailure = &automation.Error{Code: "no_window", Message: "the app has no window"}
	if message, failed := failsWith(t, func() { s.Tree() }); !failed || message != "tree: no_window: the app has no window" {
		t.Fatalf("got %q, %v", message, failed)
	}
	if err := os.WriteFile(s.Log, []byte("panic: boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	close(s.exited)
	message, failed := failsWith(t, func() { s.Tree() })
	if !failed || !strings.Contains(message, "the application has ended") || !strings.Contains(message, "The application wrote:\npanic: boom") {
		t.Fatalf("got %q, %v", message, failed)
	}
	// What the application wrote is cut to its end.
	if err := os.WriteFile(s.Log, []byte(strings.Repeat("x", 6000)+"THE END"), 0o644); err != nil {
		t.Fatal(err)
	}
	if text := s.explain(os.ErrClosed).Error(); !strings.Contains(text, "…xxx") || !strings.HasSuffix(text, "THE END") || len(text) > 4200 {
		t.Fatalf("explain returned %d bytes, ending in %q", len(text), text[len(text)-10:])
	}
	// Nothing to add when it wrote nothing.
	if err := os.WriteFile(s.Log, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.explain(os.ErrClosed); err.Error() != os.ErrClosed.Error() {
		t.Fatalf("explain = %v, want the error as it is", err)
	}
}

func TestCheckFailsWithItsDescription(t *testing.T) {
	s, _ := newSession(t)
	s.Check(true, "never seen")
	if message, failed := failsWith(t, func() { s.Check(false, "the sky is blue") }); !failed || message != "Check failed: the sky is blue" {
		t.Fatalf("got %q, %v", message, failed)
	}
}

func TestRunReportsThePassOfAScenario(t *testing.T) {
	out := captureReports(t)
	s, _ := newSession(t)
	if err := s.Run("demo", func(s *Session) { s.Step("one") }); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "e2e: scenario demo started\ne2e: one\ne2e: scenario demo passed in 0 s\n" {
		t.Fatalf("reported %q", got)
	}
}

func TestRunReportsAFailureWithWhatTheWindowShows(t *testing.T) {
	out := captureReports(t)
	s, _ := newSession(t, node("1", "", "heading", "Tasks"), keyed(node("2", "", "button", "Clear finished", press...), "clear"))
	err := s.Run("demo", func(s *Session) { s.Fail("the %s is wrong", "page") })
	if err == nil || err.Error() != "the page is wrong" {
		t.Fatalf("got %v, want the failure", err)
	}
	for _, want := range []string{
		"e2e: scenario demo started\n",
		"e2e: scenario demo FAILED: the page is wrong\n",
		"e2e: on screen:\nheading \"Tasks\"\nbutton [clear] \"Clear finished\"\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the report lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out.String(), "passed") {
		t.Errorf("a failed scenario passed:\n%s", out)
	}
}

func TestRunTurnsAPanicIntoAFailure(t *testing.T) {
	out := captureReports(t)
	s, _ := newSession(t)
	var nothing []int
	err := s.Run("demo", func(*Session) { _ = nothing[3] })
	if err == nil || !strings.Contains(err.Error(), "index out of range") {
		t.Fatalf("got %v, want the panic as an error", err)
	}
	if !strings.Contains(out.String(), "FAILED: runtime error: index out of range") {
		t.Fatalf("the panic was not reported:\n%s", out)
	}
}

func TestRunReportsTheEndOfTheApplicationAndALongScreen(t *testing.T) {
	out := captureReports(t)
	s, app := newSession(t)
	var many []automation.Node
	for i := range 200 {
		many = append(many, node(fmt.Sprint(i+1), "", "text", fmt.Sprintf("line %d", i)))
	}
	app.set(many...)
	_ = s.Run("long", func(s *Session) { s.Fail("look") })
	if !strings.Contains(out.String(), "line 149\"\n… and 50 more\n") || strings.Contains(out.String(), "line 150") {
		t.Errorf("a long screen is not cut at 150 lines:\n%s", out)
	}

	out.Reset()
	app.treeFailure = &automation.Error{Code: "failed", Message: "no tree"}
	_ = s.Run("unreadable", func(s *Session) { s.Fail("look") })
	if !strings.Contains(out.String(), "the window cannot be read: tree: failed: no tree") {
		t.Errorf("an unreadable window is not reported:\n%s", out)
	}

	out.Reset()
	if err := os.WriteFile(s.Log, []byte("segmentation fault\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	close(s.exited)
	_ = s.Run("ended", func(s *Session) { s.Fail("look") })
	if !strings.Contains(out.String(), "the application has ended; it wrote:\nsegmentation fault\n") {
		t.Errorf("the end of the application is not reported:\n%s", out)
	}
}

func TestPauseWaits(t *testing.T) {
	s, _ := newSession(t)
	start := time.Now()
	s.Pause(40 * time.Millisecond)
	if time.Since(start) < 40*time.Millisecond {
		t.Fatal("Pause returned early")
	}
}

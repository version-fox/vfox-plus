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

package screens

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// homePaired returns the frame tester of the window of a phone that is paired with the
// computer fd, with its connection running and the home tab on the screen.
func homePaired(t *testing.T, fd *pairDesktop) *ui.Tester {
	t.Helper()
	root := pairSavedRoot(t, fd.server.URL, fd.certificate, true)
	pairStartService(t, root)
	tt := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, tt, "the connection", func() bool { return root.Service().Status().Session == app.Paired })
	return tt
}

// TestTheHomeTabShowsTheSDKsInUse checks the title, the version of vfox and the SDKs in
// use: their current version, and not the SDKs that are not in use.
func TestTheHomeTabShowsTheSDKsInUse(t *testing.T) {
	fd := newPairDesktop(t)
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, "vfox 0.6.0")
	for _, text := range []string{"vfox+", "In use", "go", "1.22.0"} {
		if !hasText(tt, text) {
			t.Errorf("the home tab has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	if hasText(tt, "node") {
		t.Error("the home tab lists an SDK that is not in use")
	}
	pairSavePicture(t, tt, "home-paired")
}

// TestTheHomeTabSaysWhenNoSDKIsInUse checks the empty state of the list.
func TestTheHomeTabSaysWhenNoSDKIsInUse(t *testing.T) {
	fd := newPairDesktop(t)
	fd.inventory = `{"vfox":{"availability":"ready","version":"0.6.0"},` +
		`"installed":{"loading":false,"value":[{"Name":"node","Versions":["20.1.0"],"Current":"","HasCurrent":false}]},` +
		`"registry":{"loading":false,"value":[]},"config":{"loading":false,"value":[]},"complete":true}`
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "activeEmpty"))
	pairSavePicture(t, tt, "home-empty")
}

// TestTheHomeTabSaysWhenVfoxIsMissing checks the subtitle of a computer whose vfox has
// no version.
func TestTheHomeTabSaysWhenVfoxIsMissing(t *testing.T) {
	fd := newPairDesktop(t)
	fd.inventory = `{"vfox":{"availability":"notFound"},` +
		`"installed":{"loading":false,"value":[]},"registry":{"loading":false,"value":[]},` +
		`"config":{"loading":false,"value":[]},"complete":true}`
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "vfoxMissing"))
}

// TestTheHomeTabShowsAFailedInventoryInRed checks that a computer that does not answer
// the inventory says so, in place of the list.
func TestTheHomeTabShowsAFailedInventoryInRed(t *testing.T) {
	fd := newPairDesktop(t)
	fd.inventoryStatus = 502
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, noticeText(app.KindUnavailable))
	pairSavePicture(t, tt, "home-error")
}

// TestTheHealthOfTheSDKsOffersItsActions checks the findings of the health check: an
// SDK without its receipt is adopted, and one that drifted is installed again, which
// both start tasks; the other findings only say what they are.
func TestTheHealthOfTheSDKsOffersItsActions(t *testing.T) {
	fd := newPairDesktop(t)
	fd.doctor = `{"ok":false,"findings":[` +
		`{"id":"sdk.no-receipt","severity":"warning","sdk":"go","version":"1.21.0","message":"go 1.21.0 has no receipt"},` +
		`{"id":"sdk.drift","severity":"warning","sdk":"node","version":"20.1.0","message":"node 20.1.0 differs from its receipt"},` +
		`{"id":"vfox.path","severity":"info","sdk":"","version":"","message":"vfox is not first on the path"}` +
		`]}`
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, "Adopt")
	for _, text := range []string{"Reinstall", "go 1.21.0 has no receipt", "vfox is not first on the path"} {
		if !hasText(tt, text) {
			t.Errorf("the health check has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	if err := tt.Click("Adopt"); err != nil {
		t.Fatalf("Click(Adopt): %v", err)
	}
	pairWaitUntil(t, tt, "the task of the adoption", func() bool { return len(fd.tasksStarted()) == 1 })
	if got := fd.tasksStarted()[0]; got.Operation != remote.RequestDoctorAdopt || got.Sdk != "go" || got.Version != "1.21.0" {
		t.Errorf("the adoption started %+v, want doctorAdopt of go 1.21.0", got)
	}
	if err := tt.Click("Reinstall"); err != nil {
		t.Fatalf("Click(Reinstall): %v", err)
	}
	pairWaitUntil(t, tt, "the task of the reinstall", func() bool { return len(fd.tasksStarted()) == 2 })
	if got := fd.tasksStarted()[1]; got.Operation != remote.RequestInstall || !got.Reinstall || got.Sdk != "node" {
		t.Errorf("the reinstall started %+v, want install with reinstall of node", got)
	}
	pairSavePicture(t, tt, "home-health")
}

// TestAHealthyVfoxSaysSo checks the report without findings.
func TestAHealthyVfoxSaysSo(t *testing.T) {
	fd := newPairDesktop(t)
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "doctorNoProblems"))
}

// TestTheCommandCardCopiesTheLineOfVfox checks the line that puts vfox in the shell, and
// the copy: the clipboard holds the line, and the button shows the copy for a while.
func TestTheCommandCardCopiesTheLineOfVfox(t *testing.T) {
	fd := newPairDesktop(t)
	tt := homePaired(t, fd)
	pairWaitFor(t, tt, homeCommand)
	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("Click(Copy): %v", err)
	}
	tt.Frame()
	if got := tt.Clipboard(); got != homeCommand {
		t.Errorf("the clipboard holds %q, want %q", got, homeCommand)
	}
}

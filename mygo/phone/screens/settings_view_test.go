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
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// settingsWindow returns the window of a phone paired with f, on the settings tab.
func settingsWindow(t *testing.T, f *tsFake) (*Root, *ui.Tester) {
	t.Helper()
	root := tsRoot(t, f)
	root.App().SelectTab(app.TabSettings)
	tt := ui.NewTester(root.View, 480, 900)
	// Text typed into a field goes to the window that has the keyboard.
	tt.SetFocused(true)
	return root, tt
}

// settingsWaitCall draws the frames until the computer has a change that starts with
// prefix (a method and a path, and the body after them), and fails at the deadline.
func settingsWaitCall(t *testing.T, tt *ui.Tester, f *tsFake, prefix string) string {
	t.Helper()
	var found string
	tsWaitUntil(t, tt, "the computer to receive "+prefix, func() bool {
		for _, call := range f.tsRecorded() {
			if strings.HasPrefix(call, prefix) {
				found = call
				return true
			}
		}
		return false
	})
	return found
}

// settingsServer is the computer of the settings tests: a remote that is off, and a
// configuration of two keys, one of them empty.
func settingsServer(t *testing.T) *tsFake {
	f := tsNewFake(t)
	f.settings = remote.AppSettings{
		VfoxExecutable:   "/usr/bin/vfox",
		Language:         remote.LanguageEnglish,
		ExtraEnvironment: map[string]string{"HTTP_PROXY": "http://proxy:3128"},
	}
	f.config = []remote.ConfigEntry{
		{Key: "proxy.url", Value: ""},
		{Key: "legacyVersionFile.enable", Value: "true"},
	}
	f.web = remote.WebState{Port: 8443, Fingerprint: "AB:CD:EF:01"}
	return f
}

// settingsListening makes the computer of a settings test accept other devices, on
// its public address and one more.
func settingsListening(f *tsFake) {
	f.web.Remote = true
	f.web.Listening = true
	f.web.PublicAddress = "https://10.0.0.2:8443"
	f.web.Addresses = []string{"https://192.168.1.5:8443"}
}

// settingsWaitLoaded waits until the computer has answered the settings, which the
// fields take their values from.
func settingsWaitLoaded(t *testing.T, root *Root) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, ok := root.Service().Settings().Value(); ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the settings of the computer were not loaded")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// settingsWaitSections draws the frames until the reads of the settings tab have
// answered and the frame shows them: the connection, whose last row is "Quit", and
// vfox's configuration, whose first key is legacyVersionFile.enable. The reads are
// separate requests, and each answer changes the layout below it. A test that scrolls
// or clicks before all of them have answered aims at positions that move, and a wait
// for one section says nothing about the others. The tab's transition is over when
// the wait returns, so the layout does not change under the test's next step either.
func settingsWaitSections(t *testing.T, tt *ui.Tester) {
	t.Helper()
	tsWaitText(t, tt, "Quit")
	tsWaitText(t, tt, "legacyVersionFile.enable")
	tsSettle(tt)
}

// settingsCheckSections checks that the settings tab shows each of its sections, as the
// computer answered them. A test calls it once settingsWaitSections has returned.
func settingsCheckSections(t *testing.T, tt *ui.Tester) {
	t.Helper()
	for _, want := range []string{
		"Other devices", "Quit", "Application", "vfox config", "Interface language", "Environment variables",
		"简体中文", "English", "true", "—",
	} {
		if !tsHas(tt, want) {
			t.Errorf("the settings have no %q; the frame shows %q", want, tt.Texts())
		}
	}
}

func TestSettingsShowTheComputerAndItsSettings(t *testing.T) {
	f := settingsServer(t)
	_, tt := settingsWindow(t, f)
	settingsWaitSections(t, tt)
	settingsCheckSections(t, tt)
	tsShot(t, tt, "settings-closed")
}

// TestTheSettingsAreShownWhenTheConnectionAnswersLast holds the connection's answer until
// the configuration is shown, so that the answers come in the other order. The test used
// to check the connection's rows as soon as the configuration had answered, and it failed
// whenever the connection answered later, which a busy computer does.
func TestTheSettingsAreShownWhenTheConnectionAnswersLast(t *testing.T) {
	f := settingsServer(t)
	release := f.tsHold(t, "/api/web")
	_, tt := settingsWindow(t, f)
	tsWaitText(t, tt, "legacyVersionFile.enable")
	if tsHas(tt, "Quit") {
		t.Fatal("the connection is shown while its answer is held")
	}
	release()
	settingsWaitSections(t, tt)
	settingsCheckSections(t, tt)
}

func TestTheExecutableFieldSavesWhatTheComputerHolds(t *testing.T) {
	f := settingsServer(t)
	root, tt := settingsWindow(t, f)
	settingsWaitLoaded(t, root)
	tsWaitText(t, tt, "vfox executable")
	settingsWaitSections(t, tt)
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("Click(Save): %v", err)
	}
	if call := settingsWaitCall(t, tt, f, "PUT /api/settings"); !strings.Contains(call, `{"vfoxExecutable":"/usr/bin/vfox"}`) {
		t.Errorf("the executable was saved as %q, want the one the computer holds", call)
	}
}

func TestOtherDevicesShowTheCodeOfThePublicAddress(t *testing.T) {
	f := settingsServer(t)
	settingsListening(f)
	_, tt := settingsWindow(t, f)
	tsWaitText(t, tt, "Show code")
	tsShot(t, tt, "settings-remote-on")
	if err := tt.Click("Show code"); err != nil {
		t.Fatalf("Click(Show code): %v", err)
	}
	tsWaitText(t, tt, "Each link works once, for five minutes.")
	tsWaitText(t, tt, "Scan this code with the Android app.")
	if call := settingsWaitCall(t, tt, f, "POST /api/web/pairing"); !strings.Contains(call, "https://10.0.0.2:8443") {
		t.Errorf("the link names %q, want the public address first", call)
	}
	tsShot(t, tt, "settings-pairing-dialog")
	if err := tt.Click("192.168.1.5:8443"); err != nil {
		t.Fatalf("Click(the other address): %v", err)
	}
	if call := settingsWaitCall(t, tt, f, "POST /api/web/pairing {\"host\":\"https://192.168.1.5:8443\"}"); call == "" {
		t.Error("the other address was not asked for")
	}
	if err := tt.Click("Link and certificate"); err != nil {
		t.Fatalf("Click(Link and certificate): %v", err)
	}
	tsWaitText(t, tt, "AB:CD:EF:01")
	tsShot(t, tt, "settings-pairing-details")
}

func TestOtherDevicesShowWhyTheyCannotConnect(t *testing.T) {
	f := settingsServer(t)
	f.web.Remote = true
	f.web.Error = "bind: address already in use"
	_, tt := settingsWindow(t, f)
	tsWaitText(t, tt, "Other devices cannot connect: bind: address already in use")
	if tsHas(tt, "Show code") {
		t.Error("the pairing code is offered while the connection fails")
	}
	tsShot(t, tt, "settings-remote-error")
}

func TestTurningOtherDevicesOffAsksFirstAndSaysWhatItCosts(t *testing.T) {
	f := settingsServer(t)
	settingsListening(f)
	_, tt := settingsWindow(t, f)
	tsWaitText(t, tt, "Show code")
	if err := tt.Click("Other devices"); err != nil {
		t.Fatalf("Click(Other devices): %v", err)
	}
	tsWaitText(t, tt, "Stop other devices?")
	if !tt.HasText("this phone") {
		t.Errorf("the dialog does not say that this phone is disconnected too; the frame shows %q", tt.Texts())
	}
	for _, call := range f.tsRecorded() {
		if strings.HasPrefix(call, "PUT /api/web") {
			t.Fatalf("the access was changed before the user confirmed: %q", call)
		}
	}
	tsShot(t, tt, "settings-remote-off-confirm")
	if err := tt.Click("Stop"); err != nil {
		t.Fatalf("Click(Stop): %v", err)
	}
	settingsWaitCall(t, tt, f, `PUT /api/web {"remote":false}`)
	tsWaitUntil(t, tt, "the code to go", func() bool { return !tsHas(tt, "Show code") })
}

func TestThePortOutOfRangeIsSaidAndNotSaved(t *testing.T) {
	f := settingsServer(t)
	settingsListening(f)
	root, tt := settingsWindow(t, f)
	settingsWaitLoaded(t, root)
	tsWaitText(t, tt, "Show code")
	if err := tt.Click("1024 to 65535"); err != nil {
		t.Fatalf("Click(the port field): %v", err)
	}
	tt.Type("9")
	tsWaitText(t, tt, "Use a port from 1024 to 65535.")
	tt.Key(0, ui.KeyEnter)
	tsShot(t, tt, "settings-port-error")
	time.Sleep(20 * time.Millisecond)
	for _, call := range f.tsRecorded() {
		if strings.HasPrefix(call, "PUT /api/web") {
			t.Fatalf("an out-of-range port was sent: %q", call)
		}
	}
	for range 5 {
		tt.Key(0, ui.KeyBackspace)
	}
	tt.Type("9000")
	tt.Key(0, ui.KeyEnter)
	settingsWaitCall(t, tt, f, `PUT /api/web {"port":9000}`)
}

func TestTheConfigurationEditSavesEvenAnEmptyValue(t *testing.T) {
	f := settingsServer(t)
	_, tt := settingsWindow(t, f)
	settingsWaitSections(t, tt)
	tsWaitText(t, tt, "proxy.url")
	// The configuration is below the other sections, which the page scrolls past.
	tt.Scroll(240, 450, 0, 3000)
	tsSettle(tt)
	tsShot(t, tt, "settings-config")
	if err := tt.Click("Actions"); err != nil {
		t.Fatalf("Click(Actions): %v", err)
	}
	tsSettle(tt)
	if err := tt.Click("Edit"); err != nil {
		t.Fatalf("Click(Edit): %v", err)
	}
	tsWaitText(t, tt, "Change the value. An empty value is saved as it is.")
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("Click(Save): %v", err)
	}
	if call := settingsWaitCall(t, tt, f, "PUT /api/config"); !strings.Contains(call, `"key":"proxy.url","value":""`) {
		t.Errorf("the configuration was saved as %q, want the empty value of proxy.url", call)
	}
}

func TestTheApplicationLanguageIsSentToTheComputer(t *testing.T) {
	f := settingsServer(t)
	root, tt := settingsWindow(t, f)
	settingsWaitLoaded(t, root)
	settingsWaitSections(t, tt)
	tsWaitText(t, tt, "Application language")
	tt.Scroll(240, 450, 0, 600)
	tsSettle(tt)
	// The first "简体中文" is the application's; the screens' one is further down.
	if err := tt.Click("简体中文"); err != nil {
		t.Fatalf("Click(简体中文): %v", err)
	}
	if call := settingsWaitCall(t, tt, f, "PUT /api/settings"); !strings.Contains(call, `"language":"simplifiedChinese"`) {
		t.Errorf("the application language was sent as %q, want simplifiedChinese", call)
	}
}

func TestTheScreenLanguageIsSavedOnThePhoneAndAppliedAtOnce(t *testing.T) {
	f := settingsServer(t)
	root, _ := settingsWindow(t, f)
	var preference i18n.Preference = i18n.PreferenceSystem
	root.settingsSetScreenLanguage(kit.Ctx{}, &preference, i18n.PreferenceChinese)
	if preference != i18n.PreferenceChinese || i18n.Current() != i18n.Chinese {
		t.Errorf("the screens answer %s after the choice of zh, want zh", i18n.Current())
	}
	if got := loadUILanguage(); got != i18n.PreferenceChinese {
		t.Errorf("the saved preference is %q, want zh", got)
	}
}

func TestQuittingTheComputerAsksFirst(t *testing.T) {
	f := settingsServer(t)
	_, tt := settingsWindow(t, f)
	tsWaitText(t, tt, "Quit vfox+")
	if err := tt.Click("Quit"); err != nil {
		t.Fatalf("Click(Quit): %v", err)
	}
	tsWaitText(t, tt, "Quit vfox+?")
	for _, call := range f.tsRecorded() {
		if strings.HasPrefix(call, "POST /api/quit") {
			t.Fatal("the computer quit before the user confirmed")
		}
	}
	tsShot(t, tt, "settings-quit-confirm")
	if err := tt.Click("Quit"); err != nil {
		t.Fatalf("Click(the confirmation Quit): %v", err)
	}
	settingsWaitCall(t, tt, f, "POST /api/quit")
	tsWaitText(t, tt, "vfox+ is quitting…")
}

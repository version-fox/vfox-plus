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

package ui

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// textWith finds a text on the screen that starts with prefix.
func (ta *testApp) textWith(prefix string) (string, bool) {
	for _, text := range ta.texts() {
		if strings.HasPrefix(text, prefix) {
			return text, true
		}
	}
	return "", false
}

func (ta *testApp) mustTextWith(prefix string) string {
	ta.t.Helper()
	text, ok := ta.textWith(prefix)
	if !ok {
		ta.t.Fatalf("no text that starts with %q on screen; there is %q", prefix, ta.texts())
	}
	return text
}

// openSettings goes to the settings, scrolled to the web interface.
func (ta *testApp) openSettings() {
	ta.t.Helper()
	ta.button("Settings")
	ta.mustText("Web interface")
}

// listItem is the item of a list, if the user can reach it.
func (ta *testApp) listItem(name string) (*kit.Node, bool) { return ta.find(kit.RoleListItem, name) }

// updateSettings changes the settings, as the page of a browser would.
func (ta *testApp) updateSettings(change func(*settings.AppSettings)) {
	ta.t.Helper()
	ta.app.updateSettings(change)
	ta.settle()
}

func (ta *testApp) waitFor(description string, condition func(pair.Status) bool) {
	ta.t.Helper()
	ta.until(description, func() bool { return condition(ta.web.Status()) })
	ta.settle()
}

func TestTheWebInterfaceIsOffUntilTheUserTurnsItOnAndThenItOpensInTheBrowser(t *testing.T) {
	ta := defaultApp(t)
	ta.openSettings()
	ta.mustText("Use vfox+ in a browser on this computer. Turn it on, then choose Open in browser.")
	if _, ok := ta.listItem("Open in browser"); ok {
		t.Fatal("there is no browser to open while the web interface is off")
	}
	if status := ta.web.Status(); status.Running {
		t.Fatalf("the web interface is off by default: %v", status)
	}

	ta.item("Web interface")
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	if !ta.app.model.Settings.Settings().WebInterface {
		t.Error("the switch is the setting")
	}
	status := ta.web.Status()
	ta.mustText("Serving at " + status.LocalURL)
	if !strings.HasPrefix(status.LocalURL, "http://127.0.0.1:") {
		t.Errorf("this computer is served on the loopback interface: %q", status.LocalURL)
	}

	ta.buttonNear("Open in browser", "Open")
	if len(ta.desktop.opened) != 1 || !strings.HasPrefix(ta.desktop.opened[0], status.LocalURL+"/auth/connect/") {
		t.Fatalf("the browser gets a link that pairs it with the server of this computer, got %q", ta.desktop.opened)
	}
	ta.buttonNear("Open in browser", "Copy link")
	if len(ta.desktop.copied) != 1 || !strings.HasPrefix(ta.desktop.copied[0], status.LocalURL+"/auth/connect/") || ta.desktop.copied[0] == ta.desktop.opened[0] {
		t.Fatalf("a link of its own, for the clipboard, got %q", ta.desktop.copied)
	}
	ta.mustText("Copied to the clipboard")
	ta.snapshot("settings-web")

	ta.item("Web interface")
	ta.waitFor("the web interface to stop", func(status pair.Status) bool { return !status.Running })
	if _, ok := ta.listItem("Open in browser"); ok {
		t.Error("nothing to open once the web interface is off")
	}
	ta.mustText("Use vfox+ in a browser on this computer. Turn it on, then choose Open in browser.")
}

func TestTheWebInterfaceSaysWhyItCannotBeServed(t *testing.T) {
	ta := defaultApp(t)
	ta.openSettings()
	// Nothing to open a browser on: the links of a server that is not there are errors, not crashes.
	ta.app.openWeb()
	ta.app.copyWebLink()
	ta.settle()
	ta.mustText("Cannot make a link: the web interface is off")
	if len(ta.desktop.opened) != 0 || len(ta.desktop.copied) != 0 {
		t.Errorf("no link, so nothing opened or copied: %q %q", ta.desktop.opened, ta.desktop.copied)
	}
}

func TestTheLightweightModeAndTheDevicesNeedTheWebInterface(t *testing.T) {
	ta := defaultApp(t)
	ta.openSettings()
	for _, name := range []string{"Lightweight mode", "Other devices"} {
		item, ok := ta.listItem(name)
		if !ok {
			t.Fatalf("no item %q: %q", name, ta.texts())
		}
		if item.Click != nil {
			t.Errorf("%s cannot be switched while the web interface is off", name)
		}
	}
	if count := strings.Count(strings.Join(ta.texts(), "\n"), "Turn on the web interface first."); count != 2 {
		t.Errorf("both say what they need, got %d: %q", count, ta.texts())
	}
	ta.item("Web interface")
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	for _, name := range []string{"Lightweight mode", "Other devices"} {
		if item, _ := ta.listItem(name); item == nil || item.Click == nil {
			t.Errorf("%s can be switched now", name)
		}
	}
	if _, ok := ta.textWith("Turn on the web interface first."); ok {
		t.Error("the hint is gone with the reason for it")
	}
	ta.item("Lightweight mode")
	if !ta.app.model.Settings.Settings().Lightweight {
		t.Error("the switch is the setting")
	}
	// The application stays only while it has something to keep up.
	ta.mustText("Quit vfox+")
	ta.item("Web interface")
	ta.waitFor("the web interface to stop", func(status pair.Status) bool { return !status.Running })
	if _, ok := ta.listItem("Quit vfox+"); ok {
		t.Error("with the web interface off, closing the window quits, and there is nothing else to quit")
	}
}

func TestClosingTheWindowQuitsUnlessTheLightweightModeKeepsTheApplicationRunning(t *testing.T) {
	ta := defaultApp(t)
	a := ta.app
	a.openWindow()
	if len(ta.desktop.windows) != 1 || a.window() == nil {
		t.Fatalf("the window is open: %v", ta.desktop.windows)
	}

	// By default the window is the application.
	a.windowsClosed()
	if ta.desktop.quits != 1 {
		t.Fatalf("closing the window quits, got %d", ta.desktop.quits)
	}
	// So it does with the lightweight mode on and nothing to keep up.
	ta.updateSettings(func(s *settings.AppSettings) { s.Lightweight = true })
	a.windowsClosed()
	if ta.desktop.quits != 2 || a.window() == nil {
		t.Fatalf("without the web interface there is nothing to keep running, got %d quits", ta.desktop.quits)
	}

	// With the web interface up, the application goes on without its window.
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface = true })
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	a.navigate(destSettings)
	a.pages.portText = "stale"
	a.post(func() {})
	a.windowsClosed()
	if ta.desktop.quits != 2 {
		t.Fatalf("the application keeps running, got %d quits", ta.desktop.quits)
	}
	if a.window() != nil || !a.released {
		t.Fatal("the window is gone")
	}
	if a.shell.destination != destHome || a.pages.portText != "" || len(a.pending) != 0 {
		t.Errorf("what the window held is gone with it: %+v %+v %d", a.shell, a.pages, len(a.pending))
	}
	// What is meant for the window has no place to go, and does not pile up.
	for range 3 {
		a.post(func() { t.Error("nothing runs for a window that is not there") })
	}
	if len(a.pending) != 0 {
		t.Errorf("the work for a window that is gone was kept: %d", len(a.pending))
	}
	a.invalidate() // No window to ask for a frame.
	if !ta.web.Status().Running {
		t.Error("the web interface is what the application kept running for")
	}

	// Starting the application again opens the window again, in the state of a new start.
	a.showWindow()
	if len(ta.desktop.windows) != 2 || a.window() == nil || a.released {
		t.Fatalf("the window comes back: %v", ta.desktop.windows)
	}
	// And with a window there, it is brought to the front instead of opening another.
	a.showWindow()
	if len(ta.desktop.windows) != 2 || ta.desktop.windows[1].shown.Load() != 1 || ta.desktop.windows[1].focus.Load() != 1 {
		t.Errorf("one window, in front: %+v", ta.desktop.windows[1])
	}
	a.invalidate()
	if ta.desktop.windows[1].invalidated.Load() == 0 {
		t.Error("the window gets the frames that are asked for")
	}
	a.settleLanguage()
}

// settleLanguage lets the language set the title of the window, as a change of it does.
func (a *App) settleLanguage() {
	a.l10n.Store(nil)
	a.syncLanguage()
}

func TestTheTitleOfTheWindowFollowsTheLanguage(t *testing.T) {
	ta := defaultApp(t)
	ta.app.openWindow()
	ta.app.settleLanguage()
	if title := ta.desktop.windows[0].windowTitle(); title != "vfox+" {
		t.Errorf("got %q", title)
	}
}

func TestOtherDevicesAreAllowedOnlyAfterAConfirmation(t *testing.T) {
	ta := defaultApp(t)
	port := testsupport.FreePort(t)
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface, s.WebPort = true, port })
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	ta.openSettings()
	ta.mustText("Let phones and other computers on your network connect with a pairing link, over HTTPS. Only turn this on in a network you trust.")
	if _, ok := ta.listItem("Port"); ok {
		t.Fatal("the port is the business of whoever allows other devices")
	}

	ta.item("Other devices")
	ta.mustText("Allow other devices?")
	ta.button("Cancel")
	if ta.app.model.Settings.Settings().WebRemote || ta.web.Status().Remote {
		t.Fatal("nobody is let in without a yes")
	}
	ta.item("Other devices")
	ta.button("Allow")
	ta.waitFor("other devices to be able to connect", func(status pair.Status) bool { return status.Remote })
	ta.mustText("Listening on port " + strconv.Itoa(port))
	for _, name := range []string{"Port", "Pair a phone", "Certificate"} {
		if _, ok := ta.listItem(name); !ok {
			t.Errorf("no %q once other devices can connect: %q", name, ta.texts())
		}
	}
	ta.mustText("Made by vfox+, signed by itself")
	ta.mustText("SHA-256 fingerprint")
	fingerprint := ta.web.Status().Fingerprint
	for _, line := range fingerprintLines(fingerprint, 16) {
		ta.mustText(line)
	}
	if got := strings.Join(fingerprintLines(fingerprint, 16), ":"); got != fingerprint || len(fingerprintLines(fingerprint, 16)) != 2 {
		t.Errorf("a SHA-256 fingerprint is two lines of sixteen bytes, got %q", fingerprintLines(fingerprint, 16))
	}
	ta.snapshot("settings-web-remote")

	// Turning it off needs no confirmation.
	ta.item("Other devices")
	ta.waitFor("other devices to be shut out", func(status pair.Status) bool { return !status.Remote })
	if ta.app.model.Settings.Settings().WebRemote {
		t.Error("the switch is the setting")
	}
	if _, ok := ta.listItem("Pair a phone"); ok {
		t.Error("nobody to pair a phone with")
	}
}

func TestOtherDevicesThatCannotConnectSayWhy(t *testing.T) {
	ta := defaultApp(t)
	taken, err := net.Listen("tcp", "0.0.0.0:0") //nolint:gosec // The test takes the port that the server for other devices wants.
	if err != nil {
		t.Skipf("no interface to take a port on: %v", err)
	}
	defer func() { _ = taken.Close() }()
	port := taken.Addr().(*net.TCPAddr).Port
	if !settings.ValidWebPort(port) {
		t.Skipf("the system gave the port %d, which the settings do not accept", port)
	}
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface, s.WebRemote, s.WebPort = true, true, port })
	ta.waitFor("the verdict on the port", func(status pair.Status) bool { return status.Running && status.RemoteError != "" })
	ta.openSettings()
	ta.mustTextWith("Other devices cannot connect: ")
	for _, name := range []string{"Pair a phone", "Certificate"} {
		if _, ok := ta.listItem(name); ok {
			t.Errorf("%q has no use while other devices cannot connect", name)
		}
	}
	if _, ok := ta.listItem("Port"); !ok {
		t.Error("the port is what to change then")
	}
}

func TestThePortIsChosenInTheSettingsAndOnlyAPortThatCanBeUsed(t *testing.T) {
	ta := defaultApp(t)
	ta.updateSettings(func(s *settings.AppSettings) {
		s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(t)
	})
	ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
	ta.openSettings()
	field, ok := ta.fieldIn("Port")
	if !ok {
		t.Fatalf("no field for the port: %q", ta.texts())
	}
	for _, text := range []string{"80", "70000", "web", ""} {
		ta.fill(field, text, true)
		ta.mustText("Choose a port between 1024 and 65535.")
		if got := ta.app.model.Settings.Settings().WebPort; got == 80 || got == 70000 || got == 0 {
			t.Errorf("%q is no port, and the setting is %d", text, got)
		}
		if ta.app.pages.portText != strconv.Itoa(ta.app.model.Settings.Settings().EffectiveWebPort()) {
			t.Errorf("the field goes back to the port in use, got %q", ta.app.pages.portText)
		}
	}
	// A page of the web interface that changes the port changes what the field shows.
	elsewhere := testsupport.FreePort(t)
	ta.updateSettings(func(s *settings.AppSettings) { s.WebPort = elsewhere })
	if ta.app.pages.portText != strconv.Itoa(elsewhere) {
		t.Errorf("the field follows the settings, got %q want %d", ta.app.pages.portText, elsewhere)
	}
	field, _ = ta.fieldIn("Port")
	next := testsupport.FreePort(t)
	ta.fill(field, " "+strconv.Itoa(next)+" ", true)
	if got := ta.app.model.Settings.Settings().WebPort; got != next {
		t.Fatalf("got %d, want %d", got, next)
	}
	ta.waitFor("the server to move to the port", func(status pair.Status) bool { return status.Remote && status.Port == next })
	ta.mustText("Listening on port " + strconv.Itoa(next))
}

func TestAPhoneIsPairedWithTheCodeOfTheSettings(t *testing.T) {
	ta := defaultApp(t)
	ta.updateSettings(func(s *settings.AppSettings) {
		s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(t)
	})
	ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
	ta.openSettings()
	status := ta.web.Status()
	if len(status.Addresses) == 0 {
		item, _ := ta.listItem("Pair a phone")
		if item == nil {
			t.Fatalf("no item to pair a phone: %q", ta.texts())
		}
		ta.mustText("This computer has no address on a network, so there is nothing to scan.")
		return
	}
	ta.mustTextWith("Reachable at https://" + net.JoinHostPort(status.Addresses[0], strconv.Itoa(status.Port)))
	ta.button("Show code")
	ta.mustText("Pair a phone")
	link := func(address string) string {
		return "https://" + net.JoinHostPort(address, strconv.Itoa(status.Port)) + "/auth/connect/"
	}
	first := ta.mustTextWith(link(status.Addresses[0]))
	ta.mustTextWith("Scan the code with the Android app")
	if ta.app.pages.pairing == nil || len(ta.app.pages.pairing.matrix) < 29 {
		t.Fatalf("the code is drawn from a matrix of modules: %+v", ta.app.pages.pairing)
	}
	ta.snapshot("pair-phone")
	ta.button("Copy link")
	if len(ta.desktop.copied) != 1 || ta.desktop.copied[0] != first {
		t.Errorf("the link of the code is the one copied, got %q", ta.desktop.copied)
	}
	// A new code is a new link; the old one still pairs until it is used or expires.
	ta.button("New code")
	if second := ta.mustTextWith(link(status.Addresses[0])); second == first {
		t.Errorf("a new code has a link of its own, got %q twice", first)
	}
	if len(status.Addresses) > 1 {
		ta.button(status.Addresses[1])
		ta.mustTextWith(link(status.Addresses[1]))
	}
	ta.button("Close")
	if ta.app.pages.pairing != nil {
		t.Error("a closed dialog keeps no link")
	}
	if _, ok := ta.textWith("Scan the code with the Android app"); ok {
		t.Error("the dialog is gone")
	}
}

func TestThePairingDialogSaysWhyThereIsNoCode(t *testing.T) {
	ta := defaultApp(t)
	// No server for other devices: nothing to pair with.
	ta.app.showPairing("")
	ta.settle()
	ta.mustText("Cannot make a link: other devices cannot connect")
	if item, ok := ta.find(kit.RoleButton, "Copy link"); !ok || item.Enabled {
		t.Error("there is no link to copy")
	}
	ta.button("Close")
}

func TestTheCertificateOfTheUserIsChosenWithItsKey(t *testing.T) {
	ta := defaultApp(t)
	// One file with the certificate and the key, as the chooser of the test gives the same path twice.
	certFile, keyFile, _, err := pair.EnsureServerCert(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := os.ReadFile(certFile) //nolint:gosec // The test reads the file it just made.
	if err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile(keyFile) //nolint:gosec // Likewise.
	if err != nil {
		t.Fatal(err)
	}
	both := filepath.Join(t.TempDir(), "both.pem")
	if err := os.WriteFile(both, append(certificate, key...), 0o600); err != nil { //nolint:gosec // A file in the temporary directory of the test.
		t.Fatal(err)
	}
	ta.updateSettings(func(s *settings.AppSettings) {
		s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(t)
	})
	ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
	ta.openSettings()
	if _, ok := ta.find(kit.RoleButton, "Use the certificate that vfox+ makes"); ok {
		t.Error("nothing to go back from while the certificate is the one that vfox+ makes")
	}

	ta.desktop.chosen = both
	ta.buttonNear("Certificate", "Browse…")
	ta.until("both choosers", func() bool { return len(ta.desktop.asked) == 2 })
	if first, second := ta.desktop.asked[0], ta.desktop.asked[1]; first.Title != "Certificate file" || second.Title != "Private key file" || len(first.Extensions) == 0 {
		t.Errorf("first the certificate, then its key: %+v", ta.desktop.asked)
	}
	ta.until("the certificate", func() bool { return ta.app.model.Settings.Settings().WebCertificate == both })
	if got := ta.app.model.Settings.Settings(); got.WebKey != both {
		t.Fatalf("got %+v", got)
	}
	ta.waitFor("the server to take the certificate", func(status pair.Status) bool { return status.Remote && !status.SelfSigned })
	ta.mustText("From " + both)
	ta.mustText(fingerprintLines(ta.web.Status().Fingerprint, 16)[0])

	// A certificate that does not load is no reason to keep the one that does not.
	ta.button("Use the certificate that vfox+ makes")
	ta.waitFor("the certificate that vfox+ makes", func(status pair.Status) bool { return status.Remote && status.SelfSigned })
	if got := ta.app.model.Settings.Settings(); got.WebCertificate != "" || got.WebKey != "" {
		t.Errorf("got %+v", got)
	}

	// Giving up in the second chooser changes nothing.
	ta.desktop.chosen = ""
	ta.desktop.asked = nil
	ta.buttonNear("Certificate", "Browse…")
	ta.until("the first chooser", func() bool { return len(ta.desktop.asked) == 1 })
	ta.settle()
	if got := ta.app.model.Settings.Settings(); got.WebCertificate != "" {
		t.Errorf("a certificate without a key is no choice: %+v", got)
	}
}

func TestTheApplicationIsQuitFromTheSettingsWhenTheWindowDoesNotQuitIt(t *testing.T) {
	ta := defaultApp(t)
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface, s.Lightweight = true, true })
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	ta.openSettings()
	ta.mustText("Closing the window keeps vfox+ running. Quit to end it, and the web interface with it.")
	ta.buttonNear("Quit vfox+", "Quit")
	ta.mustText("Quit vfox+?")
	ta.button("Cancel")
	if ta.desktop.quits != 0 {
		t.Fatal("cancelled")
	}
	ta.buttonNear("Quit vfox+", "Quit")
	ta.button("Quit")
	if ta.desktop.quits != 1 {
		t.Errorf("got %d", ta.desktop.quits)
	}
}

func TestAFingerprintBreaksIntoLinesOfWholeBytes(t *testing.T) {
	for _, c := range []struct {
		fingerprint string
		perLine     int
		want        []string
	}{
		{"AA:BB:CC", 3, []string{"AA:BB:CC"}},
		{"AA:BB:CC:DD", 2, []string{"AA:BB", "CC:DD"}},
		{"AA:BB:CC:DD:EE", 2, []string{"AA:BB", "CC:DD", "EE"}},
		{"", 16, []string{""}},
	} {
		if got := fingerprintLines(c.fingerprint, c.perLine); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q in lines of %d: got %q, want %q", c.fingerprint, c.perLine, got, c.want)
		}
	}
}

func TestTheApplicationNeedsAWebInterface(t *testing.T) {
	model := defaultApp(t).app.model
	if _, err := New(Options{Model: model, Environment: map[string]string{}}); err == nil {
		t.Error("the settings switch the web interface; without one there is nothing to switch")
	}
}

func TestOldSettingsOfTheRemoteSwitchKeepTheWebInterface(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.SettingsFromJSON(map[string]interface{}{"remoteAccess": true}))
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, store)
	ta.app.model.Settings.Load()
	ta.web.Apply()
	ta.waitFor("the web interface of the last run", func(status pair.Status) bool { return status.Running })
}

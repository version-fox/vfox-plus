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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const meminfoReport = `Applications Memory Usage (in Kilobytes):
Uptime: 100 Realtime: 100

** MEMINFO in pid 4321 [dev.vfox.vfoxplus] **
                   Pss  Private  Private  SwapPss      Rss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty    Total     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------   ------
  Native Heap     5000     4900        0        0     6000    10000     6000     4000
        TOTAL    81920    60000    12000        0   130048    20000     9000     8000

 App Summary
                       Pss(KB)                        Rss(KB)
 TOTAL PSS:    81920            TOTAL RSS:   130048       TOTAL SWAP PSS:        0
`

// confirmScreen is the screen of the pairing page, with the button that trusts the certificate.
const confirmScreen = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="" class="android.widget.FrameLayout" package="dev.vfox.vfoxplus" clickable="false" enabled="true" bounds="[0,0][1080,2400]">
    <node index="0" text="Pair with 10.0.2.2" class="android.view.View" package="dev.vfox.vfoxplus" clickable="false" enabled="true" bounds="[60,200][1020,300]" />
    <node index="1" text="" content-desc="AA:BB" class="android.view.View" clickable="false" enabled="true" bounds="[60,400][1020,500]" />
    <node index="2" text="Trust and pair" class="android.view.View" package="dev.vfox.vfoxplus" clickable="true" enabled="true" bounds="[60,1800][1020,1960]" />
  </node>
</hierarchy>
`

// unreachableScreen is the dialog of the pairing when the computer did not answer, as the device dumped
// it: the title, the message, the reason that the app gives, and the button that closes it.
const unreachableScreen = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="" class="android.widget.FrameLayout" package="dev.vfox.vfoxplus" clickable="false" enabled="true" bounds="[0,0][1080,2400]">
    <node index="0" text="Trust this computer?" class="android.view.View" clickable="true" bounds="[63,884][1016,1514]">
      <node index="0" text="Trust this computer?" class="android.view.View" clickable="false" bounds="[126,1052][953,1136]" />
      <node index="1" text="The computer did not answer. Check the network, and that vfox+ runs on the computer." class="android.view.View" clickable="false" bounds="[126,1178][953,1283]" />
      <node index="2" text="remote: the request failed: dial tcp 10.0.2.2:8443: i/o timeout" class="android.view.View" clickable="false" bounds="[126,1290][953,1330]" />
      <node index="3" text="Close" class="android.widget.Button" clickable="true" bounds="[775,1346][953,1451]" />
    </node>
  </node>
</hierarchy>
`

// tabsScreen is the home page, with its tab bar at the bottom. The title "Home" is not clickable.
const tabsScreen = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="Home" class="android.view.View" clickable="false" bounds="[0,0][1080,120]" />
  <node index="1" text="Home" class="android.view.View" clickable="true" bounds="[0,2200][216,2400]" />
  <node index="2" text="SDKs" class="android.view.View" clickable="true" bounds="[216,2200][432,2400]" />
  <node index="3" text="Plugins" class="android.view.View" clickable="true" bounds="[432,2200][648,2400]" />
  <node index="4" text="Tasks" class="android.view.View" clickable="true" bounds="[648,2200][864,2400]" />
  <node index="5" text="Settings" class="android.view.View" clickable="true" bounds="[864,2200][1080,2400]" />
</hierarchy>
`

// cleanLog is a log of the app that the pairing wrote to: the info line of the pairing, and the
// messages of the software GL of the emulator, which are not errors.
const cleanLog = `11-20 10:00:00.000  2551  2551 I mygo    : vfoxplus: paired host=10.0.2.2 fingerprint=AA:BB
11-20 10:00:01.000  2551  2551 E mygo    : mygo: s_glBindAttribLocation: bind attrib 0 name position
11-20 10:00:02.000  2551  2551 W HWUI    : Failed to initialize 101010-2 format, error = EGL_SUCCESS
--------- beginning of crash
`

func pairedLogText(host, fingerprint string) string {
	return "11-20 10:00:00.000  2551  2551 I mygo    : vfoxplus: paired host=" + host + " fingerprint=" + fingerprint + "\n"
}

func TestParseMeminfo(t *testing.T) {
	pss, rss, ok := ParseMeminfo(meminfoReport)
	if !ok || pss != 81920 {
		t.Fatalf("pss = %d, %v", pss, ok)
	}
	if rss != 130048 {
		t.Fatalf("rss = %d", rss)
	}
	if _, _, ok := ParseMeminfo("No process found for: dev.vfox.vfoxplus"); ok {
		t.Fatal("parsed an empty report")
	}
}

func TestLine(t *testing.T) {
	if got := Line(81920, 130048); got != "memory (apk-cold): RSS 127 MB, PSS 80 MB in 1 processes" {
		t.Fatalf("got %q", got)
	}
	if got := PairedLine(81920, 130048); got != "memory (apk-paired): RSS 127 MB, PSS 80 MB in 1 processes" {
		t.Fatalf("got %q", got)
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run([]string{}); err == nil {
		t.Fatal("want a usage error")
	}
	err := Run([]string{"--apk", "phone.apk", "--output", "out", "--pair-link-file", "link.txt"})
	if err == nil || !strings.Contains(err.Error(), "--shots") {
		t.Fatalf("pairing without --shots: err = %v", err)
	}
}

func TestDeepLink(t *testing.T) {
	uri, host, err := DeepLink("https://10.0.2.2:8443/auth/connect/abc123")
	if err != nil {
		t.Fatal(err)
	}
	if want := "vfoxplus://pair?link=https%3A%2F%2F10.0.2.2%3A8443%2Fauth%2Fconnect%2Fabc123"; uri != want {
		t.Errorf("uri = %q, want %q", uri, want)
	}
	if host != "10.0.2.2" {
		t.Errorf("host = %q, want 10.0.2.2 without the port", host)
	}
	for _, link := range []string{
		"", "not a link", "http://127.0.0.1:8080/auth/connect/abc",
		"https://10.0.2.2:8443/app", "https://10.0.2.2:8443/auth/connect/abc def", "https://10.0.2.2:8443/auth/connect/x'y",
	} {
		if _, _, err := DeepLink(link); err == nil {
			t.Errorf("DeepLink(%q) succeeded", link)
		} else if link != "" && strings.Contains(err.Error(), link) {
			t.Errorf("the error of DeepLink(%q) names the link: %v", link, err)
		}
	}
}

func TestParseBadging(t *testing.T) {
	out := "package: name='dev.vfox.vfoxplus' versionCode='100' versionName='0.1.0'\n" +
		"sdkVersion:'26'\nlaunchable-activity: name='dev.mygo.MyGoActivity'  label='vfox+' icon=''\n"
	pkg, activity, err := parseBadging(out)
	if err != nil || pkg != "dev.vfox.vfoxplus" || activity != "dev.mygo.MyGoActivity" {
		t.Fatalf("parseBadging = %q, %q, %v", pkg, activity, err)
	}
	if _, _, err := parseBadging("package: name='dev.vfox.vfoxplus'\n"); err == nil || !strings.Contains(err.Error(), "no launchable activity") {
		t.Errorf("an APK without a launcher: err = %v", err)
	}
	if _, _, err := parseBadging("sdkVersion:'26'\n"); err == nil {
		t.Error("an APK without a package name was read")
	}
}

func TestDottedLess(t *testing.T) {
	if !dottedLess("35.0.0", "36.0.0") || dottedLess("36.0.0", "35.0.0") || !dottedLess("9.0.0", "36.0.0") {
		t.Error("build tools are not ordered by their numbers")
	}
}

func TestParseScreen(t *testing.T) {
	screen, err := parseScreen([]byte(confirmScreen))
	if err != nil {
		t.Fatal(err)
	}
	var confirm element
	for _, e := range screen.elements {
		if e.label == "Trust and pair" {
			confirm = e
		}
	}
	if !confirm.clickable {
		t.Fatalf("the confirmation is not clickable: %+v", confirm)
	}
	if x, y := confirm.center(); x != 540 || y != 1880 {
		t.Errorf("centre of the confirmation = %d, %d", x, y)
	}
	found := false
	for _, e := range screen.elements {
		found = found || e.label == "AA:BB" // The description names an element without text.
	}
	if !found {
		t.Error("the element named by its description is missing")
	}
	for name, data := range map[string]string{
		"no element": `<hierarchy rotation="0"/>`,
		"bad bounds": `<hierarchy><node text="x" bounds="[1,2]"/></hierarchy>`,
		"not XML":    `UI hierarchy dumped`,
	} {
		if _, err := parseScreen([]byte(data)); err == nil {
			t.Errorf("%s: parseScreen succeeded", name)
		}
	}
}

func TestLowestClickable(t *testing.T) {
	screen, err := parseScreen([]byte(tabsScreen))
	if err != nil {
		t.Fatal(err)
	}
	// "Home" is on the title too, but only the tab is clickable; the lowest clickable one is taken.
	home, ok := lowestClickable(screen.elements, tabs[0].label)
	if !ok {
		t.Fatal("no clickable Home")
	}
	if x, y := home.center(); x != 108 || y != 2300 {
		t.Errorf("tap of Home = %d, %d", x, y)
	}
	if _, ok := lowestClickable(screen.elements, confirmLabel); ok {
		t.Error("found a confirmation on the tabs")
	}
	if got := labelsOf(screen.elements); !strings.Contains(got, `"Settings"`) || strings.Count(got, `"Home"`) != 1 {
		t.Errorf("labelsOf = %s", got)
	}
}

func TestAppErrors(t *testing.T) {
	if got := appErrors(cleanLog); len(got) != 0 {
		t.Fatalf("appErrors(clean) = %v", got)
	}
	for name, line := range map[string]string{
		"a message of the app on the error output": "11-20 10:00:03.000  2551  2551 E mygo    : panic: nil map",
		"a crash":                     "11-20 10:00:03.000  2551  2551 E AndroidRuntime: FATAL EXCEPTION: main",
		"a fatal line of another tag": "11-20 10:00:03.000  2551  2551 F libc    : Fatal signal 11",
	} {
		if got := appErrors(cleanLog + line + "\n"); len(got) != 1 {
			t.Errorf("%s: appErrors = %v, want one error", name, got)
		}
	}
	// A warning of another tag, or an error of another tag, is not an error of the app.
	if got := appErrors("11-20 10:00:03.000  2551  2551 E SurfaceSyncGroup: Failed to receive transaction ready\n"); len(got) != 0 {
		t.Errorf("an error of the framework counted as the app's: %v", got)
	}
}

func TestAnrs(t *testing.T) {
	text := cleanLog + "11-20 10:00:05.000  1000  1000 E ActivityManager: ANR in dev.vfox.vfoxplus (dev.vfox.vfoxplus/dev.mygo.MyGoActivity)\n"
	if got := anrs(text, "dev.vfox.vfoxplus"); len(got) != 1 {
		t.Errorf("anrs = %v, want one", got)
	}
	if got := anrs(text, "dev.other"); len(got) != 0 {
		t.Errorf("anrs of another package = %v", got)
	}
}

func TestPairedLogLine(t *testing.T) {
	match := pairedLog.FindStringSubmatch(cleanLog)
	if match == nil || match[1] != "10.0.2.2" || match[2] != "AA:BB" {
		t.Fatalf("pairedLog match = %v", match)
	}
	if pairedLog.MatchString("vfoxplus: paired host=10.0.2.2 fingerprint=aa:bb") {
		t.Error("a lowercase fingerprint matched")
	}
}

// fakeDevice answers the adb commands of a session from scripted screens and logs, and keeps the
// commands and the taps that it was asked for. Its clock moves with the waits of the session, so
// that the tests run without waiting.
type fakeDevice struct {
	clock       time.Time
	screens     []string
	logs        []string
	commands    []string
	taps        []string
	screenIndex int
	logIndex    int
	// machine, when it is set, shows the screens that the taps lead to, as the app does, in place of
	// the list of screens in the order that they are dumped.
	machine *screenMachine
}

// screenMachine is a device that shows one screen at a time, and goes to another when it is tapped or when
// the key of Android goes back, as the app does. A picture that it takes holds the name of the screen and the
// theme that it was taken in, so that a test reads what each picture is of.
type screenMachine struct {
	// now is the name of the screen that is shown.
	now string
	// screens holds the dump of each screen by name, and next and back where a tap (its coordinates, as
	// "x y") and the key of Android lead from each of them.
	screens map[string]string
	next    map[string]map[string]string
	back    map[string]string
	// night is the theme of the system.
	night bool
	// seen is the screens that were dumped, in order.
	seen []string
}

// move goes to the screen that a tap or a key leads to from the screen that is shown, if there is one.
func (m *screenMachine) move(table map[string]string, key string) {
	if to, ok := table[key]; ok {
		m.now = to
	}
}

// adb answers one command of the fake device.
func (d *fakeDevice) adb(args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	d.commands = append(d.commands, command)
	switch {
	case args[0] == "install":
		return []byte("Success\n"), nil
	case command == "logcat -c", strings.HasPrefix(command, "shell pm grant "):
		return nil, nil
	case command == "logcat -d -v threadtime":
		if len(d.logs) == 0 {
			return nil, nil
		}
		text := d.logs[min(d.logIndex, len(d.logs)-1)]
		d.logIndex++
		return []byte(text), nil
	case strings.HasPrefix(command, "shell ip "), strings.HasPrefix(command, "shell getprop "),
		strings.HasPrefix(command, "shell ping "), strings.HasPrefix(command, "shell echo | nc "):
		return []byte("answer of the device to: " + strings.TrimPrefix(command, "shell ") + "\n"), nil
	case args[0] == "exec-out" && d.machine != nil:
		return []byte(fmt.Sprintf("\x89PNG fake screen=%s night=%t", d.machine.now, d.machine.night)), nil
	case args[0] == "exec-out":
		return []byte("\x89PNG fake"), nil
	case strings.HasPrefix(command, "shell am start"):
		if d.machine != nil && strings.Contains(command, "android.intent.action.VIEW") {
			// The deep link opens the dialog of the trust.
			d.machine.now = "confirm"
		}
		return []byte("Starting: Intent\nStatus: ok\n"), nil
	case command == "shell cmd uimode night yes", command == "shell cmd uimode night no":
		if d.machine != nil {
			d.machine.night = strings.HasSuffix(command, "yes")
		}
		return nil, nil
	case command == "shell input keyevent 4":
		if d.machine != nil {
			d.machine.move(d.machine.back, d.machine.now)
		}
		return nil, nil
	case strings.HasPrefix(command, "shell settings put global sysui_demo_allowed"),
		strings.HasPrefix(command, "shell am broadcast -a com.android.systemui.demo"):
		return nil, nil
	case strings.HasPrefix(command, "shell dumpsys meminfo"):
		return []byte(meminfoReport), nil
	case strings.HasPrefix(command, "shell uiautomator dump "+screenFile):
		return []byte("UI hierarchy dumped to: " + screenFile + "\n"), nil
	case strings.HasPrefix(command, "shell cat "+screenFile) && d.machine != nil:
		d.machine.seen = append(d.machine.seen, d.machine.now)
		return []byte(d.machine.screens[d.machine.now]), nil
	case strings.HasPrefix(command, "shell cat "+screenFile):
		text := d.screens[min(d.screenIndex, len(d.screens)-1)]
		d.screenIndex++
		return []byte(text), nil
	case strings.HasPrefix(command, "shell input tap "):
		tap := strings.TrimPrefix(command, "shell input tap ")
		d.taps = append(d.taps, tap)
		if d.machine != nil {
			if to, ok := d.machine.next[d.machine.now][tap]; ok {
				d.machine.now = to
			}
		}
		return nil, nil
	}
	return nil, fmt.Errorf("the fake device has no answer to %q", command)
}

// session returns the session of the fake device.
func (d *fakeDevice) session() *session {
	d.clock = time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	return &session{
		adb: d.adb,
		badging: func(aapt2, apk string) (string, string, error) {
			return "dev.vfox.vfoxplus", "dev.mygo.MyGoActivity", nil
		},
		sleep: func(wait time.Duration) { d.clock = d.clock.Add(wait) },
		now:   func() time.Time { return d.clock },
		log:   io.Discard,
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMeasureColdStart(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDevice{}
	c := config{apk: "phone.apk", outputDir: dir, screenshot: filepath.Join(dir, "cold.png"), aapt2: "aapt2", settle: 20 * time.Second}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	if got, want := readText(t, filepath.Join(dir, "memory.txt")), "memory (apk-cold): RSS 127 MB, PSS 80 MB in 1 processes\n"; got != want {
		t.Errorf("memory.txt = %q, want %q", got, want)
	}
	if !slices.Contains(d.commands, "shell am start -S -W -n dev.vfox.vfoxplus/dev.mygo.MyGoActivity") {
		t.Errorf("the cold start is missing: %q", d.commands)
	}
	if _, err := os.Stat(filepath.Join(dir, "memory-paired.txt")); err == nil {
		t.Error("a paired measurement was written without a pairing")
	}
	if _, err := os.Stat(c.screenshot); err != nil {
		t.Errorf("the screenshot of the cold start: %v", err)
	}
}

func TestMeasurePairing(t *testing.T) {
	dir := t.TempDir()
	shots := filepath.Join(dir, "shots")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{
		screens: []string{confirmScreen, tabsScreen},
		logs:    []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")},
	}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	deep := "shell am start -W -a android.intent.action.VIEW -d 'vfoxplus://pair?link=https%3A%2F%2F10.0.2.2%3A8443%2Fauth%2Fconnect%2Ftok123' -p dev.vfox.vfoxplus"
	if !slices.Contains(d.commands, deep) {
		t.Errorf("the deep link is missing: %q", d.commands)
	}
	// The confirmation, then the five tabs, each at the centre of its element.
	if want := []string{"540 1880", "108 2300", "324 2300", "540 2300", "756 2300", "972 2300"}; !slices.Equal(d.taps, want) {
		t.Errorf("taps = %v, want %v", d.taps, want)
	}
	if got, want := readText(t, filepath.Join(dir, "memory-paired.txt")), "memory (apk-paired): RSS 127 MB, PSS 80 MB in 1 processes\n"; got != want {
		t.Errorf("memory-paired.txt = %q, want %q", got, want)
	}
	for _, name := range []string{"tab-home.png", "tab-sdks.png", "tab-plugins.png", "tab-tasks.png", "tab-settings.png", "tab-home.xml", "confirm.xml", "logcat.txt"} {
		if _, err := os.Stat(filepath.Join(shots, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// The pairing link is a secret until it is used: no file of the run holds it.
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path == link {
			return err
		}
		if strings.Contains(readText(t, path), "tok123") {
			t.Errorf("%s holds the pairing link", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMeasurePairingFailsOnAnAppError(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{
		screens: []string{confirmScreen, tabsScreen},
		logs: []string{"", pairedLogText("10.0.2.2", "AA:BB"),
			pairedLogText("10.0.2.2", "AA:BB") + "11-20 10:00:03.000  2551  2551 E mygo    : panic: nil map\n"},
	}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"), aapt2: "aapt2"}
	err := d.session().measure(c)
	if err == nil || !strings.Contains(err.Error(), "panic: nil map") {
		t.Fatalf("measure = %v, want the error of the app", err)
	}
}

func TestMeasurePairingFailsOnTheWrongHost(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{screens: []string{confirmScreen}, logs: []string{pairedLogText("10.0.2.9", "AA:BB")}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"), aapt2: "aapt2"}
	err := d.session().measure(c)
	if err == nil || !strings.Contains(err.Error(), "paired with 10.0.2.9, want 10.0.2.2") {
		t.Fatalf("measure = %v, want the host of the pairing", err)
	}
}

func TestMeasurePairingFailsWithoutTheConfirmation(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	shots := filepath.Join(dir, "shots")
	d := &fakeDevice{screens: []string{tabsScreen}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	err := d.session().measure(c)
	if err == nil || !strings.Contains(err.Error(), "no confirmation button") || !strings.Contains(err.Error(), `"Settings"`) {
		t.Fatalf("measure = %v, want the missing confirmation and the labels of the screen", err)
	}
	if _, statErr := os.Stat(filepath.Join(shots, "confirm-not-found.xml")); statErr != nil {
		t.Errorf("the dump of the screen that has no confirmation: %v", statErr)
	}
}

// TestAFailedPairingLeavesItsEvidence checks what a session that fails leaves in the directory of
// the shots: the screen, the log of the app and what the device says of the network and of the
// computer, which a failure to reach the computer needs, as the log of the app has no error for it.
// The device is asked about the address of the link and not about its token.
func TestAFailedPairingLeavesItsEvidence(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	shots := filepath.Join(dir, "shots")
	d := &fakeDevice{screens: []string{tabsScreen}, logs: []string{"11-20 10:00:00.000  2551  2551 I mygo    : started\n"}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	if err := d.session().measure(c); err == nil {
		t.Fatal("the session without a confirmation did not fail")
	}
	for _, name := range []string{"failure.png", "logcat.txt", "device-network.txt", "confirm-not-found.xml"} {
		if _, err := os.Stat(filepath.Join(shots, name)); err != nil {
			t.Errorf("the failed session left no %s: %v", name, err)
		}
	}
	report := readText(t, filepath.Join(shots, "device-network.txt"))
	for _, want := range []string{"ip route", "ping -c 1 -W 2 10.0.2.2", "nc -w 3 10.0.2.2 8443"} {
		if !strings.Contains(report, want) {
			t.Errorf("the report of the network lacks %q:\n%s", want, report)
		}
	}
	for _, name := range []string{"device-network.txt", "logcat.txt"} {
		if strings.Contains(readText(t, filepath.Join(shots, name)), "tok123") {
			t.Errorf("%s holds the token of the pairing link", name)
		}
	}
	for _, command := range d.commands {
		if strings.Contains(command, "tok123") && !strings.Contains(command, "am start") {
			t.Errorf("the token of the link went into a command of the device: %q", command)
		}
	}
}

func TestAddressOfALink(t *testing.T) {
	for link, want := range map[string]string{
		"https://10.0.2.2:8443/auth/connect/abc":      "10.0.2.2 8443",
		"https://desktop.example:8369/auth/connect/a": "desktop.example 8369",
	} {
		host, port, ok := addressOf(link)
		if !ok || host+" "+port != want {
			t.Errorf("addressOf(%q) = %q %q %t, want %q", link, host, port, ok, want)
		}
	}
	// What cannot be put in a command of the shell, and a link with no port, give no address.
	for _, link := range []string{
		"https://10.0.2.2/auth/connect/abc",
		"https://[::1]:8443/auth/connect/abc",
		"https://10.0.2.2;reboot:8443/auth/connect/abc",
		"https://10.0.2.2:84a3/auth/connect/abc",
		"not a link",
	} {
		if host, port, ok := addressOf(link); ok {
			t.Errorf("addressOf(%q) = %q %q, want no address", link, host, port)
		}
	}
}

// deepLinkCommands counts the commands that open the pairing link on the device.
func deepLinkCommands(d *fakeDevice) int {
	count := 0
	for _, command := range d.commands {
		if strings.HasPrefix(command, "shell am start -W -a android.intent.action.VIEW -d 'vfoxplus://pair?link=") {
			count++
		}
	}
	return count
}

// TestMeasurePairingTriesAgainWhenTheComputerDoesNotAnswer checks the way of a person whose phone was
// not on the network yet: the dialog says that the computer did not answer, the person closes it and opens
// the link again, and the app finds the computer. The run does the same, and says that it did.
func TestMeasurePairingTriesAgainWhenTheComputerDoesNotAnswer(t *testing.T) {
	dir := t.TempDir()
	shots := filepath.Join(dir, "shots")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{
		screens: []string{unreachableScreen, unreachableScreen, confirmScreen, tabsScreen},
		logs:    []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")},
	}
	var output strings.Builder
	s := d.session()
	s.log = &output
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	if err := s.measure(c); err != nil {
		t.Fatal(err)
	}
	if got := deepLinkCommands(d); got != 2 {
		t.Errorf("the link was opened %d times, want 2: once, and again after the computer did not answer", got)
	}
	// The button that closes the dialog, then the confirmation, then the five tabs.
	if want := []string{"864 1398", "540 1880", "108 2300", "324 2300", "540 2300", "756 2300", "972 2300"}; !slices.Equal(d.taps, want) {
		t.Errorf("taps = %v, want %v", d.taps, want)
	}
	if !strings.Contains(output.String(), "the computer did not answer (attempt 1 of 3)") {
		t.Errorf("the output does not say that the run tried again:\n%s", output.String())
	}
	for _, name := range []string{"confirm-unreachable-1.xml", "confirm.xml"} {
		if _, err := os.Stat(filepath.Join(shots, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(shots, "confirm-not-found.xml")); err == nil {
		t.Error("a run that paired kept the screen of a confirmation that was not found")
	}
}

// TestMeasurePairingGivesUpWhenTheComputerNeverAnswers checks that the retries end, with the reason and
// the evidence of every attempt, and not with a run that waits for ever.
func TestMeasurePairingGivesUpWhenTheComputerNeverAnswers(t *testing.T) {
	dir := t.TempDir()
	shots := filepath.Join(dir, "shots")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{screens: []string{unreachableScreen}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	err := d.session().measure(c)
	if err == nil || !strings.Contains(err.Error(), "did not answer in 3 attempts") || !strings.Contains(err.Error(), "i/o timeout") {
		t.Fatalf("measure = %v, want the computer not to have answered in 3 attempts, with the reason that the app gave", err)
	}
	if got := deepLinkCommands(d); got != pairAttempts {
		t.Errorf("the link was opened %d times, want %d", got, pairAttempts)
	}
	for _, name := range []string{"confirm-unreachable-1.xml", "confirm-unreachable-2.xml", "confirm-not-found.xml", "failure.png", "device-network.txt"} {
		if _, err := os.Stat(filepath.Join(shots, name)); err != nil {
			t.Errorf("the failed run left no %s: %v", name, err)
		}
	}
}

// pageScreen is a page above the bar of the tabs, as the device dumps it: a title, the elements in extra, and
// the five tabs at the bottom, each at the centre that tabsScreen gives it.
func pageScreen(title, extra string) string {
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="` + title + `" class="android.view.View" clickable="false" bounds="[0,0][1080,120]" />
` + extra + `
  <node index="1" text="Home" class="android.view.View" clickable="true" bounds="[0,2200][216,2400]" />
  <node index="2" text="SDKs" class="android.view.View" clickable="true" bounds="[216,2200][432,2400]" />
  <node index="3" text="Plugins" class="android.view.View" clickable="true" bounds="[432,2200][648,2400]" />
  <node index="4" text="Tasks" class="android.view.View" clickable="true" bounds="[648,2200][864,2400]" />
  <node index="5" text="Settings" class="android.view.View" clickable="true" bounds="[864,2200][1080,2400]" />
</hierarchy>
`
}

// coverScreen is a screen that has no bar of tabs: a page that opens above them, or a dialog.
func coverScreen(title, extra string) string {
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="` + title + `" class="android.view.View" clickable="false" bounds="[0,0][1080,120]" />
` + extra + `
</hierarchy>
`
}

// guideMachine is the app as the pictures of the guide go through it: the pairing page, the dialog of the
// trust that the deep link opens, the five tabs, the detail of an SDK that the row opens (and that the key of
// Android closes), and the dialog of a plugin that its button opens (and that Cancel closes).
func guideMachine() *screenMachine {
	tabTaps := map[string]string{"108 2300": "home", "324 2300": "sdks", "540 2300": "plugins", "756 2300": "tasks", "972 2300": "settings"}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range tabTaps {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	return &screenMachine{
		now: "pairing",
		screens: map[string]string{
			"pairing":   coverScreen("Pair with the computer", ""),
			"confirm":   confirmScreen,
			"home":      pageScreen("Home", ""),
			"sdks":      pageScreen("SDKs", `  <node index="9" text="nodejs" class="android.view.View" clickable="true" bounds="[40,500][1040,700]" />`),
			"sdk":       coverScreen("nodejs", ""),
			"plugins":   pageScreen("Plugins", `  <node index="9" text="Add from source" class="android.view.View" clickable="true" bounds="[500,200][1000,300]" />`),
			"addplugin": coverScreen("Add a plugin", `  <node index="9" text="Cancel" class="android.widget.Button" clickable="true" bounds="[100,1300][300,1400]" />`),
			"tasks":     pageScreen("Tasks", ""),
			"settings":  pageScreen("Settings", ""),
		},
		next: map[string]map[string]string{
			"confirm":   {"540 1880": "home"},
			"home":      with(nil),
			"sdks":      with(map[string]string{"540 600": "sdk"}),
			"plugins":   with(map[string]string{"750 250": "addplugin"}),
			"tasks":     with(nil),
			"settings":  with(nil),
			"addplugin": {"200 1350": "plugins"},
		},
		back: map[string]string{"sdk": "sdks", "addplugin": "plugins"},
	}
}

func TestMeasurePairingTakesThePicturesOfTheGuide(t *testing.T) {
	dir := t.TempDir()
	pictures := filepath.Join(dir, "pictures")
	shots := filepath.Join(dir, "shots")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := guideMachine()
	d := &fakeDevice{machine: m, logs: []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, pictures: pictures, aapt2: "aapt2"}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}

	// Every picture is of the screen and in the theme that its name says: the one of the trust is of the
	// dialog before its button is tapped, and the dark ones are dark.
	for name, want := range map[string]string{
		"android-pairing":      "screen=pairing night=false",
		"android-dark-pairing": "screen=pairing night=true",
		"android-trust":        "screen=confirm night=false",
		"android-home":         "screen=home night=false",
		"android-sdks":         "screen=sdks night=false",
		"android-plugins":      "screen=plugins night=false",
		"android-tasks":        "screen=tasks night=false",
		"android-settings":     "screen=settings night=false",
		"android-sdk":          "screen=sdk night=false",
		"android-add-plugin":   "screen=addplugin night=false",
		"android-dark-home":    "screen=home night=true",
	} {
		if got := readText(t, filepath.Join(pictures, name+".png")); !strings.Contains(got, want) {
			t.Errorf("%s is %q, want a picture of %q", name, got, want)
		}
	}
	entries, err := os.ReadDir(pictures)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 11 {
		t.Errorf("the directory of the pictures has %d files, want the 11 of the guide", len(entries))
	}

	// The theme of the system is the light one again, and the status bar was put in the demo mode before the
	// first picture.
	if m.night {
		t.Error("the run leaves the system in the dark theme")
	}
	demo, firstShot := slices.Index(d.commands, "shell settings put global sysui_demo_allowed 1"), slices.Index(d.commands, "exec-out screencap -p")
	if demo < 0 || demo > firstShot {
		t.Errorf("the demo mode is at command %d, the first screenshot at %d: want it before", demo, firstShot)
	}
	// The dark theme was asked for twice (the pairing page and the home page), and each time given back.
	var themes []string
	for _, command := range d.commands {
		if strings.HasPrefix(command, "shell cmd uimode night ") {
			themes = append(themes, strings.TrimPrefix(command, "shell cmd uimode night "))
		}
	}
	if want := []string{"yes", "no", "yes", "no"}; !slices.Equal(themes, want) {
		t.Errorf("the themes were %v, want %v", themes, want)
	}
	// The detail of the SDK and the dialog were opened by their elements and closed, in this order: the row, the
	// key of Android, the button, Cancel.
	var steps []string
	for _, tap := range d.taps {
		switch tap {
		case "540 600", "750 250", "200 1350":
			steps = append(steps, tap)
		}
	}
	if want := []string{"540 600", "750 250", "200 1350"}; !slices.Equal(steps, want) {
		t.Errorf("the taps of the pictures were %v, want %v", steps, want)
	}
	if got := slices.Index(d.commands, "shell input keyevent 4"); got < 0 {
		t.Error("the page of the SDK was not closed with the key of Android")
	}
	// What the pictures are of is kept as dumps beside the screenshots, and the memory is still measured.
	for _, name := range []string{"picture-android-sdk.xml", "picture-android-add-plugin.xml", "picture-android-trust.xml"} {
		if _, err := os.Stat(filepath.Join(shots, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "memory-paired.txt")); err != nil {
		t.Errorf("the paired memory was not measured: %v", err)
	}
}

func TestMeasurePairingWithoutPicturesTakesNone(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{machine: guideMachine(), logs: []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"), aapt2: "aapt2"}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	for _, command := range d.commands {
		if strings.Contains(command, "uimode") || strings.Contains(command, "systemui.demo") {
			t.Errorf("a run without pictures changed the theme or the status bar: %q", command)
		}
	}
}

func TestPicturesNeedAPairedSession(t *testing.T) {
	err := Run([]string{"--apk", "phone.apk", "--output", "out", "--pictures", "pictures"})
	if err == nil || !strings.Contains(err.Error(), "--pictures needs --pair-link-file") {
		t.Errorf("Run = %v, want the pictures to need a pairing link", err)
	}
}

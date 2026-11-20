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
		text := d.logs[min(d.logIndex, len(d.logs)-1)]
		d.logIndex++
		return []byte(text), nil
	case args[0] == "exec-out":
		return []byte("\x89PNG fake"), nil
	case strings.HasPrefix(command, "shell am start"):
		return []byte("Starting: Intent\nStatus: ok\n"), nil
	case strings.HasPrefix(command, "shell dumpsys meminfo"):
		return []byte(meminfoReport), nil
	case strings.HasPrefix(command, "shell uiautomator dump "+screenFile):
		return []byte("UI hierarchy dumped to: " + screenFile + "\n"), nil
	case strings.HasPrefix(command, "shell cat "+screenFile):
		text := d.screens[min(d.screenIndex, len(d.screens)-1)]
		d.screenIndex++
		return []byte(text), nil
	case strings.HasPrefix(command, "shell input tap "):
		d.taps = append(d.taps, strings.TrimPrefix(command, "shell input tap "))
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

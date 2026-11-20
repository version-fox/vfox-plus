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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

// meminfoOf is meminfoReport with the PSS and the RSS of a reading that has other totals, in MB.
func meminfoOf(pssMB, rssMB int) string {
	return strings.NewReplacer("81920", fmt.Sprint(pssMB*1024), "130048", fmt.Sprint(rssMB*1024)).Replace(meminfoReport)
}

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

func TestMedian(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []int
		want   int
	}{
		{"none", nil, 0},
		{"one", []int{7}, 7},
		{"an odd number, out of order", []int{95, 30, 32, 31, 33}, 32},
		{"an even number is the mean of the two in the middle", []int{10, 40, 20, 30}, 25},
		{"the mean of the two in the middle is rounded up", []int{10, 20, 21, 90}, 21},
		{"the same value", []int{4, 4, 4, 4}, 4},
	} {
		if got := Median(tc.values); got != tc.want {
			t.Errorf("%s: Median(%v) = %d, want %d", tc.name, tc.values, got, tc.want)
		}
	}
	values := []int{3, 1, 2}
	Median(values)
	if !slices.Equal(values, []int{3, 1, 2}) {
		t.Errorf("Median put its argument in order: %v", values)
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
	// The dialog of the pairing is opened by the link: the check of the key back has none to open without it.
	err = Run([]string{"--apk", "phone.apk", "--output", "out", "--check-back-key"})
	if err == nil || !strings.Contains(err.Error(), "--check-back-key") {
		t.Errorf("--check-back-key without --pair-link-file: err = %v, want the usage", err)
	}
	err = Run([]string{"--apk", "phone.apk", "--output", "out", "--check-input-type"})
	if err == nil || !strings.Contains(err.Error(), "--check-input-type") {
		t.Errorf("--check-input-type without --pair-link-file: err = %v, want the usage", err)
	}
	// A measurement of no start at all reads nothing: that is a mistake, and the median of none is not a number to report.
	for _, runs := range []string{"0", "-1"} {
		err = Run([]string{"--apk", "phone.apk", "--output", "out", "--cold-runs", runs})
		if err == nil || !strings.Contains(err.Error(), "--cold-runs") {
			t.Errorf("--cold-runs %s: err = %v, want the usage", runs, err)
		}
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
	// meminfos are the reports of `dumpsys meminfo`, one for each time that the session asks (the last one
	// again once they are used up); without any, the device answers meminfoReport every time.
	meminfos     []string
	meminfoIndex int
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
	// night is the theme of the system: what was asked for. The system shows it lag screenshots later, as a slow
	// emulator does (shownNight is what a screenshot has, and delay the screenshots that are still to show the old
	// state).
	night, shownNight bool
	lag, delay        int
	// keyboard says that the device shows a soft keyboard when the dialog that adds a plugin opens, as an emulator that has no
	// hardware keyboard does, and ime that it is shown now: it takes the taps that are meant for the buttons of the dialog,
	// and the key of Android that goes back closes it and not the dialog.
	keyboard, ime bool
	// opensKeyboard are the screens that bring the soft keyboard up when a tap leads to them (the focus of a field), where the
	// device has one, and inputTypes are the input types, in hex, that the input method service reports for the field in focus on
	// a screen: none is 0, which it reports when no field has the focus.
	opensKeyboard map[string]bool
	inputTypes    map[string]string
	// strips is how many more dumps have the strip that says that the computer did not answer, over the page, as
	// the app shows it for a moment when its connection is lost, and clean is how many dumps come before them.
	strips, clean int
	// anr, when it is set, is the dialog of the system that says that a process does not answer, which is over every screen
	// and covers it: the dump is of the dialog. A tap on Wait takes it away, unless the process never answers (anrStays).
	anr, anrStays bool
	// seen is the screens that were dumped, in order.
	seen []string
	// front is the activity that Android has in front, "" for the app's own. leavesOnBack are the screens where the key of
	// Android takes the app out of the front, to the launcher (the dialog closing with it), as the app did with a dialog
	// open, and backs are the screens that were shown when the key was pressed, in order.
	front        string
	leavesOnBack map[string]bool
	backs        []string
}

// The activities that the device can have in front, as the report of the device names them.
const (
	appActivity      = "dev.vfox.vfoxplus/dev.mygo.MyGoActivity"
	launcherActivity = "com.google.android.apps.nexuslauncher/.NexusLauncherActivity"
)

// notRespondingScreen is the dialog that Android shows when System UI does not answer, as uiautomator dumps it on the emulator:
// a title and two buttons, whose bounds are those of the dump of the emulator.
const notRespondingScreen = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<hierarchy rotation="0">
  <node index="0" text="System UI isn't responding" class="android.widget.TextView" clickable="false" bounds="[133,1064][947,1135]" />
  <node index="1" text="Close app" class="android.widget.Button" clickable="true" bounds="[70,1174][1010,1300]" />
  <node index="2" text="Wait" class="android.widget.Button" clickable="true" bounds="[70,1300][1010,1426]" />
</hierarchy>
`

// waitTap is where a tap on Wait of the dialog goes.
const waitTap = "540 1363"

// stripNode is the strip of the app that says that the computer did not answer, as the accessibility tree has it.
const stripNode = `  <node index="99" text="The computer did not answer. Check the network, and that vfox+ runs on the computer." class="android.view.View" clickable="false" bounds="[0,206][1080,340]" />
`

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
		m := d.machine
		if m.delay > 0 {
			m.delay--
		}
		if m.delay == 0 {
			m.shownNight = m.night
		}
		return []byte(fmt.Sprintf("\x89PNG fake screen=%s night=%t ime=%t", m.now, m.shownNight, m.ime)), nil
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
			d.machine.delay = d.machine.lag
		}
		return nil, nil
	case command == "shell input keyevent 4":
		if d.machine != nil {
			if d.machine.ime {
				d.machine.ime = false
				return nil, nil
			}
			d.machine.backs = append(d.machine.backs, d.machine.now)
			if d.machine.leavesOnBack[d.machine.now] {
				d.machine.front = launcherActivity
			}
			d.machine.move(d.machine.back, d.machine.now)
		}
		return nil, nil
	case command == "shell dumpsys activity activities":
		front := appActivity
		if d.machine != nil && d.machine.front != "" {
			front = d.machine.front
		}
		return []byte("ACTIVITY MANAGER ACTIVITIES (dumpsys activity activities)\n  topResumedActivity=ActivityRecord{203859552 u0 " + front +
			" t27}\n  ResumedActivity: ActivityRecord{203859552 u0 " + front + " t27}\n"), nil
	case command == "shell dumpsys input_method":
		// The report names the field in focus after the lines of the clients that came before it, which have no input type.
		inputType := "0"
		if d.machine != nil {
			if v, ok := d.machine.inputTypes[d.machine.now]; ok {
				inputType = v
			}
		}
		report := "  mImeWindowVis=0\n  mInputShown=false\n"
		if d.machine != nil && d.machine.ime {
			report = "  mImeWindowVis=3\n  mInputShown=true\n"
		}
		return []byte("      inputType=0x0 imeOptions=0x0 fieldId=0x0 fieldName=null actionId=0 actionLabel=null\n" + report +
			"      curEditorInfo:\n        inputType=0x" + inputType + " imeOptions=0x2000006 privateImeOptions=null\n"), nil
	case strings.HasPrefix(command, "shell dumpsys meminfo"):
		if len(d.meminfos) == 0 {
			return []byte(meminfoReport), nil
		}
		text := d.meminfos[min(d.meminfoIndex, len(d.meminfos)-1)]
		d.meminfoIndex++
		return []byte(text), nil
	case strings.HasPrefix(command, "shell uiautomator dump "+screenFile):
		return []byte("UI hierarchy dumped to: " + screenFile + "\n"), nil
	case strings.HasPrefix(command, "shell cat "+screenFile) && d.machine != nil && d.machine.anr:
		return []byte(notRespondingScreen), nil
	case strings.HasPrefix(command, "shell cat "+screenFile) && d.machine != nil:
		d.machine.seen = append(d.machine.seen, d.machine.now)
		text := d.machine.screens[d.machine.now]
		switch {
		case d.machine.clean > 0:
			d.machine.clean--
		case d.machine.strips > 0:
			d.machine.strips--
			text = strings.Replace(text, "</hierarchy>", stripNode+"</hierarchy>", 1)
		}
		return []byte(text), nil
	case strings.HasPrefix(command, "shell cat "+screenFile):
		text := d.screens[min(d.screenIndex, len(d.screens)-1)]
		d.screenIndex++
		return []byte(text), nil
	case strings.HasPrefix(command, "shell input tap "):
		tap := strings.TrimPrefix(command, "shell input tap ")
		d.taps = append(d.taps, tap)
		if d.machine != nil && d.machine.anr {
			// The dialog is over the screen: the tap is for it, and only Wait does anything.
			if tap == waitTap && !d.machine.anrStays {
				d.machine.anr = false
			}
			return nil, nil
		}
		if d.machine != nil && !d.machine.ime {
			if to, ok := d.machine.next[d.machine.now][tap]; ok {
				d.machine.now = to
				d.machine.ime = d.machine.keyboard && d.machine.opensKeyboard[to]
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
	// One start is what a configuration that does not ask for more takes, and it leaves its whole report and its table.
	if got := readText(t, filepath.Join(dir, "meminfo-cold-1.txt")); got != meminfoReport {
		t.Errorf("meminfo-cold-1.txt = %q, want the report of the device", got)
	}
	want := "cold start 1: RSS 127 MB, PSS 80 MB\nmedian of 1: RSS 127 MB, PSS 80 MB (the PSS of the starts is from 80 to 80 MB)\n"
	if got := readText(t, filepath.Join(dir, "memory-cold-samples.txt")); got != want {
		t.Errorf("memory-cold-samples.txt = %q, want %q", got, want)
	}
	if got := strings.Count(strings.Join(d.commands, "\n"), "shell am start -S -W"); got != 1 {
		t.Errorf("the app was started cold %d times, want 1", got)
	}
}

// The readings of the cold starts in MB, as the device reports them: the first and the last are not the middle one of
// either, and the mean of each is not the median, so a line that takes the first, the last, or the mean is not the line.
var coldReadings = []struct{ pss, rss int }{{30, 100}, {31, 400}, {32, 110}, {33, 120}, {95, 105}}

func TestTheColdStartIsRepeatedAndTheLineIsTheMedian(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDevice{}
	for _, r := range coldReadings {
		d.meminfos = append(d.meminfos, meminfoOf(r.pss, r.rss))
	}
	var log bytes.Buffer
	s := d.session()
	s.log = &log
	c := config{apk: "phone.apk", outputDir: dir, aapt2: "aapt2", settle: 20 * time.Second, coldRuns: len(coldReadings)}
	if err := s.measure(c); err != nil {
		t.Fatal(err)
	}
	// The median of each, of its own: 32 MB of PSS, 110 MB of RSS (no start read both).
	if got, want := readText(t, filepath.Join(dir, "memory.txt")), "memory (apk-cold): RSS 110 MB, PSS 32 MB in 1 processes\n"; got != want {
		t.Errorf("memory.txt = %q, want %q", got, want)
	}
	// Each reading comes from a start of its own, which stops the process first (-S) as a person's start of the app would find it.
	start := "shell am start -S -W -n dev.vfox.vfoxplus/dev.mygo.MyGoActivity"
	starts, reads := 0, 0
	for _, command := range d.commands {
		switch {
		case command == start:
			starts++
		case strings.HasPrefix(command, "shell dumpsys meminfo"):
			// A reading is of the start before it: the start comes first, the settling, the reading.
			reads++
			if reads != starts {
				t.Errorf("reading %d came after %d starts, want one start before each reading", reads, starts)
			}
		}
	}
	if starts != len(coldReadings) || reads != len(coldReadings) {
		t.Errorf("starts = %d and readings = %d, want %d of each", starts, reads, len(coldReadings))
	}
	for i, r := range coldReadings {
		name := fmt.Sprintf("meminfo-cold-%d.txt", i+1)
		if got := readText(t, filepath.Join(dir, name)); got != meminfoOf(r.pss, r.rss) {
			t.Errorf("%s is not the report of the start %d: %q", name, i+1, got)
		}
	}
	want := "cold start 1: RSS 100 MB, PSS 30 MB\ncold start 2: RSS 400 MB, PSS 31 MB\ncold start 3: RSS 110 MB, PSS 32 MB\n" +
		"cold start 4: RSS 120 MB, PSS 33 MB\ncold start 5: RSS 105 MB, PSS 95 MB\n" +
		"median of 5: RSS 110 MB, PSS 32 MB (the PSS of the starts is from 30 to 95 MB)\n"
	if got := readText(t, filepath.Join(dir, "memory-cold-samples.txt")); got != want {
		t.Errorf("memory-cold-samples.txt:\n%s\nwant:\n%s", got, want)
	}
	// The log of the run shows each reading, so that the spread is in the log of CI and not only in a file.
	for _, line := range []string{"cold start 1 of 5: RSS 100 MB, PSS 30 MB", "cold start 5 of 5: RSS 105 MB, PSS 95 MB"} {
		if !strings.Contains(log.String(), line) {
			t.Errorf("the log has no %q: %q", line, log.String())
		}
	}
	// Each start settles for as long as the first: the clock moved by the settling of five starts, at least.
	if got := d.clock.Sub(time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)); got < 5*c.settle {
		t.Errorf("the run waited %s, want %s or more: every start settles", got, 5*c.settle)
	}
}

func TestAnEvenNumberOfColdStartsHasTheMeanOfTheMiddleTwo(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDevice{meminfos: []string{meminfoOf(60, 150), meminfoOf(20, 110), meminfoOf(40, 130), meminfoOf(30, 120)}}
	c := config{apk: "phone.apk", outputDir: dir, aapt2: "aapt2", settle: time.Second, coldRuns: 4}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	// 20 30 40 60: the middle two are 30 and 40; 110 120 130 150: 120 and 130.
	if got, want := readText(t, filepath.Join(dir, "memory.txt")), "memory (apk-cold): RSS 125 MB, PSS 35 MB in 1 processes\n"; got != want {
		t.Errorf("memory.txt = %q, want %q", got, want)
	}
}

func TestAColdStartWhoseReportNeverHasMemoryFailsTheRun(t *testing.T) {
	dir := t.TempDir()
	// The device has no process of the app from the second start on: every try of its reading finds the same.
	d := &fakeDevice{meminfos: []string{meminfoOf(30, 100), "No process found for: dev.vfox.vfoxplus\n"}}
	c := config{apk: "phone.apk", outputDir: dir, aapt2: "aapt2", settle: time.Second, coldRuns: 3}
	err := d.session().measure(c)
	if err == nil || !strings.Contains(err.Error(), "cold start 2") || !strings.Contains(err.Error(), "4 tries") {
		t.Fatalf("err = %v, want one that names the second start and the tries", err)
	}
	// The first reading, and the four tries of the second: the run does not ask for ever.
	if got := strings.Count(strings.Join(d.commands, "\n"), "shell dumpsys meminfo"); got != 1+meminfoTries {
		t.Errorf("the report was asked for %d times, want %d", got, 1+meminfoTries)
	}
	// A line of two readings is not what the report was asked for: no line is written.
	if _, statErr := os.Stat(filepath.Join(dir, "memory.txt")); statErr == nil {
		t.Error("memory.txt was written for a run that failed")
	}
}

// meminfoTimeout is what the service answers when the device is too busy to dump the memory in 10 s, as an emulator is in the minute after
// it booted: no report, and the run of CI that met it stopped.
const meminfoTimeout = "\n*** SERVICE 'meminfo' DUMP TIMEOUT (10000ms) EXPIRED ***\n\n"

func TestAReportThatTimesOutIsAskedForAgain(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDevice{meminfos: []string{meminfoTimeout, meminfoTimeout, meminfoOf(32, 110)}}
	var output strings.Builder
	s := d.session()
	s.log = &output
	c := config{apk: "phone.apk", outputDir: dir, aapt2: "aapt2", settle: time.Second}
	if err := s.measure(c); err != nil {
		t.Fatal(err)
	}
	if got, want := readText(t, filepath.Join(dir, "memory.txt")), "memory (apk-cold): RSS 110 MB, PSS 32 MB in 1 processes\n"; got != want {
		t.Errorf("memory.txt = %q, want %q: the reading of the third try", got, want)
	}
	if got := strings.Count(strings.Join(d.commands, "\n"), "shell dumpsys meminfo"); got != 3 {
		t.Errorf("the report was asked for %d times, want 3", got)
	}
	// The report that was kept is the one that held the memory, not a timeout.
	if got := readText(t, filepath.Join(dir, "meminfo-cold-1.txt")); got != meminfoOf(32, 110) {
		t.Errorf("meminfo-cold-1.txt = %q, want the report of the third try", got)
	}
	if !strings.Contains(output.String(), "holds no memory (try 1 of 4): asking again") {
		t.Errorf("the output does not say that the reading was asked for again:\n%s", output.String())
	}
	// The clock moved by the settling and by the waits between the tries: the tool does not hammer a device that is busy.
	if got := d.clock.Sub(time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)); got < c.settle+2*meminfoPause {
		t.Errorf("the run waited %s, want %s or more", got, c.settle+2*meminfoPause)
	}
}

func TestTheReportOfThePairedSessionThatTimesOutIsAskedForAgain(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{
		screens:  []string{confirmScreen, tabsScreen},
		logs:     []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")},
		meminfos: []string{meminfoOf(30, 100), meminfoTimeout, meminfoOf(57, 136)},
	}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"), aapt2: "aapt2"}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	if got, want := readText(t, filepath.Join(dir, "memory-paired.txt")), "memory (apk-paired): RSS 136 MB, PSS 57 MB in 1 processes\n"; got != want {
		t.Errorf("memory-paired.txt = %q, want %q", got, want)
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
		lag:           3,
		keyboard:      true,
		opensKeyboard: map[string]bool{"addplugin": true},
		now:           "pairing",
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
		back: map[string]string{"confirm": "pairing", "sdk": "sdks", "addplugin": "plugins"},
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
		got := readText(t, filepath.Join(pictures, name+".png"))
		if !strings.Contains(got, want) {
			t.Errorf("%s is %q, want a picture of %q", name, got, want)
		}
		// The device of the test shows a keyboard when the dialog opens, and no picture shows it.
		if strings.Contains(got, "ime=true") {
			t.Errorf("%s is %q, want a picture taken without the keyboard", name, got)
		}
	}
	entries, err := os.ReadDir(pictures)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 11 {
		t.Errorf("the directory of the pictures has %d files, want the 11 of the guide", len(entries))
	}

	// The theme of the system is the light one again. The status bar is left to the system: the demo mode of Android
	// throws an exception in System UI at every `enter` and `exit` on the image of CI, and leaves what listens to it half
	// in and half out, at random, so the tool does not use it (the framing cuts the bar off).
	if m.night {
		t.Error("the run leaves the system in the dark theme")
	}
	for _, command := range d.commands {
		if strings.Contains(command, "systemui.demo") || strings.Contains(command, "sysui_demo") {
			t.Errorf("the run used the demo mode of the status bar: %q", command)
		}
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

// TestAFailedSessionGivesTheThemeBack checks that a run that takes the pictures and fails, here for the lack of a
// confirmation, leaves the theme of the system light, as a run that succeeds does, and keeps the evidence of the failure.
func TestAFailedSessionGivesTheThemeBack(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{screens: []string{tabsScreen}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"),
		pictures: filepath.Join(dir, "pictures"), aapt2: "aapt2"}
	if err := d.session().measure(c); err == nil {
		t.Fatal("the session without a confirmation did not fail")
	}
	var themes []string
	for _, command := range d.commands {
		if strings.HasPrefix(command, "shell cmd uimode night ") {
			themes = append(themes, strings.TrimPrefix(command, "shell cmd uimode night "))
		}
	}
	if want := []string{"yes", "no"}; !slices.Equal(themes, want) {
		t.Errorf("the themes were %v, want %v", themes, want)
	}
	if _, err := os.Stat(filepath.Join(c.shots, "failure.png")); err != nil {
		t.Errorf("the failed run left no screenshot of the failure: %v", err)
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

// screenshotOf is a screenshot 100 px wide: a status bar of statusBarRows rows, grey, with a mark at the left whose
// place is the time that it says, over a page of the colour page, with a mark at the height of the line that the
// page shows.
func screenshotOf(t *testing.T, clock int, page color.Color, line int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, statusBarRows+200))
	for y := 0; y < statusBarRows+200; y++ {
		for x := 0; x < 100; x++ {
			if y < statusBarRows {
				img.Set(x, y, color.RGBA{200, 200, 205, 255})
			} else {
				img.Set(x, y, page)
			}
		}
	}
	for y := 40; y < 80; y++ {
		for x := 4 + 8*clock; x < 10+8*clock; x++ {
			img.Set(x, y, color.RGBA{60, 60, 60, 255})
		}
	}
	for x := 10; x < 60; x++ {
		img.Set(x, statusBarRows+20+line, color.RGBA{60, 60, 60, 255})
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestPageOf(t *testing.T) {
	light, dark := color.RGBA{250, 245, 240, 255}, color.RGBA{20, 16, 10, 255}
	// The status bar is the system's and says the time: a screen is the same screen at another minute.
	if !bytes.Equal(pageOf(screenshotOf(t, 0, light, 0)), pageOf(screenshotOf(t, 5, light, 0))) {
		t.Error("two screenshots of the same page with another time in the status bar are other pages")
	}
	if bytes.Equal(pageOf(screenshotOf(t, 0, light, 0)), pageOf(screenshotOf(t, 0, dark, 0))) {
		t.Error("a light page and a dark one are the same page")
	}
	if bytes.Equal(pageOf(screenshotOf(t, 0, light, 0)), pageOf(screenshotOf(t, 0, light, 30))) {
		t.Error("a page with a line at another height is the same page")
	}
	// A screenshot that is not a picture is all there is of it.
	if got := pageOf([]byte("not a picture")); string(got) != "not a picture" {
		t.Errorf("pageOf of what is not a picture = %q", got)
	}
	// A picture that is no higher than the status bar has no page, and that is not an error.
	flat := image.NewRGBA(image.Rect(0, 0, 4, statusBarRows-10))
	var out bytes.Buffer
	if err := png.Encode(&out, flat); err != nil {
		t.Fatal(err)
	}
	if got := pageOf(out.Bytes()); len(got) != 0 {
		t.Errorf("pageOf of a picture as high as the status bar has %d bytes, want none", len(got))
	}
}

// scriptedScreen is a session whose screenshots are the frames, one for each look, the last one for ever after. It
// counts the looks.
func scriptedScreen(frames [][]byte) (s *session, looks *int, log *strings.Builder) {
	looks, log = new(int), new(strings.Builder)
	clock := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	return &session{
		adb: func(args ...string) ([]byte, error) {
			frame := frames[min(*looks, len(frames)-1)]
			*looks++
			return frame, nil
		},
		sleep: func(wait time.Duration) { clock = clock.Add(wait) },
		now:   func() time.Time { return clock },
		log:   log,
	}, looks, log
}

// pageOfScreenshot is what the page of a screenshot is made of, which the session gives settle as the page before.
func pageOfScreenshot(screenshot []byte) []byte { return pageOf(screenshot) }

// TestTheScreenSettles checks what settle waits for: that the screen has changed from the one before the request, and
// that it has then stayed as it is for calmPolls looks in a row. Neither is enough alone: the system answers a request
// seconds late, so a screen that has not changed yet is not a screen that has settled, and one that has changed once is
// not done while it is still being drawn. The clock of the status bar goes on all the while, and is no part of it.
func TestTheScreenSettles(t *testing.T) {
	light, dark, darker := color.RGBA{250, 245, 240, 255}, color.RGBA{20, 16, 10, 255}, color.RGBA{10, 8, 4, 255}
	// The frames: the old page for longer than a calm takes (the system has not answered yet), a half drawn one that
	// stays for less than a calm takes, and the one that stays. The clock of each frame is another one.
	old := calmPolls + 2
	var frames [][]byte
	for i := range old {
		frames = append(frames, screenshotOf(t, i, light, 0))
	}
	frames = append(frames, screenshotOf(t, old, dark, 0), screenshotOf(t, old+1, dark, 0))
	for i := range calmPolls + 3 {
		frames = append(frames, screenshotOf(t, old+2+i, darker, 0))
	}
	s, looks, log := scriptedScreen(frames)
	s.settle(pageOfScreenshot(screenshotOf(t, 0, light, 0)))
	// The looks at the old page, 2 at the half drawn one, then the one that stays: 1 look that changes it and calmPolls
	// that find it unchanged.
	if want := old + 2 + 1 + calmPolls; *looks != want {
		t.Errorf("settle looked %d times, want %d: it must wait for the page that stays, and not for the clock", *looks, want)
	}
	if strings.Contains(log.String(), "warning") {
		t.Errorf("a screen that settled was reported: %s", log)
	}
}

func TestTheScreenThatNeverChangesIsGivenUpOn(t *testing.T) {
	light := color.RGBA{250, 245, 240, 255}
	s, looks, log := scriptedScreen([][]byte{screenshotOf(t, 0, light, 0)})
	s.settle(pageOfScreenshot(screenshotOf(t, 0, light, 0)))
	if want := int(calmTimeout/calmPoll) + 1; *looks != want {
		t.Errorf("settle looked %d times, want %d: one for each poll of the time that it waits", *looks, want)
	}
	if !strings.Contains(log.String(), "did not settle") || !strings.Contains(log.String(), "changed: false") {
		t.Errorf("a screen that never changed was not reported: %q", log)
	}
}

func TestTheScreenThatNeverStaysIsGivenUpOn(t *testing.T) {
	light, dark, darker := color.RGBA{250, 245, 240, 255}, color.RGBA{20, 16, 10, 255}, color.RGBA{10, 8, 4, 255}
	// A page that is drawn over and over: it changes at every look for longer than the time that settle waits.
	var frames [][]byte
	for i := range int(calmTimeout/calmPoll) + 10 {
		page := color.Color(dark)
		if i%2 == 1 {
			page = darker
		}
		frames = append(frames, screenshotOf(t, i, page, 0))
	}
	s, _, log := scriptedScreen(frames)
	s.settle(pageOfScreenshot(screenshotOf(t, 0, light, 0)))
	if !strings.Contains(log.String(), "did not settle") || !strings.Contains(log.String(), "changed: true") {
		t.Errorf("a screen that never stayed was not reported: %q", log)
	}
}

// pictureOfHome is what a picture needs of a run: the page, and the directories.
func pictureOfHome(t *testing.T, strips int) (*fakeDevice, *session, *strings.Builder, config) {
	t.Helper()
	dir := t.TempDir()
	m := guideMachine()
	m.now, m.strips = "home", strips
	d := &fakeDevice{machine: m}
	var log strings.Builder
	s := d.session()
	s.log = &log
	return d, s, &log, config{pictures: filepath.Join(dir, "pictures"), shots: filepath.Join(dir, "shots")}
}

func screenshots(d *fakeDevice) int {
	n := 0
	for _, command := range d.commands {
		if command == "exec-out screencap -p" {
			n++
		}
	}
	return n
}

// TestAPictureIsTakenAgainWhileTheComputerDoesNotAnswer checks the strip that the app shows when its connection is lost: a
// picture of a page with it is not kept. With three dumps that have the strip, the second attempt still sees it before its
// screenshot and the third is the first that is clean: the last attempt that is allowed, and a picture.
func TestAPictureIsTakenAgainWhileTheComputerDoesNotAnswer(t *testing.T) {
	d, s, log, c := pictureOfHome(t, 3)
	if err := s.picture(c, "android-home"); err != nil {
		t.Fatal(err)
	}
	if got := screenshots(d); got != pictureAttempts {
		t.Errorf("the picture was taken %d times, want %d", got, pictureAttempts)
	}
	if got := strings.Count(log.String(), "taking it again"); got != pictureAttempts-1 {
		t.Errorf("the log says %d times that the picture is taken again, want %d:\n%s", got, pictureAttempts-1, log)
	}
	if _, err := os.Stat(filepath.Join(c.pictures, "android-home.png")); err != nil {
		t.Errorf("the picture was not kept: %v", err)
	}
	// What is kept beside it is the dump of the clean screen, which says what the picture is of.
	if dump := readText(t, filepath.Join(c.shots, "picture-android-home.xml")); strings.Contains(dump, "did not answer") {
		t.Errorf("the dump that is kept has the strip:\n%s", dump)
	}
}

// TestAPictureIsTakenAgainWhenOnlyTheDumpAfterItHasTheStrip is the strip that comes while the screenshot is taken: the dump
// before it was clean, the one after it has the strip, and the screenshot is of a page that has it as well.
func TestAPictureIsTakenAgainWhenOnlyTheDumpAfterItHasTheStrip(t *testing.T) {
	d, s, _, c := pictureOfHome(t, 1)
	d.machine.clean = 1
	if err := s.picture(c, "android-home"); err != nil {
		t.Fatal(err)
	}
	if got := screenshots(d); got != 2 {
		t.Errorf("the picture was taken %d times, want twice: the dump after the first had the strip", got)
	}
}

func TestAPictureThatKeepsShowingTheStripFails(t *testing.T) {
	d, s, _, c := pictureOfHome(t, 1000)
	err := s.picture(c, "android-home")
	if err == nil || !strings.Contains(err.Error(), "shows that the computer did not answer") {
		t.Fatalf("picture = %v, want it to fail where the computer never answers", err)
	}
	if got := screenshots(d); got != pictureAttempts {
		t.Errorf("the picture was taken %d times, want %d and no more", got, pictureAttempts)
	}
}

func TestAPictureOfAScreenThatAnswersIsTakenOnce(t *testing.T) {
	d, s, _, c := pictureOfHome(t, 0)
	if err := s.picture(c, "android-home"); err != nil {
		t.Fatal(err)
	}
	if got := screenshots(d); got != 1 {
		t.Errorf("the picture was taken %d times, want once", got)
	}
}

// TestADialogThatTheKeyboardCoversIsClosed is the dialog that adds a plugin on an emulator that shows a soft keyboard: the keyboard
// takes the first tap, which was meant for Cancel, and the key of Android that goes back closes the keyboard and not the dialog, so
// the dialog needs a second one.
func TestADialogThatTheKeyboardCoversIsClosed(t *testing.T) {
	m := guideMachine()
	m.now, m.ime = "addplugin", true
	d := &fakeDevice{machine: m}
	if err := d.session().closeDialog(); err != nil {
		t.Fatalf("closeDialog = %v, want the dialog closed", err)
	}
	if m.now != "plugins" {
		t.Errorf("the screen is %q, want the page of the plugins that the dialog covered", m.now)
	}
	if got := slices.Index(d.commands, "shell input keyevent 4"); got < 0 {
		t.Error("the key of Android that goes back was not used")
	}
}

func TestADialogThatDoesNotCloseIsAnError(t *testing.T) {
	m := guideMachine()
	m.now, m.ime = "addplugin", true
	m.back = map[string]string{} // nothing goes back from here, and the keyboard takes the taps
	d := &fakeDevice{machine: m}
	err := d.session().closeDialog()
	if err == nil || !strings.Contains(err.Error(), "the dialog is still open") {
		t.Fatalf("closeDialog = %v, want the dialog that stays to be reported", err)
	}
}

// TestAPictureOfTheDialogHasNoKeyboard checks the keyboard that the emulator shows over the dialog: it is closed before the
// screenshot, and the dialog and its field are still there.
func TestAPictureOfTheDialogHasNoKeyboard(t *testing.T) {
	d, s, _, c := pictureOfHome(t, 0)
	d.machine.now, d.machine.ime = "addplugin", true
	if err := s.picture(c, "android-add-plugin"); err != nil {
		t.Fatal(err)
	}
	got := readText(t, filepath.Join(c.pictures, "android-add-plugin.png"))
	if !strings.Contains(got, "screen=addplugin") || !strings.Contains(got, "ime=false") {
		t.Errorf("the picture is %q, want the dialog without the keyboard", got)
	}
}

// TestTheDialogOfAProcessThatDoesNotAnswerIsWaitedOn is System UI that does not answer on a slow emulator: the dialog of the
// system covers the app, and the tab that the tool looks for is not on the screen. The tool taps Wait, as a person does, and
// goes on to the tab.
func TestTheDialogOfAProcessThatDoesNotAnswerIsWaitedOn(t *testing.T) {
	m := guideMachine()
	m.now, m.anr = "home", true
	d := &fakeDevice{machine: m}
	var log strings.Builder
	s := d.session()
	s.log = &log
	if _, err := s.tapWhen(tabs[1].label, tabTimeout, "SDKs tab"); err != nil {
		t.Fatalf("tapWhen = %v, want the tab after the dialog was waited on", err)
	}
	if want := []string{waitTap, "324 2300"}; !slices.Equal(d.taps, want) {
		t.Errorf("the taps were %v, want %v: Wait first, then the tab", d.taps, want)
	}
	if m.now != "sdks" {
		t.Errorf("the screen is %q, want the SDKs tab", m.now)
	}
	if !strings.Contains(log.String(), `the system says "System UI isn't responding": waiting for it`) {
		t.Errorf("the dialog that was waited on is not in the log: %q", log.String())
	}
}

// TestADialogThatNeverGoesIsNotWaitedOnForEver is the process that never answers: the tool taps Wait a few times and then
// takes the screen as it is, which says what is on it.
func TestADialogThatNeverGoesIsNotWaitedOnForEver(t *testing.T) {
	m := guideMachine()
	m.now, m.anr, m.anrStays = "home", true, true
	d := &fakeDevice{machine: m}
	s := d.session()
	_, err := s.tapWhen(tabs[1].label, 20*time.Second, "SDKs tab")
	if err == nil || !strings.Contains(err.Error(), "isn't responding") {
		t.Fatalf("tapWhen = %v, want the dialog that stays to be what the screen has", err)
	}
	for _, tap := range d.taps {
		if tap != waitTap {
			t.Errorf("a tap of %q went to a screen that is covered by the dialog", tap)
		}
	}
	if len(d.taps) == 0 {
		t.Error("Wait was never tapped")
	}
}

// pairedRun runs the paired session of a device, with the checks that set turns on, and returns the device, the output of the
// tool, the directory of its screenshots, and the error of the run.
func pairedRun(t *testing.T, m *screenMachine, set func(*config)) (*fakeDevice, string, string, error) {
	t.Helper()
	dir := t.TempDir()
	shots := filepath.Join(dir, "shots")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{machine: m, logs: []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")}}
	var output strings.Builder
	s := d.session()
	s.log = &output
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: shots, aapt2: "aapt2"}
	set(&c)
	err := s.measure(c)
	return d, output.String(), shots, err
}

// backRun is pairedRun with the key back checked.
func backRun(t *testing.T, m *screenMachine) (*fakeDevice, string, string, error) {
	t.Helper()
	return pairedRun(t, m, func(c *config) { c.checkBack = true })
}

// TestTheKeyBackWithTheDialogOfThePairingOpenLeavesTheAppInFront is the check that the app answers egoist/mygo#2088: the
// key goes back with the dialog of the trust open, the dialog closes, and the app is in front still. The pairing then
// opens the link again, as it does after a dialog that was closed, and goes on.
func TestTheKeyBackWithTheDialogOfThePairingOpenLeavesTheAppInFront(t *testing.T) {
	m := guideMachine()
	d, output, _, err := backRun(t, m)
	if err != nil {
		t.Fatal(err)
	}
	// The key was pressed with the dialog open, and once.
	if want := []string{"confirm"}; !slices.Equal(m.backs, want) {
		t.Errorf("the key back was pressed with %v shown, want %v: the dialog open", m.backs, want)
	}
	// The link, the key, the link again; the check taps nothing, so the taps are those of a pairing without it.
	var steps []string
	for _, command := range d.commands {
		switch {
		case strings.Contains(command, "android.intent.action.VIEW"):
			steps = append(steps, "link")
		case command == "shell input keyevent 4":
			steps = append(steps, "back")
		}
	}
	if want := []string{"link", "back", "link"}; !slices.Equal(steps, want) {
		t.Errorf("the steps were %v, want %v", steps, want)
	}
	if want := []string{"540 1880", "108 2300", "324 2300", "540 2300", "756 2300", "972 2300"}; !slices.Equal(d.taps, want) {
		t.Errorf("taps = %v, want %v", d.taps, want)
	}
	if !strings.Contains(output, "the key back closed the dialog of the pairing, and the app stayed in front") {
		t.Errorf("the output does not say what the check found:\n%s", output)
	}
	if m.front != "" {
		t.Errorf("the run ends with %q in front", m.front)
	}
}

// TestAKeyBackThatTakesTheAppOutOfTheFrontFailsTheRun is the app as it was in 0.10.0 of mygo: the dialog closes and the app
// goes to the launcher with it. The run fails, says which activity is in front, keeps the screen, and does not try the pairing.
func TestAKeyBackThatTakesTheAppOutOfTheFrontFailsTheRun(t *testing.T) {
	m := guideMachine()
	m.leavesOnBack = map[string]bool{"confirm": true}
	d, _, shots, err := backRun(t, m)
	if err == nil || !strings.Contains(err.Error(), "took the app out of the front") || !strings.Contains(err.Error(), launcherActivity) {
		t.Fatalf("measure = %v, want the key back to have taken the app out of the front, with the launcher named", err)
	}
	if _, statErr := os.Stat(filepath.Join(shots, "failure.png")); statErr != nil {
		t.Errorf("the failed run left no screen: %v", statErr)
	}
	if slices.Contains(d.taps, "540 1880") {
		t.Error("the pairing was tried after the app had left")
	}
}

// TestADialogThatTheKeyBackLeavesOpenFailsTheRun is the app that stays in front and keeps its dialog: a person who goes back
// is left with the dialog, so the run fails with what is on the screen.
func TestADialogThatTheKeyBackLeavesOpenFailsTheRun(t *testing.T) {
	m := guideMachine()
	delete(m.back, "confirm")
	_, _, _, err := backRun(t, m)
	if err == nil || !strings.Contains(err.Error(), "still open after the key back") || !strings.Contains(err.Error(), "Trust and pair") {
		t.Fatalf("measure = %v, want the dialog to be still open after the key back, with what the screen has", err)
	}
}

// TestTheKeyBackIsPressedOnlyWhenAskedFor is the session as it was: with no --check-back-key, no key is pressed before
// the pairing, and the activity in front is not read.
func TestTheKeyBackIsPressedOnlyWhenAskedFor(t *testing.T) {
	m := guideMachine()
	dir := t.TempDir()
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(link, []byte("https://10.0.2.2:8443/auth/connect/tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &fakeDevice{machine: m, logs: []string{"", pairedLogText("10.0.2.2", "AA:BB"), pairedLogText("10.0.2.2", "AA:BB")}}
	c := config{apk: "phone.apk", outputDir: dir, pairLinkFile: link, shots: filepath.Join(dir, "shots"), aapt2: "aapt2"}
	if err := d.session().measure(c); err != nil {
		t.Fatal(err)
	}
	if len(m.backs) != 0 {
		t.Errorf("the key back was pressed with %v shown, without being asked for", m.backs)
	}
	if slices.Contains(d.commands, "shell dumpsys activity activities") {
		t.Error("the activity in front was read without the check being asked for")
	}
	if slices.Contains(d.commands, "shell dumpsys input_method") {
		t.Error("the input method service was asked without the check of the input type being asked for")
	}
	if slices.Contains(d.taps, linkFieldTap) {
		t.Error("the field of the pairing link was tapped without the check of the input type being asked for")
	}
}

// TestACheckOfTheKeyBackWithoutADialogIsAnError is a screen that has no dialog of the pairing when the link is opened: the
// run says so, and not that the key back did something.
func TestACheckOfTheKeyBackWithoutADialogIsAnError(t *testing.T) {
	m := guideMachine()
	m.screens["confirm"] = coverScreen("Pair with the computer", "")
	_, _, _, err := backRun(t, m)
	if err == nil || !strings.Contains(err.Error(), "no dialog of the pairing") {
		t.Fatalf("measure = %v, want no dialog of the pairing to have been found", err)
	}
	if len(m.backs) != 0 {
		t.Errorf("the key back was pressed with %v shown, with no dialog to go back from", m.backs)
	}
}

func TestTheActivityInFrontIsRead(t *testing.T) {
	for _, tc := range []struct{ name, report, want string }{
		{"the report of API 29 and later", "  topResumedActivity=ActivityRecord{203859552 u0 dev.vfox.vfoxplus/dev.mygo.MyGoActivity t27}\n", appActivity},
		{"the report before it", "  mResumedActivity: ActivityRecord{1a2b3c u0 com.android.launcher3/.Launcher t2}\n", "com.android.launcher3/.Launcher"},
		{"the launcher", "  ResumedActivity: ActivityRecord{226493392 u0 " + launcherActivity + " t2}\n", launcherActivity},
	} {
		s := &session{adb: func(args ...string) ([]byte, error) { return []byte(tc.report), nil }}
		got, err := s.front()
		if err != nil || got != tc.want {
			t.Errorf("%s: front = %q, %v, want %q", tc.name, got, err, tc.want)
		}
	}
	s := &session{adb: func(args ...string) ([]byte, error) { return []byte("nothing is resumed\n"), nil }}
	if got, err := s.front(); err == nil {
		t.Errorf("a report without an activity in front gave %q", got)
	}
}

// linkFieldNode is the label of the field of the pairing link, as the device dumps it: not clickable, over the line of the field, which
// a tap on it focuses. linkFieldTap is the middle of it.
const (
	linkFieldNode = `  <node index="3" text="Pairing link" class="android.view.View" clickable="false" bounds="[178,537][995,600]" />`
	linkFieldTap  = "586 568"
)

// inputMachine is the device of the guide with the field of the pairing link on the pairing page: a tap on it focuses it, which brings
// the soft keyboard up, and the input method service reports inputType (in hex) for the field in focus.
func inputMachine(inputType string) *screenMachine {
	m := guideMachine()
	page := coverScreen("Pair with the computer", linkFieldNode)
	m.screens["pairing"] = page
	m.screens["pairing-field"] = page
	m.next["pairing"] = map[string]string{linkFieldTap: "pairing-field"}
	m.opensKeyboard["pairing-field"] = true
	m.inputTypes = map[string]string{"pairing-field": inputType}
	return m
}

func inputRun(t *testing.T, m *screenMachine) (*fakeDevice, string, string, error) {
	t.Helper()
	return pairedRun(t, m, func(c *config) { c.checkInputType = true })
}

// TestTheFieldOfThePairingLinkAsksTheKeyboardForAddresses is the field as it is now: text of the keyboard of addresses (0x10), no
// capitalization, and suggestions off (0x80000). The field is tapped, its input type read, and the keyboard that came up closed
// again, and the pairing goes on from there.
func TestTheFieldOfThePairingLinkAsksTheKeyboardForAddresses(t *testing.T) {
	m := inputMachine("80011")
	d, output, _, err := inputRun(t, m)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{linkFieldTap, "540 1880", "108 2300", "324 2300", "540 2300", "756 2300", "972 2300"}; !slices.Equal(d.taps, want) {
		t.Errorf("taps = %v, want %v: the field first, then the pairing", d.taps, want)
	}
	if !strings.Contains(output, "asks for the keyboard of addresses, with no capitalization and no corrections (input type 0x80011)") {
		t.Errorf("the output does not say what the check found:\n%s", output)
	}
	if m.ime {
		t.Error("the keyboard that the focus brought up is still shown")
	}
	// The key that closes the keyboard is pressed once, with the keyboard up, so that the key went to the keyboard and not to the
	// screen: a key back with no keyboard to close could take the app out of the front.
	if got := strings.Count(strings.Join(d.commands, "\n"), "shell input keyevent 4"); got != 1 {
		t.Errorf("the key back was pressed %d times, want once", got)
	}
	if len(m.backs) != 0 {
		t.Errorf("the key back reached the screens %v, want only the keyboard", m.backs)
	}
}

// TestAFieldThatAsksTheKeyboardForTheWrongThingFailsTheRun is the field as it was, and the other ways it can ask wrong: each says what
// was asked, and does not go on to the pairing.
func TestAFieldThatAsksTheKeyboardForTheWrongThingFailsTheRun(t *testing.T) {
	for _, tc := range []struct{ name, inputType, want string }{
		{"the plain field of Android: sentences and corrections", "c001", "does not ask for the keyboard of addresses"},
		{"the keyboard of addresses that capitalizes sentences", "84011", "capitalize"},
		{"the keyboard of addresses that capitalizes words", "82011", "capitalize"},
		{"the keyboard of addresses that capitalizes characters", "81011", "capitalize"},
		{"the keyboard of addresses with the corrections on", "8011", "corrections"},
		{"the keyboard of addresses that does not turn suggestions off", "11", "corrections"},
		{"a field of numbers", "2", "not a text field"},
		{"no field in focus", "0", "reports no field in focus"},
	} {
		m := inputMachine(tc.inputType)
		d, _, _, err := inputRun(t, m)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: measure = %v, want an error with %q", tc.name, err, tc.want)
			continue
		}
		if slices.Contains(d.taps, "540 1880") {
			t.Errorf("%s: the pairing was tried after the field was refused", tc.name)
		}
	}
}

// TestTheKeyboardIsClosedAfterAFieldThatAsksWrong is the failure that leaves a keyboard on the screen: it is closed before the
// run ends, as it is after a field that passes, so that the evidence of the failure is not the keyboard.
func TestTheKeyboardIsClosedAfterAFieldThatAsksWrong(t *testing.T) {
	m := inputMachine("c001")
	if _, _, _, err := inputRun(t, m); err == nil {
		t.Fatal("a field of the wrong input type passed")
	}
	if m.ime {
		t.Error("the keyboard is still shown after the failure")
	}
}

// TestACheckOfTheInputTypeNeedsTheFieldOnTheScreen is a pairing page without the field: the run says so.
func TestACheckOfTheInputTypeNeedsTheFieldOnTheScreen(t *testing.T) {
	m := inputMachine("80011")
	m.screens["pairing"] = coverScreen("Pair with the computer", "")
	_, _, _, err := inputRun(t, m)
	if err == nil || !strings.Contains(err.Error(), "no field of the pairing link") {
		t.Fatalf("measure = %v, want no field of the pairing link", err)
	}
}

// TestTheInputTypeIsReadFromTheFieldInFocus is the report of the service as the device writes it: other clients come before the field in
// focus, with no input type, and the field's is on the line after curEditorInfo.
func TestTheInputTypeIsReadFromTheFieldInFocus(t *testing.T) {
	report := "      inputType=0x0 imeOptions=0x0 fieldId=0x0 fieldName=null actionId=0 actionLabel=null\n" +
		"    mCurAttribute:\n      inputType=0x20001 imeOptions=0x0\n" +
		"      curEditorInfo:\n        inputType=0x80011 imeOptions=0x2000006 privateImeOptions=null\n"
	match := editorInputType.FindStringSubmatch(report)
	if match == nil || match[1] != "80011" {
		t.Fatalf("the input type of the field in focus is %v, want 80011", match)
	}
	if editorInputType.MatchString("      inputType=0x80011 imeOptions=0x0\n") {
		t.Error("an input type that is not the field in focus's was read")
	}
}

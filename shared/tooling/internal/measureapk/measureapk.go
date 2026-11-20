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

// Package measureapk implements `vfxtool measure-apk`: on a booted Android emulator it installs
// the APK of the phone app, cold-starts it, settles, and reads `dumpsys meminfo`, which is the
// memory(apk-cold) line of the memory report. It takes a screenshot of the first screen too.
//
// The reading of one cold start moves from emulator to emulator (the first screen of the same code
// read between 29 and 49 MB of PSS in nine runs of CI), so with --cold-runs N the app is started
// cold N times and the line is the median of the N readings. Every reading is kept: the whole
// report of `dumpsys meminfo` of each start (meminfo-cold-N.txt, to see which category moved), and
// a table of them (memory-cold-samples.txt), beside the line.
//
// With --pair-link-file it then drives the paired session the way a person does: the pairing
// link comes in through the deep link vfoxplus://pair, the button that trusts the certificate is
// tapped, the five tabs are visited with a screenshot and a dump of each, the log of the app is
// checked for errors, and the paired session is measured apart: memory(apk-paired), in
// memory-paired.txt, because the memory report reads one line of its artifact.
//
// With --pictures it takes the pictures of the user guide on the way, in the same session (see
// pictures.go): the pairing page in both themes, the dialog of the trust, the five tabs, the detail of an
// SDK, the dialog that adds a plugin and the home page in the dark theme, in a directory of their own.
//
// With --check-back-key it presses the key back with the dialog of the pairing open, before the pairing, and fails when
// the dialog stays open or the app is taken out of the front (see backKeepsTheApp).
//
// The screens are read from the accessibility tree (uiautomator dump): the native UI of the app
// exposes its elements with their labels and bounds (mygo's SurfaceWindow, the accessibility
// provider). CI runs it from the emulator job of .github/workflows/phone.yml. adb is on the
// PATH; aapt2 is in ANDROID_HOME, or on the PATH.
package measureapk

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The waits of the paired session. Each one ends as soon as its condition holds. The emulator
// draws its first frames slowly (docs/en/dev/memory.md), hence the long ones.
const (
	// confirmTimeout is the wait of one attempt for the dialog of the pairing to show its button or
	// its failure: the app looks for the computer for ten seconds at most.
	confirmTimeout = 45 * time.Second
	pairedTimeout  = 60 * time.Second
	tabTimeout     = 30 * time.Second
	pollInterval   = 2 * time.Second
	tabSettle      = 3 * time.Second
	// pairAttempts is how many times the link is opened when the computer does not answer, and
	// retryPause the wait before the next one.
	pairAttempts = 3
	retryPause   = 5 * time.Second
	// notRespondingTries is how many times in a row the tool taps Wait on that dialog before it takes the screen as it is.
	notRespondingTries = 3
	// screenFile is where uiautomator dumps the accessibility tree of the screen, on the device.
	screenFile = "/sdcard/vfoxplus-measure.xml"
)

// usage is the command line of measure-apk, as its errors print it.
const usage = "Usage: measure-apk --apk FILE --output DIR [--screenshot FILE] [--settle SECONDS] [--cold-runs N] [--aapt2 FILE] [--pair-link-file FILE --shots DIR [--pictures DIR] [--check-back-key]]"

var (
	totalLine = regexp.MustCompile(`(?m)^\s*TOTAL\s+(\d+)`)
	rssField  = regexp.MustCompile(`\bRSS:\s*(\d+)K?`)
	// resumedActivity is the line of `dumpsys activity activities` that names the activity in front, which is
	// `topResumedActivity=ActivityRecord{… u0 package/Activity t12}` on the emulator of CI (API 36) and
	// `mResumedActivity: ActivityRecord{…}` on the releases before.
	resumedActivity = regexp.MustCompile(`ResumedActivity[=:]\s*ActivityRecord\{\S+ u\d+ (\S+)`)
)

// The labels of the paired session, as the phone app shows them in English and in Simplified
// Chinese. The tabs are the five pages of the web interface (mygo/web/src/router.ts), in their
// order. The confirmation is the one button that trusts the certificate and pairs the phone.
var (
	confirmLabel = regexp.MustCompile(`^(Trust and pair|信任并配对)$`)
	// unreachableLabel is how the dialog of the pairing begins its message when the computer did not
	// answer (the message goes on with what to check), and closeLabel is the button that closes it.
	unreachableLabel = regexp.MustCompile(`^(The computer did not answer\.|电脑没有响应。)`)
	closeLabel       = regexp.MustCompile(`^(Close|关闭)$`)
	// notRespondingLabel is the title of the dialog that Android shows over everything when a process does not answer
	// ("System UI isn't responding"), and waitLabel is its button that goes on waiting: a person taps that one, and the other
	// ends the process. The dialog is the system's and not the app's, so it is looked for in English only, which is the language
	// of the emulator that CI starts.
	notRespondingLabel = regexp.MustCompile(`isn.t responding$`)
	waitLabel          = regexp.MustCompile(`^Wait$`)
	tabs               = []tab{
		{"home", regexp.MustCompile(`^(Home|主页)$`)},
		{"sdks", regexp.MustCompile(`^(SDKs|SDK)$`)},
		{"plugins", regexp.MustCompile(`^(Plugins|插件)$`)},
		{"tasks", regexp.MustCompile(`^(Tasks|任务)$`)},
		{"settings", regexp.MustCompile(`^(Settings|设置)$`)},
	}
	// pairedLog is the line that the app writes to its standard output when a pairing succeeds:
	// the host without its port, and the fingerprint of its certificate (as the desktop shows it).
	pairedLog = regexp.MustCompile(`(?m)vfoxplus: paired host=(\S+) fingerprint=((?:[0-9A-F]{2}:)*[0-9A-F]{2})\s*$`)
)

// tab is one tab of the app: its name for the files, and its label.
type tab struct {
	name  string
	label *regexp.Regexp
}

// ParseMeminfo returns the total PSS and RSS, in KB, of a `dumpsys meminfo
// <package>` report: the first TOTAL row holds the PSS, and the summary
// header carries "RSS: <n>K".
func ParseMeminfo(report string) (pss, rss int, ok bool) {
	pssMatch := totalLine.FindStringSubmatch(report)
	rssMatch := rssField.FindStringSubmatch(report)
	if pssMatch == nil || rssMatch == nil {
		return 0, 0, false
	}
	pss, _ = strconv.Atoi(pssMatch[1])
	rss, _ = strconv.Atoi(rssMatch[1])
	return pss, rss, true
}

// mb is a size in KB as the lines have it, in MB rounded to the nearest.
func mb(kb int) int {
	return (kb + 512) / 1024
}

// Line formats the measuring point like the desktop one, in MB rounded to
// the nearest.
func Line(pssKB, rssKB int) string {
	return fmt.Sprintf("memory (apk-cold): RSS %d MB, PSS %d MB in 1 processes", mb(rssKB), mb(pssKB))
}

// PairedLine formats the paired-session point: the same process after the pairing, with the tabs
// visited.
func PairedLine(pssKB, rssKB int) string {
	return fmt.Sprintf("memory (apk-paired): RSS %d MB, PSS %d MB in 1 processes", mb(rssKB), mb(pssKB))
}

// sample is what one reading of `dumpsys meminfo` held, in KB.
type sample struct{ pss, rss int }

// Median returns the middle of values, and for an even number of them the mean of the two in the middle,
// rounded up. It leaves values as they are. It is 0 for none.
func Median(values []int) int {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	switch n := len(sorted); {
	case n == 0:
		return 0
	case n%2 == 1:
		return sorted[n/2]
	default:
		return (sorted[n/2-1] + sorted[n/2] + 1) / 2
	}
}

// medians is the median PSS and the median RSS of the samples, each of its own: the start that read the
// median of one need not be the start that read the median of the other.
func medians(samples []sample) (pss, rss int) {
	pssAll := make([]int, len(samples))
	rssAll := make([]int, len(samples))
	for i, s := range samples {
		pssAll[i], rssAll[i] = s.pss, s.rss
	}
	return Median(pssAll), Median(rssAll)
}

// samplesText is the table of the readings, in the order of the starts, and the line that they came to.
func samplesText(samples []sample, pss, rss int) string {
	var text strings.Builder
	lowest, highest := 0, 0
	for i, s := range samples {
		fmt.Fprintf(&text, "cold start %d: RSS %d MB, PSS %d MB\n", i+1, mb(s.rss), mb(s.pss))
		if i == 0 || s.pss < lowest {
			lowest = s.pss
		}
		if i == 0 || s.pss > highest {
			highest = s.pss
		}
	}
	fmt.Fprintf(&text, "median of %d: RSS %d MB, PSS %d MB (the PSS of the starts is from %d to %d MB)\n",
		len(samples), mb(rss), mb(pss), mb(lowest), mb(highest))
	return text.String()
}

// config is what Run reads from the command line.
type config struct {
	apk, outputDir, screenshot, pairLinkFile, shots, pictures, aapt2 string
	settle                                                           time.Duration
	// coldRuns is how many times the app is started cold and read; the line of memory.txt is the
	// median of the readings. Zero is one.
	coldRuns int
	// checkBack has the paired session press the key back with the dialog of the pairing open (see backKeepsTheApp).
	checkBack bool
}

// session runs the steps against a device. Its clock and its sleep are replaceable, so that the
// tests run the waits at once.
type session struct {
	adb     func(args ...string) ([]byte, error)
	badging func(aapt2, apk string) (pkg, activity string, err error)
	sleep   func(time.Duration)
	now     func() time.Time
	log     io.Writer
}

// Run implements `measure-apk`: see the package documentation for what it measures.
func Run(args []string) error {
	flags := flag.NewFlagSet("measure-apk", flag.ContinueOnError)
	apk := flags.String("apk", "", "the APK to install")
	outputDir := flags.String("output", "", "directory receiving memory.txt (and memory-paired.txt with --pair-link-file)")
	screenshot := flags.String("screenshot", "", "PNG file for a screenshot of the first screen")
	settle := flags.Int("settle", 20, "seconds to settle after each start")
	coldRuns := flags.Int("cold-runs", 1, "how many times the app is started cold and read: the line of memory.txt is the median of the readings")
	pairLinkFile := flags.String("pair-link-file", "", "file holding a one-time pairing link of a reachable server: the paired session is driven and measured too")
	shots := flags.String("shots", "", "directory receiving the screenshots, the screen dumps and the log of the paired session (required with --pair-link-file)")
	pictures := flags.String("pictures", "", "directory receiving the pictures of the user guide, taken in the paired session (needs --pair-link-file)")
	checkBack := flags.Bool("check-back-key", false, "press the key back with the dialog of the pairing open, and fail when the dialog stays or the app leaves (needs --pair-link-file)")
	aapt2 := flags.String("aapt2", "", "the aapt2 of the build tools (default: the newest in ANDROID_HOME, or the one on the PATH)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *apk == "" || *outputDir == "" || *settle < 0 || *coldRuns < 1 {
		return errors.New(usage)
	}
	if *pairLinkFile != "" && *shots == "" {
		return errors.New("--pair-link-file needs --shots: the paired session leaves its screenshots and its log there")
	}
	if *pictures != "" && *pairLinkFile == "" {
		return errors.New("--pictures needs --pair-link-file: the pictures are taken in the paired session")
	}
	if *checkBack && *pairLinkFile == "" {
		return errors.New("--check-back-key needs --pair-link-file: the dialog of the pairing is opened by its link")
	}
	c := config{apk: *apk, outputDir: *outputDir, screenshot: *screenshot, pairLinkFile: *pairLinkFile,
		shots: *shots, pictures: *pictures, aapt2: *aapt2, settle: time.Duration(*settle) * time.Second, coldRuns: *coldRuns,
		checkBack: *checkBack}
	s := &session{adb: runAdb, badging: readBadging, sleep: time.Sleep, now: time.Now, log: os.Stderr}
	return s.measure(c)
}

// measure runs the steps of the command against the device.
func (s *session) measure(c config) error {
	aapt2, err := findAapt2(c.aapt2)
	if err != nil {
		return err
	}
	pkg, activity, err := s.badging(aapt2, c.apk)
	if err != nil {
		return err
	}
	install, err := s.adb("install", "-r", c.apk)
	if err != nil {
		return err
	}
	if !strings.Contains(string(install), "Success") {
		return fmt.Errorf("adb install: %s", strings.TrimSpace(string(install)))
	}
	// The notification permission is granted beforehand: the system's dialog for it would cover the
	// button that the paired session taps. An app that does not ask for it refuses the grant, which
	// is no failure.
	_, _ = s.adb("shell", "pm", "grant", pkg, "android.permission.POST_NOTIFICATIONS")
	if _, err := s.adb("logcat", "-c"); err != nil {
		return err
	}
	// A cold start: the process of the app is stopped first, and the launcher's activity is started.
	// What one start reads moves a good deal (see the package documentation), so the app is started as
	// many times as was asked for, and the line is the median of what the starts read.
	samples, err := s.coldSamples(c, pkg, activity)
	if err != nil {
		return err
	}
	pss, rss := medians(samples)
	line := Line(pss, rss)
	fmt.Fprintln(s.log, line)
	if err := writeFile(filepath.Join(c.outputDir, "memory.txt"), line+"\n"); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(c.outputDir, "memory-cold-samples.txt"), samplesText(samples, pss, rss)); err != nil {
		return err
	}
	if c.screenshot != "" {
		if err := s.screenshotTo(c.screenshot); err != nil {
			return err
		}
	}
	if c.pairLinkFile == "" {
		return nil
	}
	return s.paired(c, pkg)
}

// coldSamples starts the app cold as many times as c asks for (once when it asks for none), lets each start
// settle, and reads its memory. The whole report of each reading is kept in the output directory, to see
// which category of memory it was that moved when two readings differ.
func (s *session) coldSamples(c config, pkg, activity string) ([]sample, error) {
	runs := max(c.coldRuns, 1)
	samples := make([]sample, 0, runs)
	for run := 1; run <= runs; run++ {
		if err := s.start(fmt.Sprintf("am start -S -W -n %s/%s", pkg, activity)); err != nil {
			return nil, err
		}
		s.sleep(c.settle)
		report, err := s.adb("shell", "dumpsys", "meminfo", pkg)
		if err != nil {
			return nil, err
		}
		pss, rss, ok := ParseMeminfo(string(report))
		if !ok {
			return nil, fmt.Errorf("no PSS or RSS in the meminfo report of cold start %d:\n%s", run, report)
		}
		if err := writeFile(filepath.Join(c.outputDir, fmt.Sprintf("meminfo-cold-%d.txt", run)), string(report)); err != nil {
			return nil, err
		}
		fmt.Fprintf(s.log, "cold start %d of %d: RSS %d MB, PSS %d MB\n", run, runs, mb(rss), mb(pss))
		samples = append(samples, sample{pss: pss, rss: rss})
	}
	return samples, nil
}

// paired drives the paired session: the link, the confirmation, the tabs, the log and the memory.
// When it fails it leaves the evidence of the failure beside the screenshots (keepEvidence).
func (s *session) paired(c config, pkg string) (err error) {
	link, err := readLink(c.pairLinkFile)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.keepEvidence(c.shots, link)
		}
	}()
	uri, host, err := DeepLink(link)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.shots, 0o755); err != nil {
		return err
	}
	if c.pictures != "" {
		// The app has not paired yet: this is its first page.
		if err := s.unpairedPictures(c); err != nil {
			return err
		}
	}
	if c.checkBack {
		if err := s.backKeepsTheApp(pkg, uri); err != nil {
			return err
		}
	}
	confirm, err := s.confirmPairing(c, pkg, uri)
	if err != nil {
		return err
	}
	s.keepDump(c.shots, "confirm", confirm)
	fmt.Fprintln(s.log, "tapped the confirmation")
	match, err := s.waitForLog(pairedLog, pairedTimeout)
	if err != nil {
		return err
	}
	if got := match[1]; got != host {
		return fmt.Errorf("the app paired with %s, want %s", got, host)
	}
	fmt.Fprintf(s.log, "paired with %s\n", match[1])
	s.sleep(c.settle)
	for _, t := range tabs {
		shot, err := s.tapWhen(t.label, tabTimeout, t.name+" tab")
		if err != nil {
			s.keepDump(c.shots, "tab-"+t.name+"-not-found", shot)
			return err
		}
		s.sleep(tabSettle)
		if err := s.screenshotTo(filepath.Join(c.shots, "tab-"+t.name+".png")); err != nil {
			return err
		}
		s.keepDump(c.shots, "tab-"+t.name, shot)
		if err := s.picture(c, "android-"+t.name); err != nil {
			return err
		}
	}
	report, err := s.adb("shell", "dumpsys", "meminfo", pkg)
	if err != nil {
		return err
	}
	pss, rss, ok := ParseMeminfo(string(report))
	if !ok {
		return fmt.Errorf("no PSS or RSS in the paired meminfo report:\n%s", report)
	}
	line := PairedLine(pss, rss)
	fmt.Fprintln(s.log, line)
	if err := writeFile(filepath.Join(c.outputDir, "memory-paired.txt"), line+"\n"); err != nil {
		return err
	}
	// The pictures that the tabs do not give come after the measurement, which they would change (pages
	// that are opened, a theme that is changed), and before the log is read, which they write to.
	if c.pictures != "" {
		if err := s.morePictures(c); err != nil {
			return err
		}
	}
	logText, err := s.logText()
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(c.shots, "logcat.txt"), logText); err != nil {
		return err
	}
	if problems := appErrors(logText); len(problems) > 0 {
		return fmt.Errorf("the log of the app has %d error(s), the first: %s", len(problems), problems[0])
	}
	if warnings := anrs(logText, pkg); len(warnings) > 0 {
		fmt.Fprintf(s.log, "warning: the system reported an ANR for %s (not a failure): %s\n", pkg, warnings[0])
	}
	return nil
}

// backKeepsTheApp opens the pairing link, and when the dialog of the trust is up (the dialog of an app that has not paired,
// where nothing of the app takes the key back) goes back with the key of Android, as a person does, and asks two things of
// the result: the dialog is closed, and the app is still in front. mygo takes a Back that no shortcut of the window claims
// as the end of the app, and a dialog, an overlay, is not a shortcut of the window (egoist/mygo#2088), so the dialog closed
// and the app went to the background with it. The app answers that (Root.sync) and this keeps it so: nothing else can see an
// app leave, which no test of its screens does. The dialog is left closed; the pairing opens the link again.
func (s *session) backKeepsTheApp(pkg, uri string) error {
	if err := s.start(fmt.Sprintf("am start -W -a android.intent.action.VIEW -d '%s' -p %s", uri, pkg), uri); err != nil {
		return err
	}
	s.sleep(tabSettle)
	if _, err := s.awaitDialog(confirmTimeout); err != nil {
		return err
	}
	if _, err := s.adb("shell", "input", "keyevent", keyBack); err != nil {
		return err
	}
	s.sleep(tabSettle)
	front, err := s.front()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(front, pkg+"/") {
		return fmt.Errorf("the key back, with the dialog of the pairing open, took the app out of the front: %s is in front", front)
	}
	shot, err := s.dump()
	if err != nil {
		return err
	}
	if hasLabel(shot.elements, confirmLabel) || hasLabel(shot.elements, unreachableLabel) {
		return fmt.Errorf("the dialog of the pairing is still open after the key back: the screen has %s", labelsOf(shot.elements))
	}
	fmt.Fprintln(s.log, "the key back closed the dialog of the pairing, and the app stayed in front")
	return nil
}

// awaitDialog waits until the screen shows the dialog of the pairing: the button that trusts the computer, or the message that
// it did not answer. The screen it returns when it fails is the last one it dumped.
func (s *session) awaitDialog(timeout time.Duration) (screen, error) {
	deadline := s.now().Add(timeout)
	var last screen
	for {
		if shot, err := s.dump(); err == nil {
			last = shot
			if hasLabel(shot.elements, confirmLabel) || hasLabel(shot.elements, unreachableLabel) {
				return shot, nil
			}
		}
		if !s.now().Before(deadline) {
			return last, fmt.Errorf("no dialog of the pairing within %s: the screen has %s", timeout, labelsOf(last.elements))
		}
		s.sleep(pollInterval)
	}
}

// front is the activity that Android has in front, as "package/Activity".
func (s *session) front() (string, error) {
	out, err := s.adb("shell", "dumpsys", "activity", "activities")
	if err != nil {
		return "", err
	}
	match := resumedActivity.FindSubmatch(out)
	if match == nil {
		return "", errors.New("the report of the activities of the device names no activity in front")
	}
	return string(match[1]), nil
}

// confirmPairing opens the pairing link on the device and taps the button that trusts the computer,
// and returns the screen that had the button. The deep link opens the pairing page with the link
// filled in: the app does not pair by itself, the person confirms the fingerprint.
//
// The network of an emulator may not be up when the app first looks for the computer, and then the
// dialog says that the computer did not answer, as it says to a person whose phone is not on the
// network yet. The person closes the dialog and opens the link again, which probes the computer
// anew, and so does this, up to pairAttempts times. Each retry is in the output, so that a run that
// needed one is not taken for a run that did not.
func (s *session) confirmPairing(c config, pkg, uri string) (screen, error) {
	for attempt := 1; ; attempt++ {
		if err := s.start(fmt.Sprintf("am start -W -a android.intent.action.VIEW -d '%s' -p %s", uri, pkg), uri); err != nil {
			return screen{}, err
		}
		s.sleep(tabSettle)
		shot, unreachable, err := s.awaitConfirmation(confirmTimeout, func() {
			// The dialog that shows the fingerprint, before the button that trusts it is tapped.
			s.sleep(tabSettle)
			if err := s.picture(c, trustPicture); err != nil {
				fmt.Fprintf(s.log, "the picture of the dialog of the trust was not taken: %v\n", err)
			}
		})
		if err == nil && !unreachable {
			return shot, nil
		}
		if err == nil && attempt < pairAttempts {
			s.keepDump(c.shots, fmt.Sprintf("confirm-unreachable-%d", attempt), shot)
			fmt.Fprintf(s.log, "the computer did not answer (attempt %d of %d): closing the dialog and opening the link again\n", attempt, pairAttempts)
			if _, err := s.tapWhen(closeLabel, tabTimeout, "Close button"); err != nil {
				return shot, err
			}
			s.sleep(retryPause)
			continue
		}
		s.keepDump(c.shots, "confirm-not-found", shot)
		if err == nil {
			err = fmt.Errorf("the computer did not answer in %d attempts: the screen has %s", pairAttempts, labelsOf(shot.elements))
		}
		return shot, err
	}
}

// awaitConfirmation waits until the dialog of the pairing shows what it has to show: the button that
// trusts the computer, which it taps (after beforeTap, which the run uses to take a picture of the
// dialog), or the message that the computer did not answer, which it reports. The screen it returns
// when it fails is the last one it dumped.
func (s *session) awaitConfirmation(timeout time.Duration, beforeTap func()) (shot screen, unreachable bool, err error) {
	deadline := s.now().Add(timeout)
	var last screen
	for {
		if dumped, derr := s.dump(); derr == nil {
			last = dumped
			if target, ok := lowestClickable(dumped.elements, confirmLabel); ok {
				beforeTap()
				x, y := target.center()
				if _, err := s.adb("shell", fmt.Sprintf("input tap %d %d", x, y)); err != nil {
					return dumped, false, err
				}
				return dumped, false, nil
			}
			if hasLabel(dumped.elements, unreachableLabel) {
				return dumped, true, nil
			}
		}
		if !s.now().Before(deadline) {
			return last, false, fmt.Errorf("no confirmation button within %s: the screen has %s", timeout, labelsOf(last.elements))
		}
		s.sleep(pollInterval)
	}
}

// keepEvidence leaves in dir what explains a paired session that failed: the screen, the whole log of
// the app, and what the device says of its network and of the computer at the address that the link
// names (ping, and a connection to its port). A failure to reach the computer is not an error that the
// log of the app shows, and a session that stops early does not write its log otherwise. Nothing here
// fails the session, which has its error already, and nothing here holds the token of the link.
func (s *session) keepEvidence(dir, link string) {
	if dir == "" {
		return
	}
	_ = s.screenshotTo(filepath.Join(dir, "failure.png"))
	if text, err := s.logText(); err == nil {
		_ = writeFile(filepath.Join(dir, "logcat.txt"), text)
	}
	_ = writeFile(filepath.Join(dir, "device-network.txt"), s.networkReport(link))
}

// networkReport asks the device about its network, and about the address of the computer.
func (s *session) networkReport(link string) string {
	commands := []string{"ip -4 addr", "ip route", "getprop | grep -E 'dhcp|dns|net[.]'"}
	if host, port, ok := addressOf(link); ok {
		commands = append(commands, "ping -c 1 -W 2 "+host, "echo | nc -w 3 "+host+" "+port+"; echo nc_exit=$?")
	}
	var report strings.Builder
	for _, command := range commands {
		out, err := s.adb("shell", command)
		fmt.Fprintf(&report, "$ %s\n%s", command, out)
		if err != nil {
			fmt.Fprintf(&report, "[%v]\n", err)
		}
		report.WriteString("\n")
	}
	return report.String()
}

// start runs an am start command on the device, and reports the error that am prints when the
// activity does not start, which adb does not report as a failure. The secret, the pairing link
// of a deep link, is left out of that error.
func (s *session) start(command string, secret ...string) error {
	out, err := s.adb("shell", command)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(out))
	if strings.Contains(text, "Error") {
		for _, value := range secret {
			text = strings.ReplaceAll(text, value, "<redacted>")
		}
		return fmt.Errorf("the device did not start the activity: %s", text)
	}
	return nil
}

// screenshotTo saves a screenshot of the screen of the device as a PNG file.
func (s *session) screenshotTo(path string) error {
	png, err := s.adb("exec-out", "screencap", "-p")
	if err != nil {
		return err
	}
	return writeFile(path, string(png))
}

// keepDump saves the dump of a screen, for the diagnosis of a screen that was not as expected or of
// the tabs.
func (s *session) keepDump(dir, name string, shot screen) {
	if dir == "" || shot.raw == "" {
		return
	}
	_ = writeFile(filepath.Join(dir, name+".xml"), shot.raw)
}

// logText is the log of the device since it was cleared.
func (s *session) logText() (string, error) {
	out, err := s.adb("logcat", "-d", "-v", "threadtime")
	return string(out), err
}

// waitForLog waits until the log holds a line that matches pattern, and returns the match.
func (s *session) waitForLog(pattern *regexp.Regexp, timeout time.Duration) ([]string, error) {
	deadline := s.now().Add(timeout)
	for {
		if text, err := s.logText(); err == nil {
			if match := pattern.FindStringSubmatch(text); match != nil {
				return match, nil
			}
		}
		if !s.now().Before(deadline) {
			return nil, fmt.Errorf("the log has no line of the pairing within %s", timeout)
		}
		s.sleep(pollInterval)
	}
}

// tapWhen waits until the screen has a clickable element whose label matches pattern, taps the
// lowest of them (a tab bar is at the bottom), and returns the screen. The screen it returns when
// it fails is the last one it dumped.
func (s *session) tapWhen(pattern *regexp.Regexp, timeout time.Duration, what string) (screen, error) {
	deadline := s.now().Add(timeout)
	var last screen
	for {
		shot, err := s.dump()
		if err == nil {
			last = shot
			if target, ok := lowestClickable(shot.elements, pattern); ok {
				x, y := target.center()
				if _, err := s.adb("shell", fmt.Sprintf("input tap %d %d", x, y)); err != nil {
					return shot, err
				}
				return shot, nil
			}
		}
		if !s.now().Before(deadline) {
			return last, fmt.Errorf("no %s within %s: the screen has %s", what, timeout, labelsOf(last.elements))
		}
		s.sleep(pollInterval)
	}
}

// dump reads the accessibility tree of the screen, which uiautomator writes to the device. When the dialog of a process that
// does not answer is over the screen, which the system shows when the emulator is slow (it was System UI, in the middle of a theme
// changing), and covers the app so that no screen that the tool looks for is there, it taps Wait as a person does, and reads again.
func (s *session) dump() (screen, error) {
	for tries := 0; ; tries++ {
		shot, err := s.dumpOnce()
		if err != nil || tries >= notRespondingTries || !s.waitForTheProcess(shot) {
			return shot, err
		}
	}
}

// dumpOnce reads the screen once.
func (s *session) dumpOnce() (screen, error) {
	if _, err := s.adb("shell", "uiautomator dump "+screenFile); err != nil {
		return screen{}, err
	}
	data, err := s.adb("shell", "cat "+screenFile)
	if err != nil {
		return screen{}, err
	}
	return parseScreen(data)
}

// waitForTheProcess taps Wait on the dialog of a process that does not answer, when the screen shows it, and says so. It
// reports whether it tapped.
func (s *session) waitForTheProcess(shot screen) bool {
	if !hasLabel(shot.elements, notRespondingLabel) {
		return false
	}
	wait, ok := lowestClickable(shot.elements, waitLabel)
	if !ok {
		return false
	}
	for _, e := range shot.elements {
		if label := strings.TrimSpace(e.label); notRespondingLabel.MatchString(label) {
			fmt.Fprintf(s.log, "the system says %q: waiting for it\n", label)
			break
		}
	}
	x, y := wait.center()
	_, _ = s.adb("shell", fmt.Sprintf("input tap %d %d", x, y))
	s.sleep(pollInterval)
	return true
}

// runAdb runs adb with args and returns its standard output. Its error names the first argument
// and the message of adb, and not the rest of the command, which can hold a pairing link.
func runAdb(args ...string) ([]byte, error) {
	var stdout, stderr strings.Builder
	command := exec.Command("adb", args...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return []byte(stdout.String()), fmt.Errorf("adb %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return []byte(stdout.String()), nil
}

// writeFile writes text to path, making its directory first.
func writeFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

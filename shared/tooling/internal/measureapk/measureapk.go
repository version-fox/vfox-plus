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

// Package measureapk implements `vfxtool measure-apk`: on a booted Android
// emulator it installs the release APK, cold-starts it through the launcher,
// settles, reads `dumpsys meminfo`, writes the memory(apk-cold) line for the
// memory report and takes a screenshot of the cold start for the
// documentation. CI runs it from the emulator job of
// .github/workflows/android.yml; adb must be on PATH.
package measureapk

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	applicationID = regexp.MustCompile(`applicationId\s*=\s*"([^"]+)"`)
	totalLine     = regexp.MustCompile(`(?m)^\s*TOTAL\s+(\d+)`)
	rssField      = regexp.MustCompile(`\bRSS:\s*(\d+)K?`)
	// The flags of the package in `dumpsys package`: the line starts with flags=[ ... ],
	// not with pkgFlags= or privateFlags=.
	packageFlags = regexp.MustCompile(`(?m)^\s*flags=\[([^\]]*)\]`)
)

// ApplicationID reads the applicationId out of a build.gradle.kts.
func ApplicationID(gradle string) (string, bool) {
	match := applicationID.FindStringSubmatch(gradle)
	if match == nil {
		return "", false
	}
	return match[1], true
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

// Line formats the measuring point like the desktop one, in MB rounded to
// the nearest.
func Line(pssKB, rssKB int) string {
	return fmt.Sprintf("memory (apk-cold): RSS %d MB, PSS %d MB in 1 processes", (rssKB+512)/1024, (pssKB+512)/1024)
}

// PairedLine formats the paired-session point: the same process with the
// system WebView and the /app page loaded.
func PairedLine(pssKB, rssKB int) string {
	return fmt.Sprintf("memory (apk-paired): RSS %d MB, PSS %d MB in 1 processes", (rssKB+512)/1024, (pssKB+512)/1024)
}

// IsDebuggable reports whether the report of `dumpsys package` for an installed
// app lists the DEBUGGABLE flag, which only a debuggable build has.
func IsDebuggable(report string) bool {
	match := packageFlags.FindStringSubmatch(report)
	if match == nil {
		return false
	}
	for _, flag := range strings.Fields(match[1]) {
		if flag == "DEBUGGABLE" {
			return true
		}
	}
	return false
}

// pairingLinkArgs is the adb command that writes the one-time pairing link into the
// private files directory of the debuggable app, through run-as: only a debuggable
// application allows it, and no other application can write there. adb joins its
// arguments into one command for the device's shell, so the whole script is a single
// argument, quoted for that shell. The link comes on standard input and never
// reaches the command line.
func pairingLinkArgs(pkg string) []string {
	return []string{"shell", "run-as " + pkg + " sh -c 'mkdir -p files/measurement && cat > files/measurement/pairing-link'"}
}

// pairedStart is the adb command that starts the launcher activity, which takes the
// pairing link that pairingLinkArgs wrote and opens its session.
func pairedStart(pkg string) []string {
	return []string{"shell", "am", "start", "-n", pkg + "/.MainActivity"}
}

// adbInput runs adb with input on its standard input.
func adbInput(input string, args ...string) error {
	command := exec.Command("adb", args...)
	command.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("adb %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func adb(args ...string) ([]byte, error) {
	command := exec.Command("adb", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return out, fmt.Errorf("adb %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Run implements `measure-apk --apk FILE --output DIR [--gradle FILE] [--screenshot FILE] [--settle SECONDS] [--pair-url URL]`.
// Without --pair-url it does what CI does: install, cold-start through the
// launcher, settle, and write the memory(apk-cold) line. With --pair-url it
// then leaves the one-time link for the debuggable app with run-as (no other
// application can write there, and no exported component takes a link), starts
// its launcher, which opens the same session as pasting the link and tapping
// connect, settles again,
// and appends the memory(apk-paired) line: the native session with its tabs
// loaded, next to the unpaired screen. The paired line is for local
// diagnosis only: pair it against a loopback `vfox-plus-serve serve` with
// `adb reverse` carrying the port, and never upload a two-line file, which
// the memory report would refuse to parse.
func Run(args []string) error {
	flags := flag.NewFlagSet("measure-apk", flag.ContinueOnError)
	apk := flags.String("apk", "", "the APK to install")
	outputDir := flags.String("output", "", "directory receiving memory.txt")
	gradle := flags.String("gradle", "android/app/build.gradle.kts", "build file holding the applicationId")
	screenshot := flags.String("screenshot", "", "PNG file for a screenshot of the cold start")
	settle := flags.Int("settle", 20, "seconds to settle after the start")
	pairURL := flags.String("pair-url", "", "one-time pairing link of a reachable server, to also measure the paired session (a debuggable APK only)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *apk == "" || *outputDir == "" || *settle < 0 {
		return errors.New("Usage: measure-apk --apk FILE --output DIR [--gradle FILE] [--screenshot FILE] [--settle SECONDS] [--pair-url URL]")
	}
	build, err := os.ReadFile(*gradle)
	if err != nil {
		return err
	}
	pkg, ok := ApplicationID(string(build))
	if !ok {
		return fmt.Errorf("no applicationId in %s", *gradle)
	}
	if _, err := adb("install", *apk); err != nil {
		return err
	}
	if *pairURL != "" {
		// Only a debuggable build lets run-as leave the pairing link for the app: a release
		// build refuses it, so the paired line would have no session to measure.
		// The check comes before any measurement, so nothing is written for it.
		info, err := adb("shell", "dumpsys", "package", pkg)
		if err != nil {
			return err
		}
		if !IsDebuggable(string(info)) {
			return fmt.Errorf("--pair-url needs a debuggable APK: run-as leaves the pairing link for the app only in a debuggable build, and %s is not one (build the debug variant with ./gradlew assembleDebug in android/, or leave out --pair-url)", *apk)
		}
	}
	// Cold start through the launcher, like a user tapping the icon.
	if _, err := adb("shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1"); err != nil {
		return err
	}
	time.Sleep(time.Duration(*settle) * time.Second)
	report, err := adb("shell", "dumpsys", "meminfo", pkg)
	if err != nil {
		return err
	}
	pss, rss, ok := ParseMeminfo(string(report))
	if !ok {
		return fmt.Errorf("no PSS or RSS in the meminfo report:\n%s", report)
	}
	line := Line(pss, rss)
	fmt.Println(line)
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "memory.txt"), []byte(line+"\n"), 0o644); err != nil {
		return err
	}
	if *screenshot != "" {
		png, err := adb("exec-out", "screencap", "-p")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*screenshot, png, 0o644); err != nil {
			return err
		}
	}
	if *pairURL == "" {
		return nil
	}
	// The paired session: the same native session a user gets after connecting,
	// with the inventory loaded and live events streaming.
	if err := adbInput(*pairURL, pairingLinkArgs(pkg)...); err != nil {
		return err
	}
	if _, err := adb(pairedStart(pkg)...); err != nil {
		return err
	}
	time.Sleep(time.Duration(*settle) * time.Second)
	paired, err := adb("shell", "dumpsys", "meminfo", pkg)
	if err != nil {
		return err
	}
	pairedPSS, pairedRSS, ok := ParseMeminfo(string(paired))
	if !ok {
		return fmt.Errorf("no PSS or RSS in the paired meminfo report:\n%s", paired)
	}
	pairedLine := PairedLine(pairedPSS, pairedRSS)
	fmt.Println(pairedLine)
	file, err := os.OpenFile(filepath.Join(*outputDir, "memory.txt"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(file, pairedLine)
	return errors.Join(err, file.Close())
}

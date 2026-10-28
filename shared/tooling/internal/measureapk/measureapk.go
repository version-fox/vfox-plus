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

// Run implements `measure-apk --apk FILE --output DIR [--gradle FILE] [--screenshot FILE] [--settle SECONDS]`.
func Run(args []string) error {
	flags := flag.NewFlagSet("measure-apk", flag.ContinueOnError)
	apk := flags.String("apk", "", "the APK to install")
	outputDir := flags.String("output", "", "directory receiving memory.txt")
	gradle := flags.String("gradle", "android/app/build.gradle.kts", "build file holding the applicationId")
	screenshot := flags.String("screenshot", "", "PNG file for a screenshot of the cold start")
	settle := flags.Int("settle", 20, "seconds to settle after the start")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *apk == "" || *outputDir == "" || *settle < 0 {
		return errors.New("Usage: measure-apk --apk FILE --output DIR [--gradle FILE] [--screenshot FILE] [--settle SECONDS]")
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
		return os.WriteFile(*screenshot, png, 0o644)
	}
	return nil
}

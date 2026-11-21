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
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	badgingPackage  = regexp.MustCompile(`(?m)^package: name='([^']+)'`)
	badgingLauncher = regexp.MustCompile(`(?m)^launchable-activity: name='([^']+)'`)
)

// readBadging returns the package name and the launchable activity of an APK, which aapt2 reads
// from the manifest of the APK (aapt2 dump badging).
func readBadging(aapt2, apk string) (pkg, activity string, err error) {
	var out bytes.Buffer
	command := exec.Command(aapt2, "dump", "badging", apk)
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {
		return "", "", fmt.Errorf("%s dump badging %s: %w: %s", aapt2, apk, err, strings.TrimSpace(out.String()))
	}
	return parseBadging(out.String())
}

// parseBadging returns the package name and the launchable activity of the output of aapt2 dump
// badging. An APK without a launchable activity has no launcher, and cannot be started the way a
// person starts it.
func parseBadging(text string) (pkg, activity string, err error) {
	name := badgingPackage.FindStringSubmatch(text)
	if name == nil {
		return "", "", errors.New("no package name in the manifest of the APK")
	}
	launch := badgingLauncher.FindStringSubmatch(text)
	if launch == nil {
		return "", "", fmt.Errorf("%s has no launchable activity: the APK has no launcher", name[1])
	}
	return name[1], launch[1], nil
}

// findAapt2 returns the aapt2 that reads the APK: the one given, else the newest one of the build
// tools in ANDROID_HOME (or ANDROID_SDK_ROOT), else the one on the PATH.
func findAapt2(given string) (string, error) {
	if given != "" {
		return given, nil
	}
	for _, root := range []string{os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT")} {
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "build-tools", "*", "aapt2"))
		if len(matches) == 0 {
			continue
		}
		sort.Slice(matches, func(i, j int) bool {
			return dottedLess(filepath.Base(filepath.Dir(matches[i])), filepath.Base(filepath.Dir(matches[j])))
		})
		return matches[len(matches)-1], nil
	}
	path, err := exec.LookPath("aapt2")
	if err != nil {
		return "", errors.New("no aapt2: give --aapt2, set ANDROID_HOME, or put the build tools on the PATH")
	}
	return path, nil
}

// dottedLess orders versions of build tools, such as 35.0.0 before 36.0.0, by their numbers.
func dottedLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na < nb
		}
	}
	return len(pa) < len(pb)
}

// DeepLink returns the URL that opens the pairing page of the phone app with a pairing link, and the
// host of the link without its port. The app takes the link from the query of vfoxplus://pair, and
// fills it in on the pairing page: it does not pair by itself. A link is
// https://<host>:<port>/auth/connect/<token>, the address of a server of the desktop.
func DeepLink(link string) (uri, host string, err error) {
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || !strings.HasPrefix(u.Path, "/auth/connect/") ||
		strings.ContainsAny(link, "' \t\r\n") {
		// The link is not named: it is a secret until it is used.
		return "", "", errors.New("the file does not hold a pairing link (https://<host>:<port>/auth/connect/<token>)")
	}
	return "vfoxplus://pair?link=" + url.QueryEscape(link), u.Hostname(), nil
}

// addressPattern is what the host of a pairing link may look like where the tool puts it in a command
// of the device's shell: letters, digits, dots and dashes, which a host name and an IPv4 address have.
var addressPattern = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

// addressOf returns the host and the port that a pairing link names, which are no secret (the token is),
// and whether they are safe to put in a command: the host looks like a host name or an IPv4 address and
// the port is a number.
func addressOf(link string) (host, port string, ok bool) {
	u, err := url.Parse(link)
	if err != nil {
		return "", "", false
	}
	host, port = u.Hostname(), u.Port()
	if !addressPattern.MatchString(host) || port == "" || strings.Trim(port, "0123456789") != "" {
		return "", "", false
	}
	return host, port, true
}

// readLink reads the pairing link out of its file.
func readLink(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // The file of the caller, which CI writes.
	if err != nil {
		return "", err
	}
	link := strings.TrimSpace(string(data))
	if link == "" {
		return "", fmt.Errorf("%s is empty: it should hold a pairing link", path)
	}
	return link, nil
}

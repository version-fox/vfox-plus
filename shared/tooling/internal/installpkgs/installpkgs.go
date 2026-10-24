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

// Package installpkgs installs Ubuntu packages on a runner of GitHub Actions
// in two steps:
//
//  1. It downloads the packages that the cache lacks into
//     /var/cache/apt/archives, in 15 minutes at most: the fetch tool of
//     vfxtool does it, eight files at a time and the largest first, from the
//     mirrors of the runner. Then apt fetches what is left.
//  2. It installs them from the archives, which needs no network.
//
// It is a port of .github/actions/install-linux-packages/install.sh to Go,
// using only the standard library. Finally it removes the archives of versions
// that the package lists no longer have, so that the cache does not grow from
// week to week.
//
// Set APT_DOWNLOAD_SECONDS to change the time of the download (900): it has to
// end before the timeout of the job, for the action to save what was
// downloaded. APT_PARALLEL is the number of files that are downloaded at the
// same time (8; 0 leaves all of it to apt). fetch has two more,
// APT_MIN_SPEED and APT_SPEED_TIME.
package installpkgs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// archives is where apt keeps the archives it downloads.
const archives = "/var/cache/apt/archives"

// archivesDir is where the sizes are checked. It is a variable so that tests
// can point it at a temporary directory; it always holds archives.
var archivesDir = archives

// mirrorsFile lists the mirrors of the runners, one per line, the best first.
const mirrorsFile = "/etc/apt/apt-mirrors.txt"

// prefix of a mirror+file address, which stands for one of the mirrors.
const mirrorPrefix = "mirror+file:/etc/apt/apt-mirrors.txt/"

// printURIsLine parses a line of apt-get install --print-uris: 'address' name
// size hash, followed by what the address is followed by.
var printURIsLine = regexp.MustCompile(`^'([^']*)' (\S+) ([0-9]+) (\S+)`)

// Run installs the packages. args are the package names.
func Run(args []string) error {
	if len(args) == 0 {
		return errors.New("Usage: installpkgs <package...>")
	}
	packages := append([]string(nil), args...)
	seconds, err := envWithDefault("APT_DOWNLOAD_SECONDS", 900)
	if err != nil {
		return fmt.Errorf("installpkgs: invalid APT_DOWNLOAD_SECONDS: %v", err)
	}
	parallel, err := envWithDefault("APT_PARALLEL", 8)
	if err != nil {
		return fmt.Errorf("installpkgs: invalid APT_PARALLEL: %v", err)
	}
	deadline := time.Now().Unix() + seconds

	for attempt := 1; attempt <= 3; attempt++ {
		if err := aptGet("2m", "update"); err == nil {
			break
		}
		if attempt == 3 {
			fmt.Println("::error::apt-get update failed in 3 attempts.")
			return errors.New("installpkgs: apt-get update failed in 3 attempts")
		}
		time.Sleep(5 * time.Second)
	}

	if parallel != 0 {
		if err := prefetch(packages, deadline, parallel); err != nil {
			return err
		}
	}

	// apt downloads what is still missing, with what is left of the time (a
	// minute at least, which is enough to see that nothing is).
	left := deadline - time.Now().Unix()
	if left < 60 {
		left = 60
	}
	if err := aptGet(fmt.Sprintf("%ds", left), append([]string{"install", "--download-only", "-y", "--no-install-recommends"}, packages...)...); err != nil {
		fmt.Printf("::error::Could not download the packages (the limit is %d seconds); the mirrors are probably slow.\n", seconds)
		return fmt.Errorf("installpkgs: could not download the packages: %v", err)
	}

	// Killing dpkg half way would leave packages unpacked but not configured,
	// so the installation has no time limit; it needs no network.
	if err := aptGet("0", append([]string{"install", "-y", "--no-install-recommends"}, packages...)...); err != nil {
		return err
	}
	return aptGet("0", "autoclean")
}

// aptOptions are passed to every run of apt-get: a request that gets no data
// for 30 seconds fails and is tried again, up to 3 times, and the archives
// stay where they are after the installation.
func aptOptions() []string {
	return []string{
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Timeout=30",
		"-o", "Acquire::https::Timeout=30",
		"-o", "APT::Keep-Downloaded-Packages=true",
	}
}

// aptGet runs apt-get as root, for at most the time (0 is no limit), without
// questions. The time is passed to timeout, like "2m" or "300s".
func aptGet(limit string, args ...string) error {
	return runApt(limit, os.Stdout, os.Stderr, args...)
}

// runApt runs sudo apt-get with the options above.
func runApt(limit string, stdout, stderr io.Writer, args ...string) error {
	command := []string{"env", "DEBIAN_FRONTEND=noninteractive", "apt-get"}
	command = append(command, aptOptions()...)
	command = append(command, args...)
	if limit != "0" {
		command = append([]string{"timeout", limit}, command...)
	}
	cmd := exec.Command("sudo", command...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// envWithDefault reads an integer variable that is empty when unset, like the
// shell's ${APT_DOWNLOAD_SECONDS:-900}.
func envWithDefault(name string, def int64) (int64, error) {
	text := os.Getenv(name)
	if text == "" {
		return def, nil
	}
	return strconv.ParseInt(text, 10, 64)
}

// archiveSize is the size of the archive, 0 when it is not there.
func archiveSize(name string) int64 {
	info, err := os.Stat(filepath.Join(archivesDir, name))
	if err != nil {
		return 0
	}
	return info.Size()
}

// missing counts the archives of the list that are not in the directory with
// their size. Each line of the list is a size, a name, a hash and addresses.
func missing(listPath string) (int, error) {
	content, err := os.ReadFile(listPath)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		size, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		if archiveSize(fields[1]) != size {
			count++
		}
	}
	return count, nil
}

// entry is one archive to prefetch: its line for fetch, and its size to sort by.
type entry struct {
	size   int64
	fields []string
}

// parsePrintURIs parses the output of apt-get install --print-uris into
// entries of size, name, hash and one address each.
func parsePrintURIs(output string) []entry {
	var entries []entry
	for _, line := range strings.Split(output, "\n") {
		match := printURIsLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		size, err := strconv.ParseInt(match[3], 10, 64)
		if err != nil {
			continue
		}
		entries = append(entries, entry{size: size, fields: []string{match[3], match[2], match[4], match[1]}})
	}
	return entries
}

// readMirrors reads the mirrors of the runners, the best first. Without the
// file there are none, and every archive keeps the address that apt printed.
func readMirrors() []string {
	content, err := os.ReadFile(mirrorsFile)
	if err != nil {
		return nil
	}
	var mirrors []string
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			mirrors = append(mirrors, fields[0])
		}
	}
	return mirrors
}

// expand turns the address that apt printed into the address on each mirror.
// The address of a runner starts with the prefix of the mirrors file, which
// stands for one of them; anything else stays as it is.
func expand(entries []entry, mirrors []string) []entry {
	for i, e := range entries {
		address := e.fields[3]
		path, ok := strings.CutPrefix(address, mirrorPrefix)
		if !ok || len(mirrors) == 0 {
			continue
		}
		fields := append([]string(nil), e.fields[:3]...)
		for _, mirror := range mirrors {
			fields = append(fields, strings.TrimSuffix(mirror, "/")+"/"+path)
		}
		entries[i].fields = fields
	}
	// The largest come first, so that none of them is the last and alone.
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].size != entries[b].size {
			return entries[a].size > entries[b].size
		}
		return strings.Join(entries[a].fields, " ") > strings.Join(entries[b].fields, " ")
	})
	return entries
}

// prefetch makes the archives that the installation needs available in the
// directory, and checks the ones that are there already. It only saves time:
// what it does not fetch, apt does.
func prefetch(packages []string, deadline int64, parallel int64) error {
	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Println("curl is missing, apt downloads everything.")
		return nil
	}
	mirrors := readMirrors()
	list, err := os.CreateTemp("", "installpkgs-*.list")
	if err != nil {
		return err
	}
	listPath := list.Name()
	defer os.Remove(listPath)
	// apt prints the archives for an empty directory, not for the archives
	// that are in the directory, which it takes for good ones: it is shown
	// an empty directory, and fetch checks them.
	empty, err := os.MkdirTemp("", "installpkgs-empty-*")
	if err != nil {
		return err
	}
	defer os.Remove(empty)
	var output bytes.Buffer
	_ = runApt("2m", &output, io.Discard, append([]string{
		"-o", "Dir::Cache::Archives=" + empty,
		"-o", "Acquire::ForceHash=SHA256",
		"install", "--print-uris", "-y", "--no-install-recommends",
	}, packages...)...)
	for _, e := range expand(parsePrintURIs(output.String()), mirrors) {
		fmt.Fprintln(list, strings.Join(e.fields, " "))
	}
	list.Close()
	content, err := os.ReadFile(listPath)
	if err != nil {
		return err
	}
	total := 0
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) != "" {
			total++
		}
	}
	before, err := missing(listPath)
	if err != nil {
		return err
	}
	fmt.Printf("%d of %d archives are not in %s.\n", before, total, archives)
	if err := exec.Command("sudo", "mkdir", "-p", filepath.Join(archives, "partial")).Run(); err != nil {
		return err
	}
	// The fetch workers run unprivileged (the action only opens the archives
	// directory itself), so open the partial directory the same way.
	if err := exec.Command("sudo", "chmod", "a+rwx", filepath.Join(archives, "partial")).Run(); err != nil {
		return err
	}
	// One line, a size, a name, a hash and addresses, for each run of fetch,
	// and no run when there is no line. fetch ends at the deadline by itself;
	// the limit here is for what goes wrong in it.
	if parallel > 0 && total > 0 {
		fetch, err := os.Executable()
		if err != nil {
			return err
		}
		limit := deadline - time.Now().Unix() + 60
		if limit < 1 {
			limit = 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(limit)*time.Second)
		defer cancel()
		var lines [][]string
		for _, line := range strings.Split(string(content), "\n") {
			if fields := strings.Fields(line); len(fields) >= 4 {
				lines = append(lines, fields)
			}
		}
		semaphore := make(chan struct{}, parallel)
		var group sync.WaitGroup
		for _, fields := range lines {
			group.Add(1)
			semaphore <- struct{}{}
			go func(fields []string) {
				defer group.Done()
				defer func() { <-semaphore }()
				cmd := exec.CommandContext(ctx, fetch, append([]string{"fetch", archives}, fields...)...)
				cmd.Env = append(os.Environ(),
					"APT_DEADLINE="+strconv.FormatInt(deadline, 10),
					"APT_MIN_SPEED="+os.Getenv("APT_MIN_SPEED"),
					"APT_SPEED_TIME="+os.Getenv("APT_SPEED_TIME"))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				_ = cmd.Run()
			}(fields)
		}
		group.Wait()
	}
	after, err := missing(listPath)
	if err != nil {
		return err
	}
	fmt.Printf("fetch.sh downloaded %d archives.\n", before-after)
	if after > 0 {
		fmt.Printf("::warning::%d archives are left for apt.\n", after)
	}
	return nil
}

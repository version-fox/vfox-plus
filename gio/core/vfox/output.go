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

package vfox

import (
	"regexp"
	"strconv"
	"strings"

	core "github.com/version-fox/vfox-plus/gio/core"
)

// FieldSeparator is the ASCII unit separator used between the fields of
// `vfox info --format`.
const FieldSeparator = "\x1f"

// PluginInfoTemplate is the Go template passed to
// `vfox info <name> --format`.
//
// The fields are joined with FieldSeparator, which cannot appear in any of
// them, so descriptions may contain new lines and any punctuation.
const PluginInfoTemplate = `{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}`

// Parsers for the human-oriented output of the vfox CLI.
//
// vfox has no machine-readable output mode for most commands, so every format
// handled here is documented in docs/vfox-cli-contract.md and covered by the
// captured outputs in shared/fixtures/. All parsers are pure functions: they
// receive the decoded standard output of a command and never fail on
// unexpected lines, which are skipped instead.

var (
	versionPattern          = regexp.MustCompile(`(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`)
	treeItemPattern         = regexp.MustCompile(`^([│ ]*)[├└]─[─┬](.+)$`)
	availablePluginPattern  = regexp.MustCompile(`^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$`)
	downloadProgressPattern = regexp.MustCompile(`Downloading\.\.\.\s+(\d{1,3})%`)
	failedToInstallPattern  = regexp.MustCompile(`^failed to install\b`)
)

// ParseVfoxVersion parses `vfox --version`, for example `vfox version 1.0.12.
// It returns empty if no version number is found.
func ParseVfoxVersion(output string) string {
	match := versionPattern.FindStringSubmatch(core.StripANSI(output))
	if match == nil {
		return ""
	}
	return match[1]
}

// ParseCurrent parses `vfox current` without arguments into plugin name and
// current version; an empty version means no version is in effect.
//
// Each line has the form `<name> -> v<version>`, or `<name> -> N/A` when no
// version is in effect. vfox prefixes the version with a `v`, which is
// removed again. The result preserves the order of the output.
func ParseCurrent(output string) []CurrentVersion {
	var result []CurrentVersion
	for _, line := range lines(output) {
		arrow := strings.Index(line, " -> ")
		if arrow <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:arrow])
		value := strings.TrimSpace(line[arrow+4:])
		if name == "" || strings.Contains(name, " ") {
			continue
		}
		current := CurrentVersion{Name: name}
		if value != "" && value != "N/A" {
			current.Version = stripVPrefix(value)
			current.HasVersion = true
		}
		result = append(result, current)
	}
	return result
}

// CurrentVersion is the current version of one plugin: empty without
// HasVersion when no version is in effect.
type CurrentVersion struct {
	Name       string
	Version    string
	HasVersion bool
}

// SdkVersions are the installed versions of one plugin, in output order.
type SdkVersions struct {
	Name     string
	Versions []string
}

// ParseInstalledTree parses the tree printed by `vfox list` without arguments
// into the installed versions of every plugin, in output order. A plugin
// without installed versions is drawn as a leaf (`└──maven`).
//
// ```text
// All installed sdk versions
// ├─┬java
// │ ├──25.4.4.1-graalce
// │ └──25.3.4.1-graalce
// └──maven
// ```
func ParseInstalledTree(output string) []SdkVersions {
	byName := make(map[string]*SdkVersions)
	var order []string
	var current *SdkVersions
	for _, line := range lines(output) {
		match := treeItemPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		// Each level of the tree indents by two characters.
		level := utf8Runes(match[1]) / 2
		text := strings.TrimSpace(match[2])
		if text == "" {
			continue
		}
		if level == 0 {
			entry, ok := byName[text]
			if !ok {
				entry = &SdkVersions{Name: text}
				byName[text] = entry
				order = append(order, text)
			}
			current = entry
		} else if current != nil {
			current.Versions = append(current.Versions, text)
		}
	}
	result := make([]SdkVersions, 0, len(order))
	for _, name := range order {
		result = append(result, *byName[name])
	}
	return result
}

func utf8Runes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// MergeInstalledSdks combines the outputs of `vfox list` and `vfox current`
// into SDKs.
//
// Plugins that only appear in one of the outputs are still included, so that
// the result degrades gracefully if one command fails.
func MergeInstalledSdks(tree []SdkVersions, current []CurrentVersion) []core.InstalledSdk {
	versionsByName := make(map[string][]string, len(tree))
	order := []string{}
	seen := make(map[string]bool)
	for _, item := range tree {
		versionsByName[item.Name] = item.Versions
		if !seen[item.Name] {
			seen[item.Name] = true
			order = append(order, item.Name)
		}
	}
	currentByName := make(map[string]CurrentVersion, len(current))
	for _, item := range current {
		currentByName[item.Name] = item
		if !seen[item.Name] {
			seen[item.Name] = true
			order = append(order, item.Name)
		}
	}
	result := make([]core.InstalledSdk, 0, len(order))
	for _, name := range order {
		sdk := core.InstalledSdk{Name: name, Versions: versionsByName[name]}
		if item, ok := currentByName[name]; ok && item.HasVersion {
			sdk.Current = item.Version
			sdk.HasCurrent = true
		}
		result = append(result, sdk)
	}
	return result
}

// ParseSearch parses `vfox search <name>` in a non-interactive terminal.
//
// ```text
// Available versions:
//   - 24.22.0 (LTS) [npm 11.19.1]
//   - 3.9.16 (installed)
//
// ```
//
// Every entry is `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
func ParseSearch(output string) []core.AvailableVersion {
	var result []core.AvailableVersion
	for _, rawLine := range lines(output) {
		line := strings.TrimRight(rawLine, " \t")
		if !strings.HasPrefix(line, " - ") {
			continue
		}
		if version, ok := parseSearchEntry(strings.TrimSpace(line[3:])); ok {
			result = append(result, version)
		}
	}
	return result
}

func parseSearchEntry(entry string) (core.AvailableVersion, bool) {
	var version core.AvailableVersion
	if entry == "" {
		return version, false
	}
	firstSpace := strings.Index(entry, " ")
	if firstSpace < 0 {
		version.Version = entry
		return version, true
	}
	version.Version = entry[:firstSpace]
	rest := strings.TrimSpace(entry[firstSpace+1:])

	const installedSuffix = "(installed)"
	if strings.HasSuffix(rest, installedSuffix) {
		version.Installed = true
		rest = strings.TrimSpace(strings.TrimSuffix(rest, installedSuffix))
	}

	if strings.HasSuffix(rest, "]") {
		if open := strings.LastIndex(rest, "["); open >= 0 {
			version.Additions = parseAdditions(rest[open+1 : len(rest)-1])
			rest = strings.TrimSpace(rest[:open])
		}
	}

	if strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")") {
		inner := strings.TrimSpace(rest[1 : len(rest)-1])
		if inner != "" {
			version.Note, version.HasNote = inner, true
		}
	} else if rest != "" {
		version.Note, version.HasNote = rest, true
	}
	return version, true
}

func parseAdditions(text string) []core.VersionAddition {
	var result []core.VersionAddition
	for _, part := range strings.Split(text, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		name, version, _ := strings.Cut(item, " ")
		result = append(result, core.VersionAddition{Name: name, Version: strings.TrimSpace(version)})
	}
	return result
}

// ParseAvailablePlugins parses `vfox available`.
//
// ```text
// AVAILABLE PLUGINS
//
//	nodejs              ✓  https://github.com/version-fox/vfox-nodejs
//	bun                 ✗  https://github.com/arfaWong/vfox-bun
//
//	Use 'vfox add <plugin>' to install
//
// ```
func ParseAvailablePlugins(output string) []core.AvailablePlugin {
	var result []core.AvailablePlugin
	for _, line := range lines(output) {
		match := availablePluginPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		result = append(result, core.AvailablePlugin{Name: match[1], Official: match[2] == "✓", Homepage: match[3]})
	}
	return result
}

// ParsePluginInfo parses the output of `vfox info <name> --format` with
// PluginInfoTemplate.
//
// It returns false if the output does not have the expected number of fields.
func ParsePluginInfo(output string) (core.PluginInfo, bool) {
	var info core.PluginInfo
	fields := strings.Split(core.StripANSI(output), FieldSeparator)
	if len(fields) < 5 {
		return info, false
	}
	info.Name = strings.TrimSpace(fields[0])
	info.Version = strings.TrimSpace(fields[1])
	info.Homepage = strings.TrimSpace(fields[2])
	info.InstallPath = strings.TrimSpace(fields[3])
	// The description is the last field and may itself contain separators.
	info.Description = strings.TrimSpace(strings.Join(fields[4:], FieldSeparator))
	return info, true
}

// ParseInstallPath parses `vfox info <name>@<version>`, which prints the
// directory of an installed version, or `notfound`.
func ParseInstallPath(output string) (string, bool) {
	path := strings.TrimSpace(core.StripANSI(output))
	if path == "" || path == "notfound" {
		return "", false
	}
	return path, true
}

// ParseConfigList parses `vfox config --list`, which prints one
// `<key> = <value>` per line.
func ParseConfigList(output string) []core.ConfigEntry {
	var result []core.ConfigEntry
	for _, line := range lines(output) {
		equals := strings.Index(line, " = ")
		if equals <= 0 {
			// `key =` with an empty value loses its trailing space when trimmed.
			if trimmed := strings.TrimRight(line, " \t"); strings.HasSuffix(trimmed, " =") {
				result = append(result, core.ConfigEntry{Key: strings.TrimSpace(trimmed[:len(trimmed)-2])})
			}
			continue
		}
		key := strings.TrimSpace(line[:equals])
		if key == "" || strings.Contains(key, " ") {
			continue
		}
		result = append(result, core.ConfigEntry{Key: key, Value: strings.TrimSpace(line[equals+3:])})
	}
	return result
}

// ParseDownloadProgress extracts the download progress from a line of
// `vfox install` output.
//
// The progress bar is written to standard error and redrawn with carriage
// returns: `Downloading...  42% [=====>    ] (1.2 MB/s) [3s:4s]`. It returns
// the fraction between 0 and 1, and -1 if `line` is not a progress line.
func ParseDownloadProgress(line string) float64 {
	match := downloadProgressPattern.FindStringSubmatch(line)
	if match == nil {
		return -1
	}
	percent, err := strconv.Atoi(match[1])
	if err != nil {
		return -1
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return float64(percent) / 100
}

// SummarizeError picks a short, human readable error message from a failed
// command.
//
// Usage errors (`cli.Exit`) go to standard error. The error a command returns
// goes to standard output up to vfox 1.0.12 (`fmt.Println(err)`) and to
// standard error from 1.0.13, while what a command prints before failing,
// such as the reason of a failed installation, stays on standard output. So
// the lines of standard output are read first and those of standard error
// last. Progress bar redraws and blank lines are ignored, and the last
// meaningful line wins, unless it is the generic `failed to install <name>`
// trailer, in which case the more specific line printed before it is used.
func SummarizeError(stdout, stderr string) string {
	lines := append(meaningfulLines(stdout), meaningfulLines(stderr)...)
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > 1 && failedToInstallPattern.MatchString(lines[len(lines)-1]) {
		return lines[len(lines)-2]
	}
	return lines[len(lines)-1]
}

func meaningfulLines(text string) []string {
	var result []string
	for _, line := range lines(text) {
		if trimmed := strings.TrimSpace(line); trimmed != "" && ParseDownloadProgress(trimmed) < 0 {
			result = append(result, trimmed)
		}
	}
	return result
}

func stripVPrefix(version string) string {
	if strings.HasPrefix(version, "v") && len(version) > 1 {
		return version[1:]
	}
	return version
}

func lines(output string) []string {
	return core.SplitTerminalLines(core.StripANSI(output))
}

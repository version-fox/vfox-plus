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
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
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
// handled here is documented in docs/en/dev/vfox-cli-contract.md and covered by the
// captured outputs in shared/fixtures/. All parsers are pure functions: they
// receive the decoded standard output of a command and never fail on
// unexpected lines, which are skipped instead.

var (
	versionPattern          = regexp.MustCompile(`(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`)
	treeItemPattern         = regexp.MustCompile(`^([│ ]*)[├└]─[─┬](.+)$`)
	availablePluginPattern  = regexp.MustCompile(`^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$`)
	downloadProgressPattern = regexp.MustCompile(`Downloading\.\.\.\s+(\d{1,3})%`)
	failedToInstallPattern  = regexp.MustCompile(`^failed to install\b`)
	// pluginErrorPattern is the error of a hook of a plugin; the file and the line
	// number are optional, and the first `:<number>: ` ends them.
	pluginErrorPattern = regexp.MustCompile(`^plugin \[\w+\] method error: (?:.*?:\d+: )?(.+)$`)
	// tracebackEntryPattern is a line of a Lua stack traceback.
	tracebackEntryPattern = regexp.MustCompile(`^(?:\[[^\]]*\]|\S.*?:\d+): (?:in .+|\?)$|^\.\.\.`)
)

// ParseVfoxVersion parses `vfox --version`, for example `vfox version 1.1.0.
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
// Each line has the form `<name> -> <version>`, or `<name> -> N/A` when no
// version is in effect. vfox 1.0.x prefixes the version with a `v`, which is
// removed again; vfox 1.1.0 prints the stored name verbatim. The result
// preserves the order of the output.
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
// Since vfox 1.1.0 a version carries its link and alias annotations, which
// are left out:
//
// ```text
// All installed sdk versions
// ├─┬java
// │ ├──25.4.4.1-graalce
// │ └──17-system (linked → /usr/lib/jvm/java-17) [alias: j17]
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
			name := cleanInstalledName(text)
			entry, ok := byName[name]
			if !ok {
				entry = &SdkVersions{Name: name}
				byName[name] = entry
				order = append(order, name)
			}
			current = entry
		} else if current != nil {
			if version := cleanInstalledVersion(text); version != "" {
				current.Versions = append(current.Versions, version)
			}
		}
	}
	result := make([]SdkVersions, 0, len(order))
	for _, name := range order {
		result = append(result, *byName[name])
	}
	return result
}

// cleanInstalledName takes the plugin name out of a level-0 tree item.
func cleanInstalledName(text string) string {
	// A plugin line holds only its name; guard against a future suffix anyway.
	if cut, _, found := strings.Cut(text, " "); found {
		if cut != "" {
			return cut
		}
	}
	return text
}

// cleanInstalledVersion takes the version name out of a level-1 tree item,
// leaving out the annotations vfox 1.1.0 appends: ` (linked → <path>)`,
// ` [alias: <alias>]`, and the `<— current` marker of `vfox list <sdk>`.
func cleanInstalledVersion(text string) string {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "-> ")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	if cut, _, found := strings.Cut(trimmed, " "); found {
		return cut
	}
	return trimmed
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
// goes to standard error, while what a command prints before failing,
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

// meaningfulLines are the lines of an output that say something: not blank, not the
// redraw of a progress bar, and not the stack traceback that vfox prints below the
// error of a plugin. What a plugin reports reads
//
//	plugin [PreInstall] method error: /home/u/.vfox/plugin/java/hooks/pre_install.lua:42: No JDK found for 99-tem on linux/amd64.
//	stack traceback:
//		[G]: in function 'error'
//		/home/u/.vfox/plugin/java/hooks/pre_install.lua:42: in main chunk
//		[G]: ?
//
// and the sentence is in the first line: its file and line number are left out, as
// are the lines of the traceback, which say nothing the user could use.
func meaningfulLines(text string) []string {
	var result []string
	inTraceback := false
	for _, line := range lines(text) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "stack traceback:" {
			inTraceback = true
			continue
		}
		// A task keeps its lines without the indentation, so an entry is known by its
		// shape as well: `<file>:<line>: in <what>`, `[G]: in <what>` and `[G]: ?`.
		if inTraceback && (strings.HasPrefix(line, "\t") || tracebackEntryPattern.MatchString(trimmed)) {
			continue
		}
		inTraceback = false
		if trimmed != "" && ParseDownloadProgress(trimmed) < 0 {
			result = append(result, pluginErrorSentence(trimmed))
		}
	}
	return result
}

// pluginErrorSentence takes the file and the line number out of the error of a
// plugin, `plugin [PreInstall] method error: <file>:<line>: <sentence>`, and
// leaves any other line as it is.
func pluginErrorSentence(line string) string {
	if match := pluginErrorPattern.FindStringSubmatch(line); match != nil && strings.TrimSpace(match[1]) != "" {
		return strings.TrimSpace(match[1])
	}
	return line
}

// decodeJSON decodes text into value, reporting whether it succeeded.
func decodeJSON(text string, value interface{}) bool {
	return json.Unmarshal([]byte(text), value) == nil
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

// MinimumVfoxVersion is the oldest vfox release the application works with.
// Command errors go to standard error since 1.0.13, and 1.1.0 adds the link,
// doctor, alias, path.mode, install --use and plugin Detect/Fingerprint APIs
// the application relies on.
const MinimumVfoxVersion = "1.1.0"

// IsSupportedVersion reports whether version is at least MinimumVfoxVersion.
// It compares the leading numeric release (for example `1.1.0` in
// `1.1.0-beta.1`); a version without one counts as unsupported.
func IsSupportedVersion(version string) bool {
	return CompareVersions(version, MinimumVfoxVersion) >= 0
}

// CompareVersions compares two version numbers by their leading numeric
// release: 1.10.0 is newer than 1.2.0, and a suffix after the numbers (such as
// `-beta.1`) does not order them. It returns -1, 0 or 1. A version without a
// leading number sorts before any version with one.
func CompareVersions(left, right string) int {
	return compareNumericRelease(numericRelease(left), numericRelease(right))
}

func numericRelease(version string) []int {
	match := versionPattern.FindStringSubmatch(version)
	if match == nil {
		return nil
	}
	numeric := match[1]
	// Keep the leading numbers: `1.1.0-beta.1` is release `1.1.0`.
	end := 0
	for end < len(numeric) && (numeric[end] >= '0' && numeric[end] <= '9' || numeric[end] == '.') {
		end++
	}
	var release []int
	for _, part := range strings.Split(numeric[:end], ".") {
		if part == "" {
			continue
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		release = append(release, number)
	}
	return release
}

func compareNumericRelease(left, right []int) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return -1
	}
	if len(right) == 0 {
		return 1
	}
	for i := 0; i < len(left) || i < len(right); i++ {
		var l, r int
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	return 0
}

// DoctorFinding is one problem, warning or notice that `vfox doctor --json`
// reports.
type DoctorFinding struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity"`
	Sdk      string   `json:"sdk"`
	Version  string   `json:"version"`
	Message  string   `json:"message"`
	Hints    []string `json:"hints"`
	Fix      []string `json:"fix"`
	Ignored  bool     `json:"ignored"`
}

// DoctorReport is the parsed output of `vfox doctor --json`.
type DoctorReport struct {
	OK       bool            `json:"ok"`
	Findings []DoctorFinding `json:"findings"`
}

// VersionAlias is one alias that `vfox alias list` reports:
// `<sdk>@<alias> -> <sdk>@<version>`.
type VersionAlias struct {
	Sdk     string `json:"sdk"`
	Alias   string `json:"alias"`
	Target  string `json:"target"`
	Display string `json:"display"`
}

// ParseAliasList parses `vfox alias list [<sdk>]`.
//
// ```text
// java@j17 -> java@17.0.9-tem
// ```
//
// The `No aliases defined. …` message parses as no aliases.
func ParseAliasList(output string) []VersionAlias {
	var result []VersionAlias
	for _, line := range lines(output) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "No aliases defined") {
			continue
		}
		left, right, found := strings.Cut(trimmed, " -> ")
		if !found {
			continue
		}
		sdk, alias, ok := strings.Cut(strings.TrimSpace(left), "@")
		if !ok || sdk == "" || alias == "" || strings.Contains(sdk, " ") || strings.Contains(alias, " ") {
			continue
		}
		target := strings.TrimSpace(right)
		if target == "" {
			continue
		}
		result = append(result, VersionAlias{Sdk: sdk, Alias: alias, Target: target, Display: trimmed})
	}
	return result
}

// VersionDetail is what `vfox info --json <sdk>[@<version>]` says about one
// plugin or one installed version.
type VersionDetail struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	Origin    string `json:"origin"`
	Target    string `json:"target"`
	Home      string `json:"home"`
}

// ParseVersionDetailJSON parses `vfox info --json [<sdk>[@<version>]]`.
func ParseVersionDetailJSON(output string) (VersionDetail, bool) {
	var detail VersionDetail
	trimmed := strings.TrimSpace(core.StripANSI(output))
	if trimmed == "" {
		return detail, false
	}
	// The output is a JSON object; decode only the fields the GUI shows so
	// that new fields stay compatible.
	var raw struct {
		Name      *string `json:"name"`
		Version   *string `json:"version"`
		Installed *bool   `json:"installed"`
		Origin    *string `json:"origin"`
		Target    *string `json:"target"`
		Home      *string `json:"home"`
	}
	if !decodeJSON(trimmed, &raw) {
		return detail, false
	}
	if raw.Name != nil {
		detail.Name = *raw.Name
	}
	if raw.Version != nil {
		detail.Version = *raw.Version
	}
	if raw.Installed != nil {
		detail.Installed = *raw.Installed
	}
	if raw.Origin != nil {
		detail.Origin = *raw.Origin
	}
	if raw.Target != nil {
		detail.Target = *raw.Target
	}
	if raw.Home != nil {
		detail.Home = *raw.Home
	}
	if detail.Name == "" {
		return detail, false
	}
	return detail, true
}

// ParseDoctorReport parses `vfox doctor --json [<sdk>[@<version>]]`.
func ParseDoctorReport(output string) (DoctorReport, bool) {
	var report DoctorReport
	trimmed := strings.TrimSpace(core.StripANSI(output))
	if trimmed == "" {
		return report, false
	}
	var raw struct {
		OK       *bool           `json:"ok"`
		Findings []DoctorFinding `json:"findings"`
	}
	if !decodeJSON(trimmed, &raw) {
		return report, false
	}
	if raw.OK != nil {
		report.OK = *raw.OK
	}
	report.Findings = raw.Findings
	if report.Findings == nil {
		report.Findings = []DoctorFinding{}
	}
	return report, true
}

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

// Package profile holds what the application knows about particular vfox
// plugins, beyond what every plugin has in common.
//
// vfox lists, installs, selects and removes versions the same way for every
// plugin, but plugins offer more than that: vfox-java lists the JDKs of one of
// 31 distributions when `vfox search java <distribution>` names it,
// vfox-nodejs installs the newest release of a line for `nodejs@24`,
// vfox-flutter installs the release a channel points to for `flutter@stable`,
// and plugins read environment variables that move their downloads to a mirror,
// and legacy version files such as `.nvmrc`. None of this is visible in the
// output of vfox, so a profile writes it down: the arguments of a search, the
// shorthands of an install, the variables, the version files, the filters that
// suit the notes of a plugin, and how its version names are put together.
//
// A profile is a JSON file (format 2, docs/en/dev/profiles.md), which no one
// has to compile: the application ships those of the plugins of the
// version-fox organisation (shared/profile/plugins), updates them from the
// repository, reads the one that a plugin ships as lib/vfox-plus.json, and
// the user's own override every other. The sections of the files that describe
// one plugin are merged (see Merge), and a plugin that none describes still
// gets filters and variables inferred from what it reports (infer.go).
//
// A profile is plain data without a user interface framework. The desktop
// interface, the web interface and the phone app show the same profiles; the
// texts that a profile carries are written in both languages of the application
// next to the data they describe, because the web interface and the phone app
// get them over HTTP.
package profile

import (
	"runtime"
	"slices"
	"strings"
)

// Text is a text in the two languages of the application. ZH may be empty
// where it would repeat EN, as in the names of products.
type Text struct {
	EN, ZH string
}

// For returns the text in the language with the given tag (`en`, `zh`,
// `zh-Hans`, …): Simplified Chinese for any tag that starts with `zh` and has
// a translation, English otherwise.
func (t Text) For(tag string) string {
	if t.ZH != "" && strings.HasPrefix(strings.ToLower(tag), "zh") {
		return t.ZH
	}
	return t.EN
}

// Platform is the machine that vfox runs on, as `GOOS` and `GOARCH` name it.
type Platform struct {
	OS, Arch string
}

// CurrentPlatform is the platform of this process, which is the one vfox runs
// on: the application starts vfox on its own machine.
func CurrentPlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// Profile is what the application knows about one plugin, on one platform.
type Profile struct {
	// Plugin is the name of the plugin, as `vfox add` installs it.
	Plugin string
	// Search is what `vfox search <plugin> <argument>` takes, or nil when the
	// plugin ignores arguments.
	Search *Search
	// Naming tells how to take version names apart, or nil when they have no
	// structure that the application reads.
	Naming *Naming
	// Shorthands are names that `vfox install <plugin>@<name>` accepts besides
	// the versions that a search lists.
	Shorthands []Shorthand
	// Variables are the environment variables that the plugin reads.
	Variables []Variable
	// LegacyFiles are the files that vfox reads through the plugin to find the
	// version of a project.
	LegacyFiles []string
	// LegacyStrategy tells whether the setting `legacyVersionFile.strategy` of
	// vfox changes what the plugin does with those files.
	LegacyStrategy bool
	// Facets are the filters that suit the notes and the names of the versions
	// the plugin lists.
	Facets []Facet
	// Guide says in a few sentences how version names work for this plugin.
	Guide Text
	// Tips are things that are good to know before the first install.
	Tips []Text
	// Docs are the pages of the plugin's documentation worth opening: a page of
	// its docs folder, its README, or its documentation website, which comes first
	// when the plugin publishes one.
	Docs []Link
}

// Link is a page of documentation, in each language that has one.
type Link struct {
	Title Text
	URL   Text
}

// Shorthand is a version name that is not in the list of a search, and what it
// stands for.
type Shorthand struct {
	// Name is what follows `<plugin>@`.
	Name string
	// Meaning says what the name installs.
	Meaning Text
}

// Search is what a plugin takes after its name in `vfox search`.
type Search struct {
	// Label names what the user chooses, such as the distribution.
	Label Text
	// Default is the value that vfox's own behaviour amounts to without an
	// argument; choosing it passes no argument.
	Default string
	// Choices are the values a search takes.
	Choices []Choice
}

// Choice is one argument a search takes.
type Choice struct {
	// Value is the argument: the short name of a distribution.
	Value string
	// Name is how the choice is called.
	Name Text
	// Maker is who makes it, or says what it is.
	Maker Text
	// Aliases are other names the plugin accepts for it, such as the long name
	// of a distribution.
	Aliases []string
	// Popular puts the choice in front of the others.
	Popular bool
	// Everything marks a choice that lists the versions of every distribution,
	// which takes longer and gives a long list.
	Everything bool
}

// Arguments returns what follows the plugin name in `vfox search` for the
// choice value: nothing for the default and for an empty value.
func (s *Search) Arguments(value string) []string {
	if s == nil || value == "" || value == s.Default {
		return nil
	}
	return []string{value}
}

// Choice returns the choice with the given value.
func (s *Search) Choice(value string) (Choice, bool) {
	if s == nil {
		return Choice{}, false
	}
	if value == "" {
		value = s.Default
	}
	for _, choice := range s.Choices {
		if choice.Value == value {
			return choice, true
		}
	}
	return Choice{}, false
}

// Matches tells whether the choice answers to query, which is looked for in
// its value, its aliases, its name and its maker, ignoring case.
func (c Choice) Matches(query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return true
	}
	candidates := []string{c.Value, c.Name.EN, c.Name.ZH, c.Maker.EN, c.Maker.ZH}
	candidates = append(candidates, c.Aliases...)
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), needle) {
			return true
		}
	}
	return false
}

// VariableKind tells what a variable holds.
type VariableKind int

// The kinds of variables.
const (
	// VariableURL: an address that starts with http:// or https://.
	VariableURL VariableKind = iota
	// VariableChoice: one of the values of the presets.
	VariableChoice
	// VariableToggle: a switch, which is on with the value On and off with the
	// value Off, or unset when Off is empty.
	VariableToggle
	// VariableText: any text of one line, such as build options.
	VariableText
)

// String is the name of the kind in profile files and in the web API.
func (k VariableKind) String() string {
	switch k {
	case VariableChoice:
		return "choice"
	case VariableToggle:
		return "toggle"
	case VariableText:
		return "text"
	default:
		return "url"
	}
}

// Variable is an environment variable that a plugin reads.
type Variable struct {
	// Key is the name of the variable.
	Key string
	// Label names what it sets.
	Label Text
	// Help says what it does in a sentence or two.
	Help Text
	// Kind tells what it holds.
	Kind VariableKind
	// Presets are values to choose from. For VariableChoice they are the only
	// values; for VariableURL they are mirrors that were checked.
	Presets []Preset
	// On and Off are the values of a VariableToggle.
	On, Off string
	// Example is shown in an empty field.
	Example string
	// Advanced marks a variable that is for unusual systems and for tests.
	Advanced bool
	// Unvetted are the values that this profile offers and the profile of the
	// application does not: they come from the plugin, the online directory or
	// a file of the user, so the interfaces show them and ask before applying
	// them (see NeedsConfirmation).
	Unvetted []string
}

// NeedsConfirmation reports whether the value comes from a profile that the
// application did not ship, so that an interface shows it and asks before it
// applies it.
func (v Variable) NeedsConfirmation(value string) bool {
	return slices.Contains(v.Unvetted, strings.TrimSpace(value))
}

// Preset is a value of a variable, and what to call it.
type Preset struct {
	Value string
	Name  Text
}

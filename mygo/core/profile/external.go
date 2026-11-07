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

package profile

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// External profiles: curated data without a code change.
//
// A plugin repository may ship vfox-plus.json at its root describing the
// same Profile that a built-in one would hold (search choices, shorthands,
// variables, legacy files, facets, naming, guide, tips, docs). vfox-plus
// fetches it from the homepage of the plugin (see ProfileURL), with an
// optional mirror prefix such as https://gh-proxy.org/ in front, and caches
// it under the configuration directory as profiles/<plugin>.json. Files in
// that directory overlay the built-in profiles on startup; Register overlays
// anything. A plugin without any profile still works through inference (see
// infer.go).
//
// This is the declarative counterpart of https://mise-versions.jdx.dev for
// mise: new plugins are adapted by shipping data, not by changing vfox-plus.

// ProfileFileName is the file a plugin repository ships at its root.
const ProfileFileName = "vfox-plus.json"

// JSONProfile is the serialised form of a Profile. Texts are objects with
// "en" and optional "zh". Naming is "plain", "java" or "flutter".
type JSONProfile struct {
	Plugin         string           `json:"plugin"`
	Naming         string           `json:"naming,omitempty"`
	Search         *JSONSearch      `json:"search,omitempty"`
	Shorthands     []JSONShorthand  `json:"shorthands,omitempty"`
	Variables      []JSONVariable   `json:"variables,omitempty"`
	LegacyFiles    []string         `json:"legacyFiles,omitempty"`
	LegacyStrategy bool             `json:"legacyStrategy,omitempty"`
	Facets         []JSONFacet      `json:"facets,omitempty"`
	Guide          JSONText         `json:"guide,omitempty"`
	Tips           []JSONText       `json:"tips,omitempty"`
	Docs           []JSONLink       `json:"docs,omitempty"`
}

// JSONText is a bilingual text.
type JSONText struct {
	EN string `json:"en"`
	ZH string `json:"zh,omitempty"`
}

// JSONSearch is the search choices of a plugin.
type JSONSearch struct {
	Label   JSONText     `json:"label"`
	Default string       `json:"default"`
	Choices []JSONChoice `json:"choices"`
}

// JSONChoice is one search argument.
type JSONChoice struct {
	Value      string   `json:"value"`
	Name       JSONText `json:"name"`
	Maker      JSONText `json:"maker,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Popular    bool     `json:"popular,omitempty"`
	Everything bool     `json:"everything,omitempty"`
}

// JSONShorthand is an install name besides the listed versions.
type JSONShorthand struct {
	Name    string   `json:"name"`
	Meaning JSONText `json:"meaning"`
}

// JSONVariable is an environment variable a plugin reads.
type JSONVariable struct {
	Key      string       `json:"key"`
	Label    JSONText     `json:"label"`
	Help     JSONText     `json:"help"`
	Kind     string       `json:"kind,omitempty"`
	Presets  []JSONPreset `json:"presets,omitempty"`
	Example  string       `json:"example,omitempty"`
	Advanced bool         `json:"advanced,omitempty"`
}

// JSONPreset is one value of a variable.
type JSONPreset struct {
	Value string   `json:"value"`
	Name  JSONText `json:"name"`
}

// JSONFacet is a filter group.
type JSONFacet struct {
	ID     string   `json:"id"`
	Label  JSONText `json:"label"`
	Kind   string   `json:"kind,omitempty"`
	Values []string `json:"values"`
}

// JSONLink is a documentation link.
type JSONLink struct {
	Title JSONText `json:"title"`
	URL   JSONText `json:"url"`
}

func toText(t JSONText) Text { return Text{EN: t.EN, ZH: t.ZH} }

// ToProfile converts the JSON form into a Profile. Unknown naming, variable
// and facet kinds fall back to the plain defaults instead of failing, so
// that a newer file stays loadable.
func (p JSONProfile) ToProfile() Profile {
	profile := Profile{Plugin: strings.TrimSpace(p.Plugin)}
	switch strings.ToLower(strings.TrimSpace(p.Naming)) {
	case "java":
		profile.Naming = NamingJava
	case "flutter":
		profile.Naming = NamingFlutter
	default:
		profile.Naming = NamingPlain
	}
	if p.Search != nil {
		search := &Search{Label: toText(p.Search.Label), Default: p.Search.Default}
		for _, choice := range p.Search.Choices {
			search.Choices = append(search.Choices, Choice{
				Value:      choice.Value,
				Name:       toText(choice.Name),
				Maker:      toText(choice.Maker),
				Aliases:    append([]string(nil), choice.Aliases...),
				Popular:    choice.Popular,
				Everything: choice.Everything,
			})
		}
		if search.Default == "" {
			search.Default = ""
		}
		profile.Search = search
	}
	for _, shorthand := range p.Shorthands {
		if shorthand.Name == "" {
			continue
		}
		profile.Shorthands = append(profile.Shorthands, Shorthand{Name: shorthand.Name, Meaning: toText(shorthand.Meaning)})
	}
	for _, variable := range p.Variables {
		if variable.Key == "" {
			continue
		}
		kind := VariableURL
		switch strings.ToLower(strings.TrimSpace(variable.Kind)) {
		case "choice":
			kind = VariableChoice
		case "url", "":
			kind = VariableURL
		default:
			kind = VariableURL
		}
		entry := Variable{Key: variable.Key, Label: toText(variable.Label), Help: toText(variable.Help), Kind: kind, Example: variable.Example, Advanced: variable.Advanced}
		for _, preset := range variable.Presets {
			entry.Presets = append(entry.Presets, Preset{Value: preset.Value, Name: toText(preset.Name)})
		}
		profile.Variables = append(profile.Variables, entry)
	}
	profile.LegacyFiles = append([]string(nil), p.LegacyFiles...)
	profile.LegacyStrategy = p.LegacyStrategy
	for _, facet := range p.Facets {
		if facet.ID == "" || len(facet.Values) == 0 {
			continue
		}
		kind := FacetNote
		if strings.ToLower(strings.TrimSpace(facet.Kind)) == "suffix" {
			kind = FacetSuffix
		}
		profile.Facets = append(profile.Facets, Facet{ID: facet.ID, Label: toText(facet.Label), Kind: kind, Values: append([]string(nil), facet.Values...)})
	}
	profile.Guide = toText(p.Guide)
	for _, tip := range p.Tips {
		profile.Tips = append(profile.Tips, toText(tip))
	}
	for _, link := range p.Docs {
		profile.Docs = append(profile.Docs, Link{Title: toText(link.Title), URL: toText(link.URL)})
	}
	return profile
}

// ParseProfileJSON parses a vfox-plus.json file into a Profile.
func ParseProfileJSON(content []byte) (Profile, error) {
	var raw JSONProfile
	if err := json.Unmarshal(content, &raw); err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(raw.Plugin) == "" {
		return Profile{}, errMissingPlugin
	}
	return raw.ToProfile(), nil
}

// LoadExternalProfiles reads profiles/<plugin>.json under directory and
// registers each one on registry. Malformed files are skipped.
func LoadExternalProfiles(registry *Registry, directory string) int {
	if registry == nil || directory == "" {
		return 0
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0
	}
	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name())) //nolint:gosec // The path is the profiles directory.
		if err != nil {
			continue
		}
		profile, err := ParseProfileJSON(content)
		if err != nil || profile.Plugin == "" {
			continue
		}
		registry.Register(profile)
		loaded++
	}
	return loaded
}

type missingPluginError struct{}

func (missingPluginError) Error() string { return "profile has no plugin name" }

var errMissingPlugin error = missingPluginError{}

// ProfilesDir returns the directory for external profiles next to the
// settings file: <config>/profiles.
func ProfilesDir(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(settingsPath), "profiles")
}

// ProfileURL returns the URL of the vfox-plus.json of a plugin from its
// homepage, for example
// https://raw.githubusercontent.com/version-fox/vfox-gradle/main/vfox-plus.json.
// Only GitHub homepages are supported; anything else returns "".
func ProfileURL(homepage string) string {
	parsed, err := url.Parse(strings.TrimSpace(homepage))
	if err != nil || parsed.Host != "github.com" {
		return ""
	}
	segments := []string{}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) < 2 {
		return ""
	}
	owner, repo := segments[0], strings.TrimSuffix(segments[1], ".git")
	return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/main/" + ProfileFileName
}

// MirrorURL prefixes rawURL with the mirror prefix, for example
// https://gh-proxy.org/ + https://raw.githubusercontent.com/... An empty
// prefix returns the URL unchanged. It is also used by the self-update
// check to download from a mirror.
func MirrorURL(prefix, rawURL string) string {
	prefix = strings.TrimSpace(prefix)
	rawURL = strings.TrimSpace(rawURL)
	if prefix == "" || rawURL == "" {
		return rawURL
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + rawURL
}

// FetchProfile downloads the vfox-plus.json of the plugin with homepage,
// optionally through a mirror prefix such as https://gh-proxy.org/.
// A nil client uses http.DefaultClient.
func FetchProfile(ctx context.Context, client *http.Client, homepage, mirrorPrefix string) (Profile, error) {
	raw := ProfileURL(homepage)
	if raw == "" {
		return Profile{}, errNoProfileURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	for _, candidate := range []string{raw, MirrorURL(mirrorPrefix, raw)} {
		if candidate == "" {
			continue
		}
		// Without a mirror there is only one candidate.
		if mirrorPrefix == "" && candidate != raw {
			continue
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		_ = response.Body.Close()
		if err != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
			continue
		}
		if profile, err := ParseProfileJSON(body); err == nil && profile.Plugin != "" {
			return profile, nil
		}
	}
	return Profile{}, errFetchProfile
}

type profileURLError struct{ message string }

func (e profileURLError) Error() string { return e.message }

var (
	errNoProfileURL = profileURLError{"no profile URL for this homepage"}
	errFetchProfile = profileURLError{"could not fetch a profile"}
)

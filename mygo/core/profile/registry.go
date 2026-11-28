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
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Registry is the profiles that the application knows, from every source: its
// own, those of the online directory (see Catalog), those of the user and, for
// an installed plugin, the one that the plugin ships.
//
// It is safe for use from several goroutines.
type Registry struct {
	platform Platform

	mu         sync.RWMutex
	builtin    map[string]Document
	catalog    map[string]Document
	user       map[string]Document
	registered map[string]Profile
	listeners  map[int]func()
	next       int
}

// NewRegistry creates a registry for the platform that vfox runs on, with the
// profiles that the application ships (see package builtin), by plugin name.
func NewRegistry(platform Platform, builtin map[string]Document) *Registry {
	return &Registry{platform: platform, builtin: maps.Clone(builtin)}
}

// Platform is the platform whose profiles the registry gives.
func (r *Registry) Platform() Platform { return r.platform }

// Resolution is what the registry knows about a plugin.
type Resolution struct {
	Profile Profile
	// Found reports that some profile describes the plugin.
	Found bool
	// Curated reports that the application or its online directory describes
	// the plugin: a section that the profile lacks, the plugin lacks. Without
	// it, the sections that no profile carries can be inferred.
	Curated bool
	// Sections are the sections that some profile carries.
	Sections Sections
}

// Lookup returns the profile of the plugin, if there is one. The slices of the
// profile belong to the registry: do not change them.
func (r *Registry) Lookup(plugin string) (Profile, bool) {
	resolution := r.Resolve(plugin, "", nil)
	return resolution.Profile, resolution.Found
}

// Resolve returns what is known about an installed plugin that vfox calls
// name, whose metadata says that it is declared (empty when it is not known
// yet), and which ships the profile own (nil when it ships none).
//
// Every source is looked up by name, and by the declared name when it has
// nothing by name: a plugin added under an alias has the profile of the plugin
// that it is. Where several describe the plugin, each section comes from the
// first of the user, the plugin, the online directory and the application.
func (r *Registry) Resolve(name, declared string, own *Document) Resolution {
	if r == nil {
		return Resolution{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	find := func(documents map[string]Document) (Document, bool) {
		if d, ok := documents[name]; ok {
			return d, true
		}
		d, ok := documents[declared]
		return d, ok && declared != ""
	}
	if p, ok := r.registered[name]; ok {
		return Resolution{Profile: p, Found: true, Curated: true, Sections: ^Sections(0)}
	}
	if p, ok := r.registered[declared]; ok && declared != "" {
		return Resolution{Profile: p, Found: true, Curated: true, Sections: ^Sections(0)}
	}
	var layers []Document
	user, hasUser := find(r.user)
	if hasUser {
		layers = append(layers, user)
	}
	if own != nil {
		layers = append(layers, *own)
	}
	catalog, hasCatalog := find(r.catalog)
	if hasCatalog {
		layers = append(layers, catalog)
	}
	builtin, hasBuiltin := find(r.builtin)
	if hasBuiltin {
		layers = append(layers, builtin)
	}
	if len(layers) == 0 {
		return Resolution{}
	}
	merged := Merge(layers...)
	profile := merged.Resolve(r.platform)
	// A file that the plugin ships may not say what it is.
	switch {
	case profile.Plugin != "":
	case declared != "":
		profile.Plugin = declared
	default:
		profile.Plugin = name
	}
	if hasUser || hasCatalog || own != nil {
		var known []Variable
		if hasBuiltin {
			known = builtin.Resolve(r.platform).Variables
		}
		profile.Variables = markUnvetted(profile.Variables, known)
	}
	return Resolution{Profile: profile, Found: true, Curated: hasCatalog || hasBuiltin, Sections: merged.Sections()}
}

// markUnvetted returns the variables with the values that the variables of the
// application do not offer marked as unvetted.
func markUnvetted(variables, vetted []Variable) []Variable {
	var marked []Variable
	for _, variable := range variables {
		offered := map[string]bool{}
		for _, known := range vetted {
			if known.Key == variable.Key {
				for _, value := range known.values() {
					offered[value] = true
				}
			}
		}
		variable.Unvetted = nil
		for _, value := range variable.values() {
			if !offered[value] {
				variable.Unvetted = append(variable.Unvetted, value)
			}
		}
		marked = append(marked, variable)
	}
	return marked
}

// Register describes a plugin with the profile as it is, whatever the files say:
// for tests.
func (r *Registry) Register(profile Profile) {
	r.mu.Lock()
	if r.registered == nil {
		r.registered = make(map[string]Profile)
	}
	r.registered[profile.Plugin] = profile
	r.mu.Unlock()
	r.changed()
}

// SetUser replaces the profiles of the user.
func (r *Registry) SetUser(documents map[string]Document) {
	r.mu.Lock()
	r.user = maps.Clone(documents)
	r.mu.Unlock()
	r.changed()
}

// SetCatalog replaces the profiles of the online directory.
func (r *Registry) SetCatalog(documents map[string]Document) {
	r.mu.Lock()
	r.catalog = maps.Clone(documents)
	r.mu.Unlock()
	r.changed()
}

// OnChange calls listener after the profiles change, until the returned
// function is called.
func (r *Registry) OnChange(listener func()) (stop func()) {
	if r == nil {
		return func() {}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listeners == nil {
		r.listeners = make(map[int]func())
	}
	id := r.next
	r.next++
	r.listeners[id] = listener
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.listeners, id)
	}
}

func (r *Registry) changed() {
	r.mu.RLock()
	listeners := make([]func(), 0, len(r.listeners))
	for _, listener := range r.listeners {
		listeners = append(listeners, listener)
	}
	r.mu.RUnlock()
	for _, listener := range listeners {
		listener()
	}
}

// ProfilesDir returns the directory of the profiles of the user, next to the
// settings file: <config>/profiles.
func ProfilesDir(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(settingsPath), "profiles")
}

// LoadUserProfiles reads the profiles of the user, the JSON files under
// directory, by the plugin that each names. A file that names no plugin, or
// that is not a profile, is skipped; its problems are returned with its name.
func LoadUserProfiles(directory string) (map[string]Document, []Problem) {
	if directory == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil
	}
	documents := make(map[string]Document)
	var problems []Problem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		report := func(found ...Problem) {
			for _, problem := range found {
				problem.Path = strings.TrimSuffix(entry.Name()+": "+problem.Path, ": ")
				problems = append(problems, problem)
			}
		}
		content, err := readFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			report(Problem{Message: err.Error()})
			continue
		}
		d, found, err := ParseDocument(content)
		report(found...)
		switch {
		case err != nil:
			report(Problem{Message: err.Error()})
		case d.Plugin == "":
			report(Problem{Path: "plugin", Message: "is missing"})
		default:
			documents[d.Plugin] = d
		}
	}
	return documents, problems
}

// Filled is the profile of the resolution with the sections that no profile
// carries taken from inferred, for the sections that can be inferred: the
// variables, the version files, the filters, the guide and the documentation.
func (r Resolution) Filled(inferred Profile) Profile {
	filled := r.Profile
	if !r.Sections.Has(SectionVariables) {
		filled.Variables = inferred.Variables
	}
	if !r.Sections.Has(SectionLegacyFiles) {
		filled.LegacyFiles = inferred.LegacyFiles
	}
	if len(filled.Facets) == 0 {
		filled.Facets = inferred.Facets
	}
	if !r.Sections.Has(SectionGuide) {
		filled.Guide = inferred.Guide
	}
	if !r.Sections.Has(SectionDocs) {
		filled.Docs = inferred.Docs
	}
	return filled
}

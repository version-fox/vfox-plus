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

package pair

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// What the web interface learns about a plugin besides its versions: the profile
// of core/profile as JSON. The texts come in both languages, so that the page
// chooses by its own language setting without asking again.

type textDTO struct {
	EN string `json:"en"`
	ZH string `json:"zh,omitempty"`
}

func textOf(t profile.Text) textDTO { return textDTO{EN: t.EN, ZH: t.ZH} }

type choiceDTO struct {
	Value      string   `json:"value"`
	Name       textDTO  `json:"name"`
	Maker      textDTO  `json:"maker"`
	Aliases    []string `json:"aliases,omitempty"`
	Popular    bool     `json:"popular,omitempty"`
	Everything bool     `json:"everything,omitempty"`
}

type searchDTO struct {
	Label   textDTO `json:"label"`
	Default string  `json:"default"`
	// Current is the value of the choice that the plugin lists: the one that the user
	// made, the default if they made none.
	Current string      `json:"current"`
	Choices []choiceDTO `json:"choices"`
}

type shorthandDTO struct {
	Name    string  `json:"name"`
	Meaning textDTO `json:"meaning"`
}

type presetDTO struct {
	Value string  `json:"value"`
	Name  textDTO `json:"name"`
}

type variableDTO struct {
	Key      string      `json:"key"`
	Label    textDTO     `json:"label"`
	Help     textDTO     `json:"help"`
	Kind     string      `json:"kind"`
	Presets  []presetDTO `json:"presets"`
	Example  string      `json:"example,omitempty"`
	Advanced bool        `json:"advanced,omitempty"`
	// Value is what vfox runs with now, empty when the variable is not set.
	Value string `json:"value"`
}

type facetDTO struct {
	ID     string   `json:"id"`
	Label  textDTO  `json:"label"`
	Values []string `json:"values"`
}

type linkDTO struct {
	Title textDTO `json:"title"`
	URL   textDTO `json:"url"`
}

type partsDTO struct {
	Version string     `json:"version"`
	Source  *choiceDTO `json:"source,omitempty"`
	Flavors []string   `json:"flavors,omitempty"`
}

type profileDTO struct {
	Plugin         string         `json:"plugin"`
	Naming         string         `json:"naming"`
	Search         *searchDTO     `json:"search,omitempty"`
	Shorthands     []shorthandDTO `json:"shorthands"`
	Variables      []variableDTO  `json:"variables"`
	LegacyFiles    []string       `json:"legacyFiles"`
	LegacyStrategy bool           `json:"legacyStrategy"`
	Facets         []facetDTO     `json:"facets"`
	Guide          textDTO        `json:"guide"`
	Tips           []textDTO      `json:"tips"`
	Docs           []linkDTO      `json:"docs"`
	// Installed takes the names of the installed versions apart, by name.
	Installed map[string]partsDTO `json:"installed,omitempty"`
}

func choiceOf(c profile.Choice) choiceDTO {
	return choiceDTO{
		Value: c.Value, Name: textOf(c.Name), Maker: textOf(c.Maker), Aliases: c.Aliases, Popular: c.Popular,
		Everything: c.Everything,
	}
}

func partsOf(p profile.Profile, name string) (partsDTO, bool) {
	parts, ok := p.Describe(name)
	if !ok {
		return partsDTO{}, false
	}
	dto := partsDTO{Version: parts.Version, Flavors: parts.Flavors}
	if parts.SourceValue != "" {
		source := profile.Choice{Value: parts.SourceValue, Name: parts.Source}
		converted := choiceOf(source)
		dto.Source = &converted
	}
	return dto, true
}

func namingName(n profile.Naming) string {
	switch n {
	case profile.NamingJava:
		return "java"
	case profile.NamingFlutter:
		return "flutter"
	default:
		return "plain"
	}
}

// profileOf returns the profile of the plugin that the details are of. vfox names a
// plugin that was added under an alias by the alias, so a plugin without a profile
// under its own name is asked which plugin it is.
func profileOf(detail *state.SdkDetailController) (profile.Profile, bool) {
	if _, known := detail.Profile(); !known {
		awaitLoad(detail.Info.EnsureLoaded())
	}
	return detail.Profile()
}

// newProfileDTO describes the profile for the web interface, with the values of
// its variables and the installed versions it takes apart. The installed versions
// are those of the plugin of the details, which is not called what the profile
// calls it when it was added under an alias.
func newProfileDTO(model *state.AppModel, detail *state.SdkDetailController, p profile.Profile) profileDTO {
	dto := profileDTO{
		Plugin: p.Plugin, Naming: namingName(p.Naming), LegacyFiles: orEmpty(p.LegacyFiles), LegacyStrategy: p.LegacyStrategy,
		Guide: textOf(p.Guide), Shorthands: []shorthandDTO{}, Variables: []variableDTO{}, Facets: []facetDTO{},
		Tips: []textDTO{}, Docs: []linkDTO{},
	}
	if p.Search != nil {
		search := &searchDTO{
			Label: textOf(p.Search.Label), Default: p.Search.Default, Current: detail.Search(),
			Choices: make([]choiceDTO, 0, len(p.Search.Choices)),
		}
		for _, choice := range p.Search.Choices {
			search.Choices = append(search.Choices, choiceOf(choice))
		}
		dto.Search = search
	}
	for _, shorthand := range p.Shorthands {
		dto.Shorthands = append(dto.Shorthands, shorthandDTO{Name: shorthand.Name, Meaning: textOf(shorthand.Meaning)})
	}
	for _, variable := range p.Variables {
		kind := "url"
		if variable.Kind == profile.VariableChoice {
			kind = "choice"
		}
		converted := variableDTO{
			Key: variable.Key, Label: textOf(variable.Label), Help: textOf(variable.Help), Kind: kind, Example: variable.Example,
			Advanced: variable.Advanced, Value: model.PluginVariable(variable.Key), Presets: []presetDTO{},
		}
		for _, preset := range variable.Presets {
			converted.Presets = append(converted.Presets, presetDTO{Value: preset.Value, Name: textOf(preset.Name)})
		}
		dto.Variables = append(dto.Variables, converted)
	}
	for _, facet := range p.Facets {
		dto.Facets = append(dto.Facets, facetDTO{ID: facet.ID, Label: textOf(facet.Label), Values: facet.Values})
	}
	for _, tip := range p.Tips {
		dto.Tips = append(dto.Tips, textOf(tip))
	}
	for _, link := range p.Docs {
		dto.Docs = append(dto.Docs, linkDTO{Title: textOf(link.Title), URL: textOf(link.URL)})
	}
	if sdk, ok := model.InstalledSdk(detail.Name); ok {
		for _, version := range sdk.Versions {
			if parts, ok := partsOf(p, version); ok {
				if dto.Installed == nil {
					dto.Installed = make(map[string]partsDTO)
				}
				dto.Installed[version] = parts
			}
		}
	}
	return dto
}

// versionDTO is a version that can be installed. It carries what the browser would
// have to know a profile for: the version name taken apart, and the values of the
// filter chips the version has.
type versionDTO struct {
	core.AvailableVersion
	Parts  *partsDTO           `json:"parts,omitempty"`
	Facets map[string][]string `json:"facets,omitempty"`
}

// annotate adds to each version what the profile makes of it; a plugin without a
// profile keeps the plain list.
func annotate(p profile.Profile, hasProfile bool, versions []core.AvailableVersion) []versionDTO {
	result := make([]versionDTO, 0, len(versions))
	for _, version := range versions {
		entry := versionDTO{AvailableVersion: version}
		if hasProfile {
			if parts, ok := partsOf(p, version.Version); ok {
				entry.Parts = &parts
			}
			entry.Facets = p.Values(version)
		}
		result = append(result, entry)
	}
	return result
}

// serveProfile answers /api/sdks/<name>/profile.
func (s *Server) serveProfile(writer http.ResponseWriter, model *state.AppModel, detail *state.SdkDetailController) {
	known, ok := profileOf(detail)
	if !ok {
		http.Error(writer, "No profile for this plugin.", http.StatusNotFound)
		return
	}
	writeJSON(writer, newProfileDTO(model, detail, known))
}

// serveSearch answers PUT /api/sdks/<name>/search: choose what the plugin lists, by the
// value of one of the choices of its profile (the short name of a distribution for java;
// the default of the plugin is a choice too). The application keeps the choice, so that
// it is what the plugin lists from now on, here, on the desktop and after a restart.
func (s *Server) serveSearch(writer http.ResponseWriter, request *http.Request, detail *state.SdkDetailController) {
	if request.Method != http.MethodPut {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireSameOrigin(writer, request) {
		return
	}
	var req struct {
		Value *string `json:"value"`
	}
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil || req.Value == nil {
		http.Error(writer, "A choice is required.", http.StatusBadRequest)
		return
	}
	known, ok := profileOf(detail)
	if !ok || known.Search == nil {
		http.Error(writer, "No choice of what to list for this plugin.", http.StatusNotFound)
		return
	}
	if err := detail.ChooseSearch(*req.Value); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(writer, map[string]string{"value": detail.Search()})
}

// serveVariables answers PUT /api/sdks/<name>/variables: set one environment variable of
// the plugin, which the profile lists, or take it away with an empty value.
func (s *Server) serveVariables(writer http.ResponseWriter, request *http.Request, model *state.AppModel, detail *state.SdkDetailController) {
	if request.Method != http.MethodPut {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireSameOrigin(writer, request) {
		return
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil || strings.TrimSpace(req.Key) == "" {
		http.Error(writer, "A variable is required.", http.StatusBadRequest)
		return
	}
	known, ok := profileOf(detail)
	if !ok {
		http.Error(writer, "No profile for this plugin.", http.StatusNotFound)
		return
	}
	if _, ok := known.Variable(req.Key); !ok {
		http.Error(writer, detail.Name+" does not read "+req.Key+".", http.StatusBadRequest)
		return
	}
	if err := model.SetPluginVariable(known, req.Key, req.Value); err != nil {
		var invalid *profile.ValueError
		if errors.As(err, &invalid) {
			http.Error(writer, invalid.Error(), http.StatusBadRequest)
			return
		}
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(writer, map[string]string{"key": req.Key, "value": model.PluginVariable(req.Key)})
}

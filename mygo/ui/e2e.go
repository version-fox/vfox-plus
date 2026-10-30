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

//go:build e2e

package ui

import (
	"strconv"
	"strings"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
)

// startE2E prepares the application for the end-to-end tests, which vfxtool runs
// through the automation plugin of mygo (see mygo/automation_e2e.go, and
// MYGO_AUTOMATION, which names the socket): the offline test plugin gets a
// profile that takes an argument, like vfox-java does. The scenarios themselves
// are in shared/tooling; the application has no code for them.
func (a *App) startE2E() {
	if a.environment["MYGO_AUTOMATION"] == "" {
		return
	}
	a.model.Profiles.Register(testsdkProfile())
}

// testsdkProfile is the profile of the offline test plugin, which takes an argument
// like vfox-java does: `lts` lists the releases with long-term support only.
func testsdkProfile() profile.Profile {
	return profile.Profile{
		Plugin: "testsdk",
		Search: &profile.Search{
			Label: profile.Text{EN: "Channel"}, Default: "all",
			Choices: []profile.Choice{
				{Value: "all", Name: profile.Text{EN: "Every version"}, Popular: true},
				{Value: "lts", Name: profile.Text{EN: "Long-term support"}, Popular: true},
			},
		},
		Shorthands: []profile.Shorthand{{Name: "1.0.0", Meaning: profile.Text{EN: "The first release"}}},
		Facets:     []profile.Facet{{ID: "support", Label: profile.Text{EN: "Support"}, Kind: profile.FacetNote, Values: []string{"LTS"}}},
	}
}

// windowSize is the size of the window in dp: VFOX_PLUS_E2E_WINDOW, as WIDTHxHEIGHT,
// opens the window of the compact layout for its screenshots.
func (a *App) windowSize() (width, height int) {
	width, height = defaultWidth, defaultHeight
	if w, h, ok := strings.Cut(a.environment["VFOX_PLUS_E2E_WINDOW"], "x"); ok {
		wn, werr := strconv.Atoi(w)
		hn, herr := strconv.Atoi(h)
		if werr == nil && herr == nil && wn > 0 && hn > 0 {
			return wn, hn
		}
	}
	return width, height
}

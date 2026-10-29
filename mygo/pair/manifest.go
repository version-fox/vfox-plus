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
	"net/http"
)

// manifestIcon is an icon of the web app manifest.
type manifestIcon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose"`
}

// webManifest is the web app manifest of the web interface, which lets
// Chromium browsers install it as an application: a window of its own with the
// icon of vfox+. https://www.w3.org/TR/appmanifest/
type webManifest struct {
	Name            string         `json:"name"`
	ShortName       string         `json:"short_name"`
	Description     string         `json:"description"`
	ID              string         `json:"id"`
	StartURL        string         `json:"start_url"`
	Scope           string         `json:"scope"`
	Display         string         `json:"display"`
	BackgroundColor string         `json:"background_color"`
	ThemeColor      string         `json:"theme_color"`
	Icons           []manifestIcon `json:"icons"`
}

// manifestJSON is the manifest, built once. Its icon is the favicon: a vector
// icon of any size, which Chromium accepts for installing (a 144 px or larger
// PNG, SVG or WebP icon of the purpose "any"), inlined as a data URI like
// the other assets of the pages.
var manifestJSON = func() []byte {
	content, err := json.Marshal(webManifest{
		Name:        "vfox+",
		ShortName:   "vfox+",
		Description: "Install and switch SDK versions with vfox",
		ID:          "/app",
		StartURL:    "/app",
		Scope:       "/",
		Display:     "standalone",
		// The colours of the web interface: its theme-color and the light
		// background of its colour scheme.
		BackgroundColor: "#fff8f5",
		ThemeColor:      "#88521c",
		Icons:           []manifestIcon{{Src: faviconURI, Sizes: "any", Type: "image/svg+xml", Purpose: "any"}},
	})
	if err != nil {
		panic("pair: the web manifest: " + err.Error()) // The data is part of the source.
	}
	return content
}()

// serveManifest answers the web app manifest. It needs no session, like the
// intro page: it names the application and its start page and nothing else,
// and a browser may process a manifest without the page that links it (the
// specification mentions installing from a sync service), so without the
// session cookie.
func (s *Server) serveManifest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "application/manifest+json")
	writer.Header().Set("Cache-Control", "no-cache")
	_, _ = writer.Write(manifestJSON)
}

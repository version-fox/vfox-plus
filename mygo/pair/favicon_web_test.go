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

package pair_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

var faviconLinkPattern = regexp.MustCompile(`<link rel="icon" type="image/svg\+xml" href="data:image/svg\+xml,[^"]+">`)

// Every page of the server is seen in a tab: the intro that a visit without a
// link gets, the page that confirms a pairing and the web interface.
func TestPagesCarryTheFavicon(t *testing.T) {
	server := webServer(t, testModel(t, testsupport.FixtureRunner()))
	_, link, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]string{}
	status, body := webGet(t, server.BaseURL()+"/", "")
	if status != http.StatusOK {
		t.Fatalf("intro: got %d", status)
	}
	pages["intro"] = body
	status, body = webGet(t, link, "")
	if status != http.StatusOK {
		t.Fatalf("connect: got %d", status)
	}
	pages["connect"] = body
	session := webSession(t, server)
	status, body = webGet(t, server.BaseURL()+"/app", session)
	if status != http.StatusOK {
		t.Fatalf("app: got %d", status)
	}
	pages["app"] = body
	for name, page := range pages {
		head, _, _ := strings.Cut(page, "</head>")
		if len(faviconLinkPattern.FindAllString(head, -1)) != 1 {
			t.Errorf("the %s page needs one favicon link in its head", name)
		}
		if strings.Contains(page, "<!--favicon-->") {
			t.Errorf("the %s page still has the placeholder", name)
		}
		if !strings.HasPrefix(strings.TrimSpace(page), "<!--") && !strings.HasPrefix(strings.TrimSpace(page), "<!doctype html>") {
			t.Errorf("the %s page does not start as a page: %.40s", name, page)
		}
	}
}

// The manifest needs no session: a browser fetches it without the cookie of the page, and
// its icon is the favicon that the pages carry.
func TestManifestIsServed(t *testing.T) {
	server := webServer(t, testModel(t, testsupport.FixtureRunner()))
	response, err := http.Get(server.BaseURL() + "/manifest.webmanifest")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/manifest+json" {
		t.Fatalf("status %d, content type %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	var manifest struct {
		Name     string `json:"name"`
		StartURL string `json:"start_url"`
		Icons    []struct {
			Src   string `json:"src"`
			Sizes string `json:"sizes"`
			Type  string `json:"type"`
		} `json:"icons"`
	}
	if err := json.NewDecoder(response.Body).Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "vfox+" || manifest.StartURL != "/app" || len(manifest.Icons) != 1 || manifest.Icons[0].Sizes != "any" || manifest.Icons[0].Type != "image/svg+xml" {
		t.Fatalf("manifest = %+v", manifest)
	}
	status, app := webGet(t, server.BaseURL()+"/app", webSession(t, server))
	if status != http.StatusOK {
		t.Fatalf("app: got %d", status)
	}
	if !strings.Contains(app, `<link rel="manifest" href="/manifest.webmanifest">`) {
		t.Error("the web interface does not link its manifest")
	}
	href := regexp.MustCompile(`<link rel="icon" type="image/svg\+xml" href="([^"]+)">`).FindStringSubmatch(app)
	if href == nil || href[1] != manifest.Icons[0].Src {
		t.Error("the icon of the manifest is not the favicon of the page")
	}
	// The same icon is the brand mark of the navigation rail, in place of the "v+" it had.
	if !strings.Contains(app, `<img class="mark" src="`+manifest.Icons[0].Src+`"`) || strings.Contains(app, `>v+<`) {
		t.Error("the navigation rail does not show the icon")
	}
	request, err := http.NewRequest(http.MethodPost, server.BaseURL()+"/manifest.webmanifest", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	post, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = post.Body.Close() }()
	if post.StatusCode != http.StatusMethodNotAllowed || post.Header.Get("Allow") != "GET, HEAD" {
		t.Errorf("POST: status %d, Allow %q", post.StatusCode, post.Header.Get("Allow"))
	}
}

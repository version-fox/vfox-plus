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

package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var faviconLinkPattern = regexp.MustCompile(`<link rel="icon" type="image/svg\+xml" href="data:image/svg\+xml,[^"]+">`)

// Every page of the gateway is seen in a tab: the page that a browser that is not paired gets, and the
// interface that a paired browser gets.
func TestPagesCarryTheFavicon(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	pages := map[string]string{}
	status, body := unpaired(t, server).get("/")
	if status != http.StatusOK {
		t.Fatalf("the page of the gateway: got %d", status)
	}
	pages["gateway"] = body
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	status, body = b.get("/app")
	if status != http.StatusOK {
		t.Fatalf("the interface: got %d", status)
	}
	pages["interface"] = body
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

// The manifest needs no session: a browser fetches it without the cookie of the page, and its icon is the
// favicon that the pages carry. The gateway answers it itself.
func TestManifestIsServed(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	response := unpaired(t, server).send(http.MethodGet, "/manifest.webmanifest", nil, nil)
	manifestBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
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
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "vfox+" || manifest.StartURL != "/app" || len(manifest.Icons) != 1 || manifest.Icons[0].Sizes != "any" || manifest.Icons[0].Type != "image/svg+xml" {
		t.Fatalf("manifest = %+v", manifest)
	}
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	status, app := b.get("/app")
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
	post := b.send(http.MethodPost, "/manifest.webmanifest", strings.NewReader("{}"), nil)
	if post.StatusCode != http.StatusMethodNotAllowed || post.Header.Get("Allow") != "GET, HEAD" {
		t.Errorf("POST: status %d, Allow %q", post.StatusCode, post.Header.Get("Allow"))
	}
}

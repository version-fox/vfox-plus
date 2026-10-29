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

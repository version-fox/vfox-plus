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
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

// The state-changing routes refuse what the browser says is cross-origin, with
// the cross-origin protection of net/http, and let through what it says is the
// page itself and what no browser sent.
func TestStateChangingRoutesRefuseCrossOriginBrowsers(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	host := strings.TrimPrefix(server.BaseURL(), "http://")
	hostOf := func(address string) string { name, _, _ := strings.Cut(address, ":"); return name }
	const body = `{"operation":"bogus"}` // Refused with 400 once it gets past the check.
	for _, c := range []struct {
		label   string
		headers map[string]string
		want    int
	}{
		{"curl, without either header", nil, http.StatusBadRequest},
		{"the page itself, by its fetch metadata", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusBadRequest},
		{"the page itself, by its origin", map[string]string{"Origin": server.BaseURL()}, http.StatusBadRequest},
		{"a page of another site", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"a page of the same site, on another port", map[string]string{"Sec-Fetch-Site": "same-site"}, http.StatusForbidden},
		{"a page of another origin", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"another port of the same host", map[string]string{"Origin": "http://" + hostOf(host) + ":1"}, http.StatusForbidden},
		{"an opaque origin", map[string]string{"Origin": "null"}, http.StatusForbidden},
	} {
		request, err := http.NewRequest(http.MethodPost, server.BaseURL()+"/api/tasks", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		request.Header.Set("Content-Type", "application/json")
		for name, value := range c.headers {
			request.Header.Set(name, value)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != c.want {
			t.Errorf("%s: got %d, want %d", c.label, response.StatusCode, c.want)
		}
	}
}

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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

// javaRunner answers like vfox with Java installed next to the fixtures.
func javaRunner() *testsupport.FakeProcessRunner {
	return testsupport.FixtureRunner().
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n├─┬java\n│ ├──21.0.12.1+1-tem\n│ └──25.0.4+9-librca-fx\n└─┬nodejs\n  └──24.11.1\n"}).
		On("current", core.ProcessOutput{Stdout: "java -> v21.0.12.1+1-tem\nnodejs -> v24.11.1\n"})
}

type versionsAnswer []struct {
	Version string
	Note    string
	HasNote bool
	Parts   *struct {
		Version string
		Source  *struct {
			Value string
			Name  struct{ EN, ZH string }
		}
		Flavors []string
	}
	Facets map[string][]string
}

func webRequest(t *testing.T, method, url, session, body, origin string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(content)
}

func TestWebVersionsOfAChosenDistributionCarryWhatTheProfileMakesOfThem(t *testing.T) {
	runner := javaRunner()
	model := testModel(t, runner)
	server := webServer(t, model)
	session := webSession(t, server)

	status, body := webGet(t, server.BaseURL()+"/api/sdks/java/versions?search=tem", session)
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	var versions versionsAnswer
	decodeJSON(t, body, &versions)
	if len(versions) != 120 || versions[0].Version != "27+35-tem" {
		t.Fatalf("Temurin lists %d versions: %s", len(versions), body[:200])
	}
	if calls := runner.Calls("search java tem"); len(calls) != 1 {
		t.Errorf("the argument reaches vfox: %v", runner.Invocations())
	}
	first := versions[0]
	if first.Parts == nil || first.Parts.Version != "27+35" || first.Parts.Source == nil || first.Parts.Source.Value != "tem" || first.Parts.Source.Name.EN != "Temurin" {
		t.Errorf("the name is taken apart: %+v", first.Parts)
	}
	lts := 0
	for _, version := range versions {
		if len(version.Facets["support"]) == 1 && version.Facets["support"][0] == "LTS" {
			lts++
		}
	}
	if lts == 0 || lts == len(versions) {
		t.Errorf("the notes tell which versions have long-term support: %d of %d", lts, len(versions))
	}
	if first.Facets != nil {
		t.Errorf("a version without a note has no facets: %v", first.Facets)
	}

	// Without an argument the plugin lists OpenJDK, like vfox does.
	status, body = webGet(t, server.BaseURL()+"/api/sdks/java/versions", session)
	var open versionsAnswer
	decodeJSON(t, body, &open)
	if status != http.StatusOK || len(open) != 75 || open[0].Parts.Source.Value != "open" || open[0].Parts.Source.Name.ZH != "" {
		t.Errorf("got %d, %d versions", status, len(open))
	}
	if calls := runner.Calls("search java"); len(calls) != 2 || len(calls[1].Arguments) != 2 {
		t.Errorf("the default passes no argument: %v", calls)
	}

	// JavaFX builds say so in their flavours.
	status, body = webGet(t, server.BaseURL()+"/api/sdks/java/versions?search=all", session)
	var all versionsAnswer
	decodeJSON(t, body, &all)
	javafx := 0
	for _, version := range all {
		for _, flavor := range version.Parts.Flavors {
			if flavor == "JavaFX" {
				javafx++
				if len(version.Facets["flavor"]) != 1 {
					t.Fatalf("%s is JavaFX and has no such value: %+v", version.Version, version.Facets)
				}
			}
		}
	}
	// The real list of vfox-java 0.6.3 has 418 builds with JavaFX among its 2641 versions.
	if status != http.StatusOK || len(all) != 2641 || javafx != 418 {
		t.Errorf("got %d, %d versions, %d with JavaFX", status, len(all), javafx)
	}
}

func TestWebVersionsRefuseASearchThePluginDoesNotTake(t *testing.T) {
	runner := javaRunner()
	model := testModel(t, runner)
	server := webServer(t, model)
	session := webSession(t, server)
	for _, path := range []string{"/api/sdks/java/versions?search=nope", "/api/sdks/nodejs/versions?search=tem", "/api/sdks/maven/versions?search=tem"} {
		if status, body := webGet(t, server.BaseURL()+path, session); status != http.StatusBadRequest {
			t.Errorf("%s: got %d: %s", path, status, body)
		}
	}
	if calls := runner.Calls("search"); len(calls) != 0 {
		t.Errorf("nothing was searched: %v", calls)
	}
	// A plugin without a profile lists its versions as before, without annotations.
	status, body := webGet(t, server.BaseURL()+"/api/sdks/maven/versions", session)
	if status != http.StatusOK || strings.Contains(body, `"parts"`) || strings.Contains(body, `"facets"`) {
		t.Errorf("got %d: %.200s", status, body)
	}
}

func TestWebProfileDescribesWhatThePluginOffers(t *testing.T) {
	model := testModel(t, javaRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webGet(t, server.BaseURL()+"/api/sdks/java/profile", session)
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	var answer struct {
		Plugin string
		Naming string
		Search struct {
			Label   struct{ EN, ZH string }
			Default string
			Choices []struct {
				Value      string
				Everything bool
			}
		}
		Shorthands  []struct{ Name string }
		Variables   []struct{ Key, Kind, Value string }
		LegacyFiles []string
		Facets      []struct {
			ID     string
			Values []string
		}
		Tips      []struct{ EN, ZH string }
		Docs      []struct{ URL struct{ EN, ZH string } }
		Installed map[string]struct {
			Version string
			Source  struct{ Value string }
			Flavors []string
		}
	}
	decodeJSON(t, body, &answer)
	if answer.Plugin != "java" || answer.Naming != "java" || answer.Search.Default != "open" || len(answer.Search.Choices) != 32 {
		t.Errorf("got %+v", answer)
	}
	if answer.Search.Label.EN != "Distribution" || answer.Search.Label.ZH != "发行版" {
		t.Errorf("the labels come in both languages: %+v", answer.Search.Label)
	}
	if len(answer.Variables) != 1 || answer.Variables[0].Key != "VFOX_JAVA_GITHUB_MIRROR" || answer.Variables[0].Kind != "url" || answer.Variables[0].Value != "" {
		t.Errorf("variables = %+v", answer.Variables)
	}
	if len(answer.LegacyFiles) != 1 || answer.LegacyFiles[0] != ".sdkmanrc" || len(answer.Facets) != 2 || len(answer.Shorthands) == 0 || len(answer.Tips) == 0 {
		t.Errorf("got %+v", answer)
	}
	if len(answer.Docs) == 0 || !strings.Contains(answer.Docs[0].URL.ZH, "/docs/zh-CN/") {
		t.Errorf("docs = %+v", answer.Docs)
	}
	installed := answer.Installed["25.0.4+9-librca-fx"]
	if installed.Version != "25.0.4+9" || installed.Source.Value != "librca" || len(installed.Flavors) != 1 || installed.Flavors[0] != "JavaFX" {
		t.Errorf("the installed names are taken apart: %+v", answer.Installed)
	}

	if status, _ := webGet(t, server.BaseURL()+"/api/sdks/maven/profile", session); status != http.StatusNotFound {
		t.Errorf("maven has no profile, got %d", status)
	}
	if status, _ := webGet(t, server.BaseURL()+"/api/sdks/java/profile", ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
}

func TestWebVariablesOfAPluginAreSetCheckedAndRemoved(t *testing.T) {
	runner := javaRunner()
	model := testModel(t, runner)
	server := webServer(t, model)
	session := webSession(t, server)
	url := server.BaseURL() + "/api/sdks/nodejs/variables"
	const mirror = "https://npmmirror.com/mirrors/node"

	status, body := webRequest(t, http.MethodPut, url, session, `{"key":"VFOX_NODEJS_MIRROR","value":"`+mirror+`"}`, "")
	if status != http.StatusOK || !strings.Contains(body, mirror) {
		t.Fatalf("got %d: %s", status, body)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("stored %q", got)
	}
	// The profile shows the value, and so does the settings page.
	_, body = webGet(t, server.BaseURL()+"/api/sdks/nodejs/profile", session)
	if !strings.Contains(body, `"value":"`+mirror+`"`) {
		t.Errorf("the profile shows the value: %s", body)
	}
	_, body = webGet(t, server.BaseURL()+"/api/settings", session)
	if !strings.Contains(body, "VFOX_NODEJS_MIRROR") {
		t.Errorf("the settings list it: %s", body)
	}

	for _, c := range []struct {
		body   string
		method string
		origin string
		want   int
	}{
		{`{"key":"VFOX_NODEJS_MIRROR","value":"npmmirror.com"}`, http.MethodPut, "", http.StatusBadRequest},
		{`{"key":"VFOX_NODEJS_LIBC","value":"gnu"}`, http.MethodPut, "", http.StatusBadRequest},
		{`{"key":"PATH","value":"/x"}`, http.MethodPut, "", http.StatusBadRequest},
		{`{"key":"","value":"x"}`, http.MethodPut, "", http.StatusBadRequest},
		{`nonsense`, http.MethodPut, "", http.StatusBadRequest},
		{`{"key":"VFOX_NODEJS_MIRROR","value":"` + mirror + `"}`, http.MethodPut, "https://evil.example", http.StatusForbidden},
		{`{"key":"VFOX_NODEJS_MIRROR","value":""}`, http.MethodGet, "", http.StatusMethodNotAllowed},
	} {
		if status, body := webRequest(t, c.method, url, session, c.body, c.origin); status != c.want {
			t.Errorf("%s %s: got %d: %s", c.method, c.body, status, body)
		}
	}
	if status, _ := webRequest(t, http.MethodPut, server.BaseURL()+"/api/sdks/maven/variables", session, `{"key":"VFOX_NODEJS_MIRROR","value":"`+mirror+`"}`, ""); status != http.StatusNotFound {
		t.Errorf("maven has no profile, got %d", status)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("the refused requests changed nothing: %q", got)
	}

	if status, _ := webRequest(t, http.MethodPut, url, session, `{"key":"VFOX_NODEJS_MIRROR","value":""}`, ""); status != http.StatusOK {
		t.Errorf("got %d", status)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != "" {
		t.Errorf("an empty value takes it away: %q", got)
	}
	if status, _ := webRequest(t, http.MethodPut, url, "", `{"key":"VFOX_NODEJS_MIRROR","value":""}`, ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
}

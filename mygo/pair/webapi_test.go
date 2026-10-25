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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
)

func testModel(t *testing.T, runner *testsupport.FakeProcessRunner) *state.AppModel {
	t.Helper()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        runner,
		SettingsStore: settings.NewMemorySettingsStore(settings.DefaultSettings()),
		Environment:   map[string]string{"HOME": "/home/u"},
		Locator: vfox.NewVfoxLocator(map[string]string{},
			func(path string) bool { return path == "/usr/bin/vfox" }),
	})
	t.Cleanup(model.Dispose)
	model.Start()
	deadline := time.Now().Add(5 * time.Second)
	for !model.Installed.HasValue() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the installed SDKs")
		}
		time.Sleep(2 * time.Millisecond)
	}
	return model
}

func webServer(t *testing.T, model *state.AppModel) *pair.Server {
	t.Helper()
	server, err := pair.NewServer(pair.Options{
		Snapshot: func() pair.Snapshot { return pair.Snapshot{} },
		Model:    model,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

func webSession(t *testing.T, server *pair.Server) string {
	t.Helper()
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	_, _, cookies := get(t, url, "")
	for _, cookie := range cookies {
		if cookie.Name == "vfox-plus-session" {
			return cookie.Value
		}
	}
	t.Fatal("pairing must issue a session")
	return ""
}

func webGet(t *testing.T, url, session string) (int, string) {
	t.Helper()
	status, body, _ := get(t, url, session)
	return status, body
}

func webPost(t *testing.T, server *pair.Server, session, path, body, origin string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.BaseURL()+path, strings.NewReader(body))
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

func TestWebAppNeedsPairing(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	nofollow := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := nofollow.Get(server.BaseURL() + "/app")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusFound {
		t.Errorf("unpaired browsers must be redirected, got %d", response.StatusCode)
	}
	session := webSession(t, server)
	status, body := webGet(t, server.BaseURL()+"/app", session)
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	for _, want := range []string{"vfox-plus", "/api/events", "taskCount", "sdkList"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page must contain %q", want)
		}
	}
}

func TestWebInventory(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webGet(t, server.BaseURL()+"/api/inventory", session)
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	for _, want := range []string{`"availability":"ready"`, "maven", "nodejs", "proxy", `"complete":true`} {
		if !strings.Contains(body, want) {
			t.Errorf("got %s", body)
		}
	}
	if status, _ := webGet(t, server.BaseURL()+"/api/inventory", ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
}

func TestWebSdkDetail(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webGet(t, server.BaseURL()+"/api/sdks/nodejs/info", session)
	if status != http.StatusOK || !strings.Contains(body, "nodejs") {
		t.Errorf("got %d: %s", status, body)
	}
	status, body = webGet(t, server.BaseURL()+"/api/sdks/nodejs/versions", session)
	if status != http.StatusOK || !strings.Contains(body, "24") {
		t.Errorf("got %d: %s", status, body)
	}
	if status, _ := webGet(t, server.BaseURL()+"/api/sdks/nodejs/nope", session); status != http.StatusNotFound {
		t.Errorf("got %d", status)
	}
}

func TestWebTasks(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webPost(t, server, session, "/api/tasks", `{"operation":"install","sdk":"go","version":"1"}`, "")
	if status != http.StatusCreated {
		t.Fatalf("got %d: %s", status, body)
	}
	waitTaskStatus(t, server, session, body, `"status":"succeeded"`)
	status, body = webGet(t, server.BaseURL()+"/api/tasks", session)
	if status != http.StatusOK || !strings.Contains(body, `"operation":"install"`) {
		t.Errorf("got %d: %s", status, body)
	}
	if status, _ := webPost(t, server, session, "/api/tasks", `{"operation":"use","sdk":"go","version":"1","scope":"project"}`, ""); status != http.StatusBadRequest {
		t.Errorf("project scopes must be rejected, got %d", status)
	}
	if status, _ := webPost(t, server, session, "/api/tasks", `{"operation":"add","source":"mine.lua"}`, ""); status != http.StatusBadRequest {
		t.Errorf("local plugin paths must be rejected, got %d", status)
	}
	if status, _ := webPost(t, server, session, "/api/tasks", `{"operation":"bogus"}`, ""); status != http.StatusBadRequest {
		t.Errorf("got %d", status)
	}
	if status, _ := webPost(t, server, session, "/api/tasks", `{"operation":"install","sdk":"go","version":"1"}`, "https://evil.example"); status != http.StatusForbidden {
		t.Errorf("cross-site requests must be rejected, got %d", status)
	}
}

func TestWebTaskActions(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.OnStart("update --all", testsupport.NewFakeRunningProcess)
	model := testModel(t, runner)
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webPost(t, server, session, "/api/tasks", `{"operation":"update"}`, "")
	if status != http.StatusCreated {
		t.Fatalf("got %d: %s", status, body)
	}
	if !strings.Contains(body, `"status":"running"`) {
		t.Fatalf("got %s", body)
	}
	var created struct {
		ID int `json:"id"`
	}
	decodeJSON(t, body, &created)
	do := func(method, path string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, server.BaseURL()+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(content)
	}
	id := "/api/tasks/" + strconv.Itoa(created.ID)
	if status, body := do(http.MethodGet, id); status != http.StatusOK || !strings.Contains(body, `"status":"running"`) {
		t.Errorf("got %d: %s", status, body)
	}
	if status, _ := do(http.MethodPost, id+"/cancel"); status != http.StatusOK {
		t.Errorf("got %d", status)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, body := do(http.MethodGet, id)
		if strings.Contains(body, `"status":"cancelled"`) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the task was not cancelled: %s", body)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if status, body := do(http.MethodPost, id+"/retry"); status != http.StatusOK || !strings.Contains(body, `"status":"running"`) {
		t.Errorf("got %d: %s", status, body)
	}
	if status, _ := do(http.MethodDelete, "/api/tasks/finished"); status != http.StatusOK {
		t.Errorf("got %d", status)
	}
	if status, _ := do(http.MethodGet, "/api/tasks/999999"); status != http.StatusNotFound {
		t.Errorf("got %d", status)
	}
}

func decodeJSON(t *testing.T, body string, value interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), value); err != nil {
		t.Fatal(err)
	}
}

func TestWebPlugins(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webPost(t, server, session, "/api/plugins", `{"source":"https://example.com/p.zip"}`, "")
	if status != http.StatusCreated || !strings.Contains(body, `"operation":"addPlugin"`) {
		t.Errorf("got %d: %s", status, body)
	}
	for _, local := range []string{`{"source":"/etc/passwd"}`, `{"source":"mine.lua"}`, `{"source":"C:\\mine.lua"}`} {
		if status, body := webPost(t, server, session, "/api/plugins", local, ""); status != http.StatusBadRequest {
			t.Errorf("local paths must be rejected, got %d: %s", status, body)
		}
	}
	var file bytes.Buffer
	writer := multipart.NewWriter(&file)
	part, err := writer.CreateFormFile("file", "mine.lua")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("PLUGIN = {}")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, server.BaseURL()+"/api/plugins", &file)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Errorf("got %d: %s", response.StatusCode, content)
	}
	waitTaskStatus(t, server, session, string(content), `"status":"succeeded"`)
}

// waitTaskStatus polls the task of body until its JSON contains want.
func waitTaskStatus(t *testing.T, server *pair.Server, session, body, want string) {
	t.Helper()
	var created struct {
		ID int `json:"id"`
	}
	decodeJSON(t, body, &created)
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, content := webGet(t, server.BaseURL()+"/api/tasks/"+strconv.Itoa(created.ID), session)
		if status == http.StatusOK && strings.Contains(content, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("got %d: %s", status, content)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestWebSettings(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	status, body := webGet(t, server.BaseURL()+"/api/settings", session)
	if status != http.StatusOK || !strings.Contains(body, `"language"`) {
		t.Fatalf("got %d: %s", status, body)
	}
	put := func(payload string) (int, string) {
		t.Helper()
		request, err := http.NewRequest(http.MethodPut, server.BaseURL()+"/api/settings", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		request.Header.Set("Content-Type", "application/json")
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
	if status, body := put(`{"language":"simplifiedChinese"}`); status != http.StatusOK || !strings.Contains(body, "simplifiedChinese") {
		t.Errorf("got %d: %s", status, body)
	}
	if status, _ := put(`{"language":"klingon"}`); status != http.StatusBadRequest {
		t.Errorf("got %d", status)
	}
}

func TestWebConfig(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("config proxy.enable true", core.ProcessOutput{}).
		On("config --unset proxy.enable", core.ProcessOutput{})
	model := testModel(t, runner)
	server := webServer(t, model)
	session := webSession(t, server)
	put := func(payload string) (int, string) {
		t.Helper()
		request, err := http.NewRequest(http.MethodPut, server.BaseURL()+"/api/config", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		request.Header.Set("Content-Type", "application/json")
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
	if status, body := put(`{"key":"proxy.enable","value":"true"}`); status != http.StatusOK || !strings.Contains(body, "saved") {
		t.Errorf("got %d: %s", status, body)
	}
	request, err := http.NewRequest(http.MethodDelete, server.BaseURL()+"/api/config?key=proxy.enable", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Errorf("got %d", response.StatusCode)
	}
}

func TestWebEvents(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	server := webServer(t, model)
	session := webSession(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.BaseURL()+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("got %q", contentType)
	}
	reader := bufio.NewReader(response.Body)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the hello event")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "event: hello") {
			return
		}
	}
}

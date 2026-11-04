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
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// fakeControl is the application as the web interface asks it.
type fakeControl struct {
	state.Notifier
	mu      sync.Mutex
	status  pair.Status
	link    string
	linkErr error
	asked   []string
	quits   atomic.Int32
}

func (f *fakeControl) Status() pair.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeControl) RemoteLink(host string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, host)
	return f.link, f.linkErr
}

func (f *fakeControl) Quit() { f.quits.Add(1) }

// controlledServer is a server that the fake application controls.
func controlledServer(t *testing.T, control pair.Control) (*pair.Server, *state.AppModel, string) {
	t.Helper()
	model := testModel(t, testsupport.FixtureRunner())
	server, err := pair.NewServer(pair.Options{Model: model, Control: control})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, model, webSession(t, server)
}

func send(t *testing.T, method, url, session, body, origin string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
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

func TestTheWebInterfaceReadsAndWritesHowOtherDevicesConnect(t *testing.T) {
	control := &fakeControl{}
	server, model, session := controlledServer(t, control)
	url := server.BaseURL() + "/api/web"
	status, body := webGet(t, url, session)
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	var dto pair.WebDTO
	decodeJSON(t, body, &dto)
	if dto.Remote || dto.Lightweight || dto.Listening || dto.Port != settings.DefaultWebPort || dto.Error != "" {
		t.Errorf("by default, other devices are not allowed, and the port is the default one: %+v", dto)
	}
	if !strings.Contains(body, `"addresses":[]`) {
		t.Errorf("no address is an empty list, not null: %s", body)
	}
	if status, _ := webGet(t, url, ""); status != http.StatusUnauthorized {
		t.Errorf("a browser that is not paired is refused, got %d", status)
	}

	// What the servers made of the settings is what the page is told.
	control.mu.Lock()
	control.status = pair.Status{Running: true, Remote: true, Port: 9000, Addresses: []string{"192.168.1.5"}, Fingerprint: "AB:CD", SelfSigned: true, RemoteError: "x"}
	control.mu.Unlock()
	_, body = webGet(t, url, session)
	decodeJSON(t, body, &dto)
	if !dto.Listening || dto.Fingerprint != "AB:CD" || !dto.SelfSigned || dto.Error != "x" || len(dto.Addresses) != 1 || dto.Addresses[0] != "192.168.1.5" {
		t.Errorf("got %+v", dto)
	}

	status, body = send(t, http.MethodPut, url, session, `{"remote":true,"port":9000,"lightweight":true}`, "")
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	current := model.Settings.Settings()
	if !current.WebRemote || current.WebPort != 9000 || !current.Lightweight {
		t.Errorf("the settings follow the page: %+v", current)
	}
	decodeJSON(t, body, &dto)
	if !dto.Remote || dto.Port != 9000 || !dto.Lightweight {
		t.Errorf("the answer is what the settings are now: %+v", dto)
	}
	// Each part alone, and nothing else changes.
	if status, _ := send(t, http.MethodPut, url, session, `{"remote":false}`, ""); status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	if current := model.Settings.Settings(); current.WebRemote || current.WebPort != 9000 || !current.Lightweight {
		t.Errorf("only the access of other devices changed: %+v", current)
	}
	for name, payload := range map[string]string{
		"a port of the system":            `{"port":80}`,
		"a port that is none":             `{"port":70000}`,
		"a port that is not a number":     `{"port":"x"}`,
		"something that is not an object": `[1]`,
		"nothing":                         ``,
	} {
		if status, _ := send(t, http.MethodPut, url, session, payload, ""); status != http.StatusBadRequest {
			t.Errorf("%s must be refused, got %d", name, status)
		}
	}
	if status, _ := send(t, http.MethodPut, url, "", `{"remote":true}`, ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
	if status, _ := send(t, http.MethodPut, url, session, `{"remote":true}`, "https://evil.example"); status != http.StatusForbidden {
		t.Errorf("a page of another site cannot open the door to the network, got %d", status)
	}
	if model.Settings.Settings().WebRemote {
		t.Error("a refused request changes nothing")
	}
	if status, _ := send(t, http.MethodDelete, url, session, ``, ""); status != http.StatusMethodNotAllowed {
		t.Errorf("got %d", status)
	}
}

func TestTheWebInterfaceWithoutAnApplicationAroundItHasNoDevicesToConnect(t *testing.T) {
	server, _, session := controlledServer(t, nil)
	status, body := webGet(t, server.BaseURL()+"/api/web", session)
	if status != http.StatusOK || !strings.Contains(body, `"listening":false`) {
		t.Errorf("got %d: %s", status, body)
	}
	for _, path := range []string{"/api/web/pairing", "/api/quit"} {
		if status, _ := send(t, http.MethodPost, server.BaseURL()+path, session, `{}`, ""); status != http.StatusNotFound {
			t.Errorf("%s: got %d", path, status)
		}
	}
}

func TestAPairedBrowserMakesTheCodeThatAPhoneScans(t *testing.T) {
	control := &fakeControl{link: "https://192.168.1.5:8369/auth/connect/abc"}
	server, _, session := controlledServer(t, control)
	url := server.BaseURL() + "/api/web/pairing"
	status, body := send(t, http.MethodPost, url, session, ``, "")
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	var made struct {
		URL string `json:"url"`
		SVG string `json:"svg"`
	}
	decodeJSON(t, body, &made)
	if made.URL != control.link || !strings.HasPrefix(made.SVG, "<svg ") || !strings.HasSuffix(made.SVG, "</svg>") {
		t.Errorf("the link and the code of it: %+v", made)
	}
	if status, _ := send(t, http.MethodPost, url, session, `{"host":" 10.0.0.2 "}`, ""); status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	if len(control.asked) != 2 || control.asked[0] != "" || control.asked[1] != "10.0.0.2" {
		t.Errorf("the address that was asked for, without its spaces: %q", control.asked)
	}
	for _, c := range []struct {
		name    string
		err     error
		want    int
		message string
	}{
		{"other devices that are not allowed", pair.ErrNoRemote, http.StatusConflict, "not allowed"},
		{"an address that the computer has not", pair.ErrNoAddress, http.StatusConflict, "no such address"},
		{"a failure", errors.New("boom"), http.StatusInternalServerError, "Cannot make a link"},
	} {
		control.mu.Lock()
		control.linkErr = c.err
		control.mu.Unlock()
		status, body := send(t, http.MethodPost, url, session, `{}`, "")
		if status != c.want || !strings.Contains(body, c.message) {
			t.Errorf("%s: got %d: %s", c.name, status, body)
		}
	}
	for name, check := range map[string]func() int{
		"without a session": func() int { s, _ := send(t, http.MethodPost, url, "", `{}`, ""); return s },
		"from another site": func() int { s, _ := send(t, http.MethodPost, url, session, `{}`, "https://evil.example"); return s },
		"by GET":            func() int { s, _ := send(t, http.MethodGet, url, session, ``, ""); return s },
		"with a body that is no JSON": func() int {
			s, _ := send(t, http.MethodPost, url, session, `nope`, "")
			return s
		},
	} {
		if status := check(); status == http.StatusOK {
			t.Errorf("a code must not be made %s", name)
		}
	}
	if status, _ := send(t, http.MethodPost, url, "", `{}`, ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
	if status, _ := send(t, http.MethodGet, url, session, ``, ""); status != http.StatusMethodNotAllowed {
		t.Errorf("got %d", status)
	}
	if status, _ := send(t, http.MethodPost, url, session, `nope`, ""); status != http.StatusBadRequest {
		t.Errorf("got %d", status)
	}
}

func TestAPairedBrowserCanQuitTheApplicationAfterItsAnswer(t *testing.T) {
	control := &fakeControl{}
	server, _, session := controlledServer(t, control)
	url := server.BaseURL() + "/api/quit"
	for name, call := range map[string]func() int{
		"without a session": func() int { s, _ := send(t, http.MethodPost, url, "", ``, ""); return s },
		"from another site": func() int { s, _ := send(t, http.MethodPost, url, session, ``, "https://evil.example"); return s },
		"by GET":            func() int { s, _ := send(t, http.MethodGet, url, session, ``, ""); return s },
	} {
		if status := call(); status == http.StatusAccepted {
			t.Errorf("the application must not quit %s", name)
		}
	}
	time.Sleep(500 * time.Millisecond)
	if control.quits.Load() != 0 {
		t.Fatal("the application quit for a request that was refused")
	}
	status, body := send(t, http.MethodPost, url, session, ``, "")
	if status != http.StatusAccepted || !strings.Contains(body, `"quitting":true`) {
		t.Fatalf("got %d: %s", status, body)
	}
	deadline := time.Now().Add(5 * time.Second)
	for control.quits.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the application did not quit")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestThePageIsToldWhenTheConnectionsOfOtherDevicesChange(t *testing.T) {
	control := &fakeControl{}
	server, _, session := controlledServer(t, control)
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
	reader := bufio.NewReader(response.Body)
	events := make(chan string, 16)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "event: ") {
				events <- strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			}
		}
	}()
	if kind := <-events; kind != "hello" {
		t.Fatalf("got %q", kind)
	}
	control.NotifyListeners()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case kind := <-events:
			if kind == "web" {
				return
			}
		case <-deadline:
			t.Fatal("no web event")
		}
	}
}

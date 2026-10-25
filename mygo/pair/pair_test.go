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
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/pair"
)

func testSnapshot() pair.Snapshot {
	return pair.Snapshot{
		HasVfox:     true,
		VfoxVersion: "1.0.13",
		SDKs: []pair.SdkSummary{
			{Name: "nodejs", Current: "24.11.1", HasCurrent: true, Versions: []string{"24.11.1", "22.21.0"}},
		},
	}
}

func testServer(t *testing.T, snapshot pair.Snapshot) *pair.Server {
	t.Helper()
	server, err := pair.NewServer(pair.Options{Snapshot: func() pair.Snapshot { return snapshot }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

func get(t *testing.T, url, cookie string) (int, string, http.Header, []*http.Cookie) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: cookie})
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body), response.Header, response.Cookies()
}

func TestPairLinkWorksOnce(t *testing.T) {
	server := testServer(t, testSnapshot())
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, server.BaseURL()+"/auth/connect/") {
		t.Fatalf("got %q", url)
	}
	status, body, _, cookies := get(t, url, "")
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	if !strings.Contains(body, "1.0.13") || !strings.Contains(body, "nodejs") {
		t.Errorf("the page must show the instance: %q", body)
	}
	var session string
	for _, cookie := range cookies {
		if cookie.Name == "vfox-plus-session" {
			session = cookie.Value
		}
	}
	if session == "" {
		t.Fatal("pairing must issue a session")
	}
	// The link is single-use.
	if status, _, _, _ := get(t, url, ""); status != http.StatusForbidden {
		t.Errorf("a used link must be rejected, got %d", status)
	}
	// The session unlocks the read-only API.
	status, body, _, _ = get(t, server.BaseURL()+"/api/status", session)
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	if !strings.Contains(body, "24.11.1") {
		t.Errorf("got %q", body)
	}
	// Without the session the API refuses.
	if status, _, _, _ := get(t, server.BaseURL()+"/api/status", ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
}

func TestPairLinkExpires(t *testing.T) {
	server, err := pair.NewServer(pair.Options{
		TokenTTL: 10 * time.Millisecond,
		Snapshot: func() pair.Snapshot { return testSnapshot() },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _, _, _ := get(t, url, ""); status != http.StatusForbidden {
		t.Errorf("an expired link must be rejected, got %d", status)
	}
}

func TestPairListensOnLoopbackOnly(t *testing.T) {
	server := testServer(t, testSnapshot())
	if _, err := pair.NewServer(pair.Options{Bind: "0.0.0.0"}); err == nil {
		t.Error("a non-loopback bind must be refused")
	}
	if open, err := pair.NewServer(pair.Options{Bind: "0.0.0.0", AllowNonLoopback: true}); err != nil {
		t.Skipf("no non-loopback interface in this sandbox: %v", err)
	} else {
		_ = open.Close()
	}
	addr := server.Addr().String()
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("got %q", addr)
	}
	if got := server.DisplayBaseURL(); got != server.BaseURL() {
		t.Errorf("a loopback server shows its own address, got %q", got)
	}
}

func TestDisplayBaseURLMasksWildcards(t *testing.T) {
	server, err := pair.NewServer(pair.Options{
		Bind:             "0.0.0.0",
		AllowNonLoopback: true,
		Snapshot:         func() pair.Snapshot { return testSnapshot() },
	})
	if err != nil {
		t.Skipf("no non-loopback interface in this sandbox: %v", err)
	}
	defer func() { _ = server.Close() }()
	got := server.DisplayBaseURL()
	if !strings.HasPrefix(got, "http://0.0.0.0:") {
		t.Errorf("a wildcard server must show a replaceable placeholder, got %q", got)
	}
}

func TestPairPageExplainsAMissingVfox(t *testing.T) {
	server := testServer(t, pair.Snapshot{VfoxError: "executable file not found"})
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	status, body, _, _ := get(t, url, "")
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	if !strings.Contains(body, "vfox.dev") {
		t.Errorf("the page must point at the vfox installation guide: %q", body)
	}
}

func TestSummarizeKeepsInstalledVersions(t *testing.T) {
	got := pair.Summarize([]core.InstalledSdk{
		{Name: "nodejs", Current: "24.11.1", HasCurrent: true, Versions: []string{"24.11.1"}},
	})
	if len(got) != 1 || got[0].Name != "nodejs" || !got[0].HasCurrent || len(got[0].Versions) != 1 {
		t.Errorf("got %#v", got)
	}
}

func pairSession(t *testing.T, server *pair.Server) string {
	t.Helper()
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, cookies := get(t, url, "")
	for _, cookie := range cookies {
		if cookie.Name == "vfox-plus-session" {
			return cookie.Value
		}
	}
	t.Fatal("pairing must issue a session")
	return ""
}

func TestSessionExpires(t *testing.T) {
	server, err := pair.NewServer(pair.Options{
		SessionTTL: 20 * time.Millisecond,
		Snapshot:   func() pair.Snapshot { return testSnapshot() },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	session := pairSession(t, server)
	if status, _, _, _ := get(t, server.BaseURL()+"/api/status", session); status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _, _, _ := get(t, server.BaseURL()+"/api/status", session); status != http.StatusUnauthorized {
		t.Errorf("an expired session must be rejected, got %d", status)
	}
}

func TestPairedBrowserMintsLinks(t *testing.T) {
	server := testServer(t, testSnapshot())
	session := pairSession(t, server)
	// Minting needs the session.
	if status, _, _, _ := get(t, server.BaseURL()+"/api/token", ""); status != http.StatusUnauthorized {
		t.Fatalf("got %d", status)
	}
	status, body, _, _ := get(t, server.BaseURL()+"/api/token", session)
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	var minted struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body), &minted); err != nil || minted.URL == "" {
		t.Fatalf("got %q, %v", body, err)
	}
	if !strings.HasPrefix(minted.URL, server.BaseURL()+"/auth/connect/") {
		t.Errorf("got %q", minted.URL)
	}
	// The minted link pairs once.
	if status, _, _, _ := get(t, minted.URL, ""); status != http.StatusOK {
		t.Errorf("got %d", status)
	}
	if status, _, _, _ := get(t, minted.URL, ""); status != http.StatusForbidden {
		t.Errorf("got %d", status)
	}
}

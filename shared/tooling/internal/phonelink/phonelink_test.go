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

package phonelink

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServer is the pairing server of the end-to-end build: a loopback server whose one-time
// token pairs it, and which lists the server for other devices as listening after listenAfter
// polls. setCookie says whether the pairing sets the session cookie.
func fakeServer(t *testing.T, listenAfter int, setCookie bool) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	polls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/connect/tok", func(w http.ResponseWriter, r *http.Request) {
		if setCookie {
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "sess123", Path: "/", HttpOnly: true, Secure: true})
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/web", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != cookieName+"=sess123" {
			http.Error(w, "Not paired.", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"remote":true,"listening":false}`))
		case http.MethodGet:
			mu.Lock()
			polls++
			listening := polls > listenAfter
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if listening {
				_, _ = w.Write([]byte(`{"remote":true,"listening":true}`))
			} else {
				_, _ = w.Write([]byte(`{"remote":true,"listening":false}`))
			}
		default:
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/web/pairing", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != cookieName+"=sess123" || r.Method != http.MethodPost {
			http.Error(w, "Not paired.", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"https://10.0.2.2:8443/auth/connect/new","svg":"<svg/>","host":"10.0.2.2"}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestPairingLink(t *testing.T) {
	server := fakeServer(t, 2, true)
	link, err := PairingLink(context.Background(), server.URL+"/auth/connect/tok", "10.0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://10.0.2.2:8443/auth/connect/new"; link != want {
		t.Errorf("link = %q, want %q", link, want)
	}
}

func TestPairingLinkNeedsTheCookie(t *testing.T) {
	server := fakeServer(t, 0, false)
	_, err := PairingLink(context.Background(), server.URL+"/auth/connect/tok", "10.0.2.2")
	if err == nil || !strings.Contains(err.Error(), "no vfox-plus-session cookie") {
		t.Fatalf("PairingLink without the cookie: err = %v", err)
	}
}

func TestPairingLinkRefusesAnotherAddress(t *testing.T) {
	server := fakeServer(t, 0, true)
	_, err := PairingLink(context.Background(), server.URL+"/auth/connect/tok", "10.0.2.9")
	if err == nil || !strings.Contains(err.Error(), "not for 10.0.2.9") {
		t.Fatalf("PairingLink for another address: err = %v", err)
	}
}

func TestPairingLinkNamesNoToken(t *testing.T) {
	server := fakeServer(t, 0, true)
	_, err := PairingLink(context.Background(), server.URL+"/auth/connect/wrong-token", "10.0.2.2")
	if err == nil {
		t.Fatal("PairingLink with a wrong token succeeded")
	}
	if strings.Contains(err.Error(), "wrong-token") {
		t.Errorf("the error names the token: %v", err)
	}
}

func TestSplitLoopbackLink(t *testing.T) {
	base, token, err := splitLoopbackLink("http://127.0.0.1:43123/auth/connect/abc")
	if err != nil || base != "http://127.0.0.1:43123" || token != "abc" {
		t.Fatalf("splitLoopbackLink = %q, %q, %v", base, token, err)
	}
	for _, link := range []string{
		"https://127.0.0.1:43123/auth/connect/abc", // Not the plain loopback server of the build.
		"http://10.0.2.2:43123/auth/connect/abc",   // Not the loopback interface.
		"http://127.0.0.1:43123/app",
		"",
	} {
		if _, _, err := splitLoopbackLink(link); err == nil {
			t.Errorf("splitLoopbackLink(%q) succeeded", link)
		}
	}
}

func TestRunWritesTheLinkWhenTheBuildWritesItsFile(t *testing.T) {
	server := fakeServer(t, 1, true)
	dir := t.TempDir()
	loopbackFile := filepath.Join(dir, "e2e", "link.txt")
	if err := os.MkdirAll(filepath.Dir(loopbackFile), 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "phone", "pairing-link.txt")
	go func() {
		// The build writes its link when it starts, after the file was asked for.
		time.Sleep(300 * time.Millisecond)
		_ = os.WriteFile(loopbackFile, []byte(server.URL+"/auth/connect/tok\n"), 0o600)
	}()
	if err := Run([]string{"--loopback-link-file", loopbackFile, "--host", "10.0.2.2", "--output", output, "--wait", "20"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://10.0.2.2:8443/auth/connect/new\n"; string(data) != want {
		t.Errorf("output = %q, want %q", data, want)
	}
	if info, err := os.Stat(output); err == nil && runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("the output has mode %v, want 0600: the link is a secret", info.Mode().Perm())
	}
}

func TestWaitForLinkGivesUp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := waitForLink(ctx, filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil || !strings.Contains(err.Error(), "wrote no link") {
		t.Fatalf("waitForLink = %v", err)
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run([]string{}); err == nil || !strings.HasPrefix(err.Error(), "Usage: phone-link") {
		t.Fatalf("Run without arguments: err = %v", err)
	}
}

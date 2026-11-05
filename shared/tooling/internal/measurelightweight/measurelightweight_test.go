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

package measurelightweight

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionCookieValue(t *testing.T) {
	value, ok := sessionCookieValue("vfox-plus-session=abc123; Path=/; Secure; HttpOnly; SameSite=Lax")
	if !ok || value != "abc123" {
		t.Fatalf("got %q, %v", value, ok)
	}
	for _, header := range []string{"", "other=1", "vfox-plus-session=", "vfox-plus-session"} {
		if _, ok := sessionCookieValue(header); ok {
			t.Fatalf("got a cookie from %q", header)
		}
	}
}

func TestMeasureUsage(t *testing.T) {
	if err := Run([]string{}); err == nil || !strings.Contains(err.Error(), "Usage:") {
		t.Fatalf("got %v", err)
	}
	if err := Run([]string{"--settle", "-1", "application"}); err == nil {
		t.Fatal("want an error for a negative settle time")
	}
}

func TestTheEnvironmentIsAHomeOfItsOwnWithTheSettingsOfTheLightweightMode(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/elsewhere/config")
	t.Setenv("XDG_RUNTIME_DIR", "/elsewhere/run")
	t.Setenv("VFOX_PLUS_E2E_LINK", "/elsewhere/link")
	home, fixtures := t.TempDir(), t.TempDir()
	fake := filepath.Join(t.TempDir(), "fake-vfox")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // A script for the test.
		t.Fatal(err)
	}
	env, link, err := measureEnv(home, fixtures, fake)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]string{}
	for _, kv := range env {
		key, value, _ := strings.Cut(kv, "=")
		if _, twice := have[key]; twice {
			t.Errorf("%s is set twice", key)
		}
		have[key] = value
	}
	if have["HOME"] != home || have["XDG_RUNTIME_DIR"] != filepath.Join(home, "run") || have["VFOX_PLUS_FIXTURES"] != fixtures ||
		have["NO_COLOR"] != "1" || have["VFOX_PLUS_E2E_LINK"] != link || link != filepath.Join(home, "link.txt") {
		t.Errorf("got %v", have)
	}
	if _, kept := have["XDG_CONFIG_HOME"]; kept {
		t.Error("the settings are in the home of the run, not in the configuration of whoever runs the tool")
	}
	if !strings.HasPrefix(have["PATH"], filepath.Join(home, "bin")+string(os.PathListSeparator)) {
		t.Errorf("the fake vfox is first on PATH: %s", have["PATH"])
	}
	if _, err := os.Stat(filepath.Join(home, "bin", "vfox")); err != nil {
		t.Errorf("the fake vfox is there: %v", err)
	}
	stored, err := os.ReadFile(filepath.Join(home, ".config", "vfox-plus", "settings.json"))
	if err != nil || string(stored) != settings || !strings.Contains(settings, `"webInterface":true`) || !strings.Contains(settings, `"lightweight":true`) {
		t.Errorf("the settings of the run: %q, %v", stored, err)
	}
	if info, err := os.Stat(filepath.Join(home, "run")); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("the runtime directory is private: %v %v", info, err)
	}
	if _, _, err := measureEnv(home, fixtures, filepath.Join(home, "missing")); err == nil {
		t.Error("a fake vfox that is not there is an error")
	}
}

func TestTheLinkIsWaitedForUntilTheApplicationHasWrittenIt(t *testing.T) {
	file := filepath.Join(t.TempDir(), "link.txt")
	done := make(chan error, 1)
	go func() {
		time.Sleep(250 * time.Millisecond)
		// A file that is half written does not hold a link yet.
		_ = os.WriteFile(file, []byte("http://127.0.0"), 0o600)
		time.Sleep(250 * time.Millisecond)
		_ = os.WriteFile(file, []byte("http://127.0.0.1:41233/auth/connect/abc\n"), 0o600)
	}()
	link, err := waitForLink(file, done, 10*time.Second)
	if err != nil || link != "http://127.0.0.1:41233/auth/connect/abc" {
		t.Fatalf("got %q, %v", link, err)
	}
}

func TestTheLinkIsNotWaitedForWhenTheApplicationIsGoneOrTooSlow(t *testing.T) {
	done := make(chan error, 1)
	done <- errors.New("exit status 1")
	if _, err := waitForLink(filepath.Join(t.TempDir(), "link.txt"), done, 10*time.Second); err == nil || !strings.Contains(err.Error(), "exited before it wrote a pairing link") {
		t.Errorf("got %v", err)
	}
	if _, err := waitForLink(filepath.Join(t.TempDir(), "link.txt"), make(chan error), 300*time.Millisecond); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("got %v", err)
	}
}

// The link pairs through a redirect to the interface: the session is in the first answer.
func TestPairingTakesTheSessionFromTheAnswerToTheLink(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/connect/abc", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "vfox-plus-session", Value: "s1", Path: "/", Secure: true, HttpOnly: true})
		http.Redirect(w, r, "/app", http.StatusSeeOther)
	})
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("the interface")) })
	mux.HandleFunc("/api/inventory", func(w http.ResponseWriter, r *http.Request) {
		if cookie := r.Header.Get("Cookie"); cookie != "vfox-plus-session=s1" {
			http.Error(w, "not paired: "+cookie, http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"complete":true}`))
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: hello\ndata: {}\n\n"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	events, err := pairAndLoad(server.URL + "/auth/connect/abc")
	if err != nil {
		t.Fatal(err)
	}
	_ = events.Close()
	if _, err := pairAndLoad(server.URL + "/auth/connect/none"); err == nil {
		t.Error("a link that does not pair is an error")
	}
}

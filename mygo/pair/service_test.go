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
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
)

// serverPort reports the loopback port of server.
func serverPort(t *testing.T, server *pair.Server) int {
	t.Helper()
	_, port, err := net.SplitHostPort(server.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	var number int
	if _, err := fmt.Sscanf(port, "%d", &number); err != nil {
		t.Fatal(err)
	}
	return number
}

// webGetAuth answers a GET with an Authorization header and no session.
func webGetAuth(t *testing.T, url, auth string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "" {
		request.Header.Set("Authorization", auth)
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

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// serviceServer starts a server with an admin token for the service tests.
func serviceServer(t *testing.T) (*pair.Server, string) {
	t.Helper()
	server, err := pair.NewServer(pair.Options{
		Snapshot:   func() pair.Snapshot { return pair.Snapshot{} },
		AdminToken: "test-admin-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, "test-admin-token"
}

func TestServiceFileRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := pair.ServiceFilePath(dir)
	info := &pair.ServiceInfo{PID: 123, Port: 4567, AdminToken: "tok", Owner: pair.ServiceOwnerService, StartedAt: "2026-01-01T00:00:00Z"}
	if err := pair.WriteServiceInfo(path, info); err != nil {
		t.Fatal(err)
	}
	entry, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Mode().Perm() != 0o600 {
		t.Fatalf("service file is %o, want 600", entry.Mode().Perm())
	}
	got, err := pair.ReadServiceInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if *got != *info {
		t.Fatalf("got %+v, want %+v", got, info)
	}
}

func TestServiceFileInvalid(t *testing.T) {
	dir := t.TempDir()
	path := pair.ServiceFilePath(dir)
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pair.ReadServiceInfo(path); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
	if err := os.WriteFile(path, []byte(`{"port":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pair.ReadServiceInfo(path); err == nil {
		t.Fatal("want an error without an admin token")
	}
}

func TestDiscoverService(t *testing.T) {
	dir := t.TempDir()
	if got := pair.DiscoverService(dir); got != nil {
		t.Fatal("want nil without a file")
	}
	// A stale file is removed.
	stale := &pair.ServiceInfo{PID: 1, Port: 1, AdminToken: "tok", Owner: pair.ServiceOwnerService}
	if err := pair.WriteServiceInfo(pair.ServiceFilePath(dir), stale); err != nil {
		t.Fatal(err)
	}
	if got := pair.DiscoverService(dir); got != nil {
		t.Fatal("want nil for a dead server")
	}
	if _, err := os.Stat(pair.ServiceFilePath(dir)); !os.IsNotExist(err) {
		t.Fatal("want the stale file removed")
	}
	// A live server is returned.
	server, _ := serviceServer(t)
	live := &pair.ServiceInfo{PID: os.Getpid(), Port: serverPort(t, server), AdminToken: "test-admin-token", Owner: pair.ServiceOwnerDesktop}
	if err := pair.WriteServiceInfo(pair.ServiceFilePath(dir), live); err != nil {
		t.Fatal(err)
	}
	got := pair.DiscoverService(dir)
	if got == nil || got.Port != live.Port || got.Owner != live.Owner {
		t.Fatalf("got %+v, want %+v", got, live)
	}
}

func TestMintToken(t *testing.T) {
	server, _ := serviceServer(t)
	info := &pair.ServiceInfo{Port: serverPort(t, server), AdminToken: "test-admin-token"}
	url, err := pair.MintToken(info)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(url, "/auth/connect/") {
		t.Fatalf("got %q", url)
	}
	bad := &pair.ServiceInfo{Port: info.Port, AdminToken: "wrong"}
	if _, err := pair.MintToken(bad); err == nil {
		t.Fatal("want an error for a wrong token")
	}
	dead := &pair.ServiceInfo{Port: 1, AdminToken: "test-admin-token"}
	if _, err := pair.MintToken(dead); err == nil {
		t.Fatal("want an error for a dead server")
	}
}

func TestServeTokenAdmin(t *testing.T) {
	server, _ := serviceServer(t)
	// The admin bearer mints without a session.
	status, body := webGetAuth(t, server.BaseURL()+"/api/token", "Bearer test-admin-token")
	if status != http.StatusOK {
		t.Fatalf("got %d: %s", status, body)
	}
	if !contains(body, "/auth/connect/") {
		t.Fatalf("got %s", body)
	}
	// A wrong bearer answers 401 like an unpaired browser.
	if status, _ := webGetAuth(t, server.BaseURL()+"/api/token", "Bearer wrong"); status != http.StatusUnauthorized {
		t.Fatalf("got %d", status)
	}
	// Without any credential it stays 401.
	if status, _ := webGet(t, server.BaseURL()+"/api/token", ""); status != http.StatusUnauthorized {
		t.Fatalf("got %d", status)
	}
}

func TestServiceDisabled(t *testing.T) {
	dir := t.TempDir()
	if pair.ServiceDisabled(dir) {
		t.Fatal("want false without a file")
	}
	if err := pair.SetServiceDisabled(dir, true); err != nil {
		t.Fatal(err)
	}
	if !pair.ServiceDisabled(dir) {
		t.Fatal("want true after disabling")
	}
	if err := pair.SetServiceDisabled(dir, false); err != nil {
		t.Fatal(err)
	}
	if pair.ServiceDisabled(dir) {
		t.Fatal("want false after enabling")
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if pair.ServiceDisabled(dir) {
		t.Fatal("want false for invalid JSON")
	}
}

func TestServeEmbedded(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	dir := t.TempDir()
	cleanup, err := pair.ServeEmbedded(model, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	info := pair.DiscoverService(dir)
	if info == nil {
		t.Fatal("want the desktop to register itself")
	}
	if info.Owner != pair.ServiceOwnerDesktop {
		t.Fatalf("got owner %q", info.Owner)
	}
	url, err := pair.MintToken(info)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(url, "/auth/connect/") {
		t.Fatalf("got %q", url)
	}
	cleanup()
	if got := pair.DiscoverService(dir); got != nil {
		t.Fatal("want the registration removed on exit")
	}
}

func TestServeEmbeddedSkipsLive(t *testing.T) {
	server, _ := serviceServer(t)
	dir := t.TempDir()
	live := &pair.ServiceInfo{PID: os.Getpid(), Port: serverPort(t, server), AdminToken: "test-admin-token", Owner: pair.ServiceOwnerService}
	if err := pair.WriteServiceInfo(pair.ServiceFilePath(dir), live); err != nil {
		t.Fatal(err)
	}
	model := testModel(t, testsupport.FixtureRunner())
	cleanup, err := pair.ServeEmbedded(model, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	got, err := pair.ReadServiceInfo(pair.ServiceFilePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if got.Owner != pair.ServiceOwnerService {
		t.Fatalf("got owner %q, want the live one kept", got.Owner)
	}
}

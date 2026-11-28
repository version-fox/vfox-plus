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

package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// directory is an online directory that serves files, and the requests that it
// gets.
type directory struct {
	mu       sync.Mutex
	files    map[string]string
	index    string
	requests []string
}

func serve(t *testing.T) (*directory, *httptest.Server) {
	d := &directory{files: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		path := strings.TrimPrefix(r.URL.Path, "/")
		d.requests = append(d.requests, path)
		if strings.HasSuffix(path, indexFile) {
			content := d.index
			if content == "" {
				index := Index{Schema: SchemaVersion, Plugins: map[string]string{}}
				for name, file := range d.files {
					index.Plugins[name] = SHA256([]byte(file))
				}
				raw, _ := json.Marshal(index)
				content = string(raw)
			}
			_, _ = w.Write([]byte(content))
			return
		}
		name := strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".json")
		content, ok := d.files[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(content))
	}))
	t.Cleanup(server.Close)
	return d, server
}

func (d *directory) set(name, content string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if content == "" {
		delete(d.files, name)
		return
	}
	d.files[name] = content
}

func (d *directory) take() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	requests := d.requests
	d.requests = nil
	slices.Sort(requests)
	return requests
}

func refresh(t *testing.T, catalog *Catalog, want bool) {
	t.Helper()
	changed, err := catalog.Refresh(context.Background(), "")
	if err != nil || changed != want {
		t.Fatalf("Refresh = %v, %v; want %v", changed, err, want)
	}
}

func TestTheDirectoryIsKeptWhereItDiffers(t *testing.T) {
	online, server := serve(t)
	shipped := map[string][]byte{"java": []byte(`{"plugin": "java"}`), "gradle": []byte(`{"plugin": "gradle"}`)}
	online.set("java", `{"plugin": "java"}`)
	online.set("gradle", `{"plugin": "gradle", "guide": "newer"}`)
	online.set("ruby", `{"plugin": "ruby", "guide": "added"}`)
	catalog := &Catalog{Directory: t.TempDir(), Shipped: shipped, URL: server.URL + "/"}
	if documents, _ := catalog.Load(); documents != nil || catalog.Fresh(time.Hour, time.Now()) {
		t.Fatalf("an empty cache = %v", documents)
	}
	refresh(t, catalog, true)
	if got := online.take(); !slices.Equal(got, []string{"index.json", "plugins/gradle.json", "plugins/ruby.json"}) {
		t.Errorf("requests = %v: what the application ships is not fetched", got)
	}
	documents, problems := catalog.Load()
	if len(problems) != 0 || len(documents) != 2 || documents["gradle"].Resolve(linux()).Guide.EN != "newer" {
		t.Errorf("Load = %v, %v", documents, problems)
	}
	if !catalog.Fresh(time.Hour, time.Now()) || catalog.Fresh(time.Hour, time.Now().Add(2*time.Hour)) {
		t.Error("the cache is fresh for an hour")
	}
	refresh(t, catalog, false)
	if got := online.take(); !slices.Equal(got, []string{"index.json"}) {
		t.Errorf("requests = %v: what the cache holds is not fetched again", got)
	}
	online.set("ruby", "")
	refresh(t, catalog, true)
	if documents, _ := catalog.Load(); len(documents) != 1 {
		t.Errorf("a plugin that left the directory stays: %v", documents)
	}
	if _, err := os.Stat(filepath.Join(catalog.Directory, "plugins", "ruby.json")); !os.IsNotExist(err) {
		t.Errorf("its file stays: %v", err)
	}
	// Another version of the application drops the cache: what it ships may be
	// newer than what the cache holds.
	catalog.Shipped = map[string][]byte{"gradle": []byte(`{"plugin": "gradle", "guide": "newest"}`)}
	if documents, _ := catalog.Load(); documents != nil || catalog.Fresh(time.Hour, time.Now()) {
		t.Errorf("the cache of another version = %v", documents)
	}
}

func TestAFileThatDoesNotMatchTheIndexChangesNothing(t *testing.T) {
	online, server := serve(t)
	online.set("ruby", `{"plugin": "ruby", "guide": "first"}`)
	catalog := &Catalog{Directory: t.TempDir(), URL: server.URL + "/"}
	refresh(t, catalog, true)
	online.set("ruby", `{"plugin": "ruby", "guide": "second"}`)
	online.mu.Lock()
	online.index = `{"schema": 2, "plugins": {"ruby": "` + strings.Repeat("0", 64) + `"}}`
	online.mu.Unlock()
	if _, err := catalog.Refresh(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("Refresh = %v", err)
	}
	if documents, _ := catalog.Load(); documents["ruby"].Resolve(linux()).Guide.EN != "first" {
		t.Errorf("the cache changed: %v", documents)
	}
	// A file that names another plugin is refused too.
	online.mu.Lock()
	online.index = ""
	online.mu.Unlock()
	online.set("ruby", `{"plugin": "java"}`)
	if _, err := catalog.Refresh(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "describes") {
		t.Errorf("Refresh = %v", err)
	}
}

func TestAnIndexThatCannotBeTrustedIsRefused(t *testing.T) {
	for _, content := range []string{
		`{`, `{"schema": 3, "plugins": {}}`, `{"schema": 2, "plugins": {"../x": "` + strings.Repeat("0", 64) + `"}}`,
		`{"schema": 2, "plugins": {"x": "abc"}}`,
	} {
		if _, err := ParseIndex([]byte(content)); err == nil {
			t.Errorf("ParseIndex(%s) succeeded", content)
		}
	}
}

func TestTheMirrorGoesInFront(t *testing.T) {
	online, server := serve(t)
	catalog := &Catalog{Directory: t.TempDir(), URL: "https://raw.example.org/profile/"}
	refreshed, err := catalog.Refresh(context.Background(), server.URL+"/mirror")
	if err != nil || refreshed {
		t.Fatalf("Refresh = %v, %v", refreshed, err)
	}
	if got := online.take(); !slices.Equal(got, []string{"mirror/https:/raw.example.org/profile/index.json"}) &&
		!slices.Equal(got, []string{"mirror/https://raw.example.org/profile/index.json"}) {
		t.Errorf("requests = %v", got)
	}
	if _, err := (&Catalog{}).Refresh(context.Background(), ""); err == nil {
		t.Error("a catalogue without a cache cannot be refreshed")
	}
}

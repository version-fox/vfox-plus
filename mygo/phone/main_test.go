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

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// config is the part of mygo.json that names the app and its platforms.
type config struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Slug       string `json:"slug"`
	Version    string `json:"version"`
	Linux      struct {
		WebView *bool `json:"webview"`
	} `json:"linux"`
}

// readConfig reads mygo.json, which sits beside this package.
func readConfig(t *testing.T) config {
	t.Helper()
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var c config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTheConfigNamesTheApp(t *testing.T) {
	c := readConfig(t)
	if c.Name != "vfox+" || c.Identifier != "dev.vfox.vfoxplus" || c.Slug != "vfox-plus" {
		t.Errorf("name %q, identifier %q, slug %q; want the app's identity", c.Name, c.Identifier, c.Slug)
	}
	if c.Linux.WebView == nil || *c.Linux.WebView {
		t.Error("linux.webview must be false: the app shows no web page")
	}
}

func TestTheVersionIsTheDesktopsVersion(t *testing.T) {
	// `mygo.json` has its own copy of the version, which must follow the desktop's
	// mygo/VERSION until the two are released from one source.
	data, err := os.ReadFile("../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := strings.TrimSpace(string(data)), readConfig(t).Version; got != want {
		t.Errorf("mygo.json has version %q, mygo/VERSION has %q", got, want)
	}
}

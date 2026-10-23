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

// Command vfox-plus is a desktop application for vfox, built with Gio.
package main

import (
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui"
)

func main() {
	// The live heap is 10 to 12 MB. The default of 100 % lets it double before
	// the next collection, 25 % keeps what the application holds near what it
	// needs, for a few collections more.
	debug.SetGCPercent(25)
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			environment[key] = value
		}
	}
	home, _ := os.UserHomeDir()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        core.OsProcessRunner{},
		SettingsStore: settings.JSONFileSettingsStore{Path: settings.SettingsPath(environment, home)},
		Environment:   environment,
	})
	application, err := ui.New(ui.Options{Model: model, Environment: environment})
	if err != nil {
		log.Fatal(err)
	}
	application.Run()
}

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

// Command vfox-plus is a desktop application for vfox, built with mygo.
package main

import (
	"log"
	"os"
	"strings"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui"
)

func main() {
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

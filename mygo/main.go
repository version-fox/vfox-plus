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

// Command vfox-plus is the desktop application for vfox, built with mygo.
//
// It is an application and not a command: it takes no arguments, and everything
// that it can do is in its settings, the web interface among it (see package
// pair). Starting it again brings its window back.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui"
)

// maxProcessors is the most processors that the Go scheduler of the application uses. The work of
// the application is waiting, for vfox and for the window, and nothing that asks for more than a
// few processors; and each processor that the scheduler uses keeps caches of memory of its own,
// which are private memory: on a machine of 16 threads, 0.7 MB of the 15.7 MB that the
// application holds without its window.
const maxProcessors = 4

// limitProcessors keeps the Go scheduler to maxProcessors, unless the user chose a number with
// GOMAXPROCS.
func limitProcessors() {
	if os.Getenv("GOMAXPROCS") != "" {
		return
	}
	if runtime.GOMAXPROCS(0) > maxProcessors {
		runtime.GOMAXPROCS(maxProcessors)
	}
}

func main() {
	limitProcessors()
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			environment[key] = value
		}
	}
	home, _ := os.UserHomeDir()
	settingsPath := settings.SettingsPath(environment, home)
	settingsDir := filepath.Dir(settingsPath)
	taskLogDir, _ := core.UserCacheSubdir("vfox-plus", "tasks")
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        core.OsProcessRunner{},
		SettingsStore: settings.JSONFileSettingsStore{Path: settingsPath},
		Environment:   environment,
		TaskLockPath:  filepath.Join(settingsDir, "tasks.lock"),
		TaskLogDir:    taskLogDir,
	})
	// The web interface is off until the user turns it on in the settings, and
	// is served from then on by the application, for as long as it runs. The
	// settings are loaded before it starts, so that it starts as they say.
	model.Settings.Load()
	web := pair.NewHost(model, pair.HostOptions{SettingsDir: settingsDir, Quit: mygo.App.Quit})
	application, err := ui.New(ui.Options{Model: model, Environment: environment, Web: web})
	if err != nil {
		log.Fatal(err)
	}
	application.Run()
}

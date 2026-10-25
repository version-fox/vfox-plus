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
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui"
)

func main() {
	// `vfox-plus pair` pairs a browser with this machine, `vfox-plus serve`
	// keeps the pairing server up for containers and remote instances;
	// anything else starts the desktop application.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "pair":
			os.Exit(pair.Main(os.Args[2:]))
		case "serve":
			os.Exit(pair.ServeMain(os.Args[2:]))
		case "help", "--help", "-h":
			fmt.Println("Usage: vfox-plus [pair|serve]")
			fmt.Println()
			fmt.Println("  (no arguments)  Start the desktop application.")
			fmt.Println("  pair            Pair a browser with this machine over loopback.")
			fmt.Println("  serve           Keep the pairing server up for containers and remote instances.")
			return
		}
	}
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

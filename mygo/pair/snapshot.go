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

package pair

import (
	"os"
	"path/filepath"
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// localEnv returns the process environment and the home directory.
func localEnv() (map[string]string, string) {
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			environment[key] = value
		}
	}
	home, _ := os.UserHomeDir()
	return environment, home
}

// LocalSettingsDir is the directory of the GUI settings.
func LocalSettingsDir() string {
	environment, home := localEnv()
	return filepath.Dir(settings.SettingsPath(environment, home))
}

// localLaunchConfig points a vfox client at this machine: the user's
// settings, the process environment and the located vfox.
func localLaunchConfig() (vfox.VfoxLaunchConfig, map[string]string) {
	environment, _ := localEnv()
	home, _ := os.UserHomeDir()
	store := settings.JSONFileSettingsStore{Path: settings.SettingsPath(environment, home)}
	stored := store.Load()
	config := vfox.DefaultLaunchConfig()
	config.BaseEnvironment = environment
	config.ExtraEnvironment = stored.ExtraEnvironment
	if stored.VfoxExecutable != "" {
		config.Executable = stored.VfoxExecutable
	} else {
		config.Executable = vfox.NewVfoxLocator(environment, nil).Locate()
	}
	if config.Executable == "" {
		config.Executable = vfox.VfoxExecutableName
	}
	return config, environment
}

// SnapshotOf reads the state of client: the vfox version and the installed
// SDKs. A vfox that is missing or broken yields a snapshot that says so
// instead of an error, so that the pairing page can explain it.
func SnapshotOf(client *vfox.Client) Snapshot {
	version, err := client.Version()
	if err != nil {
		return Snapshot{VfoxError: err.Error()}
	}
	installed, err := client.InstalledSdks()
	if err != nil {
		return Snapshot{HasVfox: true, VfoxVersion: version, VfoxError: err.Error()}
	}
	return Snapshot{HasVfox: true, VfoxVersion: version, SDKs: Summarize(installed)}
}

// LocalSnapshot returns the read-only state of this machine.
func LocalSnapshot() func() Snapshot {
	config, _ := localLaunchConfig()
	client := vfox.NewClient(core.OsProcessRunner{}, config)
	return func() Snapshot { return SnapshotOf(client) }
}

// NewLocalModel builds the application state of this machine and starts it,
// like the desktop application does. The caller owns it and must Dispose it.
func NewLocalModel() *state.AppModel {
	_, environment := localLaunchConfig()
	home, _ := os.UserHomeDir()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        core.OsProcessRunner{},
		SettingsStore: settings.JSONFileSettingsStore{Path: settings.SettingsPath(environment, home)},
		Environment:   environment,
	})
	model.Start()
	return model
}

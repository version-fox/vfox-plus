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

package state

import (
	"maps"
	"sync"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

// AppModelOptions describes what an AppModel is made of.
type AppModelOptions struct {
	// Runner spawns vfox.
	Runner core.ProcessRunner
	// SettingsStore persists the settings.
	SettingsStore settings.SettingsStore
	// Environment is the environment of the application.
	Environment map[string]string
	// Locator finds vfox; nil uses a locator that searches `PATH` and the
	// usual places.
	Locator *vfox.VfoxLocator
}

// AppModel is the application state: every controller, wired together.
//
// The model owns the Client and keeps the derived data consistent: installed
// SDKs are reloaded whenever vfox becomes available or a task that changes
// them finishes.
type AppModel struct {
	// Cli is the vfox client shared by all controllers.
	Cli *vfox.Client
	// Settings holds the application preferences.
	Settings *SettingsController
	// Vfox reports the availability of the vfox executable.
	Vfox *VfoxController
	// Installed lists installed plugins and their versions.
	Installed *AsyncResource[[]core.InstalledSdk]
	// Registry lists plugins available in the registry.
	Registry *AsyncResource[[]core.AvailablePlugin]
	// VfoxConfig is the vfox configuration.
	VfoxConfig *VfoxConfigController
	// Tasks runs state-changing operations.
	Tasks *TaskQueue

	mu               sync.Mutex
	pluginCopies     map[int]vfox.PreparedPluginSource
	stops            []Unsubscribe
	lastAvailability VfoxAvailability
	appliedSettings  *settings.AppSettings
}

// NewAppModel creates the application state.
func NewAppModel(options AppModelOptions) *AppModel {
	locator := options.Locator
	if locator == nil {
		locator = vfox.NewVfoxLocator(options.Environment, nil)
	}
	config := vfox.DefaultLaunchConfig()
	config.BaseEnvironment = options.Environment
	model := &AppModel{
		Cli:              vfox.NewClient(options.Runner, config),
		pluginCopies:     make(map[int]vfox.PreparedPluginSource),
		lastAvailability: VfoxUnknown,
	}
	model.Settings = NewSettingsController(options.SettingsStore)
	model.Vfox = NewVfoxController(model.Cli, locator, options.Environment)
	model.Installed = NewAsyncResource(model.Cli.InstalledSdks)
	model.Registry = NewAsyncResource(model.Cli.AvailablePlugins)
	model.VfoxConfig = NewVfoxConfigController(model.Cli)
	model.Tasks = NewTaskQueue(model.Cli, 0, 0)
	model.stops = []Unsubscribe{
		model.Tasks.OnFinished(model.onTaskFinished),
		model.Vfox.AddListener(model.onVfoxChanged),
		model.Settings.AddListener(model.onSettingsChanged),
	}
	return model
}

// Start loads the settings and probes vfox. It blocks while vfox is probed,
// so a user interface calls it on its own goroutine.
func (m *AppModel) Start() {
	m.Settings.Load()
	m.applySettings()
}

// InstalledSdk returns the installed SDK called name, if it is installed.
func (m *AppModel) InstalledSdk(name string) (core.InstalledSdk, bool) {
	sdks, _ := m.Installed.Value()
	for _, sdk := range sdks {
		if sdk.Name == name {
			return sdk, true
		}
	}
	return core.InstalledSdk{}, false
}

// SdkDetail creates the controller for the detail view of name.
func (m *AppModel) SdkDetail(name string) *SdkDetailController {
	return NewSdkDetailController(m.Cli, name)
}

// Run queues command and returns its task.
func (m *AppModel) Run(command vfox.VfoxCommand) *VfoxTask {
	return m.Tasks.Enqueue(command)
}

// AddPluginFromSource adds a plugin from source, a URL or a local `.zip` or
// `.lua` file, optionally under alias.
//
// vfox moves a local `.lua` file away instead of reading it, so such a file
// is copied into the temporary directory of vfox first and vfox gets the
// copy; the copy is deleted once the task has finished. It returns an error
// if alias is not a valid plugin name.
func (m *AppModel) AddPluginFromSource(source, alias string) (*VfoxTask, error) {
	temporaryRoot := vfox.VfoxTemporaryDirectory(vfox.VfoxEnvironment(m.Cli.LaunchConfig(), false))
	prepared := vfox.PreparePluginSource(source, temporaryRoot)
	command, err := m.Cli.AddPlugin("", prepared.Path, alias)
	if err != nil {
		prepared.Dispose()
		return nil, err
	}
	task := m.Run(command)
	if prepared.IsCopy() {
		m.mu.Lock()
		m.pluginCopies[task.ID] = prepared
		m.mu.Unlock()
	}
	return task, nil
}

func (m *AppModel) onVfoxChanged() {
	availability := m.Vfox.Availability()
	m.mu.Lock()
	changed := availability != m.lastAvailability
	m.lastAvailability = availability
	m.mu.Unlock()
	if !changed || availability != VfoxReady {
		return
	}
	m.Installed.Load()
	if m.Registry.HasValue() || m.Registry.Error() != nil {
		m.Registry.Load()
	}
	if m.VfoxConfig.HasValue() || m.VfoxConfig.Error() != nil {
		m.VfoxConfig.Load()
	}
}

func (m *AppModel) onSettingsChanged() {
	current := m.Settings.Settings()
	m.mu.Lock()
	previous := m.appliedSettings
	m.mu.Unlock()
	// Before the first apply, Start applies the settings itself.
	if previous == nil || previous.VfoxExecutable == current.VfoxExecutable &&
		maps.Equal(previous.ExtraEnvironment, current.ExtraEnvironment) {
		return
	}
	go m.applySettings()
}

func (m *AppModel) applySettings() {
	current := m.Settings.Settings()
	m.mu.Lock()
	m.appliedSettings = &current
	// Force a reload of the dependent data even if vfox stays available.
	m.lastAvailability = VfoxUnknown
	m.mu.Unlock()
	m.Vfox.Apply(current)
}

func (m *AppModel) onTaskFinished(task *VfoxTask) {
	m.mu.Lock()
	prepared, ok := m.pluginCopies[task.ID]
	delete(m.pluginCopies, task.ID)
	m.mu.Unlock()
	if ok {
		prepared.Dispose()
	}
	if task.Status() == TaskCancelled {
		if _, started := task.StartedAt(); !started {
			return
		}
	}
	m.Installed.Load()
}

// Dispose releases every resource.
func (m *AppModel) Dispose() {
	for _, stop := range m.stops {
		stop()
	}
	m.mu.Lock()
	copies := m.pluginCopies
	m.pluginCopies = make(map[int]vfox.PreparedPluginSource)
	m.mu.Unlock()
	for _, prepared := range copies {
		prepared.Dispose()
	}
	m.Tasks.Dispose()
	m.Installed.Dispose()
	m.Registry.Dispose()
	m.VfoxConfig.Dispose()
	m.Vfox.Dispose()
	m.Settings.Dispose()
}

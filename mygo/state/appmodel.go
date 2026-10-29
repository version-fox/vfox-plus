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
	"fmt"
	"maps"
	"strings"
	"sync"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
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
	// TaskLockPath, when set, is held by every task while its process runs,
	// so that processes on the same machine serialize vfox mutations.
	TaskLockPath string
	// TaskLogDir, when set, spills the full log of every task to per-task
	// files while it runs. Successful and cancelled tasks delete theirs when
	// they finish; failed tasks keep theirs, pruned to the newest files.
	TaskLogDir string
	// Profiles is what the application knows about particular plugins; nil
	// takes the profiles of the plugins of the version-fox organisation, for
	// the platform the application runs on.
	Profiles *profile.Registry
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
	// Profiles is what the application knows about particular plugins.
	Profiles *profile.Registry

	mu               sync.Mutex
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
	profiles := options.Profiles
	if profiles == nil {
		profiles = profile.NewRegistry(profile.CurrentPlatform())
	}
	model := &AppModel{
		Cli:              vfox.NewClient(options.Runner, config),
		Profiles:         profiles,
		lastAvailability: VfoxUnknown,
	}
	model.Settings = NewSettingsController(options.SettingsStore)
	model.Vfox = NewVfoxController(model.Cli, locator, options.Environment)
	model.Installed = NewAsyncResource(model.Cli.InstalledSdks)
	model.Registry = NewAsyncResource(model.Cli.AvailablePlugins)
	model.VfoxConfig = NewVfoxConfigController(model.Cli)
	model.Tasks = NewTaskQueue(model.Cli, 0, 0)
	model.Tasks.LockPath = options.TaskLockPath
	model.Tasks.TaskLogDir = options.TaskLogDir
	model.stops = []Unsubscribe{
		model.Tasks.OnFinished(model.onTaskFinished),
		model.Vfox.AddListener(model.onVfoxChanged),
		model.Settings.AddListener(model.onSettingsChanged),
	}
	return model
}

// Start loads the settings and points the vfox client at them, without
// probing vfox: the first page that needs vfox triggers the probe through
// Vfox.EnsureChecked, and every settings change that affects vfox re-probes
// through Apply at once. It does not block.
func (m *AppModel) Start() {
	m.Settings.Load()
	m.configure()
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

// SdkDetail creates the controller for the detail view of name. Its search
// starts at the choice that the user made for the plugin, and the choices that
// they make are kept in the settings.
func (m *AppModel) SdkDetail(name string) *SdkDetailController {
	return NewSdkDetailController(m.Cli, name, m.Profiles, m)
}

// RememberedSearch returns what the user chose to list for the plugin, the value
// of a choice of its profile, and empty when they did not choose. It implements
// SearchMemory.
func (m *AppModel) RememberedSearch(plugin string) string {
	return m.Settings.Settings().Searches[plugin]
}

// RememberSearch keeps what the user chose to list for the plugin in the
// settings, and forgets the choice for an empty value. It implements
// SearchMemory.
func (m *AppModel) RememberSearch(plugin, value string) {
	m.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		if value == "" {
			delete(current.Searches, plugin)
			return current
		}
		if current.Searches == nil {
			current.Searches = map[string]string{}
		}
		current.Searches[plugin] = value
		return current
	})
}

// PluginVariable returns the value that vfox runs with for an environment
// variable of a plugin, empty when the variable is not set.
func (m *AppModel) PluginVariable(key string) string {
	return m.Settings.Settings().ExtraEnvironment[key]
}

// SetPluginVariable sets an environment variable that the profile lists, such
// as the download mirror of Node.js, and removes it for an empty value. The
// variable joins the extra environment of the settings, which every vfox
// process runs with, so the settings page shows it among the others, and it
// belongs to the plugin whatever name vfox knows the plugin by. It refuses a
// variable the plugin does not read, and a value the variable does not take (a
// *profile.ValueError).
func (m *AppModel) SetPluginVariable(p profile.Profile, key, value string) error {
	variable, ok := p.Variable(key)
	if !ok {
		return fmt.Errorf("%s does not read %s", p.Plugin, key)
	}
	if err := variable.Validate(value); err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	m.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		if value == "" {
			delete(current.ExtraEnvironment, key)
			return current
		}
		if current.ExtraEnvironment == nil {
			current.ExtraEnvironment = map[string]string{}
		}
		current.ExtraEnvironment[key] = value
		return current
	})
	return nil
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
//
// The copy is bound to the task when it is enqueued, so that a task which
// finishes immediately cannot outrun the registration of the copy.
func (m *AppModel) AddPluginFromSource(source, alias string) (*VfoxTask, error) {
	temporaryRoot := vfox.VfoxTemporaryDirectory(vfox.VfoxEnvironment(m.Cli.LaunchConfig(), false))
	prepared := vfox.PreparePluginSource(source, temporaryRoot)
	command, err := m.Cli.AddPlugin("", prepared.Path, alias)
	if err != nil {
		prepared.Dispose()
		return nil, err
	}
	if !prepared.IsCopy() {
		return m.Run(command), nil
	}
	return m.Tasks.EnqueueWithFinalizer(command, prepared.Dispose), nil
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
	m.configure()
	// The executable or the environment may have changed: re-probe at once.
	// The first probe, by contrast, waits for the first page that needs vfox.
	m.Vfox.Check()
}

// configure points the vfox client at the stored settings. It never spawns
// vfox: locating the executable only searches the file system.
func (m *AppModel) configure() {
	current := m.Settings.Settings()
	m.mu.Lock()
	m.appliedSettings = &current
	// Force a reload of the dependent data even if vfox stays available.
	m.lastAvailability = VfoxUnknown
	m.mu.Unlock()
	m.Vfox.Configure(current)
}

func (m *AppModel) onTaskFinished(task *VfoxTask) {
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
	// The queue runs the finalizers of the tasks that never finished, which
	// deletes the private copies of the plugin files they own.
	m.Tasks.Dispose()
	m.Installed.Dispose()
	m.Registry.Dispose()
	m.VfoxConfig.Dispose()
	m.Vfox.Dispose()
	m.Settings.Dispose()
}

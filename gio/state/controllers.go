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
	"errors"
	"log"
	"strings"
	"sync"

	core "github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
)

// SettingsController holds the AppSettings and persists every change.
type SettingsController struct {
	Notifier
	mu       sync.Mutex
	store    settings.SettingsStore
	settings settings.AppSettings
}

// NewSettingsController creates a controller on store.
func NewSettingsController(store settings.SettingsStore) *SettingsController {
	return &SettingsController{store: store, settings: settings.DefaultSettings()}
}

// Settings returns the current settings. Its maps belong to the controller:
// change a Clone, and hand it to Update.
func (c *SettingsController) Settings() settings.AppSettings {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.settings
}

// Load loads the persisted settings.
func (c *SettingsController) Load() {
	loaded := c.store.Load()
	c.mu.Lock()
	c.settings = loaded
	c.mu.Unlock()
	c.NotifyListeners()
}

// Update applies change to a copy of the settings and saves the result,
// unless nothing changed.
func (c *SettingsController) Update(change func(settings.AppSettings) settings.AppSettings) {
	c.mu.Lock()
	updated := change(c.settings.Clone())
	if updated.Equal(c.settings) {
		c.mu.Unlock()
		return
	}
	c.settings = updated
	c.mu.Unlock()
	c.NotifyListeners()
	if err := c.store.Save(updated); err != nil {
		log.Printf("saving the settings failed: %v", err)
	}
}

// VfoxAvailability tells whether vfox can be used:
//
//   - VfoxUnknown: not checked yet;
//   - VfoxChecking: the executable is being probed;
//   - VfoxReady: vfox runs and reported its version;
//   - VfoxNotFound: no vfox executable was found;
//   - VfoxBroken: an executable was found but did not behave like vfox.
type VfoxAvailability string

// The availability states.
const (
	VfoxUnknown  VfoxAvailability = "unknown"
	VfoxChecking VfoxAvailability = "checking"
	VfoxReady    VfoxAvailability = "ready"
	VfoxNotFound VfoxAvailability = "notFound"
	VfoxBroken   VfoxAvailability = "broken"
)

// VfoxController locates vfox, configures the Client and checks that vfox
// works.
type VfoxController struct {
	Notifier
	cli         *vfox.Client
	locator     *vfox.VfoxLocator
	environment map[string]string

	mu            sync.Mutex
	availability  VfoxAvailability
	executable    string
	hasExecutable bool
	custom        bool
	version       string
	hasVersion    bool
	err           error
}

// NewVfoxController creates a controller that configures cli.
func NewVfoxController(cli *vfox.Client, locator *vfox.VfoxLocator, environment map[string]string) *VfoxController {
	return &VfoxController{cli: cli, locator: locator, environment: environment, availability: VfoxUnknown}
}

// Availability reports whether vfox can be used.
func (c *VfoxController) Availability() VfoxAvailability {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.availability
}

// Executable returns the executable in use, if one was found.
func (c *VfoxController) Executable() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.executable, c.hasExecutable
}

// IsCustomExecutable reports whether the executable was chosen by the user
// rather than detected.
func (c *VfoxController) IsCustomExecutable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.custom
}

// Version returns the vfox version, once vfox is ready.
func (c *VfoxController) Version() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.version, c.hasVersion
}

// Error returns why vfox is not available, if it is not.
func (c *VfoxController) Error() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Apply points the CLI at the executable and environment from appSettings
// and checks the result. It blocks while vfox is probed.
func (c *VfoxController) Apply(appSettings settings.AppSettings) {
	custom := appSettings.VfoxExecutable != ""
	executable := appSettings.VfoxExecutable
	if !custom {
		executable = c.locator.Locate()
	}
	config := c.cli.LaunchConfig()
	config.Executable = executable
	if config.Executable == "" {
		config.Executable = vfox.VfoxExecutableName
	}
	config.BaseEnvironment = c.environment
	config.ExtraEnvironment = appSettings.ExtraEnvironment
	c.cli.SetLaunchConfig(config)
	c.mu.Lock()
	c.custom = custom
	c.executable, c.hasExecutable = executable, executable != ""
	c.mu.Unlock()
	c.Check()
}

// Check runs `vfox --version` to check that vfox works. It blocks while vfox
// is probed.
func (c *VfoxController) Check() {
	c.mu.Lock()
	c.availability = VfoxChecking
	c.mu.Unlock()
	c.NotifyListeners()
	version, err := c.cli.Version()
	c.mu.Lock()
	if err == nil {
		c.version, c.hasVersion = version, true
		c.err = nil
		c.availability = VfoxReady
	} else {
		c.version, c.hasVersion = "", false
		c.err = err
		var notFound *vfox.VfoxNotFoundError
		var failed *vfox.VfoxError
		switch {
		case errors.As(err, &notFound):
			c.availability = VfoxNotFound
		case errors.As(err, &failed):
			c.availability = VfoxBroken
		default:
			c.availability = VfoxBroken
			log.Printf("checking vfox failed: %v", err)
		}
	}
	c.mu.Unlock()
	c.NotifyListeners()
}

// VfoxConfigController is the vfox configuration (`vfox config`), editable
// key by key.
type VfoxConfigController struct {
	*AsyncResource[[]core.ConfigEntry]
	cli *vfox.Client

	mu     sync.Mutex
	saving map[string]bool
}

// NewVfoxConfigController creates a controller on cli.
func NewVfoxConfigController(cli *vfox.Client) *VfoxConfigController {
	return &VfoxConfigController{
		AsyncResource: NewAsyncResource(cli.ConfigEntries),
		cli:           cli,
		saving:        make(map[string]bool),
	}
}

// ValueFor returns the value of key, or false if vfox does not know the key.
func (c *VfoxConfigController) ValueFor(key string) (string, bool) {
	entries, ok := c.Value()
	if !ok {
		return "", false
	}
	for _, entry := range entries {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return "", false
}

// IsSaving reports whether key is being saved.
func (c *VfoxConfigController) IsSaving(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saving[key]
}

// Set sets key to newValue and reloads the configuration.
//
// It returns a *vfox.VfoxError if vfox rejects the value; the configuration
// is reloaded either way. It blocks until vfox is done.
func (c *VfoxConfigController) Set(key, newValue string) error {
	return c.save(key, func() error { return c.cli.SetConfig(key, newValue) })
}

// Reset resets key to its default and reloads the configuration. It blocks
// until vfox is done.
func (c *VfoxConfigController) Reset(key string) error {
	return c.save(key, func() error { return c.cli.UnsetConfig(key) })
}

func (c *VfoxConfigController) save(key string, action func() error) error {
	c.mu.Lock()
	c.saving[key] = true
	c.mu.Unlock()
	c.NotifyListeners()
	err := action()
	c.mu.Lock()
	delete(c.saving, key)
	c.mu.Unlock()
	<-c.Load()
	return err
}

// VersionFilter selects which versions are listed on the "available" tab.
//
//   - VersionFilterAll: every version;
//   - VersionFilterWithNote: versions carrying a note such as `LTS`;
//   - VersionFilterInstalled: versions that are installed already.
type VersionFilter string

// The version filters.
const (
	VersionFilterAll       VersionFilter = "all"
	VersionFilterWithNote  VersionFilter = "withNote"
	VersionFilterInstalled VersionFilter = "installed"
)

// SdkDetailController is the state of the detail view of one installed plugin.
type SdkDetailController struct {
	Notifier
	// Name is the plugin name.
	Name string
	// Info is the plugin metadata, loaded on demand.
	Info *AsyncResource[core.PluginInfo]
	// Versions are the installable versions, loaded on demand because the
	// search can be slow.
	Versions *AsyncResource[[]core.AvailableVersion]

	mu     sync.Mutex
	query  string
	filter VersionFilter
	stop   Unsubscribe
}

// NewSdkDetailController creates the controller for the detail view of name.
func NewSdkDetailController(cli *vfox.Client, name string) *SdkDetailController {
	c := &SdkDetailController{Name: name, filter: VersionFilterAll}
	c.Info = NewAsyncResource(func() (core.PluginInfo, error) { return cli.PluginInfo(name) })
	c.Versions = NewAsyncResource(func() ([]core.AvailableVersion, error) { return cli.SearchVersions(name) })
	c.stop = c.Versions.AddListener(c.NotifyListeners)
	return c
}

// Query returns the text the available versions are filtered by.
func (c *SdkDetailController) Query() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.query
}

// SetQuery sets the filter text.
func (c *SdkDetailController) SetQuery(value string) {
	c.mu.Lock()
	changed := value != c.query
	c.query = value
	c.mu.Unlock()
	if changed {
		c.NotifyListeners()
	}
}

// Filter returns which available versions are shown.
func (c *SdkDetailController) Filter() VersionFilter {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.filter
}

// SetFilter sets which available versions are shown.
func (c *SdkDetailController) SetFilter(value VersionFilter) {
	c.mu.Lock()
	changed := value != c.filter
	c.filter = value
	c.mu.Unlock()
	if changed {
		c.NotifyListeners()
	}
}

// VisibleVersions returns the available versions matching the query and the
// filter.
//
// installed holds the installed versions. It is passed in rather than taken
// from the search output, so that it stays current after installs without
// searching again.
func (c *SdkDetailController) VisibleVersions(installed map[string]bool) []core.AvailableVersion {
	c.mu.Lock()
	query, filter := c.query, c.filter
	c.mu.Unlock()
	versions, _ := c.Versions.Value()
	needle := strings.ToLower(strings.TrimSpace(query))
	var result []core.AvailableVersion
	for _, version := range versions {
		if matchesFilter(version, filter, installed) && matchesQuery(version, needle) {
			result = append(result, version)
		}
	}
	return result
}

func matchesFilter(version core.AvailableVersion, filter VersionFilter, installed map[string]bool) bool {
	switch filter {
	case VersionFilterWithNote:
		return version.HasNote
	case VersionFilterInstalled:
		return installed[version.Version]
	default:
		return true
	}
}

func matchesQuery(version core.AvailableVersion, needle string) bool {
	return needle == "" || strings.Contains(strings.ToLower(version.Version), needle) ||
		version.HasNote && strings.Contains(strings.ToLower(version.Note), needle)
}

// Dispose releases every resource.
func (c *SdkDetailController) Dispose() {
	c.stop()
	c.Info.Dispose()
	c.Versions.Dispose()
	c.Notifier.Dispose()
}

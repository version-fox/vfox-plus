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
	"fmt"
	"log"
	"strings"
	"sync"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
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
	// checked reports whether the current configuration was probed.
	checked bool
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
	c.Configure(appSettings)
	c.Check()
}

// Configure points the CLI at the executable and environment from
// appSettings, without probing vfox. The next EnsureChecked probes again.
// It does not notify: the settings change itself already rebuilds the views
// that show the executable, and Check notifies for the probe that follows.
func (c *VfoxController) Configure(appSettings settings.AppSettings) {
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
	c.checked = false
	c.mu.Unlock()
}

// EnsureChecked probes vfox unless this configuration was already checked.
// It blocks while vfox is probed, like Check: user interfaces call it off
// their drawing goroutine. The application starts without probing vfox at
// all; the first page that needs it triggers the probe, and every settings
// change that affects vfox re-probes through Apply at once.
func (c *VfoxController) EnsureChecked() {
	c.mu.Lock()
	checked := c.checked
	c.mu.Unlock()
	if checked {
		return
	}
	c.Check()
}

// Check runs `vfox --version` to check that vfox works. It blocks while vfox
// is probed.
func (c *VfoxController) Check() {
	c.mu.Lock()
	c.availability = VfoxChecking
	c.checked = true
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

// SearchMemory keeps what the user chose to list for a plugin, such as the
// distribution of Java, so that the choice outlives the details that were open.
type SearchMemory interface {
	// RememberedSearch returns the value of the choice made for the plugin,
	// empty when none was made.
	RememberedSearch(plugin string) string
	// RememberSearch keeps value for the plugin; empty forgets the choice.
	RememberSearch(plugin, value string)
}

// SdkDetailController is the state of the detail view of one installed plugin.
type SdkDetailController struct {
	Notifier
	// Name is the name that vfox knows the plugin by: the alias, for a plugin
	// that was added under one.
	Name string
	// Info is the plugin metadata, loaded on demand.
	Info *AsyncResource[core.PluginInfo]
	// Versions are the installable versions, loaded on demand because the
	// search can be slow. They are the versions of the chosen search: a change
	// of SetSearch empties them and loads the next list.
	Versions *AsyncResource[[]core.AvailableVersion]

	profiles *profile.Registry
	memory   SearchMemory

	mu        sync.Mutex
	known     profile.Profile
	hasKnown  bool
	query     string
	filter    VersionFilter
	search    string
	selection profile.Selection
	stops     []Unsubscribe
	// groups are the groups of the list that starts at groupsOf and has groupsLen versions.
	groups    []profile.Group
	groupsOf  *core.AvailableVersion
	groupsLen int
}

// NewSdkDetailController creates the controller for the detail view of name.
// profiles may be nil, which is the same as a registry that knows no plugin;
// memory may be nil, and then a choice is forgotten with the controller.
//
// The profile is the one of the plugin that has the name; a plugin that was
// added under an alias has none until its metadata says which plugin it is (see
// Profile). The search starts at the choice that the memory holds.
func NewSdkDetailController(cli *vfox.Client, name string, profiles *profile.Registry, memory SearchMemory) *SdkDetailController {
	c := &SdkDetailController{Name: name, filter: VersionFilterAll, profiles: profiles, memory: memory}
	c.known, c.hasKnown = profiles.Lookup(name)
	c.search = c.rememberedSearch()
	c.Info = NewAsyncResource(func() (core.PluginInfo, error) { return cli.PluginInfo(name) })
	c.Versions = NewAsyncResource(func() ([]core.AvailableVersion, error) {
		return cli.SearchVersions(name, c.SearchArguments()...)
	})
	c.stops = []Unsubscribe{c.Versions.AddListener(c.NotifyListeners), c.Info.AddListener(c.onInfo)}
	return c
}

// Profile returns what the application knows about the plugin, and whether it
// knows something.
//
// vfox names a plugin that was added under an alias (`vfox add --alias`) by the
// alias, while the plugin still says what it is: the profile of such a plugin
// is found once the Info is loaded, by the name that the plugin declares.
func (c *SdkDetailController) Profile() (profile.Profile, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.known, c.hasKnown
}

// onInfo finds the profile of a plugin that its name does not give away.
func (c *SdkDetailController) onInfo() {
	info, ok := c.Info.Value()
	if !ok {
		return
	}
	c.mu.Lock()
	if c.hasKnown {
		c.mu.Unlock()
		return
	}
	known, found := c.profiles.Lookup(info.Name)
	if !found {
		c.mu.Unlock()
		return
	}
	c.known, c.hasKnown = known, true
	search := c.rememberedSearch()
	changed := search != c.search
	c.search = search
	c.mu.Unlock()
	if changed {
		// What is listed, or being listed, answers another search.
		c.Versions.Reset()
	}
	c.NotifyListeners()
}

// rememberedSearch is the choice that the memory holds for the plugin, when
// the profile offers it, in the form that the controller keeps: empty for the
// default. The caller holds the lock, or has not published the controller yet.
func (c *SdkDetailController) rememberedSearch() string {
	if c.memory == nil || !c.hasKnown || c.known.Search == nil {
		return ""
	}
	choice, ok := c.known.Search.Choice(c.memory.RememberedSearch(c.Name))
	if !ok || choice.Value == c.known.Search.Default {
		return ""
	}
	return choice.Value
}

// Search returns the value of the chosen search: the short name of a
// distribution for vfox-java. It is the default of the plugin until
// SetSearch chooses another, and empty for a plugin whose search takes no
// argument.
func (c *SdkDetailController) Search() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasKnown || c.known.Search == nil {
		return ""
	}
	if c.search == "" {
		return c.known.Search.Default
	}
	return c.search
}

// SearchArguments returns what follows the plugin name in the `vfox search`
// of the chosen search: nothing for the default.
func (c *SdkDetailController) SearchArguments() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.known.Search.Arguments(c.search)
}

// SetSearch chooses what the next search lists, by the value of one of the
// choices of the profile. The versions of the earlier choice go away at once,
// and so do the facets chosen for them; the next list is loaded by whoever asks
// for it (Versions.EnsureLoaded), because a search can be slow and the user
// may not look at the available versions at all. It refuses a value the plugin
// does not take, and a plugin whose search takes no argument.
func (c *SdkDetailController) SetSearch(value string) error {
	_, err := c.setSearch(value)
	return err
}

// ChooseSearch is SetSearch for the choice of the user: the application keeps
// it, so that it is the search of the plugin the next time the plugin is shown,
// after a restart and by every interface of the application.
func (c *SdkDetailController) ChooseSearch(value string) error {
	stored, err := c.setSearch(value)
	if err == nil && c.memory != nil {
		c.memory.RememberSearch(c.Name, stored)
	}
	return err
}

// setSearch makes value the search, and returns it in the form that is kept:
// empty for the default of the plugin.
func (c *SdkDetailController) setSearch(value string) (string, error) {
	c.mu.Lock()
	if !c.hasKnown || c.known.Search == nil {
		c.mu.Unlock()
		return "", fmt.Errorf("%s takes no search argument", c.Name)
	}
	choice, ok := c.known.Search.Choice(value)
	if !ok {
		c.mu.Unlock()
		return "", fmt.Errorf("%s does not list %q", c.Name, value)
	}
	stored := choice.Value
	if stored == c.known.Search.Default {
		stored = ""
	}
	changed := stored != c.search
	c.search = stored
	if changed {
		// Nothing of the earlier list stays, so that it can be freed.
		c.selection = nil
		c.groups, c.groupsOf, c.groupsLen = nil, nil, 0
	}
	c.mu.Unlock()
	if changed {
		c.Versions.Reset()
	}
	return stored, nil
}

// Groups returns the filters that suit the versions that are listed, with the
// values that some of them have. The interface asks for them on every frame, and
// they only change with the list, so they are worked out once per list.
func (c *SdkDetailController) Groups() []profile.Group {
	versions, _ := c.Versions.Value()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasKnown || len(versions) == 0 {
		return nil
	}
	if c.groupsOf == &versions[0] && c.groupsLen == len(versions) {
		return c.groups
	}
	c.groups = c.known.Groups(versions)
	c.groupsOf, c.groupsLen = &versions[0], len(versions)
	return c.groups
}

// Selection returns the values that the user chose in the groups.
func (c *SdkDetailController) Selection() profile.Selection {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selection
}

// ToggleFacet chooses the value of a group, or takes the choice back.
func (c *SdkDetailController) ToggleFacet(group, value string) {
	c.mu.Lock()
	c.selection = c.selection.Toggled(group, value)
	c.mu.Unlock()
	c.NotifyListeners()
}

// ClearFacets takes back every choice of the groups.
func (c *SdkDetailController) ClearFacets() {
	c.mu.Lock()
	changed := !c.selection.Empty()
	c.selection = nil
	c.mu.Unlock()
	if changed {
		c.NotifyListeners()
	}
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
	query, filter, selection := c.query, c.filter, c.selection
	known, hasKnown := c.known, c.hasKnown
	c.mu.Unlock()
	versions, _ := c.Versions.Value()
	needle := strings.ToLower(strings.TrimSpace(query))
	var result []core.AvailableVersion
	for _, version := range versions {
		if !matchesFilter(version, filter, installed) || !matchesQuery(version, needle) {
			continue
		}
		if hasKnown && !known.Matches(selection, version) {
			continue
		}
		result = append(result, version)
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
	for _, stop := range c.stops {
		stop()
	}
	c.Info.Dispose()
	c.Versions.Dispose()
	c.Notifier.Dispose()
}

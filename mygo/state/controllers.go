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
	"os"
	"reflect"
	"slices"
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
// is probed. vfox older than the minimum supported version counts as broken,
// so that the user upgrades instead of meeting commands that do not exist yet.
func (c *VfoxController) Check() {
	c.mu.Lock()
	c.availability = VfoxChecking
	c.checked = true
	c.mu.Unlock()
	c.NotifyListeners()
	version, err := c.cli.Version()
	if err == nil && !vfox.IsSupportedVersion(version) {
		err = &vfox.VfoxError{Message: "vfox " + version + " is too old: vfox-plus needs vfox " + vfox.MinimumVfoxVersion + " or later"}
	}
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
	// Aliases are the version aliases of the plugin (`vfox alias list <name>`),
	// loaded on demand for the options tab.
	Aliases *AsyncResource[[]vfox.VersionAlias]
	// Files are what the files of the installed plugin say, loaded on demand: the
	// profile that it ships, the variables and version files that it reads and
	// whether it finds installs that are already on the machine.
	Files *AsyncResource[profile.PluginFiles]

	cli      *vfox.Client
	profiles *profile.Registry
	memory   SearchMemory

	// resolving serializes the lookups of the profile.
	resolving  sync.Mutex
	mu         sync.Mutex
	resolution profile.Resolution
	query      string
	filter     VersionFilter
	search     string
	// chosen is whether search was set, rather than remembered.
	chosen    bool
	selection profile.Selection
	stops     []Unsubscribe
	// listed is what was worked out for the last list of versions.
	listed listedFacets
}

// listedFacets are the facets of a list of versions, which starts at first and
// has count of them, and the groups of their values.
type listedFacets struct {
	first  *core.AvailableVersion
	count  int
	facets []profile.Facet
	groups []profile.Group
}

// NewSdkDetailController creates the controller for the detail view of name.
// profiles may be nil, which is the same as a registry that knows no plugin;
// memory may be nil, and then a choice is forgotten with the controller.
//
// The profile is the one of the plugin that has the name (see
// profile.Registry.Resolve). It is looked up again when the metadata of the
// plugin says which plugin it is, such as for one that was added under an
// alias, when its files are read, which may hold a profile of their own, and
// when the profiles change. The search starts at the choice that the memory
// holds.
func NewSdkDetailController(cli *vfox.Client, name string, profiles *profile.Registry, memory SearchMemory) *SdkDetailController {
	c := &SdkDetailController{Name: name, filter: VersionFilterAll, cli: cli, profiles: profiles, memory: memory}
	c.Info = NewAsyncResource(func() (core.PluginInfo, error) { return cli.PluginInfo(name) })
	c.Versions = NewAsyncResource(func() ([]core.AvailableVersion, error) {
		return cli.SearchVersions(name, c.SearchArguments()...)
	})
	c.Aliases = NewAsyncResource(func() ([]vfox.VersionAlias, error) { return cli.Aliases(name) })
	c.Files = NewAsyncResource(func() (profile.PluginFiles, error) { return c.readFiles(), nil })
	c.stops = []Unsubscribe{
		c.Versions.AddListener(c.NotifyListeners), c.Info.AddListener(c.resolve), c.Files.AddListener(c.resolve),
		c.Aliases.AddListener(c.NotifyListeners), profiles.OnChange(c.resolve),
	}
	// Looked up once the changes are listened to, so that none is missed.
	c.applyProfile()
	return c
}

// readFiles reads the files of the installed plugin, from the directory that
// the environment of vfox says. The plugin is installed under the name that
// vfox knows it by (the alias when there is one); the name that it declares is
// tried when there is no such directory.
func (c *SdkDetailController) readFiles() profile.PluginFiles {
	if c.cli == nil {
		return profile.PluginFiles{}
	}
	environment := vfox.VfoxEnvironment(c.cli.LaunchConfig(), false)
	names := []string{c.Name}
	if info, ok := c.Info.Value(); ok && info.Name != "" && info.Name != c.Name {
		names = append(names, info.Name)
	}
	for _, name := range names {
		directory := profile.PluginDir(environment, name)
		if directory == "" {
			continue
		}
		if info, err := os.Stat(directory); err != nil || !info.IsDir() {
			continue
		}
		files := profile.ReadPlugin(os.DirFS(directory))
		for _, problem := range files.Problems {
			log.Printf("the profile of the plugin %s: %s", name, problem)
		}
		return files
	}
	return profile.PluginFiles{}
}

// resolve looks the profile up again (see applyProfile). When the search that
// the list answers changes with it, what is listed, or being listed, answers
// another search: it lists the right one when somebody asked for a list, who
// would wait for it for ever otherwise.
func (c *SdkDetailController) resolve() {
	if c.applyProfile() {
		asked := c.Versions.HasValue() || c.Versions.IsLoading() || c.Versions.Error() != nil
		c.Versions.Reset()
		if asked {
			c.Versions.EnsureLoaded()
		}
	}
	c.NotifyListeners()
}

// applyProfile keeps what the registry says about the plugin, from its name,
// the name that its metadata declares and the profile that it ships, and
// reports whether the search that is listed changes with it.
func (c *SdkDetailController) applyProfile() bool {
	// One at a time: a lookup from older metadata must not follow a newer one.
	c.resolving.Lock()
	defer c.resolving.Unlock()
	var declared string
	if info, ok := c.Info.Value(); ok {
		declared = info.Name
	}
	var own *profile.Document
	if files, ok := c.Files.Value(); ok {
		own = files.Profile
	}
	resolution := c.profiles.Resolve(c.Name, declared, own)
	c.mu.Lock()
	defer c.mu.Unlock()
	if reflect.DeepEqual(resolution, c.resolution) {
		return false
	}
	before := c.resolution.Profile.Search.Arguments(c.search)
	c.resolution = resolution
	if _, offered := resolution.Profile.Search.Choice(c.search); !c.chosen || !offered {
		c.search, c.chosen = c.rememberedSearch(), false
	}
	c.listed = listedFacets{}
	return !slices.Equal(before, resolution.Profile.Search.Arguments(c.search))
}

// Profile returns what the profiles say about the plugin, and whether any
// describes it.
func (c *SdkDetailController) Profile() (profile.Profile, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolution.Profile, c.resolution.Found
}

// EffectiveProfile returns what an interface shows of the plugin: its profile,
// and the filters that its versions give when the profile has none. Unless the
// application or its online directory describes the plugin, what no profile
// carries is inferred from what vfox reports and from the files of the plugin:
// the description and homepage become the guide and the documentation, and the
// legacy files and environment variables those that the plugin reads. It lets
// interfaces offer filters and mirrors for every plugin, without a code change.
func (c *SdkDetailController) EffectiveProfile() (profile.Profile, bool) {
	c.mu.Lock()
	resolution := c.resolution
	c.mu.Unlock()
	versions, _ := c.Versions.Value()
	known := resolution.Profile
	if resolution.Curated {
		known.Facets = c.facets(versions)
		return known, true
	}
	info, hasInfo := c.Info.Value()
	if !resolution.Found && !hasInfo && len(versions) == 0 {
		return profile.Profile{Plugin: c.Name}, false
	}
	inferred := profile.InferredProfile(c.Name, info, nil)
	inferred.Facets = c.facets(versions)
	if files, ok := c.Files.Value(); ok {
		inferred.LegacyFiles = slices.Clone(files.LegacyFiles)
		inferred.Variables = profile.InferredVariables(files.EnvKeys)
	}
	if !resolution.Found {
		return inferred, true
	}
	return resolution.Filled(inferred), true
}

// CanDetect reports whether the plugin may find installs that are already on
// the machine (`vfox link --detect`): unless its files say that it cannot.
func (c *SdkDetailController) CanDetect() bool {
	files, ok := c.Files.Value()
	return !ok || files.Detect.Offered()
}

// rememberedSearch is the choice that the memory holds for the plugin, when
// the profile offers it, in the form that the controller keeps: empty for the
// default. The caller holds the lock, or has not published the controller yet.
func (c *SdkDetailController) rememberedSearch() string {
	search := c.resolution.Profile.Search
	if c.memory == nil || search == nil {
		return ""
	}
	choice, ok := search.Choice(c.memory.RememberedSearch(c.Name))
	if !ok || choice.Value == search.Default {
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
	search := c.resolution.Profile.Search
	if search == nil {
		return ""
	}
	if c.search == "" {
		return search.Default
	}
	return c.search
}

// SearchArguments returns what follows the plugin name in the `vfox search`
// of the chosen search: nothing for the default.
func (c *SdkDetailController) SearchArguments() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolution.Profile.Search.Arguments(c.search)
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
	search := c.resolution.Profile.Search
	if search == nil {
		c.mu.Unlock()
		return "", fmt.Errorf("%s takes no search argument", c.Name)
	}
	choice, ok := search.Choice(value)
	if !ok {
		c.mu.Unlock()
		return "", fmt.Errorf("%s does not list %q", c.Name, value)
	}
	stored := choice.Value
	if stored == search.Default {
		stored = ""
	}
	changed := stored != c.search
	c.search, c.chosen = stored, true
	if changed {
		// Nothing of the earlier list stays, so that it can be freed.
		c.selection = nil
		c.listed = listedFacets{}
	}
	c.mu.Unlock()
	if changed {
		c.Versions.Reset()
	}
	return stored, nil
}

// Groups returns the filters that suit the versions that are listed, with the
// values that some of them have.
func (c *SdkDetailController) Groups() []profile.Group {
	versions, _ := c.Versions.Value()
	if len(versions) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.listedLocked(versions).groups
}

// facets are the facets of the profile, or, when it has none, those that the
// versions give.
func (c *SdkDetailController) facets(versions []core.AvailableVersion) []profile.Facet {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(versions) == 0 {
		return c.resolution.Profile.Facets
	}
	return c.listedLocked(versions).facets
}

// listedLocked works out the facets and the groups of a list, once for each: the
// interface asks for them on every frame, and they only change with the list and
// with the profile (see applyProfile). The caller holds the lock.
func (c *SdkDetailController) listedLocked(versions []core.AvailableVersion) listedFacets {
	if c.listed.first != &versions[0] || c.listed.count != len(versions) {
		facets := profile.EffectiveFacets(c.resolution.Profile, versions)
		c.listed = listedFacets{
			first: &versions[0], count: len(versions), facets: facets,
			groups: profile.Profile{Facets: facets}.Groups(versions),
		}
	}
	return c.listed
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
	c.mu.Unlock()
	versions, _ := c.Versions.Value()
	needle := strings.ToLower(strings.TrimSpace(query))
	// The same facets as the groups that were chosen from.
	var filtering profile.Profile
	if !selection.Empty() {
		filtering.Facets = c.facets(versions)
	}
	var result []core.AvailableVersion
	for _, version := range versions {
		if matchesFilter(version, filter, installed) && matchesQuery(version, needle) && filtering.Matches(selection, version) {
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
	for _, stop := range c.stops {
		stop()
	}
	c.Info.Dispose()
	c.Versions.Dispose()
	c.Aliases.Dispose()
	c.Files.Dispose()
	c.Notifier.Dispose()
}

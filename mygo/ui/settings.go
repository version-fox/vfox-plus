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

package ui

import (
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// seedColors are the accent colours to choose from.
var seedColors = []uint32{
	settings.DefaultSeedColor, 0xFFD9473B, 0xFFC2407A, 0xFF7E57C2, 0xFF3F6FD8, 0xFF0B8A9A, 0xFF3E8E41, 0xFF8D7B1E,
}

// The keys of the vfox configuration that the settings page edits.
const (
	keyProxyEnable     = "proxy.enable"
	keyProxyURL        = "proxy.url"
	keyRegistry        = "registry.address"
	keyStorage         = "storage.sdkPath"
	keyLegacyEnable    = "legacyVersionFile.enable"
	keyLegacyStrategy  = "legacyVersionFile.strategy"
	keyCache           = "cache.availableHookDuration"
	keyHTTPTimeout     = "plugin.http.timeout"
	keyDownloadTimeout = "plugin.http.downloadTimeout"
	keyGitignore       = "gitignore.enable"
)

var knownConfigKeys = map[string]bool{
	keyProxyEnable: true, keyProxyURL: true, keyRegistry: true, keyStorage: true, keyLegacyEnable: true,
	keyLegacyStrategy: true, keyCache: true, keyHTTPTimeout: true, keyDownloadTimeout: true, keyGitignore: true,
}

// settingsPage holds the preferences of the application and the configuration of
// vfox.
func (a *App) settingsPage(k kit.Ctx) {
	// The version and the vfox configuration below need vfox; probing starts
	// here when the user opens Settings, off the drawing goroutine.
	if a.model.Vfox.Availability() == state.VfoxUnknown {
		go a.model.Vfox.EnsureChecked()
	}
	kit.Pane(k, func() {
		if !a.pages.configRequested && a.model.Vfox.Availability() == state.VfoxReady {
			a.pages.configRequested = true
			a.model.VfoxConfig.Load()
		}
		mui.Scroll(k.Context).Grow(1).Children(func() {
			kit.PageHeader(k, a.T("navSettings"), "", nil)
			kit.ContentWidth(k, 840, func() {
				mui.Column(k.Context).Padding(0, gapXL, gapXXL, gapXL).Children(func() {
					a.appearanceSection(k)
					a.remoteSection(k)
					a.vfoxSection(k)
					a.configSection(k)
					a.aboutSection(k)
				})
			})
		})
	}).Grow(1)
}

func (a *App) updateSettings(change func(s *settings.AppSettings)) {
	a.model.Settings.Update(func(s settings.AppSettings) settings.AppSettings {
		change(&s)
		return s
	})
}

// icon is the leading icon of an item of a list.
func icon(k kit.Ctx, name string) func(mui.Color) {
	return func(foreground mui.Color) { kit.Icon(k, name, 24, foreground, 0) }
}

// blank is the leading space of an item that has no icon, which aligns its text
// with the others.
func blank(k kit.Ctx) func(mui.Color) {
	return func(mui.Color) { kit.Gap(k, 24, 0) }
}

func (a *App) appearanceSection(k kit.Ctx) {
	current := a.model.Settings.Settings()
	items := []kit.ListItemSpec{
		{
			ID: "theme", Title: a.T("settingsTheme"), Leading: icon(k, "darkMode"), Wrap: a.wideSettingsWrap(k),
			Trailing: func() {
				kit.ButtonGroup(k, kit.ButtonGroupSpec{
					ID: "theme-group", Selected: string(current.Theme),
					OnSelect: func(value string) {
						a.updateSettings(func(s *settings.AppSettings) { s.Theme = settings.ThemePreference(value) })
					},
					Segments: []kit.Segment{
						{Value: string(settings.ThemeSystem), Label: a.T("themeSystem"), Icon: "systemMode"},
						{Value: string(settings.ThemeLight), Label: a.T("themeLight"), Icon: "lightMode"},
						{Value: string(settings.ThemeDark), Label: a.T("themeDark"), Icon: "darkMode"},
					},
				})
			},
		},
		{
			ID: "system-color", Title: a.T("settingsSystemColor"), Leading: icon(k, "palette"),
			Trailing: func() {
				kit.Switch(k, kit.SwitchSpec{
					ID: "system-color-switch", On: current.UseSystemColor, Label: a.T("settingsSystemColor"),
					OnChange: func(on bool) { a.updateSettings(func(s *settings.AppSettings) { s.UseSystemColor = on }) },
				})
			},
			OnClick: func() {
				a.updateSettings(func(s *settings.AppSettings) { s.UseSystemColor = !s.UseSystemColor })
			},
		},
		{
			ID: "color", Title: a.T("settingsColor"), Leading: blank(k),
			SubtitleWidget: func(mui.Color) {
				mui.Row(k.Context).Wrap().Gap(gapS).Margin(gapS, 0, 0, 0).Children(func() {
					for i, seed := range seedColors {
						kit.ColorSwatch(k, kit.SwatchSpec{
							ID: "swatch" + itoa(i), Seed: seed, Label: a.T("settingsColorOption", "index", i+1),
							Selected: !current.UseSystemColor && current.SeedColor == seed,
							OnClick: func() {
								a.updateSettings(func(s *settings.AppSettings) { s.SeedColor, s.UseSystemColor = seed, false })
							},
						})
					}
				})
			},
		},
		{
			ID: "language", Title: a.T("settingsLanguage"), Leading: icon(k, "language"), Wrap: a.wideSettingsWrap(k),
			Trailing: func() {
				kit.ButtonGroup(k, kit.ButtonGroupSpec{
					ID: "language-group", Selected: string(current.Language),
					OnSelect: func(value string) {
						a.updateSettings(func(s *settings.AppSettings) { s.Language = settings.LanguagePreference(value) })
					},
					Segments: []kit.Segment{
						{Value: string(settings.LanguageSystem), Label: a.T("languageSystem")},
						// Language names are always shown in their own language.
						{Value: string(settings.LanguageEnglish), Label: "English"},
						{Value: string(settings.LanguageSimplifiedChinese), Label: "简体中文"},
					},
				})
			},
		},
	}
	kit.GroupHeader(k, a.T("settingsAppearance"), nil)
	kit.ListGroup(k, items...)
}

// wideSettingsWrap tells whether the buttons at the end of the items of the
// settings move under their text: the items are as wide as the page less the
// space around them, at most 840 dp.
func (a *App) wideSettingsWrap(k kit.Ctx) bool {
	return min(a.pageWidth(k)-2*gapXL, 840) < wrapWidth
}

// environmentInitial is the text the environment field starts with.
func (a *App) environmentInitial() {
	if !a.pages.environmentReady {
		a.pages.environmentReady = true
		a.pages.environmentText = settings.FormatEnvironmentLines(a.model.Settings.Settings().ExtraEnvironment)
	}
}

// remoteSection switches remote access to this desktop: browsers on this
// machine pair with it through an embedded loopback server. Off by default;
// flipping it takes effect on the next start.
func (a *App) remoteSection(k kit.Ctx) {
	current := a.model.Settings.Settings()
	items := []kit.ListItemSpec{
		{
			ID: "remote-access", Title: a.T("settingsRemoteAccess"), Subtitle: a.T("settingsRemoteAccessHelp"),
			Leading: icon(k, "proxy"), Wrap: a.wideSettingsWrap(k),
			Trailing: func() {
				kit.Switch(k, kit.SwitchSpec{
					ID: "remote-access-switch", On: current.RemoteAccess, Label: a.T("settingsRemoteAccess"),
					OnChange: func(on bool) { a.updateSettings(func(s *settings.AppSettings) { s.RemoteAccess = on }) },
				})
			},
			OnClick: func() {
				a.updateSettings(func(s *settings.AppSettings) { s.RemoteAccess = !s.RemoteAccess })
			},
		},
	}
	kit.GroupHeader(k, a.T("settingsRemote"), nil)
	kit.ListGroup(k, items...)
}

func (a *App) vfoxSection(k kit.Ctx) {
	cs := k.Colors()
	controller := a.model.Vfox
	executable, hasExecutable := controller.Executable()
	if !hasExecutable {
		executable = "vfox"
	}
	source := a.T("settingsExecutableMissing")
	if hasExecutable {
		source = a.T("settingsExecutableDetected")
		if controller.IsCustomExecutable() {
			source = a.T("settingsExecutableCustom")
		}
	}
	parts := []string{executable, source}
	if version, ok := controller.Version(); ok {
		parts = append(parts, a.T("settingsVfoxVersion", "version", version))
	}
	a.environmentInitial()
	items := []kit.ListItemSpec{
		{
			ID: "executable", Title: a.T("settingsExecutable"), Subtitle: strings.Join(parts, " · "), Wrap: a.wideSettingsWrap(k),
			Leading: icon(k, "terminal"),
			Trailing: func() {
				mui.Row(k.Context).Gap(gapXS).Children(func() {
					if controller.Availability() == state.VfoxChecking {
						kit.LoadingIndicator(k, kit.LoadingSpec{ID: "checking", Size: 36})
					} else if controller.IsCustomExecutable() {
						kit.IconButton(k, kit.IconButtonSpec{
							ID: "detect", Icon: "refresh", Tooltip: a.T("settingsExecutableDetect"),
							OnClick: func() { a.updateSettings(func(s *settings.AppSettings) { s.VfoxExecutable = "" }) },
						})
					}
					kit.Button(k, kit.ButtonSpec{ID: "browse", Kind: kit.TonalButton, Label: a.T("actionBrowse"), OnClick: a.locateVfox})
				})
			},
		},
		{
			ID: "environment", Title: a.T("settingsEnvironment"), Leading: icon(k, "environment"),
			SubtitleWidget: func(mui.Color) {
				mui.Column(k.Context).Margin(gapS, 0, 0, 0).Children(func() {
					kit.Label(k, a.T("settingsEnvironmentHelp"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
					kit.TextField(k, kit.TextFieldSpec{
						ID: "environment", Text: &a.pages.environmentText, Lines: 3, MaxLines: 8, Monospace: true,
						Hint: "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node", Error: a.pages.environmentError,
					}).Margin(gapM, 0, 0, 0)
					kit.Button(k, kit.ButtonSpec{
						ID: "save-environment", Kind: kit.FilledButton, Label: a.T("actionSave"),
						OnClick: func() { a.saveEnvironment(a.pages.environmentText) },
					}).AlignSelf(mui.End).Margin(gapS, 0, 0, 0)
				})
			},
		},
	}
	kit.GroupHeader(k, a.T("settingsVfox"), nil)
	kit.ListGroup(k, items...)
}

// saveEnvironment stores the environment variables that vfox runs with.
func (a *App) saveEnvironment(text string) {
	parsed := settings.ParseEnvironmentLines(text)
	if len(parsed.InvalidLines) > 0 {
		lines := make([]string, len(parsed.InvalidLines))
		for i, line := range parsed.InvalidLines {
			lines[i] = itoa(line)
		}
		a.pages.environmentError = a.T("settingsEnvironmentInvalid", "count", len(lines), "lines", strings.Join(lines, ", "))
		return
	}
	a.pages.environmentError = ""
	a.updateSettings(func(s *settings.AppSettings) { s.ExtraEnvironment = parsed.Variables })
	a.notify(a.T("settingsSaved"))
}

// configSection edits the configuration of vfox, key by key.
func (a *App) configSection(k kit.Ctx) {
	cs := k.Colors()
	config := a.model.VfoxConfig
	kit.GroupHeader(k, a.T("settingsVfoxConfig"), nil)
	kit.Label(k, a.T("settingsVfoxConfigHelp"), kit.Type(kit.BodySmall), cs.OnSurfaceVariant).Margin(0, 0, gapM, gapL)
	switch {
	case a.model.Vfox.Availability() != state.VfoxReady:
	case !config.HasValue():
		mui.Column(k.Context).Height(240).Children(func() {
			a.loadingOrError(k, a.T("settingsVfoxConfigError"), config.Error(), func() { config.Load() })
		})
	default:
		var other []kit.ListItemSpec
		entries, _ := config.Value()
		for _, entry := range entries {
			if !knownConfigKeys[entry.Key] {
				other = append(other, a.configText(k, entry.Key, entry.Key, "", "", "", false))
			}
		}
		kit.ListGroup(k,
			a.configSwitch(k, keyProxyEnable, "proxy", a.T("configProxyEnable"), ""),
			a.configText(k, keyProxyURL, a.T("configProxyUrl"), "", "", "http://127.0.0.1:7890", false),
			a.configText(k, keyRegistry, a.T("configRegistry"), "registry", "", a.T("configRegistryHint"), false),
			a.configText(k, keyStorage, a.T("configStorage"), "storage", "", a.T("configStorageHint"), true),
		).Margin(0, 0, gapL, 0)
		kit.ListGroup(k,
			a.configSwitch(k, keyLegacyEnable, "legacyFiles", a.T("configLegacyEnable"), a.T("configLegacyEnableHelp")),
			a.configChoice(k, keyLegacyStrategy, a.T("configLegacyStrategy"), []kit.Choice{
				{Value: "specified", Label: a.T("strategySpecified")},
				{Value: "latest_installed", Label: a.T("strategyLatestInstalled")},
				{Value: "latest_available", Label: a.T("strategyLatestAvailable")},
			}),
			a.configSwitch(k, keyGitignore, "gitignore", a.T("configGitignore"), a.T("configGitignoreHelp")),
		).Margin(0, 0, gapL, 0)
		kit.ListGroup(k,
			a.configText(k, keyCache, a.T("configCache"), "cache", a.T("configCacheHelp"), "12h", false),
			a.configText(k, keyHTTPTimeout, a.T("configHttpTimeout"), "timeout", a.T("configTimeoutHelp"), "30s", false),
			a.configText(k, keyDownloadTimeout, a.T("configDownloadTimeout"), "", a.T("configTimeoutHelp"), "30m", false),
		)
		if len(other) > 0 {
			kit.GroupHeader(k, a.T("configOther"), nil)
			kit.ListGroup(k, other...)
		}
	}
}

// saveConfig stores a vfox configuration value in the background, and says
// whether that worked.
func (a *App) saveConfig(save func() error) {
	go func() {
		err := save()
		a.post(func() {
			if err != nil {
				a.notify(describeError(err))
				return
			}
			a.notify(a.T("settingsSaved"))
		})
	}()
}

func (a *App) configSwitch(k kit.Ctx, key, iconName, title, subtitle string) kit.ListItemSpec {
	config := a.model.VfoxConfig
	value, _ := config.ValueFor(key)
	on := value == "true"
	saving := config.IsSaving(key)
	toggle := func() {
		a.saveConfig(func() error { return config.Set(key, boolText(!on)) })
	}
	item := kit.ListItemSpec{
		ID: key, Title: title, Subtitle: subtitle,
		Leading: func(foreground mui.Color) {
			if iconName == "" {
				kit.Gap(k, 24, 0)
				return
			}
			kit.Icon(k, iconName, 24, foreground, 0)
		},
		Trailing: func() {
			kit.Switch(k, kit.SwitchSpec{ID: key + ".switch", On: on, Label: title, Disabled: saving, OnChange: func(bool) { toggle() }})
		},
	}
	if !saving {
		item.OnClick = toggle
	}
	return item
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (a *App) configChoice(k kit.Ctx, key, title string, choices []kit.Choice) kit.ListItemSpec {
	config := a.model.VfoxConfig
	value, _ := config.ValueFor(key)
	return kit.ListItemSpec{
		ID: key, Title: title, Wrap: a.wideSettingsWrap(k),
		Leading: blank(k),
		Trailing: func() {
			kit.Dropdown(k, kit.DropdownSpec{
				ID: key + ".dropdown", Value: value, Choices: choices, Disabled: config.IsSaving(key),
				OnSelect: func(selection string) {
					a.saveConfig(func() error { return config.Set(key, selection) })
				},
			})
		},
	}
}

// configField is what a text field of the configuration remembers: the value of
// the key that the field shows, the one that was saved last, and the text.
type configField struct {
	stored, saved, text string
	known               bool
}

// configFor is what the text field of the key remembers.
func (a *App) configFor(key string) *configField {
	if a.pages.configFields == nil {
		a.pages.configFields = make(map[string]*configField)
	}
	field, ok := a.pages.configFields[key]
	if !ok {
		field = &configField{}
		a.pages.configFields[key] = field
	}
	return field
}

// configText is a text field for a value of the vfox configuration, saved when the
// user presses Enter or leaves the field; an empty field resets the key.
func (a *App) configText(k kit.Ctx, key, title, iconName, help, hint string, pickDirectory bool) kit.ListItemSpec {
	config := a.model.VfoxConfig
	memory := a.configFor(key)
	stored, _ := config.ValueFor(key)
	if !memory.known || stored != memory.stored {
		memory.stored, memory.saved, memory.text, memory.known = stored, stored, stored, true
	}
	save := func(text string) {
		value := strings.TrimSpace(text)
		// Submitting and then leaving the field must not save twice. If vfox
		// rejects the value, the reload shows the stored value again.
		if value == memory.saved {
			return
		}
		memory.saved = value
		a.saveConfig(func() error {
			if value == "" {
				return config.Reset(key)
			}
			return config.Set(key, value)
		})
	}
	return kit.ListItemSpec{
		ID: key, Title: title, Subtitle: help, Wrap: a.wideSettingsWrap(k),
		Leading: func(foreground mui.Color) {
			if iconName == "" {
				kit.Gap(k, 24, 0)
				return
			}
			kit.Icon(k, iconName, 24, foreground, 0)
		},
		Trailing: func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: key + ".field", Hint: hint, Text: &memory.text, Dense: true, Disabled: config.IsSaving(key),
				OnSubmit: save, OnBlur: save,
				Trailing: func() {
					switch {
					case pickDirectory:
						kit.IconButton(k, kit.IconButtonSpec{ID: key + ".browse", Icon: "folder", Tooltip: a.T("actionBrowse"), Size: 32, OnClick: func() {
							a.choose(chooseOptions{Title: title, Initial: stored, Directory: true}, func(path string) {
								memory.text = path
								save(path)
							})
						}})
					case stored != "":
						kit.IconButton(k, kit.IconButtonSpec{ID: key + ".reset", Icon: "close", Tooltip: a.T("actionReset"), Size: 32, OnClick: func() {
							memory.text = ""
							save("")
						}})
					}
				},
			}).Width(300)
		},
	}
}

func (a *App) aboutSection(k kit.Ctx) {
	cs := k.Colors()
	open := func(link string) func() { return func() { a.openURL(link) } }
	external := func() { kit.Icon(k, "openInNew", 24, cs.OnSurfaceVariant, 0) }
	kit.GroupHeader(k, a.T("settingsAbout"), nil)
	kit.ListGroup(k,
		kit.ListItemSpec{
			ID: "about", Title: a.T("appTitle"), Subtitle: a.T("aboutVersion", "version", Version) + " · " + a.T("aboutLicense"),
			Leading: func(mui.Color) { kit.Avatar(k, kit.AvatarSpec{Name: "vfox+", Icon: "sdks", Size: 40}) },
		},
		kit.ListItemSpec{
			ID: "source", Title: a.T("aboutSource"), Subtitle: linkRepository, Leading: icon(k, "sourceCode"), Trailing: external,
			OnClick: open(linkRepository),
		},
		kit.ListItemSpec{
			ID: "docs", Title: a.T("aboutDocs"), Subtitle: linkVfoxDocs, Leading: icon(k, "docs"), Trailing: external,
			OnClick: open(linkVfoxDocs),
		},
		kit.ListItemSpec{
			ID: "licenses", Title: a.T("aboutLicenses"), Leading: icon(k, "license"),
			Trailing: func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) }, OnClick: a.showLicenses,
		},
	)
}

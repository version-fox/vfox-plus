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
	"image/color"
	"strings"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/platform"
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

// settingsPage holds the preferences of the application and the configuration
// of vfox.
func (a *App) settingsPage() kit.Widget {
	return kit.Pane(func(c kit.Ctx) layout.Dimensions {
		if !a.pages.configRequested && a.model.Vfox.Availability() == state.VfoxReady {
			a.pages.configRequested = true
			a.model.VfoxConfig.Load()
		}
		header := kit.PageHeader(a.T("navSettings"), "", nil)
		content := kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: gapXXL}, kit.ContentWidth(840, column(0,
			a.appearanceSection(), a.vfoxSection(), a.configSection(), a.aboutSection(),
		)))
		return scrollPage("settings", header, content)(c)
	})
}

func (a *App) updateSettings(change func(s *settings.AppSettings)) {
	a.model.Settings.Update(func(s settings.AppSettings) settings.AppSettings {
		change(&s)
		return s
	})
}

func (a *App) appearanceSection() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		current := a.model.Settings.Settings()
		icon := func(name string) func(color.NRGBA) kit.Widget {
			return func(col color.NRGBA) kit.Widget { return kit.Icon(name, 24, col, 0) }
		}
		blank := func(color.NRGBA) kit.Widget { return kit.Space(24, 0) }
		var swatches []kit.Widget
		for i, seed := range seedColors {
			swatches = append(swatches, kit.ColorSwatch{
				ID: "swatch" + itoa(i), Seed: seed, Label: a.T("settingsColorOption", "index", i+1),
				Selected: !current.UseSystemColor && current.SeedColor == seed,
				OnClick: func() {
					a.updateSettings(func(s *settings.AppSettings) { s.SeedColor, s.UseSystemColor = seed, false })
				},
			}.W())
		}
		items := []kit.ListItem{
			{
				ID: "theme", Title: a.T("settingsTheme"), Leading: icon("darkMode"), WrapTrailing: true,
				Trailing: kit.ButtonGroup{
					ID: "theme-group", Selected: string(current.Theme),
					OnSelect: func(value string) {
						a.updateSettings(func(s *settings.AppSettings) { s.Theme = settings.ThemePreference(value) })
					},
					Segments: []kit.Segment{
						{Value: string(settings.ThemeSystem), Label: a.T("themeSystem"), Icon: "systemMode"},
						{Value: string(settings.ThemeLight), Label: a.T("themeLight"), Icon: "lightMode"},
						{Value: string(settings.ThemeDark), Label: a.T("themeDark"), Icon: "darkMode"},
					},
				}.W(),
			},
			{
				ID: "system-color", Title: a.T("settingsSystemColor"), Leading: icon("palette"),
				Trailing: kit.Switch{
					ID: "system-color-switch", On: current.UseSystemColor, Label: a.T("settingsSystemColor"),
					OnChange: func(on bool) { a.updateSettings(func(s *settings.AppSettings) { s.UseSystemColor = on }) },
				}.W(),
				OnClick: func() {
					a.updateSettings(func(s *settings.AppSettings) { s.UseSystemColor = !s.UseSystemColor })
				},
			},
			{
				ID: "color", Title: a.T("settingsColor"), Leading: blank,
				SubtitleWidget: func(color.NRGBA) kit.Widget {
					return kit.Pad(kit.Inset{Top: gapS}, kit.Wrap{Spacing: gapS, RunSpacing: gapS}.Layout(swatches...))
				},
			},
			{
				ID: "language", Title: a.T("settingsLanguage"), Leading: icon("language"), WrapTrailing: true,
				Trailing: kit.ButtonGroup{
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
				}.W(),
			},
		}
		return column(0, kit.GroupHeader(a.T("settingsAppearance"), nil), kit.ListGroup(items...))(c)
	}
}

// environmentInitial is the text the environment field starts with.
func (a *App) environmentInitial() string {
	if !a.pages.environmentReady {
		a.pages.environmentReady = true
		a.pages.environmentText = settings.FormatEnvironmentLines(a.model.Settings.Settings().ExtraEnvironment)
	}
	return a.pages.environmentText
}

func (a *App) vfoxSection() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
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
		var trailing []kit.FlexChild
		if controller.Availability() == state.VfoxChecking {
			trailing = append(trailing, kit.Rigid(kit.LoadingIndicator{ID: "checking", Size: 36}.W()))
		} else if controller.IsCustomExecutable() {
			trailing = append(trailing, kit.Rigid(kit.IconButton{
				ID: "detect", Icon: "refresh", Tooltip: a.T("settingsExecutableDetect"),
				OnClick: func() { a.updateSettings(func(s *settings.AppSettings) { s.VfoxExecutable = "" }) },
			}.W()))
		}
		trailing = append(trailing, kit.Rigid(kit.Space(gapXS, 0)), kit.Rigid(kit.Button{
			ID: "browse", Kind: kit.TonalButton, Label: a.T("actionBrowse"), OnClick: a.locateVfox,
		}.W()))

		environment := kit.Pad(kit.Inset{Top: gapS}, column(0,
			kit.Label(a.T("settingsEnvironmentHelp"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant),
			kit.Space(0, gapM),
			kit.TextField{
				ID: "environment", Value: a.environmentInitial(), Lines: 3, MaxLines: 8, Monospace: true,
				Hint: "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node", Error: a.pages.environmentError,
			}.W(),
			kit.Space(0, gapS),
			kit.Align(layout.E, kit.Button{ID: "save-environment", Kind: kit.FilledButton, Label: a.T("actionSave"), OnClick: func() {
				a.saveEnvironment(kit.FieldText(c, "environment"))
			}}.W()),
		))
		items := []kit.ListItem{
			{
				ID: "executable", Title: a.T("settingsExecutable"), Subtitle: strings.Join(parts, " · "), WrapTrailing: true,
				Leading:  func(col color.NRGBA) kit.Widget { return kit.Icon("terminal", 24, col, 0) },
				Trailing: kit.Row(kit.FlexOpts{Cross: kit.CrossCenter, Min: true}, trailing...),
			},
			{
				ID: "environment", Title: a.T("settingsEnvironment"),
				Leading:        func(col color.NRGBA) kit.Widget { return kit.Icon("environment", 24, col, 0) },
				SubtitleWidget: func(color.NRGBA) kit.Widget { return environment },
			},
		}
		return column(0, kit.GroupHeader(a.T("settingsVfox"), nil), kit.ListGroup(items...))(c)
	}
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
func (a *App) configSection() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		config := a.model.VfoxConfig
		var body []kit.Widget
		switch {
		case a.model.Vfox.Availability() != state.VfoxReady:
		case !config.HasValue():
			body = append(body, a.loadingOrError(a.T("settingsVfoxConfigError"), config.Error(), func() { config.Load() }))
		default:
			var other []kit.ListItem
			entries, _ := config.Value()
			for _, entry := range entries {
				if !knownConfigKeys[entry.Key] {
					other = append(other, a.configText(entry.Key, entry.Key, "", "", "", false))
				}
			}
			body = append(body,
				kit.ListGroup(
					a.configSwitch(keyProxyEnable, "proxy", a.T("configProxyEnable"), ""),
					a.configText(keyProxyURL, a.T("configProxyUrl"), "", "", "http://127.0.0.1:7890", false),
					a.configText(keyRegistry, a.T("configRegistry"), "registry", "", a.T("configRegistryHint"), false),
					a.configText(keyStorage, a.T("configStorage"), "storage", "", a.T("configStorageHint"), true),
				),
				kit.Space(0, gapL),
				kit.ListGroup(
					a.configSwitch(keyLegacyEnable, "legacyFiles", a.T("configLegacyEnable"), a.T("configLegacyEnableHelp")),
					a.configChoice(keyLegacyStrategy, a.T("configLegacyStrategy"), []kit.Choice{
						{Value: "specified", Label: a.T("strategySpecified")},
						{Value: "latest_installed", Label: a.T("strategyLatestInstalled")},
						{Value: "latest_available", Label: a.T("strategyLatestAvailable")},
					}),
					a.configSwitch(keyGitignore, "gitignore", a.T("configGitignore"), a.T("configGitignoreHelp")),
				),
				kit.Space(0, gapL),
				kit.ListGroup(
					a.configText(keyCache, a.T("configCache"), "cache", a.T("configCacheHelp"), "12h", false),
					a.configText(keyHTTPTimeout, a.T("configHttpTimeout"), "timeout", a.T("configTimeoutHelp"), "30s", false),
					a.configText(keyDownloadTimeout, a.T("configDownloadTimeout"), "", a.T("configTimeoutHelp"), "30m", false),
				),
			)
			if len(other) > 0 {
				body = append(body, kit.GroupHeader(a.T("configOther"), nil), kit.ListGroup(other...))
			}
		}
		help := kit.Pad(kit.Inset{Left: gapL, Bottom: gapM},
			kit.Label(a.T("settingsVfoxConfigHelp"), kit.Type(kit.BodySmall), cs.OnSurfaceVariant))
		parts := append([]kit.Widget{kit.GroupHeader(a.T("settingsVfoxConfig"), nil), help}, body...)
		return column(0, parts...)(c)
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

func (a *App) configSwitch(key, icon, title, subtitle string) kit.ListItem {
	config := a.model.VfoxConfig
	value, _ := config.ValueFor(key)
	on := value == "true"
	saving := config.IsSaving(key)
	toggle := func() {
		a.saveConfig(func() error { return config.Set(key, boolText(!on)) })
	}
	item := kit.ListItem{
		ID: key, Title: title, Subtitle: subtitle,
		Leading: func(col color.NRGBA) kit.Widget {
			if icon == "" {
				return kit.Space(24, 0)
			}
			return kit.Icon(icon, 24, col, 0)
		},
	}
	item.Trailing = kit.Switch{ID: key + ".switch", On: on, Label: title, Disabled: saving, OnChange: func(bool) { toggle() }}.W()
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

func (a *App) configChoice(key, title string, choices []kit.Choice) kit.ListItem {
	config := a.model.VfoxConfig
	value, _ := config.ValueFor(key)
	return kit.ListItem{
		ID: key, Title: title, WrapTrailing: true,
		Leading: func(color.NRGBA) kit.Widget { return kit.Space(24, 0) },
		Trailing: kit.Dropdown{
			ID: key + ".dropdown", Value: value, Choices: choices, Disabled: config.IsSaving(key),
			OnSelect: func(selection string) {
				a.saveConfig(func() error { return config.Set(key, selection) })
			},
		}.W(),
	}
}

// configField is what a text field of the configuration remembers.
type configField struct {
	stored string
	saved  string
	known  bool
}

// configText is a text field for a value of the vfox configuration, saved when
// the user presses Enter or leaves the field; an empty field resets the key.
func (a *App) configText(key, title, icon, help, hint string, pickDirectory bool) kit.ListItem {
	config := a.model.VfoxConfig
	return kit.ListItem{
		ID: key, Title: title, Subtitle: help, WrapTrailing: true,
		Leading: func(col color.NRGBA) kit.Widget {
			if icon == "" {
				return kit.Space(24, 0)
			}
			return kit.Icon(icon, 24, col, 0)
		},
		Trailing: func(c kit.Ctx) layout.Dimensions {
			memory := kit.UseID[configField](c, key+".memory")
			stored, _ := config.ValueFor(key)
			if !memory.known || stored != memory.stored {
				memory.stored, memory.saved, memory.known = stored, stored, true
			}
			save := func(text string) {
				value := strings.TrimSpace(text)
				// Submitting and then leaving the field must not save twice. If
				// vfox rejects the value, the reload shows the stored value
				// again.
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
			var trailing kit.Widget
			if pickDirectory {
				trailing = kit.IconButton{ID: key + ".browse", Icon: "folder", Tooltip: a.T("actionBrowse"), OnClick: func() {
					a.choose(platform.ChooseOptions{Title: title, Initial: stored, Directory: true}, func(path string) {
						kit.SetFieldText(c, key+".field", path)
						save(path)
					})
				}}.W()
			} else if stored != "" {
				trailing = kit.IconButton{ID: key + ".reset", Icon: "close", Tooltip: a.T("actionReset"), OnClick: func() {
					kit.SetFieldText(c, key+".field", "")
					save("")
				}}.W()
			}
			return kit.Sized(300, 0, kit.TextField{
				ID: key + ".field", Hint: hint, Value: stored, Dense: true, Disabled: config.IsSaving(key), Trailing: trailing,
				OnSubmit: save, OnBlur: save,
			}.W())(c)
		},
	}
}

func (a *App) aboutSection() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		open := func(link string) func() { return func() { a.openURL(link) } }
		external := kit.Icon("openInNew", 24, cs.OnSurfaceVariant, 0)
		icon := func(name string) func(color.NRGBA) kit.Widget {
			return func(col color.NRGBA) kit.Widget { return kit.Icon(name, 24, col, 0) }
		}
		return column(0, kit.GroupHeader(a.T("settingsAbout"), nil), kit.ListGroup(
			kit.ListItem{
				ID: "about", Title: a.T("appTitle"), Subtitle: a.T("aboutVersion", "version", Version) + " · " + a.T("aboutLicense"),
				Leading: func(color.NRGBA) kit.Widget {
					return kit.Avatar{Name: "vfox+", Icon: "sdks", Size: 40}.W("about-avatar")
				},
			},
			kit.ListItem{
				ID: "source", Title: a.T("aboutSource"), Subtitle: linkRepository, Leading: icon("sourceCode"), Trailing: external,
				OnClick: open(linkRepository),
			},
			kit.ListItem{
				ID: "docs", Title: a.T("aboutDocs"), Subtitle: linkVfoxDocs, Leading: icon("docs"), Trailing: external,
				OnClick: open(linkVfoxDocs),
			},
			kit.ListItem{
				ID: "licenses", Title: a.T("aboutLicenses"), Leading: icon("license"),
				Trailing: kit.Icon("chevronRight", 24, cs.OnSurfaceVariant, 0), OnClick: a.showLicenses,
			},
		))(c)
	}
}

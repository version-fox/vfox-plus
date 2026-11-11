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
	"errors"
	"fmt"
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// What a plugin offers beyond the list of its versions (docs/en/dev/ui-spec.md):
// the choice of what a search lists, the filter chips that suit its notes, the
// installation by name, and the options tab with its downloads, version files,
// names and notes. The knowledge is in the profiles of core/profile.

// text is a text of a profile in the language of the application.
func (a *App) text(t profile.Text) string { return t.For(a.l10n.Load().LanguageTag()) }

// partsLine is what a version name says besides the version: the distribution
// and the flavours, which the profile of the plugin takes out of the name.
func (a *App) partsLine(parts profile.Parts) string {
	var words []string
	if parts.Source.EN != "" {
		words = append(words, a.text(parts.Source))
	}
	return strings.Join(append(words, parts.Flavors...), " · ")
}

// availableActions is the row above the available versions: the choice of what a
// search lists, where the plugin has one, and at its end the menu with the other
// ways to add an SDK that the list does not offer: by its name, or one that is
// already on the machine.
func (a *App) availableActions(k kit.Ctx, d *detailState) {
	controller := d.controller
	known, hasKnown := controller.Profile()
	menu := []kit.MenuItem{
		{Label: a.T("availableInstallByName"), Icon: "terminal", OnClick: func() { a.askInstallName(d.name, known, hasKnown, "") }},
		{Label: a.T("linkDetect"), Icon: "link", OnClick: func() {
			command, err := a.model.Cli.LinkDetect(d.name, true)
			if err != nil {
				a.notify(describeError(err))
				return
			}
			a.run(command, nil)
		}},
		{Label: a.T("linkManual"), Icon: "folder", OnClick: func() { a.askLinkPath(d.name) }},
	}
	mui.Row(k.Context).Gap(gapM).AlignItems(mui.Center).Margin(0, gapL, gapS, gapXL).Children(func() {
		if hasKnown && known.Search != nil {
			search := known.Search
			current, _ := search.Choice(controller.Search())
			kit.Button(k, kit.ButtonSpec{
				ID: "choose-search", Kind: kit.TonalButton, Icon: "registry",
				Label:   a.T("availableChoiceButton", "what", a.text(search.Label), "choice", a.text(current.Name)),
				OnClick: func() { a.chooseSearch(d) }, Lines: 2,
			}).Shrink(1)
		}
		mui.Spacer(k.Context)
		kit.IconButton(k, kit.IconButtonSpec{
			ID: "more-ways", Icon: "add", Tooltip: a.T("availableMoreWays"), Menu: menu, MenuAlignEnd: true,
		})
	})
}

// facetRow shows the filter chips of the listed versions: the channels of Flutter,
// LTS and JavaFX, the architectures. A chip narrows the list; the chips of one
// group are alternatives, and the groups narrow each other.
func (a *App) facetRow(k kit.Ctx, controller *state.SdkDetailController) {
	groups := controller.Groups()
	if len(groups) == 0 {
		return
	}
	cs := k.Colors()
	selection := controller.Selection()
	mui.Row(k.Context).Wrap().Gap(gapS).GapY(gapS).AlignItems(mui.Center).Margin(0, gapXL, gapM, gapXL).Children(func() {
		for _, group := range groups {
			kit.Label(k, a.text(group.Label), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant).Margin(0, gapXS, 0, 0)
			for _, chip := range group.Chips {
				id, value := group.ID, chip.Value
				kit.FilterChip(k, kit.ChipSpec{
					ID: "facet:" + id + ":" + value, Label: value, Selected: selection.Has(id, value),
					OnClick: func() { controller.ToggleFacet(id, value) },
				})
			}
		}
		if !selection.Empty() {
			kit.Button(k, kit.ButtonSpec{
				ID: "clear-facets", Kind: kit.TextButton, Label: a.T("availableClearFilters"), OnClick: controller.ClearFacets,
			})
		}
	})
}

// chooseSearch lets the user choose what a search lists, such as the
// distribution of Java, from a list that a field filters; the popular choices
// come first.
func (a *App) chooseSearch(d *detailState) {
	controller := d.controller
	known, hasKnown := controller.Profile()
	search := known.Search
	if !hasKnown || search == nil {
		return
	}
	var query string
	var list mui.ListState
	a.env.ShowDialog("choose-search", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		var visible []profile.Choice
		for _, choice := range search.Choices {
			if choice.Matches(query) {
				visible = append(visible, choice)
			}
		}
		list.Key = func(i int) any { return "choice:" + visible[i].Value }
		current := controller.Search()
		return kit.DialogSpec{
			Icon: "registry", Title: a.text(search.Label), MaxWidth: 560, Tall: true,
			Content: func() {
				kit.TextField(k, kit.TextFieldSpec{
					ID: "query", Hint: a.T("searchChooseHint"), LeadingIcon: "search", Dense: true, AutoFocus: true, Text: &query,
				}).Margin(0, 0, gapM, 0)
				if len(visible) == 0 {
					kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: a.T("searchChooseEmpty")}).Grow(1)
					return
				}
				mui.List(k.Context, &list, len(visible), func(i int) {
					choice := visible[i]
					selected := choice.Value == current
					details := []string{a.text(choice.Maker), choice.Value}
					if a.text(choice.Maker) == "" {
						details = details[1:]
					}
					kit.ListItem(k, kit.ListItemSpec{
						ID: "choice:" + choice.Value, Title: a.text(choice.Name), Subtitle: strings.Join(details, " · "),
						Position: kit.PositionOf(i, len(visible)), Selected: selected,
						Leading: func(foreground mui.Color) {
							if selected {
								kit.Icon(k, "check", 24, foreground, 0)
							} else {
								kit.Gap(k, 24, 0)
							}
						},
						Trailing: func() {
							if choice.Value == search.Default {
								kit.Tag(k, kit.TagSpec{Label: a.T("searchChoiceDefault")})
							}
						},
						OnClick: func() {
							closeDialog()
							a.applySearch(d, choice.Value)
						},
					}).Margin(0, 0, hairline, 0)
				}).Grow(1)
			},
			Actions: []func(){func() {
				kit.Button(k, kit.ButtonSpec{ID: "close", Kind: kit.TextButton, Label: a.T("actionClose"), OnClick: closeDialog})
			}},
		}
	})
}

// applySearch makes the choice the search of the plugin, and lists it. The choice
// is kept in the settings, so it is the search of the plugin from now on.
func (a *App) applySearch(d *detailState, value string) {
	if err := d.controller.ChooseSearch(value); err != nil {
		a.notify(describeError(err))
		return
	}
	d.available = mui.ListState{}
	d.controller.Versions.EnsureLoaded()
	a.invalidate()
}

// askInstallName installs a version by the name that the user types, which vfox
// resolves while it installs: `24` for the newest Node.js 24, `stable` for the
// release of a Flutter channel, `21-tem` for the newest Temurin 21. The names
// that the profile suggests are chips that fill the field.
func (a *App) askInstallName(name string, p profile.Profile, hasProfile bool, initial string) {
	text, problem := initial, ""
	useAfter := false
	guide := ""
	if hasProfile {
		guide = a.text(p.Guide)
	}
	a.env.ShowDialog("install-by-name", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		cs := k.Colors()
		submit := func(use bool) func() {
			return func() {
				spec := strings.TrimSpace(text)
				switch {
				case spec == "":
					problem = a.T("installNameRequired")
				case strings.HasPrefix(spec, "-") || strings.ContainsAny(spec, " \t\r\n@"):
					problem = a.T("installNameInvalid")
				default:
					var command vfox.VfoxCommand
					var err error
					if use {
						command, err = a.model.Cli.InstallWithOptions(name, spec, vfox.InstallOptions{Use: true})
					} else {
						command, err = a.model.Cli.Install(name, spec)
					}
					if err != nil {
						problem = a.T("installNameInvalid")
						return
					}
					closeDialog()
					a.model.Run(command)
					a.navigate(destTasks)
				}
			}
		}
		return kit.DialogSpec{
			Icon: "terminal", Title: a.T("installNameTitle", "name", name), MaxWidth: 560,
			Content: func() {
				kit.TextField(k, kit.TextFieldSpec{
					ID: "name", Label: a.T("installNameField"), Helper: guide, Error: problem,
					AutoFocus: true, Text: &text, Monospace: true, OnSubmit: func(string) { submit(useAfter)() },
					OnChange: func(string) { problem = "" },
				})
				if hasProfile && len(p.Shorthands) > 0 {
					kit.Label(k, a.T("installNameExamples"), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant).Margin(gapL, 0, gapS, 0)
					meaning := ""
					mui.Row(k.Context).Wrap().Gap(gapS).GapY(gapS).Children(func() {
						for _, shorthand := range p.Shorthands {
							chosen := strings.TrimSpace(text) == shorthand.Name
							if chosen {
								meaning = a.text(shorthand.Meaning)
							}
							kit.FilterChip(k, kit.ChipSpec{
								ID: "shorthand:" + shorthand.Name, Label: shorthand.Name, Selected: chosen,
								OnClick: func() { text, problem = shorthand.Name, "" },
							})
						}
					})
					if meaning != "" {
						kit.Label(k, meaning, kit.Type(kit.BodyMedium), cs.OnSurface).Margin(gapM, 0, 0, 0)
					}
				}
				a.switchRow(k, "use-after", a.T("installUseAfter"), useAfter, func(on bool) { useAfter = on })
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Icon: "install", Label: a.T("versionInstall"), OnClick: submit(useAfter)})
				},
			},
		}
	})
}

// switchRow is a switch of a dialog with its label in front of it: the label is the text of the row, and
// names the switch for assistive technology.
func (a *App) switchRow(k kit.Ctx, id, label string, on bool, change func(bool)) {
	cs := k.Colors()
	mui.Row(k.Context).AlignItems(mui.Center).Gap(gapM).Margin(gapL, 0, 0, 0).Children(func() {
		kit.Label(k, label, kit.Type(kit.BodyLarge), cs.OnSurface).Grow(1)
		kit.Switch(k, kit.SwitchSpec{ID: id, On: on, Label: label, OnChange: change})
	})
}

// askLinkPath links the SDK already on the machine at a directory the user
// names, as the version they name.
func (a *App) askLinkPath(name string) {
	version, path, force := "", "", false
	versionProblem, pathProblem := "", ""
	a.env.ShowDialog("link-manual", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		submit := func() {
			versionProblem, pathProblem = "", ""
			trimmedVersion, trimmedPath := strings.TrimSpace(version), strings.TrimSpace(path)
			if trimmedVersion == "" || strings.HasPrefix(trimmedVersion, "-") || strings.ContainsAny(trimmedVersion, " \t\r\n@") {
				versionProblem = a.T("installNameInvalid")
			}
			if trimmedPath == "" {
				pathProblem = a.T("installNameRequired")
			}
			if versionProblem != "" || pathProblem != "" {
				return
			}
			command, err := a.model.Cli.Link(name, trimmedVersion, trimmedPath, force)
			if err != nil {
				versionProblem = a.T("installNameInvalid")
				return
			}
			closeDialog()
			a.run(command, nil)
			a.navigate(destTasks)
		}
		return kit.DialogSpec{
			Icon: "link", Title: a.T("linkPathTitle", "name", name), MaxWidth: 560,
			Content: func() {
				kit.TextField(k, kit.TextFieldSpec{
					ID: "version", Label: a.T("linkVersionField"), Hint: a.T("linkVersionHint"), Error: versionProblem,
					AutoFocus: true, Text: &version, Monospace: true, OnSubmit: func(string) { submit() },
					OnChange: func(string) { versionProblem = "" },
				})
				kit.TextField(k, kit.TextFieldSpec{
					ID: "path", Label: a.T("linkPathField"), Helper: a.T("linkPathHelp"), Error: pathProblem,
					Text: &path, Monospace: true, OnSubmit: func(string) { submit() },
					OnChange: func(string) { pathProblem = "" },
					Trailing: func() {
						kit.IconButton(k, kit.IconButtonSpec{ID: "browse", Icon: "folder", Tooltip: a.T("actionBrowse"), Size: 32, OnClick: func() {
							a.choose(chooseOptions{Title: a.T("linkPathField"), Initial: strings.TrimSpace(path), Directory: true}, func(picked string) {
								path = picked
								pathProblem = ""
							})
						}})
					},
				}).Margin(gapM, 0, 0, 0)
				a.switchRow(k, "force", a.T("linkForce"), force, func(on bool) { force = on })
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Icon: "link", Label: a.T("linkConfirm"), OnClick: submit})
				},
			},
		}
	})
}

// optionsBody is the tab of what is particular to a plugin: how its version names
// work, the environment variables that it reads, the version files that vfox reads
// through it, the names that it takes besides the listed versions, notes and
// documentation.
func (a *App) optionsBody(k kit.Ctx, d *detailState) detailBody {
	p, _ := d.controller.EffectiveProfile()
	return single("options", func() {
		mui.Column(k.Context).Padding(0, gapXL, gapXXL, gapXL).Children(func() {
			a.variablesSection(k, p)
			a.legacySection(k, p)
			a.aliasSection(k, d)
			a.tipsSection(k, p)
			a.docsSection(k, p)
		})
	})
}

// longText is the content of a list item that is a few sentences long.
func (a *App) longText(k kit.Ctx, text string, style kit.TextStyle) func(mui.Color) {
	return func(foreground mui.Color) { kit.Label(k, text, style, foreground) }
}

func (a *App) variablesSection(k kit.Ctx, p profile.Profile) {
	if len(p.Variables) == 0 {
		return
	}
	kit.GroupHeader(k, a.T("optionsVariables"), nil)
	var items []kit.ListItemSpec
	for _, variable := range p.Variables {
		value := a.model.PluginVariable(variable.Key)
		subtitle := a.T("optionsVariableUnset")
		if value != "" {
			subtitle = value
			if preset, ok := variable.PresetFor(value); ok {
				subtitle = a.text(preset.Name) + " · " + value
			}
		}
		items = append(items, kit.ListItemSpec{
			ID: "variable:" + variable.Key, Wrap: a.itemWraps(),
			TitleWidget: func(foreground mui.Color) {
				mui.Row(k.Context).Wrap().Gap(gapS).Children(func() {
					kit.Label(k, a.text(variable.Label), kit.Type(kit.BodyLarge), foreground)
					if variable.Advanced {
						kit.Tag(k, kit.TagSpec{Label: a.T("optionsVariableAdvanced")})
					}
				})
			},
			Name: a.text(variable.Label),
			SubtitleWidget: func(secondary mui.Color) {
				mui.Column(k.Context).AlignItems(mui.Start).Children(func() {
					// The name of the variable is one word that cannot be broken: where the room is the least it is cut, and
					// at 200 % it is not larger than at 130 %. The value is a few words or an address, which wrap.
					kit.Line(k, variable.Key, kit.Type(kit.BodyMedium), secondary, kit.ScaleCap(1.3)).Shrink(1)
					kit.Label(k, subtitle, kit.Type(kit.BodyMedium), secondary, kit.Lines(2))
				})
			},
			Trailing: func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "change:" + variable.Key, Kind: kit.TonalButton, Label: a.T("optionsVariableChange"),
					OnClick: func() { a.askVariable(p, variable) },
				})
			},
		})
	}
	kit.ListGroup(k, items...)
}

// askVariable sets an environment variable that the plugin reads: a mirror
// address that is typed or picked from the ones that were checked, or one of the
// values of a choice.
func (a *App) askVariable(p profile.Profile, variable profile.Variable) {
	text, problem := a.model.PluginVariable(variable.Key), ""
	a.env.ShowDialog("variable", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		cs := k.Colors()
		submit := func() {
			err := a.model.SetPluginVariable(p, variable.Key, text)
			var invalid *profile.ValueError
			switch {
			case errors.As(err, &invalid) && invalid.Reason == profile.ReasonChoice:
				problem = a.T("variableInvalidChoice")
			case err != nil:
				problem = a.T("variableInvalidAddress")
			default:
				closeDialog()
				a.notify(a.T("settingsSaved"))
			}
		}
		return kit.DialogSpec{
			Icon: "install", Title: a.text(variable.Label), MaxWidth: 560,
			Content: func() {
				kit.Label(k, a.text(variable.Help), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				kit.Label(k, variable.Key, kit.Type(kit.LabelMedium), cs.OnSurfaceVariant, kit.Monospace()).Margin(gapS, 0, 0, 0)
				if variable.Kind == profile.VariableURL {
					kit.TextField(k, kit.TextFieldSpec{
						ID: "value", Label: a.T("variableAddress"), Hint: variable.Example, Error: problem, AutoFocus: true, Text: &text,
						Monospace: true, OnSubmit: func(string) { submit() }, OnChange: func(string) { problem = "" },
					}).Margin(gapL, 0, 0, 0)
				} else if problem != "" {
					kit.Label(k, problem, kit.Type(kit.BodySmall), cs.Error).Margin(gapS, 0, 0, 0)
				}
				kit.Label(k, a.T("variablePresets"), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant).Margin(gapL, 0, gapS, 0)
				mui.Row(k.Context).Wrap().Gap(gapS).GapY(gapS).Children(func() {
					kit.FilterChip(k, kit.ChipSpec{
						ID: "preset:default", Label: a.T("variableDefaultOption"), Selected: strings.TrimSpace(text) == "",
						OnClick: func() { text, problem = "", "" },
					})
					for _, preset := range variable.Presets {
						kit.FilterChip(k, kit.ChipSpec{
							ID: "preset:" + preset.Value, Label: a.text(preset.Name), Selected: strings.TrimSpace(text) == preset.Value,
							OnClick: func() { text, problem = preset.Value, "" },
						})
					}
				})
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Label: a.T("actionSave"), OnClick: submit})
				},
			},
		}
	})
}

func (a *App) legacySection(k kit.Ctx, p profile.Profile) {
	if len(p.LegacyFiles) == 0 {
		return
	}
	a.model.VfoxConfig.EnsureLoaded()
	if !a.model.VfoxConfig.HasValue() {
		return
	}
	kit.GroupHeader(k, a.T("optionsLegacy"), nil)
	enable := a.configSwitch(k, keyLegacyEnable, "", a.T("configLegacyEnable"), "")
	enable.Leading, enable.Wrap = nil, a.itemWraps()
	enable.SubtitleWidget = func(mui.Color) {
		mui.Row(k.Context).Wrap().Gap(gapS).GapY(gapS).Margin(gapS, 0, 0, 0).Children(func() {
			for _, file := range p.LegacyFiles {
				kit.Tag(k, kit.TagSpec{Label: file})
			}
		})
	}
	items := []kit.ListItemSpec{enable}
	if p.LegacyStrategy {
		strategy := a.configChoice(k, keyLegacyStrategy, a.T("configLegacyStrategy"), []kit.Choice{
			{Value: "specified", Label: a.T("strategySpecified")},
			{Value: "latest_installed", Label: a.T("strategyLatestInstalled")},
			{Value: "latest_available", Label: a.T("strategyLatestAvailable")},
		})
		strategy.Leading = nil
		items = append(items, strategy)
	}
	kit.ListGroup(k, items...)
}

func (a *App) aliasSection(k kit.Ctx, d *detailState) {
	kit.GroupHeader(k, a.T("aliasTitle"), nil)
	aliases := d.controller.Aliases
	switch {
	case !aliases.HasValue() && aliases.Error() == nil:
		aliases.EnsureLoaded()
		mui.Column(k.Context).Height(120).Children(func() {
			a.loadingOrError(k, a.T("settingsVfoxConfigError"), nil, func() { aliases.Load() })
		})
		return
	case aliases.Error() != nil:
		mui.Column(k.Context).Height(120).Children(func() {
			a.loadingOrError(k, a.T("settingsVfoxConfigError"), aliases.Error(), func() { aliases.Load() })
		})
		return
	}
	values, _ := aliases.Value()
	var items []kit.ListItemSpec
	for _, alias := range values {
		if alias.Sdk != d.name {
			continue
		}
		items = append(items, kit.ListItemSpec{
			ID: "alias:" + alias.Alias, Title: alias.Alias, Subtitle: alias.Target,
			Trailing: func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "unset:" + alias.Alias, Kind: kit.TextButton, Label: a.T("aliasUnset"), Danger: true,
					OnClick: func() {
						command, err := a.model.Cli.AliasUnset(d.name, alias.Alias)
						if err != nil {
							a.notify(describeError(err))
							return
						}
						a.run(command, nil)
					},
				})
			},
		})
	}
	if len(items) > 0 {
		kit.ListGroup(k, items...)
	}
	mui.Row(k.Context).Margin(gapM, 0, 0, gapL).Children(func() {
		kit.Button(k, kit.ButtonSpec{
			ID: "alias-set", Kind: kit.TonalButton, Icon: "add", Label: a.T("aliasSet"),
			OnClick: func() { a.askAliasSet(d) },
		})
		kit.IconButton(k, kit.IconButtonSpec{
			ID: "alias-refresh", Icon: "refresh", Tooltip: a.T("actionRefresh"),
			OnClick: func() { aliases.Load() },
		})
	})
}

func (a *App) askAliasSet(d *detailState) {
	a.askAliasSetFor(d, "")
}

func (a *App) askAliasSetFor(d *detailState, initialVersion string) {
	alias, version := "", initialVersion
	aliasProblem, versionProblem := "", ""
	if version == "" {
		if sdk, ok := a.model.InstalledSdk(d.name); ok && len(sdk.Versions) > 0 {
			if sdk.HasCurrent {
				version = sdk.Current
			} else {
				version = sdk.Versions[0]
			}
		}
	}
	a.env.ShowDialog("alias-set", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		submit := func() {
			aliasProblem, versionProblem = "", ""
			trimmedAlias, trimmedVersion := strings.TrimSpace(alias), strings.TrimSpace(version)
			if trimmedAlias == "" || strings.HasPrefix(trimmedAlias, "-") || strings.ContainsAny(trimmedAlias, " \t\r\n@") {
				aliasProblem = a.T("installNameInvalid")
			}
			if trimmedVersion == "" || strings.HasPrefix(trimmedVersion, "-") || strings.ContainsAny(trimmedVersion, " \t\r\n@") {
				versionProblem = a.T("installNameInvalid")
			}
			if aliasProblem != "" || versionProblem != "" {
				return
			}
			command, err := a.model.Cli.AliasSet(d.name, trimmedAlias, trimmedVersion)
			if err != nil {
				aliasProblem = a.T("installNameInvalid")
				return
			}
			closeDialog()
			a.run(command, nil)
		}
		return kit.DialogSpec{
			Icon: "alias", Title: a.T("aliasDialogTitle", "name", d.name), MaxWidth: 560,
			Content: func() {
				kit.TextField(k, kit.TextFieldSpec{
					ID: "alias", Label: a.T("aliasField"), Hint: a.T("aliasHint"), Error: aliasProblem,
					AutoFocus: true, Text: &alias, Monospace: true, OnSubmit: func(string) { submit() },
					OnChange: func(string) { aliasProblem = "" },
				})
				kit.TextField(k, kit.TextFieldSpec{
					ID: "version", Label: a.T("linkVersionField"), Hint: a.T("linkVersionHint"), Error: versionProblem,
					Text: &version, Monospace: true, OnSubmit: func(string) { submit() },
					OnChange: func(string) { versionProblem = "" },
				}).Margin(gapM, 0, 0, 0)
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Icon: "add", Label: a.T("aliasConfirm"), OnClick: submit})
				},
			},
		}
	})
}

func (a *App) tipsSection(k kit.Ctx, p profile.Profile) {
	if len(p.Tips) == 0 {
		return
	}
	kit.GroupHeader(k, a.T("optionsTips"), nil)
	var items []kit.ListItemSpec
	for i, tip := range p.Tips {
		items = append(items, kit.ListItemSpec{
			ID:          fmt.Sprintf("tip:%d", i),
			TitleWidget: a.longText(k, a.text(tip), kit.Type(kit.BodyMedium)),
		})
	}
	kit.ListGroup(k, items...)
}

func (a *App) docsSection(k kit.Ctx, p profile.Profile) {
	if len(p.Docs) == 0 {
		return
	}
	cs := k.Colors()
	kit.GroupHeader(k, a.T("optionsDocs"), nil)
	var items []kit.ListItemSpec
	for _, link := range p.Docs {
		address := a.text(link.URL)
		items = append(items, kit.ListItemSpec{
			ID: "doc:" + link.URL.EN, Title: a.text(link.Title),
			Trailing: func() { kit.Icon(k, "openInNew", 24, cs.OnSurfaceVariant, 0) },
			OnClick:  func() { a.openURL(address) },
		})
	}
	kit.ListGroup(k, items...)
}

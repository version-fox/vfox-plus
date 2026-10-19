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
	"regexp"
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// confirmation describes a dialog that asks the user to confirm an action.
type confirmation struct {
	Title, Body, ConfirmLabel, Icon string
	// Destructive draws the confirmation in the colour of errors.
	Destructive bool
	OnConfirm   func()
}

// confirm asks the user to confirm an action.
func (a *App) confirm(id string, question confirmation) {
	a.env.ShowDialog(id, func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon: question.Icon, IconError: question.Destructive, Title: question.Title,
			Content: func() {
				kit.Label(k, question.Body, kit.Type(kit.BodyMedium), k.Colors().OnSurfaceVariant)
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "confirm", Kind: kit.FilledButton, Label: question.ConfirmLabel, Danger: question.Destructive,
						OnClick: func() {
							closeDialog()
							question.OnConfirm()
						},
					})
				},
			},
		}
	})
}

// chooseOptions say what the user chooses with the file chooser of the desktop.
type chooseOptions struct {
	Title string
	// Initial is the directory or the file the chooser starts at.
	Initial string
	// Directory chooses directories instead of files.
	Directory bool
	// FilterLabel and Extensions limit the files that are offered.
	FilterLabel string
	Extensions  []string
}

// choose lets the user choose a file or a directory, and runs onChosen with it
// on the main thread.
func (a *App) choose(options chooseOptions, onChosen func(path string)) {
	go func() {
		path := a.desktop.choose(options)
		if path == "" {
			return
		}
		a.post(func() { onChosen(path) })
	}()
}

func (a *App) home() string { return a.environment["HOME"] }

// browseButton is the button that opens a chooser to fill a field.
func (a *App) browseButton(k kit.Ctx, id string, onClick func()) mui.Element {
	return kit.IconButton(k, kit.IconButtonSpec{
		ID: id, Kind: kit.TonalIconButton, Icon: "folder", Tooltip: a.T("actionBrowse"), OnClick: onClick,
	})
}

// askProjectDirectory asks for the project that a version is used in, and uses it
// there.
func (a *App) askProjectDirectory(name, version string) {
	var problem, path string
	a.env.ShowDialog("project-directory", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		submit := func() {
			directory := strings.TrimSpace(path)
			if directory == "" || !isDirectory(directory) {
				problem = a.T("useProjectInvalid")
				return
			}
			closeDialog()
			a.run(a.model.Cli.Use(name, version, core.UseScopeProject, directory))
		}
		browse := func() {
			a.choose(chooseOptions{Title: a.T("useProjectDirectory"), Initial: a.home(), Directory: true}, func(chosen string) {
				path = chosen
				problem = ""
			})
		}
		return kit.DialogSpec{
			Icon: "projectScope", Title: a.T("useProjectTitle", "name", name, "version", version), MaxWidth: 560,
			Content: func() {
				kit.Label(k, a.T("useProjectBody"), kit.Type(kit.BodyMedium), k.Colors().OnSurfaceVariant)
				mui.Row(k.Context).Gap(gapS).AlignItems(mui.Start).Margin(gapL, 0, 0, 0).Children(func() {
					kit.TextField(k, kit.TextFieldSpec{
						ID: "path", Label: a.T("useProjectDirectory"), Error: problem, AutoFocus: true, Text: &path,
						OnSubmit: func(string) { submit() },
					}).Grow(1).Shrink(1)
					a.browseButton(k, "browse", browse).Margin(gapS, 0, 0, 0)
				})
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Label: a.T("useProjectConfirm"), OnClick: submit})
				},
			},
		}
	})
}

var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// askCustomPlugin asks for the source of a plugin that is not in the registry, and
// adds it.
func (a *App) askCustomPlugin() {
	var sourceProblem, aliasProblem, source, alias string
	a.env.ShowDialog("custom-plugin", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		submit := func() {
			from, as := strings.TrimSpace(source), strings.TrimSpace(alias)
			sourceProblem, aliasProblem = "", ""
			if from == "" {
				sourceProblem = a.T("pluginsCustomRequired")
			}
			if as != "" && !pluginNamePattern.MatchString(as) {
				aliasProblem = a.T("pluginsCustomInvalidName")
			}
			if sourceProblem != "" || aliasProblem != "" {
				return
			}
			closeDialog()
			go func() {
				if _, err := a.model.AddPluginFromSource(from, as); err != nil {
					a.post(func() { a.notify(describeError(err)) })
				}
			}()
		}
		browse := func() {
			a.choose(chooseOptions{
				Title: a.T("pluginsCustomSource"), Initial: a.home(), FilterLabel: a.T("pluginsCustomFileType"),
				Extensions: []string{"zip", "lua"},
			}, func(chosen string) {
				source = chosen
				sourceProblem = ""
			})
		}
		return kit.DialogSpec{
			Icon: "plugins", Title: a.T("pluginsCustomTitle"), MaxWidth: 560,
			Content: func() {
				mui.Row(k.Context).Gap(gapS).AlignItems(mui.Start).Children(func() {
					kit.TextField(k, kit.TextFieldSpec{
						ID: "source", Label: a.T("pluginsCustomSource"), Helper: a.T("pluginsCustomSourceHelp"), Error: sourceProblem,
						AutoFocus: true, Text: &source, OnSubmit: func(string) { submit() },
					}).Grow(1).Shrink(1)
					a.browseButton(k, "browse", browse).Margin(gapS, 0, 0, 0)
				})
				kit.TextField(k, kit.TextFieldSpec{
					ID: "alias", Label: a.T("pluginsCustomAlias"), Helper: a.T("pluginsCustomAliasHelp"), Error: aliasProblem,
					Text: &alias, OnSubmit: func(string) { submit() },
				}).Margin(gapL, 0, 0, 0)
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "confirm", Kind: kit.FilledButton, Label: a.T("pluginsCustomConfirm"), OnClick: submit})
				},
			},
		}
	})
}

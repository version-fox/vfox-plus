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
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/platform"
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
	a.env.ShowDialog(id, func(c kit.Ctx, closeDialog func()) kit.Dialog {
		return kit.Dialog{
			Icon: question.Icon, IconError: question.Destructive, Title: question.Title,
			Content: kit.Label(question.Body, kit.Type(kit.BodyMedium), colors(c).OnSurfaceVariant),
			Actions: []kit.Widget{
				kit.Button{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog}.W(),
				kit.Button{
					ID: "confirm", Kind: kit.FilledButton, Label: question.ConfirmLabel, Danger: question.Destructive,
					OnClick: func() {
						closeDialog()
						question.OnConfirm()
					},
				}.W(),
			},
		}
	})
}

// choose lets the user choose a file or a directory, and runs onChosen with it
// on the goroutine of the window.
func (a *App) choose(options platform.ChooseOptions, onChosen func(path string)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		path, err := platform.Choose(ctx, a.portal.Load(), options)
		if err != nil || path == "" {
			return
		}
		a.post(func() { onChosen(path) })
	}()
}

func (a *App) home() string { return a.environment["HOME"] }

// browseButton is the button that opens a chooser to fill a field.
func (a *App) browseButton(id string, onClick func()) kit.Widget {
	return kit.IconButton{ID: id, Kind: kit.TonalIconButton, Icon: "folder", Tooltip: a.T("actionBrowse"), OnClick: onClick}.W()
}

// askProjectDirectory asks for the project that a version is used in, and uses
// it there.
func (a *App) askProjectDirectory(name, version string) {
	var problem string
	a.env.ShowDialog("project-directory", func(c kit.Ctx, closeDialog func()) kit.Dialog {
		submit := func() {
			path := strings.TrimSpace(kit.FieldText(c, "path"))
			if path == "" || !isDirectory(path) {
				problem = a.T("useProjectInvalid")
				return
			}
			closeDialog()
			a.run(a.model.Cli.Use(name, version, core.UseScopeProject, path))
		}
		field := kit.TextField{ID: "path", Label: a.T("useProjectDirectory"), Error: problem, AutoFocus: true, OnSubmit: func(string) { submit() }}
		browse := func() {
			a.choose(platform.ChooseOptions{Title: a.T("useProjectDirectory"), Initial: a.home(), Directory: true}, func(path string) {
				kit.SetFieldText(c, "path", path)
				problem = ""
			})
		}
		return kit.Dialog{
			Icon: "projectScope", Title: a.T("useProjectTitle", "name", name, "version", version), MaxWidth: 560,
			Content: kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true},
				kit.Rigid(kit.Label(a.T("useProjectBody"), kit.Type(kit.BodyMedium), colors(c).OnSurfaceVariant)),
				kit.Rigid(kit.Space(0, gapL)),
				kit.Rigid(kit.Row(kit.FlexOpts{Gap: gapS, Cross: kit.CrossStart},
					kit.Expanded(1, field.W()), kit.Rigid(kit.Pad(kit.Inset{Top: gapS}, a.browseButton("browse", browse))))),
			),
			Actions: []kit.Widget{
				kit.Button{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog}.W(),
				kit.Button{ID: "confirm", Kind: kit.FilledButton, Label: a.T("useProjectConfirm"), OnClick: submit}.W(),
			},
		}
	})
}

var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// askCustomPlugin asks for the source of a plugin that is not in the registry,
// and adds it.
func (a *App) askCustomPlugin() {
	var sourceProblem, aliasProblem string
	a.env.ShowDialog("custom-plugin", func(c kit.Ctx, closeDialog func()) kit.Dialog {
		submit := func() {
			source := strings.TrimSpace(kit.FieldText(c, "source"))
			alias := strings.TrimSpace(kit.FieldText(c, "alias"))
			sourceProblem, aliasProblem = "", ""
			if source == "" {
				sourceProblem = a.T("pluginsCustomRequired")
			}
			if alias != "" && !pluginNamePattern.MatchString(alias) {
				aliasProblem = a.T("pluginsCustomInvalidName")
			}
			if sourceProblem != "" || aliasProblem != "" {
				return
			}
			closeDialog()
			go func() {
				if _, err := a.model.AddPluginFromSource(source, alias); err != nil {
					a.post(func() { a.notify(describeError(err)) })
				}
			}()
		}
		source := kit.TextField{
			ID: "source", Label: a.T("pluginsCustomSource"), Helper: a.T("pluginsCustomSourceHelp"), Error: sourceProblem,
			AutoFocus: true, OnSubmit: func(string) { submit() },
		}
		alias := kit.TextField{
			ID: "alias", Label: a.T("pluginsCustomAlias"), Helper: a.T("pluginsCustomAliasHelp"), Error: aliasProblem,
			OnSubmit: func(string) { submit() },
		}
		browse := func() {
			a.choose(platform.ChooseOptions{
				Title: a.T("pluginsCustomSource"), Initial: a.home(), FilterLabel: a.T("pluginsCustomFileType"),
				Extensions: []string{"zip", "lua"},
			}, func(path string) {
				kit.SetFieldText(c, "source", path)
				sourceProblem = ""
			})
		}
		return kit.Dialog{
			Icon: "plugins", Title: a.T("pluginsCustomTitle"), MaxWidth: 560,
			Content: kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true},
				kit.Rigid(kit.Row(kit.FlexOpts{Gap: gapS, Cross: kit.CrossStart},
					kit.Expanded(1, source.W()), kit.Rigid(kit.Pad(kit.Inset{Top: gapS}, a.browseButton("browse", browse))))),
				kit.Rigid(kit.Space(0, gapL)),
				kit.Rigid(alias.W()),
			),
			Actions: []kit.Widget{
				kit.Button{ID: "cancel", Kind: kit.TextButton, Label: a.T("actionCancel"), OnClick: closeDialog}.W(),
				kit.Button{ID: "confirm", Kind: kit.FilledButton, Label: a.T("pluginsCustomConfirm"), OnClick: submit}.W(),
			},
		}
	})
}

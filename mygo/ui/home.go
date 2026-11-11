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
	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// activationCommand is the line that makes a shell use the versions that vfox+
// chooses.
const activationCommand = `eval "$(vfox activate bash)"`

// The links that the user interface opens.
const (
	linkVfoxInstall = "https://vfox.dev/guides/quick-start.html"
	linkVfoxDocs    = "https://vfox.dev/"
	linkRepository  = "https://github.com/version-fox/vfox-plus"
)

// homePage is the overview: what is in use, how healthy it is, and the line that a terminal needs.
func (a *App) homePage(k kit.Ctx) {
	kit.Pane(k, func() {
		sdks, ok := a.model.Installed.Value()
		if !ok {
			a.loadingOrError(k, a.T("sdksLoadError"), a.model.Installed.Error(), func() { a.model.Installed.Load() })
			return
		}
		var active []core.InstalledSdk
		for _, sdk := range sdks {
			if sdk.HasCurrent {
				active = append(active, sdk)
			}
		}
		subtitle := a.T("homeSubtitle", "version", orQuestionMark(a.model.Vfox.Version))
		if executable, ok := a.model.Vfox.Executable(); ok {
			subtitle += " · " + executable
		}
		mui.Scroll(k.Context).Grow(1).Children(func() {
			kit.PageHeader(k, a.T("appTitle"), subtitle, nil, func() {
				a.refreshButton(k, "refresh", a.model.Installed.IsLoading(), func() { a.model.Installed.Load() })
			})
			kit.ContentWidth(k, 1080, func() {
				mui.Column(k.Context).Padding(0, gapXL, gapXL, gapXL).Children(func() {
					kit.GroupHeader(k, a.T("homeActiveTitle"), nil)
					a.activeList(k, active)
					a.doctorCard(k)
					a.shellCard(k).Margin(gapXL, 0, 0, 0)
				})
			})
		})
	}).Grow(1)
}

// activeList is the segmented list of the plugins that have a version in use.
func (a *App) activeList(k kit.Ctx, active []core.InstalledSdk) {
	cs := k.Colors()
	if len(active) == 0 {
		kit.ListGroup(k, kit.ListItemSpec{
			ID: "empty", Title: a.T("homeActiveEmpty"), OnClick: func() { a.showSdk("") },
			Leading: func(foreground mui.Color) { kit.Icon(k, "info", 24, foreground, 0) },
		})
		return
	}
	items := make([]kit.ListItemSpec, len(active))
	for i, sdk := range active {
		items[i] = kit.ListItemSpec{
			ID: "sdk:" + sdk.Name, Title: sdk.Name, Subtitle: sdk.Current,
			Leading:  func(mui.Color) { avatar(k, sdk.Name, 40) },
			Trailing: func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) },
			OnClick:  func() { a.showSdk(sdk.Name) },
		}
	}
	kit.ListGroup(k, items...)
}

// doctorCard shows the health of vfox and the installed SDKs.
func (a *App) doctorCard(k kit.Ctx) {
	if a.model.Vfox.Availability() != state.VfoxReady {
		return
	}
	doctor := a.model.Doctor
	if !doctor.HasValue() && doctor.Error() == nil && !doctor.IsLoading() {
		doctor.EnsureLoaded()
	}
	cs := k.Colors()
	kit.GroupHeader(k, a.T("doctorTitle"), nil)
	switch {
	case doctor.IsLoading():
		kit.ListGroup(k, kit.ListItemSpec{
			ID: "doctor-loading", Title: a.T("doctorCheck"), Leading: icon(k, "health"),
			Trailing: func() { kit.LoadingIndicator(k, kit.LoadingSpec{ID: "doctor-loading", Size: 24}) },
		})
	case doctor.Error() != nil:
		kit.ListGroup(k, kit.ListItemSpec{
			ID: "doctor-error", Title: a.T("doctorCheck"), Subtitle: describeError(doctor.Error()),
			Leading: icon(k, "health"),
			Trailing: func() {
				kit.Button(k, kit.ButtonSpec{ID: "doctor-retry", Kind: kit.TonalButton, Label: a.T("doctorCheck"), OnClick: func() { doctor.Load() }})
			},
		})
	default:
		report, _ := doctor.Value()
		if len(report.Findings) == 0 {
			kit.ListGroup(k, kit.ListItemSpec{
				ID: "doctor-ok", Title: a.T("doctorNoProblems"), Leading: icon(k, "health"),
				Trailing: func() {
					kit.Button(k, kit.ButtonSpec{ID: "doctor-check", Kind: kit.TextButton, Label: a.T("doctorCheck"), OnClick: func() { doctor.Load() }})
				},
			})
			return
		}
		var items []kit.ListItemSpec
		for _, finding := range report.Findings {
			finding := finding
			title := finding.Message
			if title == "" {
				title = finding.ID
			}
			subtitle := finding.ID
			if finding.Sdk != "" {
				subtitle = finding.Sdk
				if finding.Version != "" {
					subtitle += "@" + finding.Version
				}
				subtitle += " · " + finding.Severity
			}
			items = append(items, kit.ListItemSpec{
				ID:    "doctor:" + finding.ID + ":" + finding.Sdk + "@" + finding.Version,
				Title: title, Subtitle: subtitle, Wrap: a.itemWraps(),
				Leading: icon(k, "health"),
				Trailing: func() {
					mui.Row(k.Context).Gap(gapXS).Children(func() {
						switch finding.ID {
						case "sdk.no-receipt":
							kit.Button(k, kit.ButtonSpec{
								ID: "adopt:" + finding.Sdk + "@" + finding.Version, Kind: kit.TonalButton, Label: a.T("doctorAdopt"),
								OnClick: func() {
									command, err := a.model.Cli.DoctorAdopt(finding.Sdk, finding.Version)
									if err != nil {
										a.notify(describeError(err))
										return
									}
									a.run(command, nil)
								},
							})
						case "sdk.drift":
							kit.Button(k, kit.ButtonSpec{
								ID: "reinstall:" + finding.Sdk + "@" + finding.Version, Kind: kit.TonalButton, Label: a.T("versionReinstall"),
								OnClick: func() {
									command, err := a.model.Cli.InstallWithOptions(finding.Sdk, finding.Version, vfox.InstallOptions{Reinstall: true})
									if err != nil {
										a.notify(describeError(err))
										return
									}
									a.run(command, nil)
								},
							})
						}
						kit.IconButton(k, kit.IconButtonSpec{
							ID: "doctor-check-one", Icon: "refresh", Tooltip: a.T("actionRefresh"),
							OnClick: func() { doctor.Load() },
						})
					})
				},
			})
			if len(items) >= 5 {
				break
			}
		}
		kit.ListGroup(k, items...)
		kit.Label(k, a.T("doctorProblems", "count", len(report.Findings)), kit.Type(kit.BodySmall), cs.OnSurfaceVariant).Margin(gapS, 0, 0, gapL)
	}
}

func (a *App) shellCard(k kit.Ctx) mui.Element {
	cs := k.Colors()
	return kit.Card(k, kit.CardSpec{
		ID: "shell", Color: cs.SurfaceContainerHighest, Radius: kit.CornerLarge,
		Children: func() {
			mui.Column(k.Context).Padding(gapXL).Children(func() {
				mui.Row(k.Context).Gap(gapM).Children(func() {
					kit.Icon(k, "terminal", 24, cs.Primary, 0)
					kit.Line(k, a.T("homeShellTitle"), kit.Emph(kit.TitleMedium), cs.OnSurface).Grow(1)
				})
				kit.Label(k, a.T("homeShellBody"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant).Margin(gapS, 0, 0, 0)
				kit.Card(k, kit.CardSpec{
					ID: "command", Color: cs.SurfaceContainerLowest, Radius: kit.CornerMedium,
					Children: func() {
						mui.Row(k.Context).Padding(0, gapXS, 0, gapL).Children(func() {
							kit.Label(k, activationCommand, kit.Type(kit.BodyMedium), cs.OnSurface, kit.Selectable(), kit.Monospace()).
								Grow(1).Shrink(1)
							kit.IconButton(k, kit.IconButtonSpec{ID: "copy-activation", Icon: "copy", Tooltip: a.T("actionCopy"), OnClick: func() {
								a.copy(activationCommand)
								a.notify(a.T("actionCopied"))
							}})
						})
					},
				}).Margin(gapM, 0, 0, 0)
			})
		},
	})
}

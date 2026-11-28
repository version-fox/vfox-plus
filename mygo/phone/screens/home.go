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

package screens

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// homeCopiedFor is how long the copy button shows that the command was copied.
const homeCopiedFor = 1200 * time.Millisecond

// homeState is what the home tab keeps between frames: the shell the person chose, and
// the command that was copied and when.
type homeState struct {
	shell   string
	command string
	copied  time.Time
}

// homeTab is the home tab. It shows the version of vfox with a refresh, the SDKs that
// are in use, the health of vfox and of the SDKs, and how a terminal activates vfox.
// Each part reads its resource without waiting; the reads load what they need.
func (r *Root) homeTab(k kit.Ctx) {
	inventory := r.service.Inventory()
	column := ui.Column(k.Context).Fill()
	state := ui.Local(column, "home", func() homeState { return homeState{} })
	column.Children(func() {
		r.homeHeader(k, inventory)
		ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
			ui.Column(k.Context).Padding(8, 16, 24, 16).Gap(16).Children(func() {
				r.homeActive(k, inventory)
				r.homeHealth(k)
				r.homeShellCard(k, inventory, state)
			})
		})
	})
}

// homeHeader is the title of the home tab: the name of the app, and the version of
// vfox under it, with the button that refreshes everything.
func (r *Root) homeHeader(k kit.Ctx, inventory app.Resource[remote.Inventory]) {
	kit.PageHeader(k, i18n.Text("appName"), homeVersionLine(inventory), nil, func() {
		kit.IconButton(k, kit.IconButtonSpec{
			ID:      "refresh",
			Icon:    "refresh",
			Tooltip: i18n.Text("refresh"),
			OnClick: r.service.Refresh,
		})
	})
}

// homeVersionLine returns the subtitle of the home tab: the version of vfox, or the
// message that vfox is missing. A failed inventory leaves the subtitle empty: the
// sections below say why nothing is listed.
func homeVersionLine(inventory app.Resource[remote.Inventory]) string {
	value, loaded := inventory.Value()
	switch {
	case loaded && value.Vfox.Version != "":
		return strings.ReplaceAll(i18n.Text("homeVersion"), "{version}", value.Vfox.Version)
	case loaded:
		return i18n.Text("vfoxMissing")
	case inventory.Err() != nil:
		return ""
	}
	return i18n.Text("loading")
}

// homeSection is the heading of a part of the home tab.
func homeSection(k kit.Ctx, title string) {
	kit.Label(k, title, kit.Type(kit.TitleMedium), k.Colors().OnSurface)
}

// homeActive lists the SDKs in use: each one's current version, which opens the SDK.
// The list is the installed SDKs whose version is set, from the inventory.
func (r *Root) homeActive(k kit.Ctx, inventory app.Resource[remote.Inventory]) {
	cs := k.Colors()
	homeSection(k, i18n.Text("active"))
	value, loaded := inventory.Value()
	switch {
	case !loaded && inventory.Err() != nil:
		kit.Label(k, noticeText(app.KindOf(inventory.Err())), kit.Type(kit.BodyMedium), cs.Error)
		return
	case !loaded:
		kit.Label(k, i18n.Text("loading"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	case value.Installed.Error != "":
		kit.Label(k, value.Installed.Error, kit.Type(kit.BodyMedium), cs.Error)
		return
	}
	var items []kit.ListItemSpec
	for _, sdk := range value.Installed.Value {
		if !sdk.HasCurrent {
			continue
		}
		name, current := sdk.Name, sdk.Current
		items = append(items, kit.ListItemSpec{
			ID:       name,
			Name:     name,
			Leading:  func(ui.Color) { sdkAvatar(k, name, 40) },
			Title:    name,
			Subtitle: current,
			Trailing: func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) },
			OnClick:  func() { r.app.Push(app.Page{Kind: app.PageSDK, Name: name}) },
		})
	}
	if len(items) == 0 {
		kit.Label(k, i18n.Text("activeEmpty"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	}
	kit.ListGroup(k, items...)
}

// homeActivation is the activation of the chosen shell, or the first one, which is the
// shell of the person at the computer.
func homeActivation(activations []remote.Activation, shell string) remote.Activation {
	for _, activation := range activations {
		if activation.Shell == shell {
			return activation
		}
	}
	return activations[0]
}

// homeInstruction says what to do with the command of an activation.
func homeInstruction(activation remote.Activation) string {
	switch activation.Kind {
	case remote.ActivationOnce:
		return strings.ReplaceAll(i18n.Text("shellOnce"), "{shell}", activation.Name)
	case remote.ActivationScript:
		return i18n.Text("shellScript")
	}
	return strings.ReplaceAll(i18n.Text("shellStartup"), "{file}", activation.File)
}

// homeShellCard says how a terminal of the computer comes to use the versions that vfox
// selects: the shells of its system, what to do for the chosen one, and the command with
// the button that copies it. The button shows a check mark for a moment after the copy.
// There is no card for a computer that says nothing about its shells.
func (r *Root) homeShellCard(k kit.Ctx, inventory app.Resource[remote.Inventory], state *homeState) {
	value, loaded := inventory.Value()
	if !loaded || len(value.Vfox.Activations) == 0 {
		return
	}
	cs := k.Colors()
	services := k.Services()
	activations := value.Vfox.Activations
	chosen := homeActivation(activations, state.shell)
	shells := make([]kit.Choice, len(activations))
	for i, activation := range activations {
		shells[i] = kit.Choice{Value: activation.Shell, Label: activation.Name}
	}
	copied := state.command == chosen.Command && time.Since(state.copied) < homeCopiedFor
	icon := "copy"
	if copied {
		icon = "check"
	}
	kit.Card(k, kit.CardSpec{
		ID:      "shell",
		Color:   cs.SurfaceContainerHighest,
		Radius:  kit.CornerLarge,
		Content: cs.OnSurface,
		Name:    i18n.Text("shellTitle"),
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(12).Children(func() {
				ui.Row(k.Context).Gap(12).Children(func() {
					kit.Icon(k, "terminal", 24, cs.Primary, 0)
					kit.Label(k, i18n.Text("shellTitle"), kit.Emph(kit.TitleMedium), cs.OnSurface, kit.Heading()).Grow(1).Shrink(1)
				})
				ui.Row(k.Context).Children(func() {
					kit.Dropdown(k, kit.DropdownSpec{
						ID:       "activationShell",
						Value:    chosen.Shell,
						Choices:  shells,
						Color:    cs.SurfaceContainerLowest,
						OnSelect: func(shell string) { state.shell = shell },
					}).Shrink(1)
				})
				kit.Label(k, homeInstruction(chosen), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Breakable())
				kit.Card(k, kit.CardSpec{
					ID:      "command",
					Color:   cs.SurfaceContainerLowest,
					Radius:  kit.CornerMedium,
					Content: cs.OnSurface,
					Children: func() {
						ui.Row(k.Context).Padding(4, 4, 4, 16).Gap(8).Children(func() {
							kit.Label(k, chosen.Command, kit.Type(kit.BodyMedium), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable()).
								Grow(1).Shrink(1)
							kit.IconButton(k, kit.IconButtonSpec{
								ID:      "copyCommand",
								Icon:    icon,
								Tooltip: i18n.Text("copyCommand"),
								OnClick: func() {
									services.WriteClipboard(chosen.Command)
									state.command, state.copied = chosen.Command, time.Now()
									// The window is drawn again when the mark goes back to a copy.
									time.AfterFunc(homeCopiedFor, services.Invalidate)
								},
							})
						})
					},
				})
				if chosen.Kind == remote.ActivationScript {
					ui.Row(k.Context).Children(func() {
						kit.Button(k, kit.ButtonSpec{
							ID:      "activationScript",
							Kind:    kit.TextButton,
							Label:   i18n.Text("shellGetScript"),
							Icon:    "openInNew",
							OnClick: func() { r.sdkOpenLink(services, chosen.URL) },
						})
					})
				}
			})
		},
	})
}

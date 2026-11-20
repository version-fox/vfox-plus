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

// The choice of what a plugin's search lists, which the web offers as "Choose": a
// button that names the choice in use, and a chooser that searches the choices. The
// choice is kept by the computer, and every interface of it shares it.

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// availChoiceButton is the button above the versions that names the search choice of
// the plugin, and opens the chooser. Its label is the name of the choice that the
// plugin lists now, or its value when the profile does not name it.
func (r *Root) availChoiceButton(k kit.Ctx, name string, profile *remote.Profile) {
	value := availSearchValue(profile)
	label := value
	if choice, ok := availChoiceOf(profile.Search, value); ok {
		label = choice.Name.In(availChinese())
	}
	env := k.Env
	search := *profile.Search
	kit.Button(k, kit.ButtonSpec{
		ID:      "choose",
		Kind:    kit.TonalButton,
		Label:   label,
		Icon:    "registry",
		OnClick: func() { r.availOpenChooser(env, name, search, value) },
	})
}

// availChooser is what the chooser keeps while it is open: the text of its search.
type availChooser struct {
	query string
}

// availOpenChooser opens the chooser of the search choice of the plugin name: its
// choices, which the search narrows, and the one that the person picks. The computer
// keeps the choice, and the versions list follows it once the profile is read again.
func (r *Root) availOpenChooser(env *kit.Env, name string, search remote.ProfileSearch, current string) {
	form := &availChooser{}
	env.ShowDialog("availChooser", func(c kit.Ctx, close func()) kit.DialogSpec {
		query := strings.ToLower(availTrim(form.query))
		var rows []remote.Choice
		for _, choice := range search.Choices {
			if availChoiceMatches(choice, query) {
				rows = append(rows, choice)
			}
		}
		return kit.DialogSpec{
			Icon:  "registry",
			Title: search.Label.In(availChinese()),
			Tall:  true,
			Content: func() {
				// The list and its search take the width of the dialog, which otherwise
				// fits its widest choice, and the hint of the search is cut short.
				ui.Column(c.Context).Width(kit.DialogContentWidth(c, 0)).Grow(1).Shrink(1).Gap(12).Children(func() {
					kit.SearchBar(c, kit.SearchBarSpec{
						ID:           "query",
						Hint:         i18n.Text("chooseHint"),
						Text:         &form.query,
						ClearTooltip: i18n.Text("clearSearch"),
					})
					if len(rows) == 0 {
						kit.StatusMessage(c, kit.StatusSpec{Icon: "notFound", Title: i18n.Text("chooseEmpty")})
						return
					}
					ui.Scroll(c.Context).Grow(1).Shrink(1).Children(func() {
						ui.Column(c.Context).Children(func() {
							items := make([]kit.ListItemSpec, 0, len(rows))
							for _, choice := range rows {
								items = append(items, r.availChoiceItem(c, name, search, current, choice, close))
							}
							kit.ListGroup(c, items...)
						})
					})
				})
			},
			Actions: []func(){func() {
				kit.Button(c, kit.ButtonSpec{ID: "close", Kind: kit.TextButton, Label: i18n.Text("close"), OnClick: close})
			}},
		}
	})
}

// availChoiceItem is the row of one choice in the chooser: its name, its maker and its
// value, a check on the choice in use, and the tag of the default. Choosing it closes
// the chooser and saves the choice on the computer.
func (r *Root) availChoiceItem(c kit.Ctx, name string, search remote.ProfileSearch, current string, choice remote.Choice, close func()) kit.ListItemSpec {
	value := choice.Value
	chosen := value == current
	maker := choice.Maker.In(availChinese())
	item := kit.ListItemSpec{
		ID:       value,
		Title:    choice.Name.In(availChinese()),
		Selected: chosen,
		// The maker, then the value of the choice, which is the identifier that a search names.
		SubtitleWidget: func(foreground ui.Color) {
			ui.Row(c.Context).Wrap().Gap(8).GapY(2).AlignItems(ui.Center).Children(func() {
				if maker != "" {
					kit.Label(c, maker, kit.Type(kit.BodyMedium), foreground, kit.Breakable())
				}
				kit.Label(c, value, kit.Type(kit.BodyMedium), foreground, kit.Monospace(), kit.Breakable())
			})
		},
		Leading: func(foreground ui.Color) {
			if chosen {
				kit.Icon(c, "check", 24, foreground, 0)
			} else {
				kit.Gap(c, 24, 24)
			}
		},
		OnClick: func() {
			close()
			r.Do(func(ctx context.Context) error {
				return r.Service().SetSearch(ctx, name, value)
			}, "")
		},
	}
	if value == search.Default {
		item.Trailing = func() {
			kit.Tag(c, kit.TagSpec{Label: i18n.Text("default")})
		}
	}
	return item
}

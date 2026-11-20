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
	"context"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pluginsAddForm is what the add dialog keeps while it is open: the address of the
// source, and the alias that names the plugin, which is optional.
type pluginsAddForm struct {
	source string
	alias  string
}

// pluginsOpenAdd opens the dialog that adds a plugin from the address of its source,
// with an optional alias. A plugin file cannot be uploaded from the phone, so the
// dialog has the address and nothing else to fill in. The address is checked for
// being empty only, as the web checks it; the computer checks the rest.
func (r *Root) pluginsOpenAdd(env *kit.Env, landed *atomic.Bool) {
	form := &pluginsAddForm{}
	env.ShowDialog("pluginAdd", func(c kit.Ctx, close func()) kit.DialogSpec {
		submit := func() {
			source := strings.TrimSpace(form.source)
			if source == "" {
				return
			}
			alias := strings.TrimSpace(form.alias)
			close()
			r.availThen(landed, func(ctx context.Context) error {
				_, err := r.Service().AddPlugin(ctx, source, alias)
				return err
			})
		}
		return kit.DialogSpec{
			Icon:  "add",
			Title: i18n.Text("addPluginTitle"),
			Content: func() {
				ui.Column(c.Context).Gap(12).Children(func() {
					kit.TextField(c, kit.TextFieldSpec{
						ID:          "source",
						Label:       i18n.Text("sourceUrl"),
						Hint:        i18n.Text("pluginSourceHint"),
						Text:        &form.source,
						LeadingIcon: "link",
						AutoFocus:   true,
						OnSubmit:    func(string) { submit() },
					})
					kit.TextField(c, kit.TextFieldSpec{
						ID:       "alias",
						Label:    i18n.Text("aliasLabel"),
						Text:     &form.alias,
						OnSubmit: func(string) { submit() },
					})
				})
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:       "add",
						Kind:     kit.FilledButton,
						Label:    i18n.Text("add"),
						Icon:     "add",
						Disabled: strings.TrimSpace(form.source) == "",
						OnClick:  submit,
					})
				},
			},
		}
	})
}

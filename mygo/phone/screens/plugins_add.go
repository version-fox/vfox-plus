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
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo"
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

// pluginsOpenAdd opens the dialog that adds a plugin, as the web interface does: from the
// address of its source, with an optional alias, or from a file that the user chooses with
// the picker of the system (a .zip archive or a .lua script), under "Upload & add", with the
// alias that was typed. The address is checked for being empty only, as the web checks it;
// the computer checks the rest.
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
		choose := func() {
			alias := strings.TrimSpace(form.alias)
			close()
			r.pluginsUploadFile(landed, alias)
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
					kit.Label(c, i18n.Text("uploadAdd"), kit.Type(kit.LabelLarge), c.Colors().OnSurfaceVariant)
					kit.Button(c, kit.ButtonSpec{
						ID:      "chooseFile",
						Kind:    kit.TonalButton,
						Label:   i18n.Text("chooseFile"),
						Icon:    "folder",
						OnClick: choose,
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

// pluginsUploadFile has the user choose a plugin file, in the background, and adds the
// plugin from it, as the web does when a file is chosen: the dialog is closed, and the tasks
// tab follows when the computer has the file. A user who goes back from the picker chooses
// nothing, and nothing happens. The copy that the picker made is removed when the upload
// ends, whatever its outcome.
func (r *Root) pluginsUploadFile(landed *atomic.Bool, alias string) {
	r.Do(func(ctx context.Context) error {
		path, err := r.pickFile(ctx)
		if err != nil || path == "" {
			return err
		}
		defer forgetPicked(path)
		if _, err := r.Service().UploadPlugin(ctx, path, alias); err != nil {
			return err
		}
		landed.Store(true)
		r.invalidate()
		return nil
	}, "")
}

// pickPluginFile shows the picker of the system and returns the path of the file that the
// user chose, or "" when the user chose nothing. On a phone the picker gives a copy of the
// file in the cache of the app. No filter is set: the picker of Android filters by the type of
// a file and not by its extension, and it knows no type for a .lua script, so the filter of
// .zip and .lua would hide the scripts. The service checks the name of the file instead.
func pickPluginFile(context.Context) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: i18n.Text("chooseFile")})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// forgetPicked removes the copy that the picker of the system made of a file, in the cache
// of the app, where each pick has a directory of its own. It removes nothing else: on a
// computer the picker names the file of the user, which stays where it is.
func forgetPicked(path string) {
	if cache, err := os.UserCacheDir(); err == nil {
		forgetPickedIn(cache, path)
	}
}

// forgetPickedIn removes the directory of the pick of path, when path is the file of one
// pick in the directory picked of cache: picked/<number>/<name>.
func forgetPickedIn(cache, path string) {
	picked := filepath.Join(cache, "picked")
	rel, err := filepath.Rel(picked, filepath.Clean(path))
	if err != nil {
		return
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 2 || parts[0] == "" || strings.Trim(parts[0], "0123456789") != "" || parts[1] == "" || parts[1] == ".." {
		return
	}
	_ = os.RemoveAll(filepath.Join(picked, parts[0]))
}

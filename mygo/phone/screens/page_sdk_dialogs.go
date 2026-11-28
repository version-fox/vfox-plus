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

// The dialogs of an SDK's page: naming an alias, changing a variable, and the two
// confirmations that the web asks before it uninstalls a version or removes a plugin.

import (
	"context"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// sdkCancelAction is the button that closes a dialog without doing anything.
func sdkCancelAction(c kit.Ctx, close func()) func() {
	return func() {
		kit.Button(c, kit.ButtonSpec{
			ID:      "sdkCancel",
			Kind:    kit.TextButton,
			Label:   i18n.Text("cancel"),
			OnClick: close,
		})
	}
}

// sdkOpenAlias opens the dialog that names an alias of a version of the plugin. The
// version it names is version when one is given; otherwise it is the version in use,
// or else the first installed one, as the web chooses it.
func (r *Root) sdkOpenAlias(name, version string) {
	if version == "" {
		version = r.sdkDefaultVersion(name)
	}
	alias, target := "", version
	r.env.ShowDialog("sdkAlias/"+name, func(c kit.Ctx, close func()) kit.DialogSpec {
		submit := func() {
			a, v := strings.TrimSpace(alias), strings.TrimSpace(target)
			if a == "" || v == "" {
				return
			}
			close()
			r.sdkRun(remote.TaskRequest{Operation: remote.RequestAliasSet, Sdk: name, Alias: a, Version: v})
		}
		return kit.DialogSpec{
			Icon:  "alias",
			Title: i18n.Text("aliasSet"),
			Content: func() {
				ui.Column(c.Context).Gap(16).Children(func() {
					kit.TextField(c, kit.TextFieldSpec{
						ID:        "sdkAliasName",
						Label:     i18n.Text("aliasField"),
						Text:      &alias,
						Monospace: true,
						AutoFocus: true,
						OnSubmit:  func(string) { submit() },
					})
					kit.TextField(c, kit.TextFieldSpec{
						ID:        "sdkAliasVersion",
						Label:     i18n.Text("linkVersionField"),
						Text:      &target,
						Monospace: true,
						OnSubmit:  func(string) { submit() },
					})
				})
			},
			Actions: []func(){
				sdkCancelAction(c, close),
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:      "sdkAliasSave",
						Icon:    "alias",
						Label:   i18n.Text("aliasSet"),
						OnClick: submit,
					})
				},
			},
		}
	})
}

// sdkOpenVariable opens the dialog that sets a variable of the plugin: what the
// variable is for, its key, a field for its address when it is a URL, and the quick
// picks of its presets and of the default. A save that fails is said in the dialog,
// which stays open; a save that works closes it and says so.
func (r *Root) sdkOpenVariable(name string, variable remote.Variable) {
	value := variable.Value
	save := &sdkSave{}
	r.env.ShowDialog("sdkVariable/"+name+"/"+variable.Key, func(c kit.Ctx, close func()) kit.DialogSpec {
		cs := c.Colors()
		chinese := sdkChinese()
		state := save.poll()
		switch {
		case state.saved:
			close()
			c.Env.ShowSnackbar(kit.Snack{Text: i18n.Text("saved")})
		case state.dismissed:
			close()
		}
		submit := func() {
			key, text := variable.Key, strings.TrimSpace(value)
			// The question takes the place of the dialog.
			if variable.NeedsConfirmation(text) {
				close()
				r.sdkChangeVariable(c.Env, name, variable, text)
				return
			}
			if !save.begin() {
				return
			}
			r.Do(func(ctx context.Context) error {
				notice := save.end(r.service.SetVariable(ctx, name, key, text))
				r.invalidate()
				return notice
			}, "")
		}
		return kit.DialogSpec{
			Icon:  "options",
			Title: variable.Label.In(chinese),
			Content: func() {
				ui.Column(c.Context).Gap(12).Children(func() {
					if help := variable.Help.In(chinese); help != "" {
						kit.Label(c, help, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
					}
					kit.Label(c, variable.Key, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Monospace())
					if variable.Kind == remote.VariableURL || variable.Kind == remote.VariableText {
						label := i18n.Text("address")
						if variable.Kind == remote.VariableText {
							label = i18n.Text("variableValue")
						}
						kit.TextField(c, kit.TextFieldSpec{
							ID:        "sdkVariableValue",
							Label:     label,
							Hint:      variable.Example,
							Text:      &value,
							Monospace: true,
							Error:     state.problem,
							OnChange:  func(string) { save.edited() },
							OnSubmit:  func(string) { submit() },
						})
					} else if state.problem != "" {
						kit.Label(c, state.problem, kit.Type(kit.BodyMedium), cs.Error)
					}
					kit.Label(c, i18n.Text("quickPicks"), kit.Type(kit.TitleSmall), cs.OnSurfaceVariant)
					ui.Row(c.Context).Wrap().Gap(8).GapY(8).Children(func() {
						pick := func(id, label, to string) {
							kit.FilterChip(c, kit.ChipSpec{
								ID:       id,
								Label:    label,
								Selected: strings.TrimSpace(value) == to,
								OnClick: func() {
									value = to
									save.edited()
								},
							})
						}
						pick("sdkDefaultValue", i18n.Text("defaultOption"), "")
						for _, preset := range variable.Presets {
							pick("sdkPreset/"+preset.Value, preset.Name.In(chinese), preset.Value)
						}
					})
				})
			},
			Actions: []func(){
				sdkCancelAction(c, close),
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:       "sdkVariableSave",
						Label:    i18n.Text("save"),
						Disabled: state.busy,
						OnClick:  submit,
					})
				},
			},
		}
	})
}

// sdkChangeVariable sets a variable of the plugin name to value, after a
// question for a value from a profile that the application of the computer did
// not ship.
func (r *Root) sdkChangeVariable(env *kit.Env, name string, variable remote.Variable, value string) {
	set := func() {
		r.Do(func(ctx context.Context) error {
			err := r.service.SetVariable(ctx, name, variable.Key, value)
			r.invalidate()
			return err
		}, "saved")
	}
	if !variable.NeedsConfirmation(value) {
		set()
		return
	}
	body := strings.NewReplacer("{key}", variable.Key, "{value}", value).Replace(i18n.Text("variableConfirmBody"))
	confirmDialog(env, "sdkVariableConfirm/"+name+"/"+variable.Key, "warning",
		i18n.Text("variableConfirmTitle"), body, i18n.Text("variableConfirmAction"), false, set)
}

// sdkConfirmUninstall asks before a version is uninstalled, and then queues it.
func (r *Root) sdkConfirmUninstall(name, version string) {
	r.env.ShowDialog("sdkUninstall/"+name+"/"+version, func(c kit.Ctx, close func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon:      "uninstall",
			IconError: true,
			Title:     i18n.Text("uninstall"),
			Content: func() {
				kit.Label(c, version, kit.Type(kit.BodyLarge), c.Colors().OnSurface, kit.Monospace())
			},
			Actions: []func(){
				sdkCancelAction(c, close),
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:     "sdkUninstallConfirm",
						Label:  i18n.Text("uninstall"),
						Danger: true,
						OnClick: func() {
							close()
							r.sdkRun(remote.TaskRequest{Operation: remote.RequestUninstall, Sdk: name, Version: version})
						},
					})
				},
			},
		}
	})
}

// sdkConfirmRemove asks before the plugin is removed, and then queues the removal. The
// page of the plugin closes with it, since the plugin goes away.
func (r *Root) sdkConfirmRemove(name string) {
	r.env.ShowDialog("sdkRemove/"+name, func(c kit.Ctx, close func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon:      "removePlugin",
			IconError: true,
			Title:     i18n.Text("remove"),
			Content: func() {
				kit.Label(c, name, kit.Type(kit.BodyLarge), c.Colors().OnSurface, kit.Monospace())
			},
			Actions: []func(){
				sdkCancelAction(c, close),
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:     "sdkRemoveConfirm",
						Label:  i18n.Text("remove"),
						Danger: true,
						OnClick: func() {
							close()
							r.sdkRun(remote.TaskRequest{Operation: remote.RequestRemove, Sdk: name})
							if top, ok := r.app.Top(); ok && top.Kind == app.PageSDK && top.Name == name {
								r.app.Pop()
							}
						},
					})
				},
			},
		}
	})
}

// sdkSave is what a dialog that saves a value keeps of the save: whether it runs in
// the background, the problem it had, and whether it worked or ended the connection.
// The background writes it, and the frames read it on the main thread (poll).
type sdkSave struct {
	mu        sync.Mutex
	busy      bool
	problem   string
	saved     bool
	dismissed bool
}

// sdkSaveState is a copy of an sdkSave, which one frame reads.
type sdkSaveState struct {
	busy      bool
	problem   string
	saved     bool
	dismissed bool
}

// begin starts a save, and reports false, changing nothing, when one runs already.
func (s *sdkSave) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return false
	}
	s.busy, s.problem = true, ""
	return true
}

// end ends the save that err is the result of. A save that failed is said in the
// dialog, except when the connection to the computer ended: then the dialog goes, and
// the error is returned, to be shown as the service's notice.
func (s *sdkSave) end(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	if err == nil {
		s.saved = true
		return nil
	}
	switch app.KindOf(err) {
	case app.KindCanceled:
		return nil
	case app.KindUnpaired, app.KindCertificateChanged:
		s.dismissed = true
		return err
	}
	s.problem = sdkErrorText(err)
	return nil
}

// edited forgets the problem of the last save, once the value is changed again.
func (s *sdkSave) edited() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.problem = ""
}

// poll returns the state of the save, and forgets the outcome that it reports once:
// a save that worked closes its dialog and says so one time.
func (s *sdkSave) poll() sdkSaveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := sdkSaveState{busy: s.busy, problem: s.problem, saved: s.saved, dismissed: s.dismissed}
	s.saved, s.dismissed = false, false
	return state
}

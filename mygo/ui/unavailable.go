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
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// unavailablePage tells the user that vfox cannot be used and what to do about
// it.
func (a *App) unavailablePage(k kit.Ctx) {
	kit.Pane(k, func() {
		vfoxController := a.model.Vfox
		switch vfoxController.Availability() {
		case state.VfoxNotFound, state.VfoxBroken:
			missing := vfoxController.Availability() == state.VfoxNotFound
			executable, ok := vfoxController.Executable()
			if !ok {
				executable = "vfox"
			}
			message := kit.StatusSpec{
				Icon: "notFound", Shape: shapes.ShapeClover4, Title: a.T("vfoxMissingTitle"), Body: a.T("vfoxMissingBody"),
			}
			if !missing {
				message = kit.StatusSpec{
					Icon: "warning", Shape: shapes.ShapeSoftBurst, IsError: true, Title: a.T("vfoxBrokenTitle"),
					Body: a.T("vfoxBrokenBody", "path", executable),
				}
			}
			message.Actions = []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "guide", Kind: kit.FilledButton, Icon: "openInNew", Label: a.T("vfoxInstallGuide"),
						OnClick: func() { a.openURL(linkVfoxInstall) },
					})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "locate", Kind: kit.TonalButton, Icon: "folder", Label: a.T("vfoxLocate"), OnClick: a.locateVfox,
					})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "retry", Kind: kit.OutlinedButton, Icon: "retry", Label: a.T("actionRetry"),
						OnClick: func() { go a.model.Vfox.Apply(a.model.Settings.Settings()) },
					})
				},
			}
			kit.StatusMessage(k, message)
		default:
			a.loadingView(k, a.T("vfoxCheckingTitle"))
		}
	}).Grow(1)
}

// locateVfox lets the user show where vfox is.
func (a *App) locateVfox() {
	a.choose(chooseOptions{Title: a.T("vfoxLocate"), Initial: "/usr/local/bin"}, func(path string) {
		a.model.Settings.Update(func(s settings.AppSettings) settings.AppSettings {
			s.VfoxExecutable = path
			return s
		})
	})
}

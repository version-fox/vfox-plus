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
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// sessionStrip is the strip above the tabs that tells the state of the connection
// while the tabs are shown: the check of a saved computer that is under way, and a
// computer that does not answer, with the button that tries again. The other states
// need no strip: a phone that must pair again, or whose certificate changed, shows the
// pairing page; a paired phone shows nothing.
func (r *Root) sessionStrip(k kit.Ctx) {
	cs := k.Colors()
	switch r.service.Status().Session {
	case app.Connecting:
		ui.Row(k.Context).Background(cs.SurfaceContainerHigh).Padding(12, 16, 12, 16).Gap(12).Children(func() {
			kit.LoadingIndicator(k, kit.LoadingSpec{ID: "connecting", Size: 24, Label: i18n.Text("sessionConnecting")})
			kit.Label(k, i18n.Text("sessionConnecting"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant).Grow(1).Shrink(1)
		})
	case app.Unreachable:
		ui.Row(k.Context).Background(cs.ErrorContainer).Padding(8, 16, 8, 16).Gap(8).Children(func() {
			kit.Icon(k, "warning", 24, cs.OnErrorContainer, 0)
			kit.Label(k, noticeText(app.KindUnreachable), kit.Type(kit.BodyMedium), cs.OnErrorContainer).Grow(1).Shrink(1)
			kit.Button(k, kit.ButtonSpec{
				ID:      "retry",
				Kind:    kit.TextButton,
				Label:   i18n.Text("retry"),
				Icon:    "retry",
				OnClick: r.service.Refresh,
			})
		})
	}
}

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
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pairPage is the pairing with a new link, above the tabs, from the settings or a deep
// link. The phone stays paired until the new pairing succeeds, and then the home tab
// shows the new computer; pageScreen draws the bar with its way back.
func (r *Root) pairPage(k kit.Ctx) {
	r.pairForm(k)
}

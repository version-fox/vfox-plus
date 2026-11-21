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

import "github.com/egoist/mygo/ui"

// routerHistory is the history of the window's router, which the pages above the
// tabs are shown in.
type routerHistory struct {
	router *ui.Router
}

// Path returns the path of the page shown.
func (h routerHistory) Path() string {
	return h.router.Path()
}

// Push shows the page at path after the one shown.
func (h routerHistory) Push(path string) {
	h.router.Push(path)
}

// Go moves delta entries through the history of the router.
func (h routerHistory) Go(delta int) {
	h.router.Go(delta)
}

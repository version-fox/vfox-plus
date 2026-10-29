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
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/assets"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pageStates is what the pages remember while the user moves between them.
type pageStates struct {
	sdksQuery string
	sdksList  mui.ListState
	detail    *detailState
	// searches is what the user chose to list for the plugins whose search takes an
	// argument, such as the distribution of Java; it outlives the details.
	searches map[string]string
	// detailWidth is the width in dp that the details of a plugin are built in.
	detailWidth float32

	pluginsRequested bool
	pluginsQuery     string
	pluginsFilter    string
	pluginsList      mui.ListState

	tasksList mui.ListState

	configRequested  bool
	configFields     map[string]*configField
	environmentReady bool
	environmentText  string
	environmentError string

	licenses []string
}

// licenseChunk is how many lines of the licenses are one item of the list that
// shows them.
const licenseChunk = 40

// showLicenses opens the licenses of everything the application is built from.
func (a *App) showLicenses() {
	if a.pages.licenses == nil {
		content, err := assets.Files.ReadFile("licenses.txt")
		if err != nil {
			a.notify(describeError(err))
			return
		}
		lines := strings.Split(strings.ReplaceAll(string(content), "\r", ""), "\n")
		for start := 0; start < len(lines); start += licenseChunk {
			end := min(start+licenseChunk, len(lines))
			a.pages.licenses = append(a.pages.licenses, strings.Join(lines[start:end], "\n"))
		}
	}
	chunks := a.pages.licenses
	var list mui.ListState
	a.env.ShowDialog("licenses", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		cs := k.Colors()
		style := kit.Type(kit.BodySmall)
		list.Key = func(i int) any { return i }
		return kit.DialogSpec{
			Title: a.T("aboutLicenses"), MaxWidth: 760, Tall: true,
			Content: func() {
				mui.List(k.Context, &list, len(chunks), func(i int) {
					kit.Label(k, chunks[i], style, cs.OnSurfaceVariant, kit.Selectable(), kit.Monospace())
				}).Grow(1)
			},
			Actions: []func(){func() {
				kit.Button(k, kit.ButtonSpec{ID: "close", Kind: kit.TextButton, Label: a.T("actionClose"), OnClick: closeDialog})
			}},
		}
	})
}

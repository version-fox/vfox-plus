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

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/assets"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

// pageStates is what the pages remember while the user moves between them.
type pageStates struct {
	sdksQuery string
	detail    *detailState

	pluginsRequested bool
	pluginsQuery     string
	pluginsFilter    string

	configRequested  bool
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
	a.env.ShowDialog("licenses", func(c kit.Ctx, closeDialog func()) kit.Dialog {
		cs := colors(c)
		style := kit.Type(kit.BodySmall)
		return kit.Dialog{
			Title: a.T("aboutLicenses"), MaxWidth: 760, Tall: true,
			Content: kit.List("licenses", kit.ListSource{
				Count: len(chunks),
				Item: func(c kit.Ctx, i int) layout.Dimensions {
					return kit.SelectableText("chunk"+itoa(i), chunks[i], style, cs.OnSurfaceVariant, kit.Monospace())(c)
				},
			}),
			Actions: []kit.Widget{kit.Button{ID: "close", Kind: kit.TextButton, Label: a.T("actionClose"), OnClick: closeDialog}.W()},
		}
	})
}

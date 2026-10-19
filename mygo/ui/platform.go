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
	"log"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// loadIcons registers the icon fonts with the toolkit. The interface works
// without its icons (a button that has only an icon keeps its name and its
// tooltip), so a failure is reported and not fatal. Where registering a font means
// writing it to a cache directory, which can be read-only, the fonts get a second
// try in a directory of their own.
func loadIcons(outlined, filled, codepoints []byte) {
	err := kit.LoadIcons(outlined, filled, codepoints)
	if err == nil {
		return
	}
	log.Printf("cannot register the icon fonts: %v", err)
	if !ownFontCache() {
		log.Print("the interface goes without icons")
		return
	}
	if err := kit.LoadIcons(outlined, filled, codepoints); err != nil {
		log.Printf("the interface goes without icons: %v", err)
	}
}

// openURL opens a web address in the browser of the user.
func openURL(address string) {
	go func() {
		if err := mygo.Shell.OpenExternal(address); err != nil {
			log.Printf("cannot open %s: %v", address, err)
		}
	}()
}

// copyText puts text on the clipboard.
func copyText(text string) { mygo.Clipboard.WriteText(text) }

// chooseFile lets the user choose a file or a directory with the file chooser of
// the desktop. It blocks until the user has answered, so it runs on a goroutine
// of its own, and returns an empty path when the user gives up.
func (a *App) chooseFile(options chooseOptions) string {
	var filters []mygo.FileFilter
	if len(options.Extensions) > 0 {
		filters = []mygo.FileFilter{{Name: options.FilterLabel, Extensions: options.Extensions}}
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: a.win, Title: options.Title, DefaultPath: options.Initial, Filters: filters, Directory: options.Directory,
	})
	if err != nil {
		log.Printf("the file chooser failed: %v", err)
		return ""
	}
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

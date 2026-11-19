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
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"strings"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// loadIcons registers the icons with the toolkit. The interface works without
// its icons (a button that has only an icon keeps its name and its tooltip), so
// a failure is reported and not fatal.
func loadIcons(table []byte, files fs.FS) {
	if err := kit.LoadIcons(table, files); err != nil {
		log.Printf("the interface goes without icons: %v", err)
	}
}

// shellOpen opens an address with the default application of the system. It is
// a variable so that the tests can see what would be handed to the system.
var shellOpen = func(address string) error { return mygo.Shell.OpenExternal(address) }

// webAddressOf returns the address without its surrounding space when it is an
// absolute http or https address with a host, the only kind the application
// opens. A plugin names its homepage in its own metadata, so any other scheme
// (file:, javascript:, a protocol handler of the system) is refused.
func webAddressOf(address string) (string, bool) {
	trimmed := strings.TrimSpace(address)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	// url.Parse lowercases the scheme, so HTTPS and https are the same here.
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	return trimmed, true
}

// openWebAddress opens a web address in the browser of the user. Any other
// address is refused, and the system never sees it.
func openWebAddress(address string) error {
	web, ok := webAddressOf(address)
	if !ok {
		return fmt.Errorf("refusing to open %q: only http and https addresses are opened", address)
	}
	return shellOpen(web)
}

// openURL opens a web address in the browser of the user, without waiting for it.
func openURL(address string) {
	go func() {
		if err := openWebAddress(address); err != nil {
			log.Printf("cannot open %q: %v", address, err)
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
	var parent *mygo.Window
	if win := a.window(); win != nil {
		parent, _ = win.(*mygo.Window)
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: parent, Title: options.Title, DefaultPath: options.Initial, Filters: filters, Directory: options.Directory,
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

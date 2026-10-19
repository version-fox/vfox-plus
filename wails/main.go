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

// Package main is vfox+, the Wails implementation: the web front end of
// ../web in a Wails v3 window, with the native side of package host.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/version-fox/vfox-plus/wails/internal/host"
)

// The web front end, built for Wails by build.sh into frontend/dist. The
// repository only holds frontend/README.md.
//
//go:embed all:frontend
var assets embed.FS

// exitCode is the exit code that the front end asked for.
var exitCode atomic.Int32

// desktop implements host.Desktop with the managers of the Wails application.
type desktop struct{}

func (desktop) OpenURL(url string) error {
	return application.Get().Browser.OpenURL(url)
}

func (desktop) PickDirectory(initial string) (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true)
	if initial != "" {
		dialog = dialog.SetDirectory(initial)
	}
	return dialog.PromptForSingleSelection()
}

func (desktop) PickFile(initial, label string, extensions []string) (string, error) {
	dialog := application.Get().Dialog.OpenFile().CanChooseFiles(true)
	if initial != "" {
		dialog = dialog.SetDirectory(initial)
	}
	if len(extensions) > 0 {
		patterns := make([]string, len(extensions))
		for i, extension := range extensions {
			patterns[i] = "*." + extension
		}
		dialog = dialog.AddFilter(label, strings.Join(patterns, ";"))
	}
	return dialog.PromptForSingleSelection()
}

func (desktop) CopyText(text string) error {
	if !application.Get().Clipboard.SetText(text) {
		return fmt.Errorf("the clipboard is not available")
	}
	return nil
}

func (desktop) Emit(event host.Event) {
	application.Get().Event.Emit(host.ProcessEventName, event)
}

func (desktop) Exit(code int) {
	exitCode.Store(int32(code)) //nolint:gosec // Exit codes are small.
	// Quit returns from Run below, which exits with the code. Should the
	// shutdown hang, the process ends anyway.
	time.AfterFunc(5*time.Second, func() { os.Exit(code) })
	go application.Get().Quit()
}

func main() {
	// Accelerated compositing maps Mesa's software OpenGL and its compiler
	// when there is no real GPU; the pages of vfox+ do not need it. See
	// docs/memory.md and the WebKitGTK documentation of this variable.
	if _, set := os.LookupEnv("WEBKIT_DISABLE_COMPOSITING_MODE"); !set {
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
	}
	frontend, err := fs.Sub(assets, "frontend/dist")
	if err == nil {
		_, err = fs.Stat(frontend, "index.html")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "vfox+ was built without its front end; build it with build.sh:", err)
		os.Exit(1)
	}
	service := host.NewService(desktop{})
	app := application.New(application.Options{
		Name:        "vfox+",
		Description: "Material 3 Expressive desktop GUI for the vfox version manager",
		Services:    []application.Service{application.NewService(service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(frontend)},
		// The program name follows the ID, so that the window matches the
		// desktop entry (StartupWMClass and the Wayland app ID).
		Linux:      application.LinuxOptions{ApplicationID: host.AppID},
		OnShutdown: service.Shutdown,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "vfox+",
		Width:     1280,
		Height:    820,
		MinWidth:  360,
		MinHeight: 480,
		URL:       "/",
	})
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(int(exitCode.Load()))
}

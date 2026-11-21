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

// Command phone is the vfox+ app for phones: the native user interface of mygo,
// which pairs the phone with a computer that runs vfox+ and shows its SDKs,
// plugins and tasks.
//
// On Linux, `go run ./phone` opens the app in a window of a phone's size, for
// development; `mygo build -platform android` builds the Android app.
package main

import (
	"context"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/screens"
)

const (
	// windowWidth and windowHeight are a phone in portrait, in dp, as the window
	// opens on a computer. A phone shows the app on its whole screen.
	windowWidth, windowHeight = 480, 900
	minWidth, minHeight       = 360, 640
)

func main() {
	root, err := screens.New()
	if err != nil {
		log.Fatal(err)
	}
	service := root.Service()
	// The connection lives as long as the app. Run ends when the app quits; the
	// system's suspension ends the event stream and resumes it, as the app goes to
	// the background and comes back.
	ctx, stop := context.WithCancel(context.Background())
	mygo.App.OnQuit(stop)
	mygo.Power.OnSuspend(service.Suspend)
	mygo.Power.OnResume(service.Resume)
	// A deep link of the pairing page opens it, and asks the user to confirm the link;
	// it is registered before Run, so that the link the app was started with reaches it.
	mygo.App.OnOpenURL(root.OpenURL)
	go service.Run(ctx)
	mygo.App.WhenReady(func() {
		screens.ApplyUILanguage(mygo.App.PreferredLanguages())
		window := mygo.NewWindow(mygo.WindowOptions{
			Title:     i18n.Text("appName"),
			Width:     windowWidth,
			Height:    windowHeight,
			MinWidth:  minWidth,
			MinHeight: minHeight,
			Content:   ui.View(root.View),
		})
		// The service asks the window for a frame when its state changes, from its
		// own goroutines; Invalidate is safe for that.
		root.Attach(window.Invalidate)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

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

package measureapk

// The pictures of the user guide of the phone app (docs/en/user/android.md), taken in the paired
// session of the emulator job by the steps of a person, so that the guide shows what the app is and
// not what a drawing of it was: the pairing page in the light theme and in the dark one, the dialog
// that asks to trust the computer, the five tabs, the detail of an SDK, the dialog that adds a plugin,
// and the home page in the dark theme.
//
// The computer is the end-to-end build of the desktop application against the fake vfox, which has
// SDKs installed and plugins to offer, so the pictures show what a person who uses the app has. The
// status bar is that of the demo mode of Android, which fixes the clock, shows a full battery and a good
// network, and no notification: the pictures are alike for it. The app is in English, which both guides
// use, and the pictures are of the screen as it comes, 1080 by 2400; `vfxtool frame-phone` gives them the
// shape of a phone display afterwards.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// The names of the pictures that are not of a tab (those are "android-" and the name of the tab). The
// guides link them: docs/en/dev/contributing.md lists them all.
const (
	pairingPicture     = "android-pairing"
	darkPairingPicture = "android-dark-pairing"
	trustPicture       = "android-trust"
	sdkPicture         = "android-sdk"
	addPluginPicture   = "android-add-plugin"
	darkHomePicture    = "android-dark-home"
)

var (
	// sdkRowLabel is the row of the SDK whose detail is pictured: Node.js is installed in the fake inventory,
	// with versions, in every inventory that the pictures are taken with.
	sdkRowLabel = regexp.MustCompile(`^nodejs$`)
	// addPluginLabel is the button that opens the dialog that adds a plugin, and cancelLabel the one that closes it.
	addPluginLabel = regexp.MustCompile(`^(Add from source|从来源添加)$`)
	cancelLabel    = regexp.MustCompile(`^(Cancel|取消)$`)
)

// keyBack is the key of Android that goes back.
const keyBack = "4"

// demoMode puts the status bar of the device in the demo mode of Android, or takes it out. Nothing here
// fails the run: a status bar that is not fixed makes a picture that is not like the others, and no more.
func (s *session) demoMode(on bool) {
	if !on {
		s.demo("exit")
		return
	}
	_, _ = s.adb("shell", "settings", "put", "global", "sysui_demo_allowed", "1")
	for _, command := range [][]string{
		{"enter"},
		{"clock", "-e", "hhmm", "1000"},
		{"battery", "-e", "level", "100", "-e", "plugged", "false"},
		{"network", "-e", "wifi", "show", "-e", "level", "4", "-e", "fully", "true"},
		{"network", "-e", "mobile", "hide"},
		{"notifications", "-e", "visible", "false"},
	} {
		s.demo(command...)
	}
}

// demo sends one command of the demo mode to the system interface.
func (s *session) demo(command ...string) {
	args := append([]string{"shell", "am", "broadcast", "-a", "com.android.systemui.demo", "-e", "command"}, command...)
	_, _ = s.adb(args...)
}

// night sets the theme of the system, which the app follows: dark or light.
func (s *session) night(dark bool) error {
	mode := "no"
	if dark {
		mode = "yes"
	}
	_, err := s.adb("shell", "cmd", "uimode", "night", mode)
	return err
}

// picture saves the screen as the picture name of the user guide, when the run takes the pictures, and
// keeps the dump of the screen beside the screenshots, which says what the picture is of when the labels
// of a screen change.
func (s *session) picture(c config, name string) error {
	if c.pictures == "" {
		return nil
	}
	if err := s.screenshotTo(filepath.Join(c.pictures, name+".png")); err != nil {
		return err
	}
	if shot, err := s.dump(); err == nil {
		s.keepDump(c.shots, "picture-"+name, shot)
	}
	fmt.Fprintf(s.log, "picture %s\n", name)
	return nil
}

// unpairedPictures takes the pictures of the pairing page, in the light theme and in the dark one, of the
// app that has not paired yet, and puts the status bar in the demo mode for all the pictures that follow. The
// theme is light again when it returns.
func (s *session) unpairedPictures(c config) error {
	if err := os.MkdirAll(c.pictures, 0o755); err != nil {
		return err
	}
	s.demoMode(true)
	s.sleep(tabSettle)
	if err := s.picture(c, pairingPicture); err != nil {
		return err
	}
	return s.darkPicture(c, darkPairingPicture, nil)
}

// darkPicture puts the system in the dark theme, takes the picture of what show leaves on the screen (nothing,
// when it is nil), and puts the light theme back.
func (s *session) darkPicture(c config, name string, show func() error) error {
	if err := s.night(true); err != nil {
		return err
	}
	defer func() { _ = s.night(false) }()
	s.sleep(tabSettle)
	if show != nil {
		if err := show(); err != nil {
			return err
		}
		s.sleep(tabSettle)
	}
	return s.picture(c, name)
}

// openTab taps the tab of the bar with the name, and waits for its page.
func (s *session) openTab(c config, name string) error {
	for _, t := range tabs {
		if t.name != name {
			continue
		}
		shot, err := s.tapWhen(t.label, tabTimeout, t.name+" tab")
		if err != nil {
			s.keepDump(c.shots, "tab-"+t.name+"-not-found", shot)
			return err
		}
		s.sleep(tabSettle)
		return nil
	}
	return fmt.Errorf("no tab is named %q", name)
}

// morePictures takes the pictures that the tabs do not give, after the five tabs: the detail of an SDK, the
// dialog that adds a plugin, and the home page in the dark theme.
func (s *session) morePictures(c config) error {
	// The detail of an SDK: the row opens a page above the tabs, and the key of Android goes back from it.
	if err := s.openTab(c, "sdks"); err != nil {
		return err
	}
	shot, err := s.tapWhen(sdkRowLabel, tabTimeout, "row of the SDK")
	if err != nil {
		s.keepDump(c.shots, "sdk-row-not-found", shot)
		return err
	}
	s.sleep(tabSettle)
	if err := s.picture(c, sdkPicture); err != nil {
		return err
	}
	if _, err := s.adb("shell", "input", "keyevent", keyBack); err != nil {
		return err
	}
	s.sleep(tabSettle)

	// The dialog that adds a plugin, which has the source and, under it, the upload of a file.
	if err := s.openTab(c, "plugins"); err != nil {
		return err
	}
	shot, err = s.tapWhen(addPluginLabel, tabTimeout, "button that adds a plugin")
	if err != nil {
		s.keepDump(c.shots, "add-plugin-not-found", shot)
		return err
	}
	s.sleep(tabSettle)
	if err := s.picture(c, addPluginPicture); err != nil {
		return err
	}
	if _, err := s.tapWhen(cancelLabel, tabTimeout, "button that cancels the dialog"); err != nil {
		// The dialog goes with the key of Android as well, and a page that is covered cannot show its tabs.
		_, _ = s.adb("shell", "input", "keyevent", keyBack)
	}
	s.sleep(tabSettle)

	// The home page in the dark theme.
	return s.darkPicture(c, darkHomePicture, func() error { return s.openTab(c, "home") })
}

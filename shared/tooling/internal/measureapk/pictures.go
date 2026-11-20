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
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"time"
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

const (
	// statusBarRows is how many rows of a screenshot, from the top, hold the status bar: 118 px on the
	// 1080 by 2400 of the pictures, and a few more.
	statusBarRows = 120
	// statusIcons is how many icons the demo mode that the tool asks for shows at the right of the status bar: the
	// wifi and the battery. The clock is at the left.
	statusIcons = 2
	// calmPoll is the time between two looks at the status bar while it settles, calmPolls the number of
	// looks in a row that must find it unchanged, and calmTimeout when the tool stops waiting for it.
	calmPoll    = time.Second
	calmPolls   = 4
	calmTimeout = 60 * time.Second
	// pictureAttempts is how many times a picture is taken while the app says that the computer did not answer, and
	// dialogAttempts how many times the tool tries to close a dialog.
	pictureAttempts = 3
	dialogAttempts  = 3
)

// demoMode puts the status bar of the device in the demo mode of Android, or takes it out. Nothing here
// fails the run: a status bar that is not fixed makes a picture that is not like the others, and no more.
func (s *session) demoMode(on bool) {
	if !on {
		s.demo("exit")
		return
	}
	before := s.statusBar()
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
	s.settleStatusBar(before)
}

// demo sends one command of the demo mode to the system interface.
func (s *session) demo(command ...string) {
	args := append([]string{"shell", "am", "broadcast", "-a", "com.android.systemui.demo", "-e", "command"}, command...)
	_, _ = s.adb(args...)
}

// theme sets the theme of the system, which the app follows: dark or light. It returns when the status bar
// has followed.
func (s *session) theme(dark bool) error {
	before := s.statusBar()
	mode := "no"
	if dark {
		mode = "yes"
	}
	if _, err := s.adb("shell", "cmd", "uimode", "night", mode); err != nil {
		return err
	}
	s.settleStatusBar(before)
	return nil
}

// barOf reads the status bar of a screenshot: what its rows are made of, which tells one bar from another whatever the
// page under it does, and how many icons it shows at the right, which tells a bar that is done from one that is half
// drawn. A screenshot that cannot be read as a picture stands for itself, and has all the icons that were asked for.
func barOf(screenshot []byte) (pixels []byte, icons int) {
	img, err := png.Decode(bytes.NewReader(screenshot))
	if err != nil {
		return screenshot, statusIcons
	}
	bounds := img.Bounds()
	rows := min(statusBarRows, bounds.Dy())
	pixels = make([]byte, 0, rows*bounds.Dx()*4)
	for y := bounds.Min.Y; y < bounds.Min.Y+rows; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return pixels, iconsOf(img, rows)
}

// iconsOf counts the icons that can be seen at the right of the status bar: the runs of columns, in the right 45 % of
// the rows of the bar, that hold something that is not the colour of the bar, a gap of the colour of the bar apart. What
// is white on a cream bar, as the icons of the mobile network are, is not seen by a person and is not counted; and on a
// dark bar the icons that are still dark are not either, which is how the bar that has not followed the theme yet is told.
func iconsOf(img image.Image, rows int) int {
	const visible = 40 // the difference of luminance, of 255, that a person sees
	bounds := img.Bounds()
	luminance := func(x, y int) int {
		r, g, b, _ := img.At(x, y).RGBA()
		return int((299*(r>>8) + 587*(g>>8) + 114*(b>>8)) / 1000)
	}
	bar := luminance(bounds.Max.X-3, bounds.Min.Y+4)
	gap := max(bounds.Dx()/108, 3)
	count, last := 0, -gap
	inside := false
	for x := bounds.Min.X + bounds.Dx()*55/100; x < bounds.Max.X; x++ {
		seen := false
		for y := bounds.Min.Y + rows/6; y < bounds.Min.Y+rows && !seen; y++ {
			seen = max(luminance(x, y)-bar, bar-luminance(x, y)) > visible
		}
		switch {
		case seen && !inside:
			inside, count = true, count+1
			last = x
		case seen:
			last = x
		case inside && x-last >= gap:
			inside = false
		}
	}
	return count
}

// statusBar takes a screenshot and returns what its status bar is made of, or nothing when the device gives none.
func (s *session) statusBar() []byte {
	pixels, _ := s.statusBarLook()
	return pixels
}

// statusBarLook takes a screenshot and reads its status bar (barOf).
func (s *session) statusBarLook() (pixels []byte, icons int) {
	screenshot, err := s.adb("exec-out", "screencap", "-p")
	if err != nil {
		return nil, statusIcons
	}
	return barOf(screenshot)
}

// settleStatusBar waits until the status bar is no longer the one that the tool saw before it asked for a
// change (the demo mode, a theme), shows the icons that were asked for, and stays as it is for calmPolls looks in a row.
// The system draws the bar when it can, which on a slow emulator is seconds after the request has returned, and in
// steps: the clock and the battery, then the icon of the wifi, many seconds later on a runner; or the dark bar, with its
// icons still dark. A bar that has stayed as it is for a while is not done for that: it has not got the wifi yet. A picture
// taken before is of a bar half drawn, and not like the others. It gives up after calmTimeout, says so, and goes on.
func (s *session) settleStatusBar(before []byte) {
	deadline := s.now().Add(calmTimeout)
	var last []byte
	changed, still, icons := false, 0, 0
	for {
		var bar []byte
		bar, icons = s.statusBarLook()
		changed = changed || !bytes.Equal(bar, before)
		if last != nil && bytes.Equal(bar, last) {
			still++
		} else {
			still = 0
		}
		last = bar
		if changed && still >= calmPolls && icons >= statusIcons {
			return
		}
		if !s.now().Before(deadline) {
			fmt.Fprintf(s.log, "warning: the status bar did not settle in %s (changed: %t, icons seen: %d of %d): the pictures that follow may show it half drawn\n", calmTimeout, changed, icons, statusIcons)
			return
		}
		s.sleep(calmPoll)
	}
}

// picture saves the screen as the picture name of the user guide, when the run takes the pictures, and
// keeps the dump of the screen beside the screenshots, which says what the picture is of when the labels
// of a screen change.
//
// When the computer does not answer, the app says so in a strip above the page, and the picture must not show it.
// On a slow emulator the app says it for a moment, after a change of theme for one. The screen is looked at
// before the screenshot and after it, and the picture is taken again while either has the strip.
func (s *session) picture(c config, name string) error {
	if c.pictures == "" {
		return nil
	}
	s.hideKeyboard()
	path := filepath.Join(c.pictures, name+".png")
	var kept screen
	for attempt := 1; ; attempt++ {
		before, _ := s.dump()
		if err := s.screenshotTo(path); err != nil {
			return err
		}
		after, err := s.dump()
		if err == nil {
			kept = after
		}
		if !hasLabel(before.elements, unreachableLabel) && !hasLabel(after.elements, unreachableLabel) {
			break
		}
		if attempt == pictureAttempts {
			s.keepDump(c.shots, "picture-"+name, kept)
			return fmt.Errorf("the picture %s shows that the computer did not answer, in each of %d attempts", name, pictureAttempts)
		}
		fmt.Fprintf(s.log, "the picture %s shows that the computer did not answer (attempt %d of %d): taking it again\n", name, attempt, pictureAttempts)
		s.sleep(tabSettle)
	}
	s.keepDump(c.shots, "picture-"+name, kept)
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
	if err := s.picture(c, pairingPicture); err != nil {
		return err
	}
	return s.darkPicture(c, darkPairingPicture, nil)
}

// darkPicture puts the system in the dark theme, takes the picture of what show leaves on the screen (nothing,
// when it is nil), and puts the light theme back.
func (s *session) darkPicture(c config, name string, show func() error) error {
	if err := s.theme(true); err != nil {
		return err
	}
	defer func() { _ = s.theme(false) }()
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
	if err := s.closeDialog(); err != nil {
		return err
	}

	// The home page in the dark theme.
	return s.darkPicture(c, darkHomePicture, func() error { return s.openTab(c, "home") })
}

// closeDialog cancels the dialog that is open, and returns when it has gone: the page that it covers cannot show its
// tabs before. It taps the button, and when the dialog is still there it goes back with the key of Android, as a
// person does, up to dialogAttempts times in all.
func (s *session) closeDialog() error {
	for attempt := 1; attempt <= dialogAttempts; attempt++ {
		shot, err := s.dump()
		if err == nil && !hasLabel(shot.elements, cancelLabel) {
			return nil
		}
		if attempt == 1 {
			if target, ok := lowestClickable(shot.elements, cancelLabel); ok {
				x, y := target.center()
				_, _ = s.adb("shell", fmt.Sprintf("input tap %d %d", x, y))
				s.sleep(tabSettle)
				continue
			}
		}
		_, _ = s.adb("shell", "input", "keyevent", keyBack)
		s.sleep(tabSettle)
	}
	shot, err := s.dump()
	if err == nil && !hasLabel(shot.elements, cancelLabel) {
		return nil
	}
	return fmt.Errorf("the dialog is still open after %d attempts to close it: the screen has %s", dialogAttempts, labelsOf(shot.elements))
}

// hideKeyboard closes the soft keyboard when one is shown. An emulator that has no hardware keyboard shows it when a field of
// the app has the focus, and the field of the dialog that adds a plugin has it as the dialog opens: the keyboard covers the
// lower half of the dialog and its buttons, which the tool would then tap in place of the button. The key of Android that goes
// back closes the keyboard, and leaves the dialog and its field.
func (s *session) hideKeyboard() {
	out, err := s.adb("shell", "dumpsys", "input_method")
	if err != nil || !bytes.Contains(out, []byte("mInputShown=true")) {
		return
	}
	_, _ = s.adb("shell", "input", "keyevent", keyBack)
	s.sleep(tabSettle)
}

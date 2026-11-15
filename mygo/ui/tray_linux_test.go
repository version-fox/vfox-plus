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

//go:build linux

package ui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/assets"
)

// Where the session bus cannot be reached there is no host for tray icons, and making an icon says so
// without an error that stops the application. (That a host that is there is found is shown by driving
// the application against one, which needs a desktop: see docs/en/dev/architecture.md, "The tray".)
func TestADesktopWhoseSessionBusCannotBeReachedHasNoTrayHost(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+t.TempDir()+"/no-bus")
	picture, err := assets.Files.ReadFile("tray.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newNativeTray(trayOptions{icon: picture}); !errors.Is(err, errNoTrayHost) {
		t.Errorf("an icon on a bus that is not there: %v, want no host of the icons", err)
	}
}

// TestTheIconIsItsPictureAsThePanelTakesIt checks the picture of the application, as the item has it.
func TestTheIconIsItsPictureAsThePanelTakesIt(t *testing.T) {
	picture, err := assets.Files.ReadFile("tray.png")
	if err != nil {
		t.Fatal(err)
	}
	pixmap, err := sniPixmapOf(picture)
	if err != nil {
		t.Fatalf("the picture is not a PNG: %v", err)
	}
	if pixmap.Width != 32 || pixmap.Height != 32 || len(pixmap.Pixels) != 4*32*32 {
		t.Fatalf("the picture is %d by %d with %d bytes", pixmap.Width, pixmap.Height, len(pixmap.Pixels))
	}
	// The fox is opaque at its middle, and the corner is clear.
	middle := 4 * (16*32 + 16)
	if pixmap.Pixels[middle] != 0xFF || pixmap.Pixels[0] != 0 {
		t.Errorf("alpha %d at the middle and %d at the corner, want opaque and clear", pixmap.Pixels[middle], pixmap.Pixels[0])
	}
}

// TestAPixelHasItsAlphaFirst checks the order that the protocol has the bytes of a pixel in.
func TestAPixelHasItsAlphaFirst(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 40})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	pixmap, err := sniPixmapOf(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{40, 10, 20, 30}; !bytes.Equal(pixmap.Pixels, want) {
		t.Errorf("pixels %v, want %v", pixmap.Pixels, want)
	}
}

// TestAnIconIsAnItemOfItsOwnOnTheBus checks the names of the icons: the GNOME Shell takes names of this
// form, and each icon has a name that no other icon of the process has.
func TestAnIconIsAnItemOfItsOwnOnTheBus(t *testing.T) {
	shell := regexp.MustCompile(`^org\.freedesktop\.StatusNotifierItem-[0-9]+-[0-9]+$`)
	first, second := sniBusName(4242, 1), sniBusName(4242, 2)
	if first != "org.freedesktop.StatusNotifierItem-4242-1" || !shell.MatchString(first) || !shell.MatchString(second) || first == second {
		t.Errorf("the names %q and %q", first, second)
	}
}

// TestTheMenuLaysOutItsEntriesAsTheChildrenOfItsRoot checks what a desktop reads of the menu: the entries
// are the children of the root, a separator is an entry that does nothing, and an underscore of a label
// is doubled, as a single one would mark the mnemonic of the next character.
func TestTheMenuLaysOutItsEntriesAsTheChildrenOfItsRoot(t *testing.T) {
	click := func() {}
	layout := sniLayout([]trayEntry{{label: "Open vfox+", click: click}, {}, {label: "Save_as", click: click}})
	if layout.ID != 0 || layout.Properties["children-display"].Value() != "submenu" || len(layout.Children) != 3 {
		t.Fatalf("the root %+v", layout)
	}
	first := layout.Children[0].Value().(sniMenuNode)
	separator := layout.Children[1].Value().(sniMenuNode)
	last := layout.Children[2].Value().(sniMenuNode)
	if first.ID != 1 || first.Properties["label"].Value() != "Open vfox+" {
		t.Errorf("the first entry %+v", first)
	}
	if separator.ID != 2 || separator.Properties["type"].Value() != "separator" {
		t.Errorf("the separator %+v", separator)
	}
	if last.ID != 3 || last.Properties["label"].Value() != "Save__as" {
		t.Errorf("the last entry %+v", last)
	}
}

// TestAChosenEntryRunsItsClickAndNothingElse checks which click a chosen id runs: the entry's, and
// nothing for the root, a separator, or an id that the menu does not have.
func TestAChosenEntryRunsItsClickAndNothingElse(t *testing.T) {
	var called []string
	tray := &sniTray{entries: []trayEntry{
		{label: "A", click: func() { called = append(called, "A") }},
		{},
		{label: "B", click: func() { called = append(called, "B") }},
	}}
	for _, id := range []int32{0, 1, 2, 3, 4} {
		if click := tray.clickOf(id); click != nil {
			click()
		}
	}
	if strings.Join(called, "") != "AB" {
		t.Errorf("called %v, want the entries A and B", called)
	}
}

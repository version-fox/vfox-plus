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

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// sdkAvatarCase is what the web interface computes for a name (mygo/web/src/dom.ts:
// hashName, then the av-p, av-s or av-t tone from the hash modulo 3, the av-0 to
// av-11 shape from the hash modulo 12, and the first letter in upper case). The
// values were computed by running hashName and avatar of that file, with its types
// removed, in node.
type sdkAvatarCase struct {
	name   string
	hash   uint32
	tone   int
	shape  int
	letter string
}

var sdkAvatarCases = []sdkAvatarCase{
	{name: "go", hash: 3304, tone: 1, shape: 4, letter: "G"},
	{name: "java", hash: 3254818, tone: 1, shape: 10, letter: "J"},
	{name: "nodejs", hash: 3254797003, tone: 1, shape: 7, letter: "N"},
	{name: "flutter", hash: 3534632988, tone: 0, shape: 0, letter: "F"},
	{name: "dotnet", hash: 2968990644, tone: 0, shape: 0, letter: "D"},
	{name: "kotlin", hash: 3169392897, tone: 0, shape: 9, letter: "K"},
	{name: "cmake", hash: 94772433, tone: 0, shape: 9, letter: "C"},
	{name: "maven", hash: 103670155, tone: 1, shape: 7, letter: "M"},
	{name: "php", hash: 110968, tone: 1, shape: 4, letter: "P"},
	{name: "dart", hash: 3075967, tone: 1, shape: 7, letter: "D"},
	{name: "Go", hash: 2312, tone: 2, shape: 8, letter: "G"},
	{name: "ü", hash: 252, tone: 0, shape: 0, letter: "Ü"},
	{name: "日本", hash: 835047, tone: 0, shape: 3, letter: "日"},
	{name: "a", hash: 97, tone: 1, shape: 1, letter: "A"},
	{name: "", hash: 0, tone: 0, shape: 0, letter: "?"},
	{name: "zig", hash: 120600, tone: 0, shape: 0, letter: "Z"},
	{name: "vfox-plus", hash: 2055927374, tone: 2, shape: 2, letter: "V"},
	{name: "terraform-lint", hash: 638723932, tone: 1, shape: 4, letter: "T"},
}

func TestSDKAvatarFollowsTheHashOfTheWeb(t *testing.T) {
	for _, c := range sdkAvatarCases {
		hash := sdkNameHash(c.name)
		if hash != c.hash {
			t.Errorf("hash of %q is %d, the web's is %d", c.name, hash, c.hash)
		}
		if tone := sdkTone(hash); tone != c.tone {
			t.Errorf("tone of %q is %d, the web's is %d", c.name, tone, c.tone)
		}
		if got := sdkShape(hash); got != shapes.AvatarShapes[c.shape] {
			t.Errorf("shape of %q is not the web's av-%d", c.name, c.shape)
		}
		if letter := sdkInitial(c.name); letter != c.letter {
			t.Errorf("letter of %q is %q, the web's is %q", c.name, letter, c.letter)
		}
	}
}

func TestSDKAvatarColoursAreTheToneOfThePalette(t *testing.T) {
	palette := kit.PaletteFor(seedColor, false)
	cases := []struct {
		tone                int
		background, content ui.Color
	}{
		{tone: 0, background: palette.PrimaryFixed, content: palette.OnPrimaryFixed},
		{tone: 1, background: palette.SecondaryContainer, content: palette.OnSecondaryContainer},
		{tone: 2, background: palette.TertiaryContainer, content: palette.OnTertiaryContainer},
	}
	for _, c := range cases {
		background, content := sdkTonePair(&palette, c.tone)
		if background != c.background || content != c.content {
			t.Errorf("tone %d is %v on %v, want %v on %v", c.tone, content, background, c.content, c.background)
		}
	}
	// The primary tone is the web's av-p, which is primary-fixed, #FFDCC1 in the
	// light scheme of the same seed.
	if want := kit.ARGB(0xFFFFDCC1); palette.PrimaryFixed != want {
		t.Errorf("the primary fixed colour is %v, the web's is %v", palette.PrimaryFixed, want)
	}
}

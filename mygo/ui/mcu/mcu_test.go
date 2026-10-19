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

package mcu_test

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/ui/mcu"
)

func hex(argb uint32) string { return fmt.Sprintf("#%06x", argb&0xFFFFFF) }

// golden is the output of testdata/generate.mjs: for every seed, the roles
// of the light and the dark scheme, as the JavaScript implementation
// computes them.
func golden(t *testing.T) map[string]map[string]map[string]string {
	t.Helper()
	content, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]map[string]map[string]string
	if err := json.Unmarshal(content, &data); err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("the golden file is empty")
	}
	return data
}

func TestSchemesMatchTheJavaScriptImplementation(t *testing.T) {
	data := golden(t)
	seeds := make([]string, 0, len(data))
	for seed := range data {
		seeds = append(seeds, seed)
	}
	sort.Strings(seeds)
	checked, mismatches := 0, 0
	for _, seedHex := range seeds {
		value, err := strconv.ParseUint(strings.TrimPrefix(seedHex, "#"), 16, 32)
		if err != nil {
			t.Fatal(err)
		}
		seed := 0xFF000000 | uint32(value)
		for _, dark := range []bool{false, true} {
			mode := "light"
			if dark {
				mode = "dark"
			}
			expected := data[seedHex][mode]
			for _, token := range mcu.Generate(seed, dark).Tokens() {
				want, ok := expected[token.Name]
				if !ok {
					t.Fatalf("the golden file has no role %s", token.Name)
				}
				checked++
				if got := hex(token.ARGB); got != want {
					mismatches++
					if mismatches <= 20 {
						t.Errorf("seed %s %s %s: got %s, want %s", seedHex, mode, token.Name, got, want)
					}
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d of %d colours differ", mismatches, checked)
	}
	if checked < 82*2*48 {
		t.Errorf("only %d colours were checked", checked)
	}
}

func TestTheDefaultSeedGivesTheOrangeScheme(t *testing.T) {
	light := mcu.Generate(0xFFF28C1B, false)
	dark := mcu.Generate(0xFFF28C1B, true)
	for name, pair := range map[string][2]string{
		"surface":           {hex(light.Surface), "#fff8f5"},
		"primary":           {hex(light.Primary), "#88521c"},
		"primary-container": {hex(light.PrimaryContainer), "#ffdcc1"},
		"on-surface":        {hex(light.OnSurface), "#221a14"},
		"dark primary":      {hex(dark.Primary), "#ffb779"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s: got %s, want %s", name, pair[0], pair[1])
		}
	}
	if light.Surface>>24 != 0xFF {
		t.Error("colours are opaque")
	}
}

func TestHctRoundTrips(t *testing.T) {
	for _, argb := range []uint32{0xFF000000, 0xFFFFFFFF, 0xFFF28C1B, 0xFF0061A4, 0xFF808080} {
		hct := mcu.HctFromARGB(argb)
		if hct.ARGB() != argb {
			t.Errorf("%s: got %s", hex(argb), hex(hct.ARGB()))
		}
		again := mcu.HctFrom(hct.Hue, hct.Chroma, hct.Tone)
		if hex(again.ARGB()) != hex(argb) {
			t.Errorf("%s: solving for %.1f, %.1f, %.1f gave %s", hex(argb), hct.Hue, hct.Chroma, hct.Tone, hex(again.ARGB()))
		}
	}
	// A chroma of 0 is a grey; 8 bits per channel leave a little chroma.
	if gray := mcu.HctFrom(10, 0, 50); gray.Chroma > 3 {
		t.Errorf("a chroma of 0 is grey, got %v", gray.Chroma)
	}
}

func TestTonalPaletteTones(t *testing.T) {
	palette := mcu.NewTonalPalette(60, 36)
	if hex(palette.Tone(0)) != "#000000" || hex(palette.Tone(100)) != "#ffffff" {
		t.Errorf("got %s %s", hex(palette.Tone(0)), hex(palette.Tone(100)))
	}
	first := palette.Tone(40)
	if again := palette.Tone(40); again != first {
		t.Error("tones are stable")
	}
	// Tone 99 of a yellow is the average of its neighbours.
	yellow := mcu.NewTonalPalette(110, 36)
	if a, b, c := yellow.Tone(98), yellow.Tone(99), yellow.Tone(100); b == a && b == c {
		t.Errorf("got %s %s %s", hex(a), hex(b), hex(c))
	}
	if h := palette.Hct(40); h.Tone < 39 || h.Tone > 41 {
		t.Errorf("got tone %v", h.Tone)
	}
}

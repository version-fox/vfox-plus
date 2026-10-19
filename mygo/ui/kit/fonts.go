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

package kit

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The font families of the interface. mygo draws text with the text engine of
// the desktop, which finds these through fontconfig: the sans-serif font of the
// desktop is the font of Material 3 on Linux (Ubuntu Sans on Ubuntu), and what
// it lacks, such as Chinese, comes from the next family and then from the
// fonts the system chooses.
const (
	// UIFamilies is the font of the interface.
	UIFamilies = "Ubuntu Sans, Ubuntu, Noto Sans, sans-serif"
	// MonoFamilies is the monospaced font.
	MonoFamilies = "Ubuntu Sans Mono, Ubuntu Mono, Noto Sans Mono, monospace"

	// The icon fonts are static instances of Material Symbols Rounded, the
	// outlined (fill 0) and the filled (fill 1) one: mygo draws a font at the
	// weight and not at the variations of its axes, so the fill axis, which the
	// interface animates, is the cross-fade of the two.
	iconFamily       = "VfoxSymbols"
	iconFilledFamily = "VfoxSymbolsFilled"
)

var (
	iconMu    sync.RWMutex
	iconCodes map[string]string
)

// LoadIcons registers the icon fonts, the outlined and the filled one, and the
// code points of the icons, which tools/prepare.sh generates from
// shared/icons.json: {"icons": {"<meaning>": {"codepoint": 57669}}}.
func LoadIcons(outlined, filled, codepoints []byte) error {
	var table struct {
		Icons map[string]struct {
			Codepoint rune `json:"codepoint"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(codepoints, &table); err != nil {
		return fmt.Errorf("the code points of the icons: %w", err)
	}
	codes := make(map[string]string, len(table.Icons))
	for name, icon := range table.Icons {
		codes[name] = string(icon.Codepoint)
	}
	if err := ui.RegisterFont(outlined, iconFamily); err != nil {
		return fmt.Errorf("the icon font: %w", err)
	}
	if err := ui.RegisterFont(filled, iconFilledFamily); err != nil {
		return fmt.Errorf("the filled icon font: %w", err)
	}
	iconMu.Lock()
	iconCodes = codes
	iconMu.Unlock()
	return nil
}

// iconGlyph is the character that draws the icon with the given meaning (a key
// of shared/icons.json); it is empty when the icon is not in the font.
func iconGlyph(name string) string {
	iconMu.RLock()
	defer iconMu.RUnlock()
	return iconCodes[name]
}

// HasIcon reports whether the icon fonts know the icon.
func HasIcon(name string) bool { return iconGlyph(name) != "" }

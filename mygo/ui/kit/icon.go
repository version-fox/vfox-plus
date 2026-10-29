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
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The icons are the SVGs of Material Symbols Rounded (weight 400, grade 0,
// optical size 24) in shared/icons, drawn by mygo's ui.Icon in the colour of
// the text: SYMBOL.svg is the outlined icon (fill 0) and SYMBOL-fill.svg the
// filled one (fill 1). The meanings of the interface name the symbols
// (shared/icons.json). An SVG is parsed when its icon is first drawn, so that
// the icons a session never shows cost nothing.

// iconSet is what LoadIcons registered.
type iconSet struct {
	symbols map[string]string // by meaning
	files   fs.FS

	mu     sync.Mutex
	parsed map[string]*ui.SVG // by file name; nil for a file that cannot be parsed
}

var (
	iconMu  sync.RWMutex
	current *iconSet
)

// LoadIcons registers the icons of the interface. table is shared/icons.json,
// {"icons": {"<meaning>": "<symbol>"}}, and files holds the SVGs of the
// symbols, SYMBOL.svg and SYMBOL-fill.svg. It fails, and registers nothing,
// when a symbol lacks one of its two files.
func LoadIcons(table []byte, files fs.FS) error {
	var parsed struct {
		Icons map[string]string `json:"icons"`
	}
	if err := json.Unmarshal(table, &parsed); err != nil {
		return fmt.Errorf("the table of the icons: %w", err)
	}
	var missing []string
	for _, symbol := range parsed.Icons {
		for _, name := range []string{symbol + ".svg", symbol + "-fill.svg"} {
			if _, err := fs.Stat(files, name); err != nil {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("no icon file: %s", strings.Join(missing, ", "))
	}
	set := &iconSet{symbols: parsed.Icons, files: files, parsed: map[string]*ui.SVG{}}
	iconMu.Lock()
	current = set
	iconMu.Unlock()
	return nil
}

// svg returns the outlined and the filled icon of a meaning, or nil for both
// when the meaning is not registered (or its files cannot be parsed).
func (s *iconSet) svg(meaning string) (outlined, filled *ui.SVG) {
	symbol, ok := s.symbols[meaning]
	if !ok {
		return nil, nil
	}
	outlined, filled = s.parse(symbol+".svg"), s.parse(symbol+"-fill.svg")
	if outlined == nil || filled == nil {
		return nil, nil
	}
	return outlined, filled
}

func (s *iconSet) parse(name string) *ui.SVG {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svg, done := s.parsed[name]; done {
		return svg
	}
	var svg *ui.SVG
	if data, err := fs.ReadFile(s.files, name); err == nil {
		if parsed, err := ui.ParseSVG(data); err == nil {
			svg = parsed
		}
	}
	s.parsed[name] = svg
	return svg
}

func iconsOf(meaning string) (outlined, filled *ui.SVG) {
	iconMu.RLock()
	set := current
	iconMu.RUnlock()
	if set == nil {
		return nil, nil
	}
	return set.svg(meaning)
}

// HasIcon reports whether the interface has an icon with that meaning.
func HasIcon(name string) bool {
	outlined, _ := iconsOf(name)
	return outlined != nil
}

// Icon draws an icon of Material Symbols Rounded, by its meaning (a key of
// shared/icons.json), in a square of size dp. fill, from 0 (outlined) to 1
// (filled), is the fill of the symbol: the icon of the selected destination of
// the navigation is filled. A fill between the two cross-fades the outlined
// and the filled icon. An icon that is not registered leaves its square empty.
func Icon(c Ctx, name string, size float32, col ui.Color, fill float32) ui.Element {
	outlined, filled := iconsOf(name)
	fill = clamp01(fill)
	// An element is added to the tree where it is made, so the square is only
	// made when it is the icon.
	switch {
	case outlined == nil:
		return ui.Box(c.Context).Size(size, size).Shrink(0)
	case fill < 0.005:
		return iconSVG(c, outlined, size, col)
	case fill > 0.995:
		return iconSVG(c, filled, size, col)
	}
	return ui.Box(c.Context).Size(size, size).Shrink(0).Children(func() {
		iconSVG(c, outlined, size, col).Opacity(1 - fill)
		iconSVG(c, filled, size, col).Opacity(fill).Absolute().Top(0).Left(0)
	})
}

func iconSVG(c Ctx, svg *ui.SVG, size float32, col ui.Color) ui.Element {
	return ui.Icon(c.Context, svg).TextColor(col).Size(size, size).Shrink(0)
}

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
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	giofont "gioui.org/font"
	"gioui.org/text"
	fontapi "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// The typefaces of the interface.
const (
	uiTypeface   giofont.Typeface = "UI"
	monoTypeface giofont.Typeface = "Mono"
	iconTypeface giofont.Typeface = "Icons"
)

// iconFills is how many fill levels of the icon font are available: the fill
// of the icon of a navigation destination animates from 0 to 1, and Gio draws
// a face at a fixed variation, so there is a face for a few levels between.
const iconFills = 5

// Fonts are the faces of the interface and the shaper that draws them.
//
// The sans-serif font of the system is the font of the interface, as
// Material 3 on the Linux desktop wants it (Ubuntu on Ubuntu). Gio draws a
// face at a fixed set of variations, which a variable font does not have, so
// the weights that the type scale uses are faces of their own, and so are the
// fills of the icon font; the system fonts remain the fallback for scripts
// that the fonts of the interface lack, such as Chinese.
type Fonts struct {
	// Shaper shapes and lays out text.
	Shaper *text.Shaper
	// UIFont names the font of the interface, for the log.
	UIFont string
	// icons are the code points of the icons, by meaning.
	icons map[string]rune
}

// IconRune is the code point of the icon with the given meaning (the keys of
// shared/icons.json), or the replacement character.
func (f *Fonts) IconRune(name string) rune {
	if r, ok := f.icons[name]; ok {
		return r
	}
	return '\uFFFD'
}

// UI is the font of the interface at a weight from 100 to 900.
func (f *Fonts) UI(weight int) giofont.Font {
	return giofont.Font{Typeface: uiTypeface, Weight: giofont.Weight(weight - 400)}
}

// Mono is the monospaced font.
func (f *Fonts) Mono() giofont.Font { return giofont.Font{Typeface: monoTypeface} }

// Icon is the icon font at the fill level closest to fill (0 to 1).
func (f *Fonts) Icon(fill float32) giofont.Font {
	level := int(clamp01(fill)*(iconFills-1) + 0.5)
	return giofont.Font{Typeface: giofont.Typeface("Icons" + strconv.Itoa(level))}
}

// faceOfVariation is a face of a font with variations set.
type faceOfVariation struct {
	font       *fontapi.Font
	variations []fontapi.Variation
}

// Face implements giofont.Face.
func (f faceOfVariation) Face() *fontapi.Face {
	face := fontapi.NewFace(f.font)
	if len(f.variations) > 0 {
		face.SetVariations(f.variations)
	}
	return face
}

// parseFont parses a font of the data. Gio tells faces apart by their *Font,
// so every variation of a variable font needs a Font of its own: a copy of
// the parsed one, which shares its tables (parsing the data again for each
// would keep a copy of the tables for each in the memory).
func parseFont(data []byte, index int) (*fontapi.Font, bool, error) {
	if index > 0 || bytes.HasPrefix(data, []byte("ttcf")) {
		loaders, err := opentype.NewLoaders(bytes.NewReader(data))
		if err != nil {
			return nil, false, err
		}
		if index >= len(loaders) {
			index = 0
		}
		f, err := fontapi.NewFont(loaders[index])
		return f, loaders[index].HasTable(opentype.MustNewTag("fvar")), err
	}
	loader, err := opentype.NewLoader(bytes.NewReader(data))
	if err != nil {
		return nil, false, err
	}
	f, err := fontapi.NewFont(loader)
	return f, loader.HasTable(opentype.MustNewTag("fvar")), err
}

// fontSource is a font file of the system.
type fontSource struct {
	path     string
	index    int
	variable bool
}

// matchFont asks fontconfig for the font that matches a pattern, and the
// families it belongs to.
func matchFont(pattern string) (fontSource, []string, bool) {
	out, err := exec.Command("fc-match", "-f", "%{family}\t%{file}\t%{index}\t%{variable}", pattern).Output() //nolint:gosec // The pattern is made of the names of fonts above.
	if err != nil {
		return fontSource{}, nil, false
	}
	parts := strings.Split(string(out), "\t")
	if len(parts) < 4 || parts[1] == "" {
		return fontSource{}, nil, false
	}
	index, _ := strconv.Atoi(parts[2])
	return fontSource{path: parts[1], index: index, variable: strings.EqualFold(strings.TrimSpace(parts[3]), "true")},
		strings.Split(parts[0], ","), true
}

// findFont looks for the first of the families that is installed: fontconfig
// answers for every name, so the family of the answer must be the one asked
// for, unless it is the generic family, which always matches. The pattern asks
// for a language that the fonts of the interface cover, so that a locale of
// the desktop that needs another font does not pick a huge one.
func findFont(families ...string) (fontSource, string, bool) {
	for _, family := range families {
		source, matched, ok := matchFont(family + ":lang=en")
		if !ok {
			continue
		}
		if family == "sans-serif" || family == "monospace" {
			return source, family, true
		}
		for _, name := range matched {
			if strings.EqualFold(strings.TrimSpace(name), family) {
				return source, family, true
			}
		}
	}
	return fontSource{}, "", false
}

// The fonts that the interface prefers, in this order: Ubuntu on Ubuntu, as
// the design system asks, and else the sans-serif font of the desktop.
var (
	sansFamilies = []string{"Ubuntu Sans", "Ubuntu", "sans-serif"}
	monoFamilies = []string{"Ubuntu Sans Mono", "Ubuntu Mono", "monospace"}
)

// LoadFonts creates the fonts of the interface. iconFont is the icon font, with a
// fill axis, and iconCodes the code points of its icons, as written by
// shared/tools/icon_font.py.
func LoadFonts(iconFont, iconCodes []byte) (*Fonts, error) {
	var faces []text.FontFace
	name := loadUIFaces(&faces)
	if mono, ok := loadMono(); ok {
		faces = append(faces, mono)
	}
	iconFaces, err := loadIcons(iconFont)
	if err != nil {
		return nil, err
	}
	faces = append(faces, iconFaces...)
	codes, err := parseIconCodes(iconCodes)
	if err != nil {
		return nil, err
	}
	return &Fonts{Shaper: text.NewShaper(text.WithCollection(faces)), UIFont: name, icons: codes}, nil
}

// parseIconCodes reads the code points of the icons.
func parseIconCodes(data []byte) (map[string]rune, error) {
	var parsed struct {
		Icons map[string]struct {
			Codepoint int `json:"codepoint"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	codes := make(map[string]rune, len(parsed.Icons))
	for name, icon := range parsed.Icons {
		if icon.Codepoint > 0 && icon.Codepoint <= unicode.MaxRune {
			codes[name] = rune(icon.Codepoint)
		}
	}
	return codes, nil
}

// weights are the weights that the type scale uses.
var weights = [...]int{400, 500, 700}

// variationWeight is the value of the weight axis of a variable font for a
// weight of the type scale. Gio blends the edges of glyphs without the
// correction of gamma that other toolkits apply, which makes dark text on a
// light surface look lighter than the weight says, so the axis is set a little
// above it.
func variationWeight(weight int) float32 { return float32(weight) + 40 }

func loadUIFaces(faces *[]text.FontFace) string {
	if regular, family, ok := findFont(sansFamilies...); ok {
		if added := loadFamily(faces, regular, family+":lang=en", uiTypeface); added {
			return regular.path
		}
	}
	// No usable system font: the Go fonts, which are always there.
	for _, face := range []struct {
		data   []byte
		weight int
	}{{goregular.TTF, 400}, {gomedium.TTF, 500}, {gobold.TTF, 700}} {
		if f, _, err := parseFont(face.data, 0); err == nil {
			*faces = append(*faces, text.FontFace{
				Font: giofont.Font{Typeface: uiTypeface, Weight: giofont.Weight(face.weight - 400)},
				Face: faceOfVariation{font: f},
			})
		}
	}
	return "Go"
}

// loadFamily loads the three weights of the family of regular. A variable
// font gives them as variations; static fonts are looked up for each weight.
func loadFamily(faces *[]text.FontFace, regular fontSource, pattern string, typeface giofont.Typeface) bool {
	data, err := os.ReadFile(regular.path)
	if err != nil {
		return false
	}
	// Fontconfig lists the instances of a variable font as fonts that are not
	// variable: the font file says.
	if _, variable, err := parseFont(data, regular.index); err == nil {
		regular.variable = variable
	}
	if regular.variable {
		parsed, _, err := parseFont(data, regular.index)
		if err != nil {
			return false
		}
		for _, weight := range weights {
			// A Font of its own for every weight, which shares the tables of
			// the one that was parsed.
			clone := *parsed
			f := &clone
			*faces = append(*faces, text.FontFace{
				Font: giofont.Font{Typeface: typeface, Weight: giofont.Weight(weight - 400)},
				Face: faceOfVariation{font: f, variations: []fontapi.Variation{
					{Tag: opentype.MustNewTag("wght"), Value: variationWeight(weight)},
				}},
			})
		}
		return true
	}
	added := false
	for _, weight := range weights {
		source := regular
		switch weight {
		case 500:
			if m, _, ok := matchFont(pattern + ":weight=medium"); ok {
				source = m
			}
		case 700:
			if b, _, ok := matchFont(pattern + ":weight=bold"); ok {
				source = b
			}
		}
		fileData := data
		if source.path != regular.path {
			if d, err := os.ReadFile(source.path); err == nil {
				fileData = d
			}
		}
		f, _, err := parseFont(fileData, source.index)
		if err != nil {
			continue
		}
		*faces = append(*faces, text.FontFace{
			Font: giofont.Font{Typeface: typeface, Weight: giofont.Weight(weight - 400)},
			Face: faceOfVariation{font: f},
		})
		added = true
	}
	return added
}

func loadMono() (text.FontFace, bool) {
	if source, _, ok := findFont(monoFamilies...); ok {
		if data, err := os.ReadFile(source.path); err == nil {
			if f, variable, err := parseFont(data, source.index); err == nil {
				face := faceOfVariation{font: f}
				if variable {
					face.variations = []fontapi.Variation{{Tag: opentype.MustNewTag("wght"), Value: 400}}
				}
				return text.FontFace{Font: giofont.Font{Typeface: monoTypeface}, Face: face}, true
			}
		}
	}
	if f, _, err := parseFont(gomono.TTF, 0); err == nil {
		return text.FontFace{Font: giofont.Font{Typeface: monoTypeface}, Face: faceOfVariation{font: f}}, true
	}
	return text.FontFace{}, false
}

func loadIcons(data []byte) ([]text.FontFace, error) {
	var faces []text.FontFace
	for level := 0; level < iconFills; level++ {
		f, _, err := parseFont(data, 0)
		if err != nil {
			return nil, err
		}
		faces = append(faces, text.FontFace{
			Font: giofont.Font{Typeface: giofont.Typeface("Icons" + strconv.Itoa(level))},
			Face: faceOfVariation{font: f, variations: []fontapi.Variation{
				{Tag: opentype.MustNewTag("FILL"), Value: float32(level) / (iconFills - 1)},
			}},
		})
	}
	return faces, nil
}

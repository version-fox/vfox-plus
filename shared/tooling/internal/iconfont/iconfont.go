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

// Package iconfont downloads and subsets the Material Symbols Rounded
// variable font for the icons in shared/icons.json. It is a pure-Go
// replacement for shared/tools/icon_font.py: the subset step is implemented
// natively (TrueType subsetting plus variable-font instancing), so no
// fontTools installation is needed.
//
// Behavior is kept equivalent to the Python tool and to fontTools
// varLib.instancer for the FILL, GRAD, opsz and wght axes:
//
//   - subset keeps the codepoints of the icons plus the [a-z0-9_] ligature
//     characters, glyph 0, GSUB ligature lookups pruned with
//     layout_closure=false semantics, and all name records;
//   - pinning bakes gvar deltas (with IUP-inferred deltas and phantom-point
//     metrics) into glyf, drops pinned axes from fvar (dropping fvar, gvar
//     and avar on a full pin), prunes STAT and the name table, and updates
//     OS/2 metrics like the instancer does.
//
// Known behavior deltas vs fontTools are documented on the relevant
// functions. In particular, WOFF2 output compresses the font data with the
// Brotli library; and IUP
// re-optimization of surviving gvar tuples is skipped (rendering-identical,
// marginally larger).
package iconfont

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Pinned upstream revision and files, mirroring icon_font.py.
const commit = "10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7"
const baseURL = "https://raw.githubusercontent.com/google/material-design-icons/" + commit
const fontName = "MaterialSymbolsRounded[FILL,GRAD,opsz,wght]"

type pinnedFile struct {
	repoPath string
	sha256   string
}

var pinnedFiles = []struct {
	name string
	file pinnedFile
}{
	{fontName + ".ttf", pinnedFile{"variablefont/" + fontName + ".ttf", "7e74e36533ccee12c2b81e75bfcf3ccf123d3bb7e7e46d65d94445f23db0c97c"}},
	{fontName + ".codepoints", pinnedFile{"variablefont/" + fontName + ".codepoints", "a7404d9e5a1b63f496cf05ace45d386075886b5a7f123fba9f29527cb6006fde"}},
	{"LICENSE", pinnedFile{"LICENSE", "58d1e17ffe5109a7ae296caafcadfdbe6a7d176f0bc4ab01e12a689b0499d8bd"}},
}

// ligatureChars mirrors LIGATURE_CHARACTERS in icon_font.py.
const ligatureChars = "abcdefghijklmnopqrstuvwxyz0123456789_"

// dropTables mirrors fontTools subset Options._drop_tables_default.
var subsetDropTables = []string{
	"JSTF", "DSIG", "EBDT", "EBLC", "EBSC", "PCLT", "LTSH",
	"Feat", "Glat", "Gloc", "Silf", "Sill",
}

// Run implements `iconfont download DIR` and
// `iconfont subset --fonts DIR --output FILE [--json FILE] [--pin AXIS=VALUE...]`.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: iconfont <download|subset> [...]; try `iconfont download DIR` or `iconfont subset --fonts DIR --output FILE`")
	}
	switch args[0] {
	case "download":
		if len(args) != 2 {
			return fmt.Errorf("usage: iconfont download DIR")
		}
		return download(args[1])
	case "subset":
		return runSubset(args[1:])
	default:
		return fmt.Errorf("unknown iconfont command %q (want download or subset)", args[0])
	}
}

// quotePath percent-encodes a repository path like urllib.parse.quote with
// safe='/', so URLs match the Python tool byte-for-byte.
func quotePath(path string) string {
	var sb strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' ||
			'0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' ||
			c == '_' || c == '.' || c == '-' || c == '~' {
			sb.WriteByte(c)
		} else {
			fmt.Fprintf(&sb, "%%%02X", c)
		}
	}
	return sb.String()
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func download(dir string) error {
	files := make([]pinnedFile, len(pinnedFiles))
	for i, pf := range pinnedFiles {
		files[i] = pf.file
	}
	names := make([]string, len(pinnedFiles))
	for i, pf := range pinnedFiles {
		names[i] = pf.name
	}
	return downloadFiles(dir, baseURL, names, files, &http.Client{Timeout: 120 * time.Second})
}

func downloadFiles(dir, base string, names []string, files []pinnedFile, client *http.Client) error {
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	for i, pf := range files {
		target := filepath.Join(dir, names[i])
		if sum, err := sha256File(target); err == nil && sum == pf.sha256 {
			continue
		}
		url := base + "/" + quotePath(pf.repoPath)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("download %s: %w", url, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("download %s: %w", url, err)
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("download %s: HTTP %s", url, resp.Status)
		}
		partial := target + ".part"
		if err := os.WriteFile(partial, body, 0o666); err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		actual := hex.EncodeToString(sum[:])
		if actual != pf.sha256 {
			os.Remove(partial)
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", url, pf.sha256, actual)
		}
		if err := os.Rename(partial, target); err != nil {
			return err
		}
		fmt.Printf("Downloaded %s\n", target)
	}
	return nil
}

func readCodepoints(path string) (map[string]uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]uint32{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			v, err := strconv.ParseUint(parts[1], 16, 32)
			if err != nil {
				return nil, fmt.Errorf("bad codepoints line %q", line)
			}
			out[parts[0]] = uint32(v)
		}
	}
	return out, nil
}

type pinFlag map[string]float64

func parsePin(text string) (string, float64, error) {
	axis, value, _ := strings.Cut(text, "=")
	if !isVariableAxis(axis) || value == "" {
		return "", 0, fmt.Errorf("expected AXIS=VALUE with AXIS one of FILL, GRAD, opsz, wght: %s", text)
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", 0, fmt.Errorf("expected AXIS=VALUE with AXIS one of FILL, GRAD, opsz, wght: %s", text)
	}
	return axis, v, nil
}

// findIconsJSON locates shared/icons.json from the current directory or the
// executable, walking up through ancestors.
func findIconsJSON() (string, error) {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		for d := cwd; ; {
			candidates = append(candidates, filepath.Join(d, "shared", "icons.json"), filepath.Join(d, "icons.json"))
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		for d := filepath.Dir(exe); ; {
			candidates = append(candidates, filepath.Join(d, "shared", "icons.json"), filepath.Join(d, "icons.json"))
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("cannot find shared/icons.json (searched upwards from working directory and executable)")
}

func runSubset(args []string) error {
	fs := flag.NewFlagSet("iconfont subset", flag.ContinueOnError)
	fontsDir := fs.String("fonts", "", "directory of `download`")
	output := fs.String("output", "", "the .ttf or .woff2 to write")
	jsonOut := fs.String("json", "", "also write the codepoints of the icons")
	var pins []string
	fs.Func("pin", "pin an axis, e.g. wght=400", func(s string) error {
		pins = append(pins, s)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *fontsDir == "" || *output == "" {
		return fmt.Errorf("usage: iconfont subset --fonts DIR --output FILE [--json FILE] [--pin AXIS=VALUE]")
	}
	pinMap := map[string]float64{}
	for _, p := range pins {
		axis, v, err := parsePin(p)
		if err != nil {
			return err
		}
		pinMap[axis] = v
	}
	iconsPath, err := findIconsJSON()
	if err != nil {
		return err
	}
	return subsetCmd(*fontsDir, *output, *jsonOut, pinMap, iconsPath)
}

func subsetCmd(fontsDir, output, jsonOut string, pins map[string]float64, iconsPath string) error {
	for _, pf := range pinnedFiles {
		p := filepath.Join(fontsDir, pf.name)
		sum, err := sha256File(p)
		if err != nil || sum != pf.file.sha256 {
			return fmt.Errorf("%s is missing or corrupt; run `vfxtool iconfont download %s`", p, fontsDir)
		}
	}
	data, err := os.ReadFile(filepath.Join(fontsDir, fontName+".ttf"))
	if err != nil {
		return err
	}
	iconsData, err := os.ReadFile(iconsPath)
	if err != nil {
		return err
	}
	var iconsFile struct {
		Icons map[string]string `json:"icons"`
	}
	if err := json.Unmarshal(iconsData, &iconsFile); err != nil {
		return fmt.Errorf("parse %s: %w", iconsPath, err)
	}
	codepoints, err := readCodepoints(filepath.Join(fontsDir, fontName+".codepoints"))
	if err != nil {
		return err
	}
	var missing []string
	for _, symbol := range iconsFile.Icons {
		if _, ok := codepoints[symbol]; !ok {
			missing = append(missing, symbol)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("unknown Material Symbols: %s", strings.Join(missing, ", "))
	}
	font, err := subsetFont(data, iconsFile.Icons, codepoints, pins)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(output); dir != "" {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return err
		}
	}
	var written []byte
	if strings.HasSuffix(output, ".woff2") {
		written = font.woff2
	} else {
		written = font.sfnt
	}
	if written == nil {
		return fmt.Errorf("cannot determine output flavor from %q (want .ttf or .woff2)", output)
	}
	if err := os.WriteFile(output, written, 0o666); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%d KB, %d icons)\n", output, len(written)/1024, len(iconsFile.Icons))
	if jsonOut != "" {
		if err := writeJSON(jsonOut, iconsFile.Icons, codepoints); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", jsonOut)
	}
	return nil
}

// writeJSON writes the icon codepoint mapping, mirroring the --json output
// of icon_font.py (indent 2, sorted keys, trailing newline).
func writeJSON(path string, icons map[string]string, codepoints map[string]uint32) error {
	mapping := map[string]any{
		"font":  "Material Symbols Rounded",
		"icons": map[string]any{},
	}
	inner := mapping["icons"].(map[string]any)
	for meaning, symbol := range icons {
		inner[meaning] = map[string]any{"symbol": symbol, "codepoint": codepoints[symbol]}
	}
	encoded, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return err
		}
	}
	return os.WriteFile(path, encoded, 0o666)
}

// subsetFont runs the full pipeline and returns both serializations.
type subsetResult struct {
	sfnt  []byte
	woff2 []byte
}

func subsetFont(data []byte, icons map[string]string, codepoints map[string]uint32, pins map[string]float64) (*subsetResult, error) {
	f, err := parseSFNT(data)
	if err != nil {
		return nil, err
	}
	s, err := newSubsetter(f, icons, codepoints)
	if err != nil {
		return nil, err
	}
	if err := s.run(icons, codepoints, pins); err != nil {
		return nil, err
	}
	sfnt, err := s.writeSFNT()
	if err != nil {
		return nil, err
	}
	woff2, err := s.writeWOFF2()
	if err != nil {
		return nil, err
	}
	return &subsetResult{sfnt: sfnt, woff2: woff2}, nil
}

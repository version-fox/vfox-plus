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

package prepare

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	getter "github.com/hashicorp/go-getter/v2"
)

// The icons of the desktop interface are Material Symbols Rounded, which the npm
// package @iconify-json/material-symbols carries in one file, icons.json, in the
// IconifyJSON format: the body of an SVG for each icon, and aliases of them.
// shared/icons.json names the package, its version and the SHA-512 of its
// tarball, and says which symbols the interface uses; `vfxtool prepare` fetches
// the tarball once, into a cache, checks it and cuts those symbols out of it as
// SVG files. No icon is committed.

// defaultRegistry is where npm packages come from. VFOX_PLUS_NPM_REGISTRY names
// another one (a mirror serves the same paths).
const defaultRegistry = "https://registry.npmjs.org"

// iconSource is the package that the icons come from, as shared/icons.json pins it.
type iconSource struct {
	Package string `json:"package"`
	Version string `json:"version"`
	SHA512  string `json:"sha512"`
}

// iconManifest is shared/icons.json, of which prepare reads what it needs.
type iconManifest struct {
	Source iconSource        `json:"source"`
	Icons  map[string]string `json:"icons"`
}

// readManifest reads the manifest of the icons.
func readManifest(file string) (iconManifest, error) {
	var manifest iconManifest
	data, err := os.ReadFile(file) //nolint:gosec // The manifest of the repository.
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("%s: %w", file, err)
	}
	source := manifest.Source
	if source.Package == "" || source.Version == "" {
		return manifest, fmt.Errorf("%s: \"source\" has no package and version", file)
	}
	if raw, err := hex.DecodeString(source.SHA512); err != nil || len(raw) != sha512.Size {
		return manifest, fmt.Errorf("%s: \"source\" has no SHA-512 of the tarball (128 hexadecimal digits)", file)
	}
	if len(manifest.Icons) == 0 {
		return manifest, fmt.Errorf("%s: no icons", file)
	}
	return manifest, nil
}

// symbols are the symbols of the interface, each once, in order.
func (m iconManifest) symbols() []string {
	seen := map[string]bool{}
	var symbols []string
	for _, symbol := range m.Icons {
		if !seen[symbol] {
			seen[symbol] = true
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	return symbols
}

// tarballName is the file name of the package of an npm registry:
// material-symbols-1.2.97.tgz for @iconify-json/material-symbols 1.2.97.
func (s iconSource) tarballName() string {
	return path.Base(s.Package) + "-" + s.Version + ".tgz"
}

// url is the address of the tarball at a registry.
func (s iconSource) url(registry string) string {
	return strings.TrimSuffix(registry, "/") + "/" + s.Package + "/-/" + s.tarballName()
}

// iconCacheDir is where the tarball is kept: $VFOX_PLUS_ICONS, which can be filled
// beforehand on a machine without a network, else vfox-plus/icons in the cache
// directory of the user.
func iconCacheDir(lookup func(string) string, userCache func() (string, error)) (string, error) {
	if dir := lookup("VFOX_PLUS_ICONS"); dir != "" {
		return dir, nil
	}
	base, err := userCache()
	if err != nil {
		return "", fmt.Errorf("no cache directory for the icons (set VFOX_PLUS_ICONS): %w", err)
	}
	return filepath.Join(base, "vfox-plus", "icons"), nil
}

// checkTarball reports whether file is the package that the source pins.
func checkTarball(file string, source iconSource) error {
	f, err := os.Open(file) //nolint:gosec // A file of the cache.
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	hash := sha512.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != source.SHA512 {
		return fmt.Errorf("%s has the SHA-512 %s, not %s", file, got, source.SHA512)
	}
	return nil
}

// downloadAttempts is how often a download is tried: the registries drop a
// connection now and then. retryPause is the wait after an attempt that failed
// (a variable, so that the tests do not wait).
const downloadAttempts = 3

var retryPause = func(attempt int) time.Duration { return time.Duration(attempt) * 2 * time.Second }

// ensureTarball returns the tarball of the source in dir. A tarball that is
// there is checked and used; otherwise it is downloaded from registry, with the
// checksum that the source pins, into dir.
func ensureTarball(ctx context.Context, source iconSource, dir, registry string, log io.Writer) (string, error) {
	file := filepath.Join(dir, source.tarballName())
	if err := checkTarball(file, source); err == nil {
		return file, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	address := source.url(registry)
	partial := file + ".part"
	defer func() { _ = os.Remove(partial) }()
	transfer := &getter.Client{
		// HTTP only, and no unpacking: the tarball is read as it is.
		Getters:       []getter.Getter{&getter.HttpGetter{XTerraformGetDisabled: true}},
		Decompressors: map[string]getter.Decompressor{},
	}
	var failure error
	for attempt := 1; attempt <= downloadAttempts; attempt++ {
		_, _ = fmt.Fprintf(log, "Downloading %s\n", address)
		_ = os.Remove(partial)
		request := &getter.Request{
			Src:     address + "?checksum=sha512:" + source.SHA512,
			Dst:     partial,
			GetMode: getter.ModeFile,
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		_, failure = transfer.Get(attemptCtx, request)
		cancel()
		if failure == nil {
			break
		}
		var mismatch *getter.ChecksumError
		if errors.As(failure, &mismatch) || ctx.Err() != nil {
			break // Another try gets the same file, or no time.
		}
		select {
		case <-ctx.Done():
		case <-time.After(retryPause(attempt)):
		}
	}
	if failure != nil {
		return "", fmt.Errorf("cannot download %s %s: %w\n(a mirror of the registry: set VFOX_PLUS_NPM_REGISTRY; no network: put %s in a directory and name it with VFOX_PLUS_ICONS)",
			source.Package, source.Version, failure, source.tarballName())
	}
	if err := os.Rename(partial, file); err != nil {
		return "", err
	}
	return file, nil
}

// iconSet is the part of icons.json of a package that the icons need.
type iconSet struct {
	Icons   map[string]iconBody  `json:"icons"`
	Aliases map[string]iconAlias `json:"aliases"`
	Width   float64              `json:"width"`
	Height  float64              `json:"height"`
}

// iconBody is an icon: the markup of its shapes, and the box of its own when it
// has one.
type iconBody struct {
	Body   string   `json:"body"`
	Left   float64  `json:"left"`
	Top    float64  `json:"top"`
	Width  *float64 `json:"width"`
	Height *float64 `json:"height"`
	// Transformations of an icon are not drawn here, so they are refused.
	Rotate *int  `json:"rotate"`
	HFlip  *bool `json:"hFlip"`
	VFlip  *bool `json:"vFlip"`
}

// iconAlias is another name of an icon.
type iconAlias struct {
	Parent string `json:"parent"`
	iconBody
}

// readIconSet reads icons.json out of the tarball of the package.
func readIconSet(tarball string) (iconSet, error) {
	var set iconSet
	f, err := os.Open(tarball) //nolint:gosec // A file of the cache.
	if err != nil {
		return set, err
	}
	defer func() { _ = f.Close() }()
	zipped, err := gzip.NewReader(f)
	if err != nil {
		return set, fmt.Errorf("%s: %w", tarball, err)
	}
	defer func() { _ = zipped.Close() }()
	archive := tar.NewReader(zipped)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return set, fmt.Errorf("%s has no package/icons.json", tarball)
		}
		if err != nil {
			return set, fmt.Errorf("%s: %w", tarball, err)
		}
		if header.Name != "package/icons.json" {
			continue
		}
		if err := json.NewDecoder(archive).Decode(&set); err != nil {
			return set, fmt.Errorf("%s: package/icons.json: %w", tarball, err)
		}
		if set.Width == 0 || set.Height == 0 {
			set.Width, set.Height = 24, 24
		}
		return set, nil
	}
}

// find returns the icon of a name, which may be an alias (of an alias).
func (s iconSet) find(name string) (iconBody, error) {
	for range 8 {
		if alias, ok := s.Aliases[name]; ok {
			if alias.hasTransformation() {
				return iconBody{}, fmt.Errorf("%s is an alias that transforms %s", name, alias.Parent)
			}
			name = alias.Parent
			continue
		}
		icon, ok := s.Icons[name]
		if !ok {
			return iconBody{}, fmt.Errorf("the package has no icon %s", name)
		}
		if icon.hasTransformation() {
			return iconBody{}, fmt.Errorf("%s is transformed", name)
		}
		return icon, nil
	}
	return iconBody{}, fmt.Errorf("%s: too many aliases", name)
}

func (b iconBody) hasTransformation() bool {
	return (b.Rotate != nil && *b.Rotate != 0) || (b.HFlip != nil && *b.HFlip) || (b.VFlip != nil && *b.VFlip)
}

// svg is the icon as a file: its body in a box of its own or of the set.
func (s iconSet) svg(icon iconBody) string {
	width, height := s.Width, s.Height
	if icon.Width != nil {
		width = *icon.Width
	}
	if icon.Height != nil {
		height = *icon.Height
	}
	number := func(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="` + number(icon.Left) + " " + number(icon.Top) + " " + number(width) + " " + number(height) +
		`" width="` + number(width) + `" height="` + number(height) + `">` + icon.Body + "</svg>\n"
}

// iconNames are the names that the package gives the outlined and the filled
// rounded icon of a symbol: home-outline-rounded and home-rounded for home,
// folder-open-outline-rounded for folder_open.
func iconNames(symbol string) (outlined, filled string) {
	base := strings.ReplaceAll(symbol, "_", "-")
	return base + "-outline-rounded", base + "-rounded"
}

// writeIcons writes SYMBOL.svg (outlined) and SYMBOL-fill.svg (filled) of every
// symbol into dir, and fails, naming them all, when the package lacks icons.
func writeIcons(set iconSet, symbols []string, dir string) error {
	type file struct{ name, content string }
	var files []file
	var problems []string
	for _, symbol := range symbols {
		outlined, filled := iconNames(symbol)
		for _, variant := range []struct{ file, name string }{{symbol + ".svg", outlined}, {symbol + "-fill.svg", filled}} {
			icon, err := set.find(variant.name)
			if err != nil {
				problems = append(problems, symbol+": "+err.Error())
				continue
			}
			files = append(files, file{variant.file, set.svg(icon)})
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("icons of shared/icons.json that the package cannot give:\n  %s", strings.Join(problems, "\n  "))
	}
	for _, f := range files {
		target := filepath.Join(dir, f.name)
		if err := os.WriteFile(target, []byte(f.content), 0o666); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	return nil
}

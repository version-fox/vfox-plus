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

// The online directory: the profiles of the repository of the application, as
// they are on its main branch, so that a plugin is adapted without a release.
// Its index.json lists every profile with the SHA-256 of its file. The
// application keeps in its cache the files that differ from those it ships.

package profile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/update"
)

// CatalogURL is the online directory: shared/profile of the repository of the
// application.
const CatalogURL = "https://raw.githubusercontent.com/version-fox/vfox-plus/main/shared/profile/"

// The names of the files of the directory, and of its cache.
const (
	indexFile   = "index.json"
	shippedFile = "shipped.sha256"
	pluginsDir  = "plugins"
)

// maxPlugins is the most plugins that an index lists.
const maxPlugins = 1000

// Index is index.json of the online directory.
type Index struct {
	Schema int `json:"schema"`
	// Plugins is the SHA-256, in hexadecimal, of plugins/<name>.json by name.
	Plugins map[string]string `json:"plugins"`
}

// ParseIndex reads and checks index.json.
func ParseIndex(content []byte) (Index, error) {
	var index Index
	if err := json.Unmarshal(content, &index); err != nil {
		return Index{}, fmt.Errorf("the index of the directory: %w", err)
	}
	if index.Schema != SchemaVersion {
		return Index{}, fmt.Errorf("the index of the directory has schema %d, not %d", index.Schema, SchemaVersion)
	}
	if len(index.Plugins) > maxPlugins {
		return Index{}, fmt.Errorf("the index of the directory lists more than %d plugins", maxPlugins)
	}
	for name, sum := range index.Plugins {
		if decoded, err := hex.DecodeString(sum); !isPluginName(name) || err != nil || len(decoded) != sha256.Size {
			return Index{}, fmt.Errorf("the index of the directory has a bad entry for %q", name)
		}
	}
	return index, nil
}

// SHA256 is the checksum of a file, as the index writes it.
func SHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Catalog keeps the online directory in a cache.
type Catalog struct {
	// Directory holds the cache.
	Directory string
	// Shipped are the files of the profiles that the application ships, by
	// plugin name: the cache keeps only the files that differ from them, and is
	// dropped when they change, since an older file of the directory would
	// otherwise count before the newer one that the application ships.
	Shipped map[string][]byte
	// URL is the online directory; empty means CatalogURL.
	URL string
	// Client fetches the files; nil means a client with a timeout.
	Client *http.Client
}

// shippedDigest tells the profiles that the application ships apart from those
// of another version of it.
func (c *Catalog) shippedDigest() string {
	var lines strings.Builder
	for _, name := range slices.Sorted(maps.Keys(c.Shipped)) {
		fmt.Fprintf(&lines, "%s %s\n", name, SHA256(c.Shipped[name]))
	}
	return SHA256([]byte(lines.String()))
}

// cachedIndex is the index that the cache holds, if it belongs to the profiles
// that the application ships now.
func (c *Catalog) cachedIndex() (Index, bool) {
	if c == nil || c.Directory == "" {
		return Index{}, false
	}
	digest, err := readFile(filepath.Join(c.Directory, shippedFile))
	if err != nil || strings.TrimSpace(string(digest)) != c.shippedDigest() {
		return Index{}, false
	}
	content, err := readFile(filepath.Join(c.Directory, indexFile))
	if err != nil {
		return Index{}, false
	}
	index, err := ParseIndex(content)
	return index, err == nil
}

// Load reads the profiles that the cache holds, by plugin name.
func (c *Catalog) Load() (map[string]Document, []Problem) {
	index, ok := c.cachedIndex()
	if !ok {
		return nil, nil
	}
	documents := make(map[string]Document)
	var problems []Problem
	for _, name := range slices.Sorted(maps.Keys(index.Plugins)) {
		sum := index.Plugins[name]
		if shipped, ok := c.Shipped[name]; ok && SHA256(shipped) == sum {
			continue
		}
		content, err := readFile(filepath.Join(c.Directory, pluginsDir, name+".json"))
		if err != nil || SHA256(content) != sum {
			if !errors.Is(err, fs.ErrNotExist) {
				problems = append(problems, Problem{Path: name, Message: "the cached file does not match the index"})
			}
			continue
		}
		d, found, err := parseCatalogFile(name, content)
		problems = append(problems, found...)
		if err != nil {
			problems = append(problems, Problem{Path: name, Message: err.Error()})
			continue
		}
		documents[name] = d
	}
	return documents, problems
}

// parseCatalogFile reads the file of the plugin name of the directory, which
// has to say that it describes that plugin.
func parseCatalogFile(name string, content []byte) (Document, []Problem, error) {
	d, found, err := ParseDocument(content)
	for i := range found {
		found[i].Path = strings.TrimSuffix(name+": "+found[i].Path, ": ")
	}
	if err == nil && d.Plugin != name {
		err = fmt.Errorf("the file describes %q, not %q", d.Plugin, name)
	}
	return d, found, err
}

// Fresh reports whether the directory was fetched less than age ago, for the
// profiles that the application ships now.
func (c *Catalog) Fresh(age time.Duration, now time.Time) bool {
	if _, ok := c.cachedIndex(); !ok {
		return false
	}
	info, err := os.Stat(filepath.Join(c.Directory, indexFile))
	if err != nil {
		return false
	}
	since := now.Sub(info.ModTime())
	return since >= 0 && since < age
}

// Refresh fetches the directory into the cache: through the mirror prefix, such
// as https://gh-proxy.org/, when there is one. It fetches only the files that
// differ from those the application ships and the cache holds, checks each
// against the index, and changes the cache only when every one of them arrived.
// It reports whether what Load gives changed.
func (c *Catalog) Refresh(ctx context.Context, mirror string) (changed bool, err error) {
	if c == nil || c.Directory == "" {
		return false, errors.New("the online directory has no cache")
	}
	content, err := c.fetch(ctx, mirror, indexFile)
	if err != nil {
		return false, err
	}
	index, err := ParseIndex(content)
	if err != nil {
		return false, err
	}
	previous, _ := c.cachedIndex()
	downloaded := make(map[string][]byte)
	wanted := make(map[string]string)
	for _, name := range slices.Sorted(maps.Keys(index.Plugins)) {
		sum := index.Plugins[name]
		if shipped, ok := c.Shipped[name]; ok && SHA256(shipped) == sum {
			continue
		}
		wanted[name] = sum
		cached, err := readFile(filepath.Join(c.Directory, pluginsDir, name+".json"))
		if err == nil && SHA256(cached) == sum {
			continue
		}
		file, err := c.fetch(ctx, mirror, pluginsDir+"/"+name+".json")
		if err != nil {
			return false, err
		}
		if SHA256(file) != sum {
			return false, fmt.Errorf("the profile of %s does not match the index", name)
		}
		if _, _, err := parseCatalogFile(name, file); err != nil {
			return false, fmt.Errorf("the profile of %s: %w", name, err)
		}
		downloaded[name] = file
	}
	before := make(map[string]string)
	for name, sum := range previous.Plugins {
		if shipped, ok := c.Shipped[name]; !ok || SHA256(shipped) != sum {
			before[name] = sum
		}
	}
	if err := os.MkdirAll(filepath.Join(c.Directory, pluginsDir), 0o755); err != nil { //nolint:gosec // A cache of public files.
		return false, err
	}
	for name, file := range downloaded {
		if err := writeFile(filepath.Join(c.Directory, pluginsDir, name+".json"), file); err != nil {
			return false, err
		}
	}
	if err := writeFile(filepath.Join(c.Directory, shippedFile), []byte(c.shippedDigest()+"\n")); err != nil {
		return false, err
	}
	// The index goes last: the cache counts once it is written.
	if err := writeFile(filepath.Join(c.Directory, indexFile), content); err != nil {
		return false, err
	}
	// What the index no longer asks for goes, which costs nothing if it fails.
	entries, _ := os.ReadDir(filepath.Join(c.Directory, pluginsDir))
	for _, entry := range entries {
		if _, ok := wanted[strings.TrimSuffix(entry.Name(), ".json")]; !ok {
			_ = os.Remove(filepath.Join(c.Directory, pluginsDir, entry.Name()))
		}
	}
	return !maps.Equal(before, wanted), nil
}

// fetch gets a file of the directory.
func (c *Catalog) fetch(ctx context.Context, mirror, name string) ([]byte, error) {
	base := c.URL
	if base == "" {
		base = CatalogURL
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	address := update.MirrorURL(mirror, base+name)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", address, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: %s", address, response.Status)
	}
	content, err := readLimited(response.Body, maxFile)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", address, err)
	}
	return content, nil
}

// writeFile replaces a file at once, so that nobody reads half of it.
func writeFile(name string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(name), ".profile-*")
	if err != nil {
		return err
	}
	_, err = temporary.Write(content)
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), name)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
	}
	return err
}

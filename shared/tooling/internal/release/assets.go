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

package release

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Asset is one kind of file that a release uploads: the files that match Pattern
// in Dir, exactly Count of them. A versioned asset carries the version in its name.
type Asset struct {
	Dir       string
	Pattern   string
	Count     int
	Versioned bool
}

// Assets are the kinds of file that a release uploads, in the order of the release
// list: the .deb and the .tar.gz of each Linux architecture, the APK, the snap, and
// the Windows packages. Each architecture has its own .msix, and the bundle holds
// both; the release signs them (Sign the Windows packages). A kind that is missing,
// or doubled, stops the release before it is published.
var Assets = []Asset{
	{Dir: "dist", Pattern: "*.deb", Count: 1, Versioned: true},
	{Dir: "dist", Pattern: "*.tar.gz", Count: 1, Versioned: true},
	{Dir: "dist-arm64", Pattern: "*.deb", Count: 1, Versioned: true},
	{Dir: "dist-arm64", Pattern: "*.tar.gz", Count: 1, Versioned: true},
	{Dir: "dist-android", Pattern: "*.apk", Count: 1, Versioned: true},
	{Dir: "dist-snap", Pattern: "*.snap", Count: 1, Versioned: true},
	{Dir: "dist-windows", Pattern: "*.msix", Count: 2, Versioned: true},
	{Dir: "dist-windows", Pattern: "*.msixbundle", Count: 1, Versioned: true},
}

// CheckAssets returns the files that the release of version uploads, in the order
// of Assets, relative to root and with slashes. It reports an error when a kind does
// not have the count it needs, when a versioned file does not carry version in its
// name, or when two files have the same name.
func CheckAssets(root, version string) ([]string, error) {
	if !versionPattern.MatchString(version) {
		return nil, fmt.Errorf("Not a version (major.minor.patch, without v): %s", version)
	}
	var files []string
	seen := map[string]string{}
	for _, asset := range Assets {
		matches, err := filepath.Glob(filepath.Join(root, asset.Dir, asset.Pattern))
		if err != nil {
			return nil, err
		}
		if len(matches) != asset.Count {
			found := ""
			if len(matches) > 0 {
				found = " (" + strings.Join(baseNames(matches), ", ") + ")"
			}
			return nil, fmt.Errorf("%s/%s: %d files, want %d%s", asset.Dir, asset.Pattern, len(matches), asset.Count, found)
		}
		for _, match := range matches {
			name := filepath.Base(match)
			shown := filepath.Join(asset.Dir, name)
			if asset.Versioned && !strings.Contains(name, version) {
				return nil, fmt.Errorf("%s does not carry the version %s", shown, version)
			}
			if other, ok := seen[name]; ok {
				return nil, fmt.Errorf("%s and %s have the same name", other, shown)
			}
			seen[name] = shown
			rel, err := filepath.Rel(root, match)
			if err != nil {
				return nil, err
			}
			files = append(files, filepath.ToSlash(rel))
		}
	}
	return files, nil
}

// baseNames returns the file names of the paths.
func baseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = filepath.Base(path)
	}
	return names
}

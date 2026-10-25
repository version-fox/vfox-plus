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

package buildmsix

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// msixManifest is the subset of AppxManifest.xml that validation checks.
type msixManifest struct {
	Identity msixIdentity `xml:"Identity"`
}

type msixIdentity struct {
	Name                  string `xml:"Name,attr"`
	Publisher             string `xml:"Publisher,attr"`
	Version               string `xml:"Version,attr"`
	ProcessorArchitecture string `xml:"ProcessorArchitecture,attr"`
}

// bundleManifest is the subset of AppxBundleManifest.xml that validation checks.
type bundleManifest struct {
	Identity bundleIdentity  `xml:"Identity"`
	Packages []bundlePackage `xml:"Packages>Package"`
}

type bundleIdentity struct {
	Name      string `xml:"Name,attr"`
	Publisher string `xml:"Publisher,attr"`
	Version   string `xml:"Version,attr"`
}

type bundlePackage struct {
	Type         string `xml:"Type,attr"`
	Version      string `xml:"Version,attr"`
	Architecture string `xml:"Architecture,attr"`
	FileName     string `xml:"FileName,attr"`
	Offset       int64  `xml:"Offset,attr"`
	Size         int64  `xml:"Size,attr"`
}

// blockMapDoc is AppxBlockMap.xml.
type blockMapDoc struct {
	Files []blockMapEntry `xml:"File"`
}

type blockMapEntry struct {
	Name    string         `xml:"Name,attr"`
	Size    int64          `xml:"Size,attr"`
	LfhSize int64          `xml:"LfhSize,attr"`
	Blocks  []blockMapHash `xml:"Block"`
}

type blockMapHash struct {
	Hash string `xml:"Hash,attr"`
}

var versionSuffix = regexp.MustCompile(`_(\d+\.\d+\.\d+)(?:_([^_]+))?\.(?:msix|msixbundle)$`)

// ValidateDir validates every .msix and .msixbundle directly inside dir: ZIP
// layout, manifests, block maps, bundle offsets and version consistency.
// It reports the first problem found.
func ValidateDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Printf("validate-msix: %d entries in %s: %s\n", len(names), dir, strings.Join(names, ", "))
	var packages []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".msix") && !strings.HasSuffix(name, ".msixbundle") ||
			strings.HasSuffix(name, ".msixbundle") {
			packages = append(packages, filepath.Join(dir, name))
		}
	}
	if len(packages) == 0 {
		return fmt.Errorf("no .msix or .msixbundle in %s", dir)
	}
	sort.Strings(packages)
	for _, path := range packages {
		data, err := os.ReadFile(path) //nolint:gosec // The caller names the validation input.
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".msixbundle") {
			if err := ValidateBundle(data, path); err != nil {
				return err
			}
		} else if err := ValidateMsix(data, path); err != nil {
			return err
		}
		fmt.Println("validated", path)
	}
	return nil
}

// ValidateMsix validates one .msix package.
func ValidateMsix(data []byte, source string) error {
	files, err := checkZipLayout(data, source, true)
	if err != nil {
		return err
	}
	for _, want := range []string{"AppxManifest.xml", "[Content_Types].xml", "AppxBlockMap.xml"} {
		if _, ok := files[want]; !ok {
			return fmt.Errorf("%s misses %s", source, want)
		}
	}
	var manifest msixManifest
	if err := xml.Unmarshal(files["AppxManifest.xml"], &manifest); err != nil {
		return fmt.Errorf("%s has an invalid AppxManifest.xml: %w", source, err)
	}
	id := manifest.Identity
	if id.Name == "" || id.Publisher == "" || id.Version == "" || id.ProcessorArchitecture == "" {
		return fmt.Errorf("%s manifest identity is incomplete: %+v", source, id)
	}
	switch id.ProcessorArchitecture {
	case "x64", "x86", "arm", "arm64", "neutral":
	default:
		return fmt.Errorf("%s has an unknown architecture: %s", source, id.ProcessorArchitecture)
	}
	if err := checkBlockMap(data, source, files); err != nil {
		return err
	}
	return checkFilenameVersion(source, id.Version, "")
}

// ValidateBundle validates one regular .msixbundle.
func ValidateBundle(data []byte, source string) error {
	files, err := checkZipLayout(data, source, false)
	if err != nil {
		return err
	}
	for _, want := range []string{"AppxMetadata/AppxBundleManifest.xml", "[Content_Types].xml", "AppxBlockMap.xml"} {
		if _, ok := files[want]; !ok {
			return fmt.Errorf("%s misses %s", source, want)
		}
	}
	var manifest bundleManifest
	if err := xml.Unmarshal(files["AppxMetadata/AppxBundleManifest.xml"], &manifest); err != nil {
		return fmt.Errorf("%s has an invalid AppxBundleManifest.xml: %w", source, err)
	}
	if manifest.Identity.Name == "" || manifest.Identity.Publisher == "" || manifest.Identity.Version == "" {
		return fmt.Errorf("%s bundle identity is incomplete: %+v", source, manifest.Identity)
	}
	if len(manifest.Packages) == 0 {
		return fmt.Errorf("%s lists no packages", source)
	}
	headers, err := parseLocalHeaders(data)
	if err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	for _, pkg := range manifest.Packages {
		info, ok := headers[pkg.FileName]
		if !ok {
			return fmt.Errorf("%s lists %s, which is not in the bundle", source, pkg.FileName)
		}
		if info.dataSize != pkg.Size {
			return fmt.Errorf("%s lists %s with size %d, want %d", source, pkg.FileName, pkg.Size, info.dataSize)
		}
		if info.dataOffset != pkg.Offset {
			return fmt.Errorf("%s lists %s with offset %d, want %d", source, pkg.FileName, pkg.Offset, info.dataOffset)
		}
	}
	if err := checkBlockMap(data, source, files); err != nil {
		return err
	}
	for _, pkg := range manifest.Packages {
		if err := checkFilenameVersion(pkg.FileName, pkg.Version, source); err != nil {
			return err
		}
	}
	return checkFilenameVersion(source, manifest.Identity.Version, "")
}

// checkZipLayout checks the deterministic ZIP layout: no compression and a
// single timestamp for every entry. sorted reports whether the local entries
// must be in name order (.msix payloads are sorted; bundle payloads come
// first so their offsets are stable).
func checkZipLayout(data []byte, source string, sorted bool) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%s is not a ZIP: %w", source, err)
	}
	files := map[string][]byte{}
	var names []string
	var modified string
	for _, file := range reader.File {
		if file.Method != zip.Store {
			return nil, fmt.Errorf("%s stores %s compressed: packages use Store", source, file.Name)
		}
		stamp := file.Modified.UTC().Format("2006-01-02 15:04:05")
		if modified == "" {
			modified = stamp
		} else if stamp != modified {
			return nil, fmt.Errorf("%s has mixed timestamps: %s vs %s", source, modified, stamp)
		}
		content, err := readZipFile(reader, file.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		files[file.Name] = content
		names = append(names, file.Name)
	}
	if sorted && !sort.StringsAreSorted(names) {
		return nil, fmt.Errorf("%s entries are not sorted", source)
	}
	if _, err := parseLocalHeaders(data); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	return files, nil
}

func readZipFile(reader *zip.Reader, name string) ([]byte, error) {
	for _, file := range reader.File {
		if file.Name == name {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			_, err = buf.ReadFrom(rc)
			closeErr := rc.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("%s is missing", name)
}

// checkBlockMap verifies AppxBlockMap.xml against the ZIP: it must list every
// entry except itself, with matching sizes, header sizes and block hashes.
func checkBlockMap(data []byte, source string, files map[string][]byte) error {
	raw, ok := files["AppxBlockMap.xml"]
	if !ok {
		return fmt.Errorf("%s misses AppxBlockMap.xml", source)
	}
	var doc blockMapDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s has an invalid AppxBlockMap.xml: %w", source, err)
	}
	headers, err := parseLocalHeaders(data)
	if err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	listed := map[string]bool{}
	for _, entry := range doc.Files {
		listed[entry.Name] = true
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		content, ok := files[name]
		if !ok {
			return fmt.Errorf("%s block map lists %s, which is not packaged", source, entry.Name)
		}
		if entry.Size != int64(len(content)) {
			return fmt.Errorf("%s block map lists %s with size %d, want %d", source, entry.Name, entry.Size, len(content))
		}
		info, ok := headers[name]
		if !ok {
			return fmt.Errorf("%s has no local header for %s", source, entry.Name)
		}
		if entry.LfhSize != info.headerSize {
			return fmt.Errorf("%s block map lists %s with LfhSize %d, want %d", source, entry.Name, entry.LfhSize, info.headerSize)
		}
		want := blockHashes(content)
		if len(entry.Blocks) != len(want) {
			return fmt.Errorf("%s block map lists %s with %d blocks, want %d", source, entry.Name, len(entry.Blocks), len(want))
		}
		for i, block := range entry.Blocks {
			if block.Hash != want[i] {
				return fmt.Errorf("%s block map hash %d of %s does not match", source, i, entry.Name)
			}
		}
	}
	for name := range files {
		if name == "AppxBlockMap.xml" {
			continue
		}
		if !listed[name] && !listed[toWindowsPath(name)] {
			return fmt.Errorf("%s block map misses %s", source, name)
		}
	}
	return nil
}

// checkFilenameVersion checks the manifest version against the version in the
// file name (<pkg>_<major>.<minor>.<patch>[_<arch>].msix[bundle]); names
// without a version carry no check. context names the enclosing bundle for
// errors about embedded packages, or is empty.
func checkFilenameVersion(filename, manifestVersion, context string) error {
	match := versionSuffix.FindStringSubmatch(filename)
	if match == nil {
		return nil
	}
	quad, err := ToQuadVersion(match[1])
	if err != nil {
		return err
	}
	if manifestVersion != quad {
		where := filename
		if context != "" {
			where = context + ": " + filename
		}
		return fmt.Errorf("%s manifest version is %s, want %s", where, manifestVersion, quad)
	}
	return nil
}

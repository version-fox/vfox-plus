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
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	if err := checkManifestOrder(files["AppxManifest.xml"], source); err != nil {
		return err
	}
	if err := checkMsixContentTypes(files, source); err != nil {
		return err
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
	if err := checkBundleBlockMap(data, source, files); err != nil {
		return err
	}
	if err := checkBundleContentTypes(files, manifest.Packages, source); err != nil {
		return err
	}
	for _, pkg := range manifest.Packages {
		if err := checkFilenameVersion(pkg.FileName, pkg.Version, source); err != nil {
			return err
		}
	}
	// Each payload carries its own block map and is validated through it
	// when its slice is installed; validate it the same way here.
	for _, pkg := range manifest.Packages {
		payload, ok := files[pkg.FileName]
		if !ok {
			return fmt.Errorf("%s lists %s, which is not in the bundle", source, pkg.FileName)
		}
		if err := ValidateMsix(payload, source+": "+pkg.FileName); err != nil {
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
		if entry.Name == "[Content_Types].xml" {
			return fmt.Errorf("%s block map must not list [Content_Types].xml", source)
		}
		if err := verifyBlockEntry(entry, files, headers, source); err != nil {
			return err
		}
	}
	for name := range files {
		if name == "AppxBlockMap.xml" || name == "[Content_Types].xml" {
			continue
		}
		if !listed[name] && !listed[toWindowsPath(name)] {
			return fmt.Errorf("%s block map misses %s", source, name)
		}
	}
	return nil
}

// verifyBlockEntry checks one block map entry against the packaged file:
// presence, size, local header size and every block hash.
func verifyBlockEntry(entry blockMapEntry, files map[string][]byte, headers map[string]localInfo, source string) error {
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
	return nil
}

// checkBundleBlockMap verifies the bundle AppxBlockMap.xml: like
// MakeAppx output, it lists exactly the bundle manifest, whose payloads
// validate through their own embedded block maps.
func checkBundleBlockMap(data []byte, source string, files map[string][]byte) error {
	raw, ok := files["AppxBlockMap.xml"]
	if !ok {
		return fmt.Errorf("%s misses AppxBlockMap.xml", source)
	}
	var doc blockMapDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s has an invalid AppxBlockMap.xml: %w", source, err)
	}
	if len(doc.Files) != 1 || doc.Files[0].Name != `AppxMetadata\AppxBundleManifest.xml` {
		return fmt.Errorf("%s block map must list only AppxMetadata\\AppxBundleManifest.xml", source)
	}
	headers, err := parseLocalHeaders(data)
	if err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	return verifyBlockEntry(doc.Files[0], files, headers, source)
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

// packageChildOrder is the schema order of the direct children of Package:
// Windows rejects manifests with anything else there (such as a top-level
// Description, which belongs inside Properties).
var packageChildOrder = []string{"Identity", "Properties", "Resources", "Dependencies", "Capabilities", "Applications"}

// propertiesChildOrder is the schema order of the children of Properties.
var propertiesChildOrder = []string{"Framework", "DisplayName", "PublisherDisplayName", "Description", "Logo"}

// checkManifestOrder verifies the element order of AppxManifest.xml against
// the schema: unknown or misplaced elements fail Windows deployment with
// 0x80080204, which this check catches without Windows.
func checkManifestOrder(manifest []byte, source string) error {
	decoder := xml.NewDecoder(bytes.NewReader(manifest))
	var stack []string
	var pkgChildren, propChildren []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s has an invalid AppxManifest.xml: %w", source, err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			if end, ok := token.(xml.EndElement); ok && len(stack) > 0 {
				stack = stack[:len(stack)-1]
				_ = end
			}
			continue
		}
		stack = append(stack, start.Name.Local)
		switch len(stack) {
		case 2:
			if stack[0] == "Package" {
				pkgChildren = append(pkgChildren, start.Name.Local)
			}
		case 3:
			if stack[0] == "Package" && stack[1] == "Properties" {
				propChildren = append(propChildren, start.Name.Local)
			}
		}
	}
	if err := checkElementOrder(pkgChildren, packageChildOrder, "Package", source); err != nil {
		return err
	}
	return checkElementOrder(propChildren, propertiesChildOrder, "Properties", source)
}

func checkElementOrder(got, want []string, parent, source string) error {
	previous := -1
	for _, name := range got {
		index := slices.Index(want, name)
		if index < 0 {
			return fmt.Errorf("%s has unexpected %s element %q", source, parent, name)
		}
		if index <= previous {
			return fmt.Errorf("%s has %s element %q out of order", source, parent, name)
		}
		previous = index
	}
	return nil
}

// contentTypesDoc is [Content_Types].xml of an OPC container.
type contentTypesDoc struct {
	Defaults []struct {
		Extension   string `xml:"Extension,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Default"`
	Overrides []struct {
		PartName    string `xml:"PartName,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Override"`
}

// resolve returns the content type of a part: an Override wins over the
// Default of its extension, as the OPC specification says.
func (d contentTypesDoc) resolve(part string) (string, bool) {
	part = strings.TrimPrefix(part, "/")
	for _, override := range d.Overrides {
		if strings.TrimPrefix(override.PartName, "/") == part {
			return override.ContentType, true
		}
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(part)), ".")
	for _, def := range d.Defaults {
		if strings.EqualFold(def.Extension, extension) {
			return def.ContentType, true
		}
	}
	return "", false
}

// parseContentTypes parses [Content_Types].xml of files.
func parseContentTypes(files map[string][]byte, source string) (contentTypesDoc, error) {
	var doc contentTypesDoc
	raw, ok := files["[Content_Types].xml"]
	if !ok {
		return doc, fmt.Errorf("%s misses [Content_Types].xml", source)
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return doc, fmt.Errorf("%s has an invalid [Content_Types].xml: %w", source, err)
	}
	return doc, nil
}

// checkMsixContentTypes verifies that every part of a .msix package resolves
// to a content type and that the footprint files use the MSIX ones.
func checkMsixContentTypes(files map[string][]byte, source string) error {
	doc, err := parseContentTypes(files, source)
	if err != nil {
		return err
	}
	for part, want := range map[string]string{
		"AppxManifest.xml": "application/vnd.ms-appx.manifest+xml",
		"AppxBlockMap.xml": "application/vnd.ms-appx.blockmap+xml",
	} {
		got, ok := doc.resolve(part)
		if !ok {
			return fmt.Errorf("%s declares no content type for %s", source, part)
		}
		if got != want {
			return fmt.Errorf("%s declares content type %q for %s, want %q", source, got, part, want)
		}
	}
	for name := range files {
		if name == "AppxManifest.xml" || name == "AppxBlockMap.xml" || name == "[Content_Types].xml" {
			continue
		}
		if _, ok := doc.resolve(name); !ok {
			return fmt.Errorf("%s declares no content type for %s", source, name)
		}
	}
	return nil
}

// checkBundleContentTypes verifies that every payload package of a bundle
// resolves to application/vnd.ms-appx. Windows recognises bundle payloads by
// that content type; when a payload resolves to something else it is treated
// as an ordinary part, which then fails block map validation with
// APPX_E_BLOCK_MAP_INVALID (0x80080205).
func checkBundleContentTypes(files map[string][]byte, packages []bundlePackage, source string) error {
	doc, err := parseContentTypes(files, source)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		got, ok := doc.resolve(pkg.FileName)
		if !ok {
			return fmt.Errorf("%s declares no content type for %s", source, pkg.FileName)
		}
		if got != "application/vnd.ms-appx" {
			return fmt.Errorf("%s declares content type %q for %s, want %q", source, got, pkg.FileName, "application/vnd.ms-appx")
		}
	}
	return nil
}

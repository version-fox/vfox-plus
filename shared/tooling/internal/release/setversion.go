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
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SetVersion writes version into the files that carry it, in the checkout at root:
// mygo/VERSION, the Version constant of mygo/ui/app.go, and the release list of the
// metainfo. The metainfo gets an entry dated date unless it has one for version
// already. The files are read back to check them, and the metainfo must stay
// well-formed XML.
func SetVersion(root, version, date string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("Not a version (major.minor.patch, without v): %s", version)
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("The date of the release must be YYYY-MM-DD: %s", date)
	}
	path := func(file string) string { return filepath.Join(root, filepath.FromSlash(file)) }

	if err := os.WriteFile(path(VersionFile), []byte(version+"\n"), 0o644); err != nil {
		return err
	}

	app, err := os.ReadFile(path(AppFile))
	if err != nil {
		return err
	}
	if n := len(versionConst.FindAllIndex(app, -1)); n != 1 {
		return fmt.Errorf("%s must have one Version constant, found %d", AppFile, n)
	}
	app = versionConst.ReplaceAllLiteral(app, []byte(`const Version = "`+version+`"`))
	if err := os.WriteFile(path(AppFile), app, 0o644); err != nil {
		return err
	}

	metainfo, err := os.ReadFile(path(MetainfoFile))
	if err != nil {
		return err
	}
	if !bytes.Contains(metainfo, []byte(`<release version="`+version+`"`)) {
		const releases = "<releases>\n"
		if !bytes.Contains(metainfo, []byte(releases)) {
			return fmt.Errorf("%s has no <releases> list", MetainfoFile)
		}
		entry := `    <release version="` + version + `" date="` + date + `"/>` + "\n"
		metainfo = bytes.Replace(metainfo, []byte(releases), []byte(releases+entry), 1)
		if err := os.WriteFile(path(MetainfoFile), metainfo, 0o644); err != nil {
			return err
		}
	}

	return verify(root, version)
}

// verify reads the files back: they must say version, and the metainfo must be
// well-formed XML.
func verify(root, version string) error {
	read := func(file string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		return string(data), err
	}
	if err := checkFiles(version, read); err != nil {
		return err
	}
	metainfo, err := read(MetainfoFile)
	if err != nil {
		return err
	}
	return wellFormed(metainfo)
}

// wellFormed reports an error unless text is well-formed XML.
func wellFormed(text string) error {
	dec := xml.NewDecoder(strings.NewReader(text))
	for {
		_, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s is not well-formed XML: %w", MetainfoFile, err)
		}
	}
}

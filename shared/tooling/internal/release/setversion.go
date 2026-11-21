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
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// SetVersion writes version into the files that carry it, in the checkout at root:
// mygo/VERSION, the Version constant of mygo/ui/app.go, the version of the phone app's
// configuration (PhoneConfigFile), and the release list of the metainfo. The metainfo
// gets an entry dated date unless it has one for version already. The files are read
// back to check them, and the metainfo must stay well-formed XML.
func SetVersion(root, version, date string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("Not a version (major.minor.patch, without v): %s", version)
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("The date of the release must be YYYY-MM-DD: %s", date)
	}
	path := func(file string) string { return filepath.Join(root, filepath.FromSlash(file)) }

	// Every file is read and changed in memory first, so that a file that cannot be
	// changed leaves the checkout as it was.
	app, err := os.ReadFile(path(AppFile))
	if err != nil {
		return err
	}
	if n := len(versionConst.FindAllIndex(app, -1)); n != 1 {
		return fmt.Errorf("%s must have one Version constant, found %d", AppFile, n)
	}
	app = versionConst.ReplaceAllLiteral(app, []byte(`const Version = "`+version+`"`))

	phone, err := os.ReadFile(path(PhoneConfigFile))
	if err != nil {
		return err
	}
	phone, err = setPhoneVersion(phone, version)
	if err != nil {
		return fmt.Errorf("%s: %w", PhoneConfigFile, err)
	}

	metainfo, err := os.ReadFile(path(MetainfoFile))
	if err != nil {
		return err
	}
	metainfoChanged := false
	if !bytes.Contains(metainfo, []byte(`<release version="`+version+`"`)) {
		const releases = "<releases>\n"
		if !bytes.Contains(metainfo, []byte(releases)) {
			return fmt.Errorf("%s has no <releases> list", MetainfoFile)
		}
		entry := `    <release version="` + version + `" date="` + date + `"/>` + "\n"
		metainfo = bytes.Replace(metainfo, []byte(releases), []byte(releases+entry), 1)
		metainfoChanged = true
	}

	if err := os.WriteFile(path(VersionFile), []byte(version+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(path(AppFile), app, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(path(PhoneConfigFile), phone, 0o644); err != nil {
		return err
	}
	if metainfoChanged {
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

// phoneVersionField matches the "version" line of the phone app's configuration: one line
// of the file, as the configuration is written by hand, with its indentation kept.
var phoneVersionField = regexp.MustCompile(`(?m)^([ \t]*)"version"([ \t]*):([ \t]*)"[^"]*"`)

// phoneConfig is the part of the phone app's configuration that the release reads.
type phoneConfig struct {
	Version string `json:"version"`
}

// phoneConfigVersion returns the version of the phone app's configuration text.
func phoneConfigVersion(text string) (string, error) {
	var config phoneConfig
	if err := json.Unmarshal([]byte(text), &config); err != nil {
		return "", fmt.Errorf("cannot read the version, which is JSON with a string: %w", err)
	}
	return config.Version, nil
}

// setPhoneVersion returns the configuration with its version set to version, and nothing
// else changed: the text keeps its order and layout, only the value of the one "version"
// line moves, and the result must decode to the same fields as before.
func setPhoneVersion(config []byte, version string) ([]byte, error) {
	before, err := phoneConfigVersion(string(config))
	if err != nil {
		return nil, err
	}
	if n := len(phoneVersionField.FindAllIndex(config, -1)); n != 1 {
		return nil, fmt.Errorf("must have one version field, found %d", n)
	}
	updated := phoneVersionField.ReplaceAll(config, []byte(`${1}"version"${2}:${3}"`+version+`"`))
	if after, err := phoneConfigVersion(string(updated)); err != nil || after != version {
		return nil, fmt.Errorf("the version did not change to %s (it is %q, was %q)", version, after, before)
	}
	if err := sameFieldsExceptVersion(config, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// sameFieldsExceptVersion reports an error unless the two configurations decode to the same
// fields, without the version.
func sameFieldsExceptVersion(old, updated []byte) error {
	var a, b map[string]any
	if err := json.Unmarshal(old, &a); err != nil {
		return err
	}
	if err := json.Unmarshal(updated, &b); err != nil {
		return err
	}
	delete(a, "version")
	delete(b, "version")
	if !reflect.DeepEqual(a, b) {
		return errors.New("the change would alter other fields of the configuration")
	}
	return nil
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

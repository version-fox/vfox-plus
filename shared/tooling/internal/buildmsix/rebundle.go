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
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// signatureEntry is the part that signing adds to an .msix package.
const signatureEntry = "AppxSignature.p7x"

// RunRebundle is `vfxtool rebundle`. It puts signed packages into their bundle
// again, after the signing has made them larger (see Rebundle). args are the
// command line without the program name:
//
//	rebundle --package NAME --version VERSION --publisher PUBLISHER [--identity ID] <directory>
//
// The packages are the .msix files of x64 and arm64 in the directory. A package
// that is not signed is refused: its bundle would not install.
func RunRebundle(args []string) error {
	flags := flag.NewFlagSet("rebundle", flag.ContinueOnError)
	pkg := flags.String("package", "", "package file name prefix, e.g. vfox-plus (required)")
	version := flags.String("version", "", "application version, e.g. 0.1.0 (required)")
	identity := flags.String("identity", DefaultIdentity, "package identity name")
	publisher := flags.String("publisher", "", "publisher of the signed packages: the subject of their certificate (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("Usage: rebundle --package NAME --version VERSION --publisher PUBLISHER [--identity ID] <directory>")
	}
	if *pkg == "" || *version == "" || *publisher == "" {
		return fmt.Errorf("--package, --version and --publisher are required")
	}
	dir := flags.Arg(0)
	var arches []string
	for _, arch := range []string{"x64", "arm64"} {
		name := filepath.Join(dir, fmt.Sprintf("%s_%s_%s.msix", *pkg, *version, arch))
		data, err := os.ReadFile(name) //nolint:gosec // The caller names the directory of its own packages.
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		signed, err := isSigned(data)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !signed {
			return fmt.Errorf("%s is not signed: sign the packages before they go into the bundle again", name)
		}
		arches = append(arches, arch)
	}
	if len(arches) == 0 {
		return fmt.Errorf("no %s_%s_<arch>.msix in %s", *pkg, *version, dir)
	}
	path, err := Rebundle(RebundleConfig{
		Dir:           dir,
		Package:       *pkg,
		Version:       *version,
		Identity:      *identity,
		Publisher:     *publisher,
		Architectures: arches,
	})
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// isSigned reports whether the .msix package in data has the signature part.
func isSigned(data []byte) (bool, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false, err
	}
	for _, file := range reader.File {
		if file.Name == signatureEntry {
			return true, nil
		}
	}
	return false, nil
}

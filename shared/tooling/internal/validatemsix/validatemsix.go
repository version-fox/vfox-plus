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

// Package validatemsix implements `vfxtool validate-msix`, which checks the
// .msix and .msixbundle files without the Windows SDK: ZIP layout, manifests,
// block maps, bundle offsets, version consistency, the publisher and both of the
// architectures of a bundle. Without a --publisher the packages are unsigned and
// carry the unsigned publisher; with one, they carry that publisher, the subject
// of the certificate that signs them. With --self-sign it also checks the files
// that `vfxtool sign-msix` needs next to the bundle, which the CI artifact of
// mygo.yml has and a release does not.
package validatemsix

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
)

// selfSignInputs are the files that sign-msix needs next to the unsigned bundle: the
// executables of both architectures and vfxtool.exe (the CI artifact of mygo.yml,
// Stage the Windows self-sign inputs).
var selfSignInputs = []string{
	"stage-x64/vfox-plus.exe",
	"stage-arm64/vfox-plus.exe",
	"vfxtool.exe",
}

const usage = "Usage: validate-msix [--self-sign] [--publisher PUBLISHER] <directory>"

// Run validates the packages in a directory. args are the command line
// without the program name:
//
//	validate-msix [--self-sign] [--publisher PUBLISHER] <directory>
func Run(args []string) error {
	flags := flag.NewFlagSet("validate-msix", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	selfSign := flags.Bool("self-sign", false, "also check the files that sign-msix needs next to the bundle")
	publisher := flags.String("publisher", "", "the publisher that the packages carry: the subject of their certificate")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%s: %w", usage, err)
	}
	if flags.NArg() != 1 || flags.Arg(0) == "" {
		return fmt.Errorf("%s", usage)
	}
	dir := flags.Arg(0)
	if err := buildmsix.ValidateDirFor(dir, *publisher); err != nil {
		return err
	}
	if *selfSign {
		return checkSelfSignInputs(dir)
	}
	return nil
}

// checkSelfSignInputs reports an error unless every self-sign input is in dir.
func checkSelfSignInputs(dir string) error {
	for _, rel := range selfSignInputs {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() {
			return fmt.Errorf("missing self-sign input: %s", rel)
		}
	}
	return nil
}

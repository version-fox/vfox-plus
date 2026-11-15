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
// unsigned .msix and .msixbundle files without the Windows SDK: ZIP layout,
// manifests, block maps, bundle offsets, version consistency, the publisher of the
// unsigned bundle and both of its architectures. With --self-sign it also checks
// the files that `vfxtool sign-msix` needs next to the bundle, which a release has
// and a sandbox does not.
package validatemsix

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
)

// selfSignInputs are the files that sign-msix needs next to the unsigned bundle: the
// executables of both architectures and vfxtool.exe (release.yml, Stage the Windows
// self-sign inputs).
var selfSignInputs = []string{
	"stage-x64/vfox-plus.exe",
	"stage-arm64/vfox-plus.exe",
	"vfxtool.exe",
}

// Run validates the packages in a directory. args are the command line
// without the program name:
//
//	validate-msix [--self-sign] <directory>
func Run(args []string) error {
	const usage = "Usage: validate-msix [--self-sign] <directory>"
	selfSign := len(args) > 0 && args[0] == "--self-sign"
	if selfSign {
		args = args[1:]
	}
	if len(args) != 1 || args[0] == "" || args[0][0] == '-' {
		return fmt.Errorf("%s", usage)
	}
	if err := buildmsix.ValidateDir(args[0]); err != nil {
		return err
	}
	if selfSign {
		return checkSelfSignInputs(args[0])
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

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
// manifests, block maps, bundle offsets and version consistency.
package validatemsix

import (
	"fmt"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
)

// Run validates the packages in a directory. args are the command line
// without the program name:
//
//	validate-msix <directory>
func Run(args []string) error {
	if len(args) != 1 || args[0] == "" || args[0][0] == '-' {
		return fmt.Errorf("Usage: validate-msix <directory>")
	}
	return buildmsix.ValidateDir(args[0])
}

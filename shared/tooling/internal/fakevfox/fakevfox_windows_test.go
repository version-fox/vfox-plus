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

//go:build windows

package fakevfox

import "testing"

// The application runs the commands that are not about a project in the root of the system drive, C:\, and
// the fake vfox has to take that for the root, as the CLI contract has it.
func TestTheRootOfADriveIsTheRootOnWindows(t *testing.T) {
	for dir, want := range map[string]bool{
		`C:\`:       true,
		`D:\`:       true,
		`C:\Users`:  false,
		`C:\Users\`: false,
		`C:`:        false, // The current directory of the drive, which is not the root.
	} {
		if got := isRoot(dir); got != want {
			t.Errorf("isRoot(%q) = %t, want %t", dir, got, want)
		}
	}
}

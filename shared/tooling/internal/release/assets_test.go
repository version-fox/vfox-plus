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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// releaseFiles are the files of a release of version, as the workflow builds them.
func releaseFiles(version string) []string {
	return []string{
		"dist/vfox-plus_" + version + "_amd64.deb",
		"dist/vfox-plus-" + version + "-linux-x64.tar.gz",
		"dist-arm64/vfox-plus_" + version + "_arm64.deb",
		"dist-arm64/vfox-plus-" + version + "-linux-arm64.tar.gz",
		"dist-android/vfox-plus_" + version + ".apk",
		"dist-snap/vfox-plus_" + version + "_amd64.snap",
		"dist-windows/vfox-plus_" + version + "_x64.msix",
		"dist-windows/vfox-plus_" + version + "_arm64.msix",
		"dist-windows/vfox-plus_" + version + ".msixbundle",
	}
}

// makeFiles creates the files, empty, under root.
func makeFiles(t *testing.T, root string, files []string) {
	t.Helper()
	for _, file := range files {
		writeFile(t, root, file, "")
	}
}

func TestCheckAssetsListsEveryFileOnce(t *testing.T) {
	root := t.TempDir()
	makeFiles(t, root, releaseFiles("1.0.0"))

	got, err := CheckAssets(root, "1.0.0")
	if err != nil {
		t.Fatalf("CheckAssets: %v", err)
	}
	want := releaseFiles("1.0.0")
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("CheckAssets = %v\nwant %v", got, want)
	}
}

func TestCheckAssetsRefusesAReleaseThatIsNotComplete(t *testing.T) {
	cases := []struct {
		name    string
		version string
		change  func(t *testing.T, root string)
		wantErr string
	}{
		{name: "the snap is missing", version: "1.0.0",
			change: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, "dist-snap", "vfox-plus_1.0.0_amd64.snap")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "dist-snap/*.snap: 0 files, want 1"},
		{name: "a .deb is doubled", version: "1.0.0",
			change: func(t *testing.T, root string) {
				writeFile(t, root, "dist/vfox-plus_1.0.0_arm64.deb", "")
			},
			wantErr: "dist/*.deb: 2 files, want 1"},
		{name: "one architecture of the .msix is missing", version: "1.0.0",
			change: func(t *testing.T, root string) {
				if err := os.Remove(filepath.Join(root, "dist-windows", "vfox-plus_1.0.0_arm64.msix")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "dist-windows/*.msix: 1 files, want 2"},
		{name: "a file carries another version", version: "1.0.0",
			change: func(t *testing.T, root string) {
				if err := os.Rename(filepath.Join(root, "dist-android", "vfox-plus_1.0.0.apk"),
					filepath.Join(root, "dist-android", "vfox-plus_0.9.0.apk")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "dist-android/vfox-plus_0.9.0.apk does not carry the version 1.0.0"},
		{name: "the version is not major.minor.patch", version: "1.0",
			wantErr: "Not a version"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			makeFiles(t, root, releaseFiles("1.0.0"))
			if c.change != nil {
				c.change(t, root)
			}
			_, err := CheckAssets(root, c.version)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("CheckAssets error = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}

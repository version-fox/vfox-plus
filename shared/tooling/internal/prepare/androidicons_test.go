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

package prepare

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeAndroidIcons has the symbols that the tests of the Android drawables ask for:
// one path, two paths, a filled variant, a box that is not 24, an origin that is not
// 0,0 and a shape that is not a path.
const fakeAndroidIcons = `{
  "icons": {
    "qr-code-scanner-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M1 1h2z\"/>"},
    "flashlight-on-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M2 2h2z\"/><path fill=\"currentColor\" d=\"m5 5l.5.5z\"/>"},
    "flashlight-on-rounded": {"body": "<path fill=\"currentColor\" d=\"M3 3h2z\"/>"},
    "arrow-forward-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M4 4h2z\"/>"},
    "small-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M5 5h2z\"/>", "width": 20, "height": 20},
    "moved-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M6 6h2z\"/>", "left": 1},
    "round-outline-rounded": {"body": "<circle cx=\"12\" cy=\"12\" r=\"4\"/>"}
  },
  "width": 24,
  "height": 24
}`

func androidManifest(t *testing.T, icons map[string]string, filled, mirrored []string) iconManifest {
	t.Helper()
	return iconManifest{
		Source:  iconSource{Package: "@iconify-json/material-symbols", Version: "1.2.97"},
		Icons:   map[string]string{"back": "arrow_back"},
		Android: androidIcons{Icons: icons, Filled: filled, Mirrored: mirrored},
	}
}

func fakeSet(t *testing.T) iconSet {
	t.Helper()
	set, err := readIconSet(writeTarball(t, fakeAndroidIcons))
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestAndroidName(t *testing.T) {
	for name, want := range map[string]string{"close": "close", "scanQr": "scan_qr", "torchOn": "torch_on", "linkOff": "link_off", "a1B": "a1_b"} {
		if got := androidName(name); got != want {
			t.Errorf("androidName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestAndroidFilesAreTheVectorDrawablesOfTheSection(t *testing.T) {
	manifest := androidManifest(t,
		map[string]string{"scanQr": "qr_code_scanner", "torchOn": "flashlight_on", "forward": "arrow_forward"},
		[]string{"torchOn"}, []string{"forward"})
	files, problems := androidFiles(fakeSet(t), manifest)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if got := strings.Join(slices.Sorted(maps.Keys(files)), " "); got != "ic_forward.xml ic_scan_qr.xml ic_torch_on.xml ic_torch_on_fill.xml" {
		t.Fatalf("files: %s", got)
	}

	scan := files["ic_scan_qr.xml"]
	for _, want := range []string{
		"Copyright 2026 Han Li and contributors", // the header of every source file
		androidMarker,
		`Material Symbols Rounded "qr_code_scanner" (outlined), @iconify-json/material-symbols 1.2.97`,
		`android:width="24dp"`, `android:viewportHeight="24"`,
		`android:tint="?attr/colorControlNormal"`,
		`android:pathData="M1 1h2z"`,
	} {
		if !strings.Contains(scan, want) {
			t.Errorf("ic_scan_qr.xml lacks %q:\n%s", want, scan)
		}
	}
	if strings.Contains(scan, "autoMirrored") {
		t.Error("a drawable that is not mirrored says so")
	}
	if !strings.Contains(files["ic_forward.xml"], `android:autoMirrored="true"`) {
		t.Error("the mirrored drawable does not turn around")
	}
	// Two paths are two elements, and the filled variant is the filled symbol.
	if got := strings.Count(files["ic_torch_on.xml"], "<path"); got != 2 {
		t.Errorf("ic_torch_on.xml has %d paths, want 2", got)
	}
	if fill := files["ic_torch_on_fill.xml"]; !strings.Contains(fill, "(filled)") || !strings.Contains(fill, `android:pathData="M3 3h2z"`) {
		t.Errorf("the filled variant is not the filled symbol:\n%s", fill)
	}
}

func TestAnIconOfAnotherBoxKeepsIt(t *testing.T) {
	files, problems := androidFiles(fakeSet(t), androidManifest(t, map[string]string{"small": "small"}, nil, nil))
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	if got := files["ic_small.xml"]; !strings.Contains(got, `android:width="20dp"`) || !strings.Contains(got, `android:viewportWidth="20"`) {
		t.Errorf("the box of the icon is lost:\n%s", got)
	}
}

func TestAndroidFilesNameEveryIconThatCannotBeMade(t *testing.T) {
	manifest := androidManifest(t,
		map[string]string{"gone": "not_there", "moved": "moved", "round": "round", "scanQr": "qr_code_scanner"}, nil, nil)
	files, problems := androidFiles(fakeSet(t), manifest)
	if len(files) != 1 || files["ic_scan_qr.xml"] == "" {
		t.Errorf("the icons that work are still made: %v", files)
	}
	if len(problems) != 3 {
		t.Fatalf("problems: %v", problems)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"gone: the package has no icon not-there-outline-rounded", "moved: moved-outline-rounded starts somewhere else", "round: round-outline-rounded is not made of paths"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the problems do not say %q:\n%s", want, joined)
		}
	}
}

func TestWriteAndroidIconsKeepsTheHandMadeAndRemovesTheStaleOnes(t *testing.T) {
	dir := t.TempDir()
	handMade := filepath.Join(dir, "ic_mark.xml")
	if err := os.WriteFile(handMade, []byte("<vector/>"), 0o666); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "background.xml")
	if err := os.WriteFile(other, []byte("<shape/> "+androidMarker), 0o666); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "ic_old.xml")
	if err := os.WriteFile(stale, []byte("<!-- "+androidMarker+" -->"), 0o666); err != nil {
		t.Fatal(err)
	}
	manifest := androidManifest(t, map[string]string{"scanQr": "qr_code_scanner"}, nil, nil)
	written, err := writeAndroidIcons(fakeSet(t), manifest, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(written, " ") != "ic_scan_qr.xml" {
		t.Errorf("written: %v", written)
	}
	if _, err := os.Stat(filepath.Join(dir, "ic_scan_qr.xml")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(handMade); err != nil {
		t.Error("a hand-made drawable is gone")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("a file that is not an icon is gone")
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("the icon that the section no longer names is still there")
	}
	// Nothing is written when one icon cannot be made.
	broken := androidManifest(t, map[string]string{"scanQr": "qr_code_scanner", "gone": "not_there"}, nil, nil)
	other2 := t.TempDir()
	if _, err := writeAndroidIcons(fakeSet(t), broken, other2); err == nil || !strings.Contains(err.Error(), "gone") {
		t.Errorf("want an error that names gone, got %v", err)
	}
	if entries, _ := os.ReadDir(other2); len(entries) != 0 {
		t.Errorf("files of a failed run: %v", entries)
	}
}

func TestAndroidSectionIsChecked(t *testing.T) {
	shared := map[string]string{"back": "arrow_back"}
	good := androidIcons{Icons: map[string]string{"back": "arrow_back", "scanQr": "qr_code_scanner"}, Filled: []string{"scanQr"}, Mirrored: []string{"back"}}
	if err := good.validate(shared); err != nil {
		t.Fatal(err)
	}
	for name, section := range map[string]androidIcons{
		"a name in snake case":   {Icons: map[string]string{"scan_qr": "qr_code_scanner"}},
		"a name with a capital":  {Icons: map[string]string{"ScanQr": "qr_code_scanner"}},
		"a symbol with a dash":   {Icons: map[string]string{"scanQr": "qr-code-scanner"}},
		"a symbol of the other":  {Icons: map[string]string{"back": "arrow_forward"}},
		"filled that is no icon": {Icons: map[string]string{"scanQr": "qr_code_scanner"}, Filled: []string{"nothing"}},
		"mirrored that is none":  {Icons: map[string]string{"scanQr": "qr_code_scanner"}, Mirrored: []string{"nothing"}},
	} {
		if err := section.validate(shared); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestResolveRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "shared", "icons.json"), []byte("{}"), 0o666); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveRepositoryRoot(root); err != nil || got != root {
		t.Errorf("the flag: %q, %v", got, err)
	}
	if _, err := resolveRepositoryRoot(t.TempDir()); err == nil {
		t.Error("a directory without shared/icons.json is accepted")
	}
	nested := filepath.Join(root, "android", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	if got, err := resolveRepositoryRoot(""); err != nil || got != root {
		t.Errorf("located from below: %q, %v", got, err)
	}
}

func TestRunAndroidIconsRefusesWhatItCannotUse(t *testing.T) {
	if err := RunAndroidIcons([]string{"extra"}); err == nil {
		t.Error("an argument that is not a flag is accepted")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("ab", 64)
	manifest := `{"source": {"package": "@a/b", "version": "1.0.0", "sha512": "` + hash + `"}, "icons": {"home": "home"}}`
	if err := os.WriteFile(filepath.Join(root, "shared", "icons.json"), []byte(manifest), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := RunAndroidIcons([]string{"--root", root}); err == nil || !strings.Contains(err.Error(), `no "android" icons`) {
		t.Errorf("a manifest without the section: %v", err)
	}
}

// The drawables that are committed are the ones of the manifest: every icon of the
// "android" section has its file, from the package that the manifest pins, and no
// file that android-icons wrote is left over. (A new version of the package, or a
// new icon, needs `vfxtool android-icons`.)
func TestTheAndroidDrawablesOfTheRepository(t *testing.T) {
	manifest, err := readManifest(filepath.Join("..", "..", "..", "..", "shared", "icons.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Android.Icons) == 0 {
		t.Fatal(`shared/icons.json has no "android" icons`)
	}
	dir := filepath.Join("..", "..", "..", "..", "android", "app", "src", "main", "res", "drawable")
	want := map[string]string{}
	for name, symbol := range manifest.Android.Icons {
		want["ic_"+androidName(name)+".xml"] = `Material Symbols Rounded "` + symbol + `" (outlined), ` + manifest.Source.Package + " " + manifest.Source.Version
		if slices.Contains(manifest.Android.Filled, name) {
			want["ic_"+androidName(name)+"_fill.xml"] = `Material Symbols Rounded "` + symbol + `" (filled), ` + manifest.Source.Package + " " + manifest.Source.Version
		}
	}
	for file, line := range want {
		content, err := os.ReadFile(filepath.Join(dir, file)) //nolint:gosec // A file of the repository.
		if err != nil {
			t.Errorf("%s: %v (run `vfxtool android-icons`)", file, err)
			continue
		}
		if !strings.Contains(string(content), androidMarker) || !strings.Contains(string(content), line) {
			t.Errorf("%s is not %q (run `vfxtool android-icons`)", file, line)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join(dir, entry.Name())) //nolint:gosec // A file of the repository.
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := want[entry.Name()]; !ok && strings.Contains(string(content), androidMarker) {
			t.Errorf("%s was written by android-icons and no icon of the section needs it", entry.Name())
		}
	}
}

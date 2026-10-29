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

package kit

import (
	"bytes"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/assets"
)

const testSeed = 0xFFF28C1B

var loadIconsOnce sync.Once

// loadTestIcons registers the icons that `vfxtool prepare` collected.
func loadTestIcons(t testing.TB) {
	t.Helper()
	var err error
	loadIconsOnce.Do(func() {
		var table []byte
		var files fs.FS
		if table, err = assets.Files.ReadFile("icons.json"); err != nil {
			return
		}
		if files, err = fs.Sub(assets.Files, "icons"); err != nil {
			return
		}
		err = LoadIcons(table, files)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// testEnv is the environment of a window with the theme of the vfox orange.
func testEnv(t testing.TB, dark bool) *Env {
	t.Helper()
	loadTestIcons(t)
	env := NewEnv(NewTheme(testSeed, dark, false, 1))
	env.Reg.CollectText = true
	return env
}

// view makes a view of mygo of a function of the context of the kit, in the
// theme of the environment, as the application does.
func view(env *Env, build func(c Ctx)) func(c *ui.Context) {
	return func(c *ui.Context) {
		c.SetTheme(env.Theme.UI())
		build(env.Begin(c))
	}
}

// snapshot renders what build draws to a file in the directory that
// VFOX_PLUS_SNAPSHOTS names, if it is set: the way to look at the components.
func snapshot(t testing.TB, env *Env, name string, w, h int, build func(c Ctx)) {
	t.Helper()
	saveImage(t, name, ui.Render(view(env, build), w, h, 1))
}

// saveImage writes an image to the directory that VFOX_PLUS_SNAPSHOTS names, if
// it is set.
func saveImage(t testing.TB, name string, img image.Image) {
	t.Helper()
	dir := os.Getenv("VFOX_PLUS_SNAPSHOTS")
	if dir == "" {
		return
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // The directory is the choice of whoever looks at the pages.
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), encoded.Bytes(), 0o600); err != nil { //nolint:gosec // Likewise.
		t.Fatal(err)
	}
}

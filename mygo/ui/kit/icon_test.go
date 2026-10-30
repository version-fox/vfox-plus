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
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/assets"
)

const squareSVG = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path d="M200-200h560v-560H200Z"/></svg>`

func TestEveryIconOfTheTableIsThereAndParses(t *testing.T) {
	loadTestIcons(t)
	table, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Icons map[string]string `json:"icons"`
	}
	if err := json.Unmarshal(table, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Icons) < 50 {
		t.Fatalf("the table of the icons has %d meanings", len(parsed.Icons))
	}
	for meaning := range parsed.Icons {
		if outlined, filled := iconsOf(meaning); outlined == nil || filled == nil {
			t.Errorf("the icon %q has no outlined and filled SVG that parse", meaning)
		}
		if !HasIcon(meaning) {
			t.Errorf("HasIcon(%q) is false", meaning)
		}
	}
}

func TestLoadIconsNamesTheFilesThatAreMissing(t *testing.T) {
	files := fstest.MapFS{"add.svg": {Data: []byte(squareSVG)}}
	err := LoadIcons([]byte(`{"icons": {"add": "add", "back": "arrow_back"}}`), files)
	if err == nil {
		t.Fatal("LoadIcons succeeded without the files of the icons")
	}
	for _, name := range []string{"add-fill.svg", "arrow_back.svg", "arrow_back-fill.svg"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the error %q does not name %s", err, name)
		}
	}
	if strings.Contains(err.Error(), " add.svg") {
		t.Errorf("the error %q names a file that is there", err)
	}
	if err := LoadIcons([]byte(`{"icons": `), files); err == nil {
		t.Error("LoadIcons accepted a table that is not JSON")
	}
}

func TestAnIconThatDoesNotParseIsLeftOut(t *testing.T) {
	set := &iconSet{
		symbols: map[string]string{"good": "good", "bad": "bad"},
		files: fstest.MapFS{
			"good.svg": {Data: []byte(squareSVG)}, "good-fill.svg": {Data: []byte(squareSVG)},
			"bad.svg": {Data: []byte(squareSVG)}, "bad-fill.svg": {Data: []byte("not an SVG")},
		},
		parsed: map[string]*ui.SVG{},
	}
	if outlined, filled := set.svg("good"); outlined == nil || filled == nil {
		t.Error("an icon with two good files is left out")
	}
	if outlined, filled := set.svg("bad"); outlined != nil || filled != nil {
		t.Error("an icon with a file that does not parse is drawn")
	}
	if outlined, filled := set.svg("unknown"); outlined != nil || filled != nil {
		t.Error("an icon that is not in the table is drawn")
	}
}

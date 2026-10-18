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

package platform_test

import (
	"reflect"
	"testing"

	"github.com/version-fox/vfox-plus/gio/ui/platform"
)

func TestParseAccent(t *testing.T) {
	if got := platform.ParseAccent([]any{0.2, 0.4, 0.8}); got == nil || *got != 0xFF3366CC {
		t.Errorf("got %v", got)
	}
	if got := platform.ParseAccent([]any{1.0, 0.0, 0.5}); got == nil || *got != 0xFFFF0080 {
		t.Errorf("got %v", got)
	}
	// The portal reports negative components when no accent colour is set.
	for name, value := range map[string]any{
		"unset":   []any{-1.0, -1.0, -1.0},
		"short":   []any{0.2, 0.4},
		"strings": []any{"a", "b", "c"},
		"nil":     nil,
		"big":     []any{2.0, 0.0, 0.0},
	} {
		if got := platform.ParseAccent(value); got != nil {
			t.Errorf("%s: got %x", name, *got)
		}
	}
}

func TestButtonLayoutsKeepTheWindowButtonsAndTheirSides(t *testing.T) {
	for layout, want := range map[string]platform.WindowButtons{
		"appmenu:minimize,maximize,close": {Right: []string{"minimize", "maximize", "close"}},
		"appmenu:close":                   {Right: []string{"close"}},
		"close,minimize:":                 {Left: []string{"close", "minimize"}},
		"icon,close:maximize":             {Left: []string{"close"}, Right: []string{"maximize"}},
		"close":                           {Left: []string{"close"}},
		":":                               {},
		"":                                {},
		"appmenu:spacer, minimize ,bogus": {Right: []string{"minimize"}},
	} {
		got := platform.ParseButtonLayout(layout)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", layout, got, want)
		}
	}
	if got := platform.DefaultWindowButtons(); len(got.Left) != 0 || len(got.Right) != 3 {
		t.Errorf("got %v", got)
	}
}

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

//go:build !linux

package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestTheMenuOfTheIconCallsWhatItsEntriesDo(t *testing.T) {
	var called []string
	menu := trayMenu([]trayEntry{
		{label: "A", click: func() { called = append(called, "A") }},
		{},
		{label: "B", click: func() { called = append(called, "B") }},
	})
	items := menu.Items()
	if len(items) != 3 || items[0].Label != "A" || items[2].Label != "B" {
		t.Fatalf("items = %+v", items)
	}
	if items[1].Type != mygo.MenuItemSeparator {
		t.Errorf("the empty entry is a %v, want a separator", items[1].Type)
	}
	for _, item := range []*mygo.MenuItem{items[0], items[2], items[0]} {
		item.Click(item, nil)
	}
	if strings.Join(called, "") != "ABA" {
		t.Errorf("called %v", called)
	}
}

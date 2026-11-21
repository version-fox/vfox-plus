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

package app

import "testing"

func TestTheCacheForgetsTheLeastRecentlyReadEntry(t *testing.T) {
	c := newCache[int, string](2)
	c.at(1)
	c.at(2)
	c.at(1) // 1 is read again, so 2 is the one read least recently
	c.at(3)
	if _, ok := c.entries[2]; ok {
		t.Error("the least recently read entry is still kept")
	}
	if _, ok := c.entries[1]; !ok {
		t.Error("the entry read again was forgotten")
	}
	if len(c.entries) != 2 {
		t.Errorf("%d entries kept, want the limit of 2", len(c.entries))
	}
}

func TestTheCacheKeepsAnEntryThatIsLoading(t *testing.T) {
	c := newCache[int, string](1)
	busy := c.at(1)
	busy.res.Begin()
	c.at(2)
	if _, ok := c.entries[1]; !ok {
		t.Error("an entry that is loading was forgotten")
	}
}

func TestTheCacheMarksTheEntriesThatMatch(t *testing.T) {
	c := newCache[string, int](4)
	c.at("a")
	c.at("b")
	c.mark(func(key string) bool { return key == "a" })
	if !c.entries["a"].dirty || c.entries["b"].dirty {
		t.Error("the match of a marked the wrong entries")
	}
	c.mark(nil)
	if !c.entries["b"].dirty {
		t.Error("a nil match did not mark every entry")
	}
}

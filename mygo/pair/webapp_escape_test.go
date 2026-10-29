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

package pair

import (
	"bytes"
	"regexp"
	"testing"
)

// The page builds its HTML from strings, and writes attributes in double quotes
// and in single quotes (data-use='...'). A value with a quote in it would end the
// attribute there and could start one of its own, so esc has to escape both
// quotes along with & < and >. CodeQL reports it, as
// js/incomplete-html-attribute-sanitization, for every attribute that gets a
// value of esc while esc leaves the single quote alone.
func TestWebappEscapesBothQuotes(t *testing.T) {
	definition := regexp.MustCompile(`function esc\(s\) \{[^\n]*\}`).Find(webappHTML)
	if definition == nil {
		t.Fatal("the page has no esc function")
	}
	for _, want := range []string{`/[&<>"']/g`, `"&":"&amp;"`, `"<":"&lt;"`, `">":"&gt;"`, `'"':"&quot;"`, `"'":"&#39;"`} {
		if !bytes.Contains(definition, []byte(want)) {
			t.Errorf("esc lacks %s: %s", want, definition)
		}
	}
}

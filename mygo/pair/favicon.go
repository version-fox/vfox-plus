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
	_ "embed"
	"strings"
)

// faviconSVG is the favicon of the pages: the plate and the fox of the
// application icon without its halo, and with a larger fox, which still reads
// at 16 px. A test compares it with the master of the icon.
//
//go:embed favicon.svg
var faviconSVG string

// faviconPlaceholder marks the place of the favicon in the head of a page.
const faviconPlaceholder = "<!--favicon-->"

// faviconLink is the link of the favicon. The SVG is inlined as a data URI,
// like the other assets of the pages: the server answers no request for an
// icon, and the web interface stays a single page without requests of its own.
var faviconLink = `<link rel="icon" type="image/svg+xml" href="` + faviconDataURI(faviconSVG) + `">`

// withFavicon puts the favicon in a page text with a faviconPlaceholder.
func withFavicon(page string) string {
	return strings.Replace(page, faviconPlaceholder, faviconLink, 1)
}

// faviconDataURI turns an SVG into a data URI that is safe in an HTML
// attribute in double quotes: no comments, no declaration, one space where
// the file has white space, single quotes inside, and %-escapes for the
// characters that HTML or the URL parser would take for their own.
func faviconDataURI(svg string) string {
	svg = dropBetween(svg, "<!--", "-->")
	svg = dropBetween(svg, "<?", "?>")
	svg = strings.ReplaceAll(svg, `"`, `'`)
	svg = strings.Join(strings.Fields(svg), " ")
	svg = strings.ReplaceAll(svg, "> <", "><")
	var uri strings.Builder
	uri.WriteString("data:image/svg+xml,")
	for i := 0; i < len(svg); i++ {
		switch c := svg[i]; c {
		case '%':
			uri.WriteString("%25")
		case '#':
			uri.WriteString("%23")
		case '<':
			uri.WriteString("%3C")
		case '>':
			uri.WriteString("%3E")
		case '&':
			uri.WriteString("%26")
		default:
			uri.WriteByte(c)
		}
	}
	return uri.String()
}

// dropBetween removes every part of s from the text from to the text to, both included.
func dropBetween(s, from, to string) string {
	for {
		start := strings.Index(s, from)
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], to)
		if end < 0 {
			return s
		}
		s = s[:start] + s[start+end+len(to):]
	}
}

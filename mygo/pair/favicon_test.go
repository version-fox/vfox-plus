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
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFaviconDataURI(t *testing.T) {
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<!-- a comment
  that spans lines -->
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8">
  <path fill="url(#a)" d="M1,2
     L3,4 Z"/>
  <path fill="#fff" d="M0,0 &amp; 100%"/>
</svg>
`
	want := "data:image/svg+xml," +
		"%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 8 8'%3E" +
		"%3Cpath fill='url(%23a)' d='M1,2 L3,4 Z'/%3E" +
		"%3Cpath fill='%23fff' d='M0,0 %26amp; 100%25'/%3E" +
		"%3C/svg%3E"
	if got := faviconDataURI(svg); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := dropBetween("a<!-- b", "<!--", "-->"); got != "a<!-- b" {
		t.Errorf("an unterminated comment is left alone, got %q", got)
	}
}

// decodeDataURI undoes the escapes of faviconDataURI.
func decodeDataURI(uri string) string {
	payload := strings.TrimPrefix(uri, "data:image/svg+xml,")
	return strings.NewReplacer("%3C", "<", "%3E", ">", "%23", "#", "%26", "&", "%25", "%").Replace(payload)
}

func TestFaviconLink(t *testing.T) {
	prefix := `<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,`
	if !strings.HasPrefix(faviconLink, prefix) || !strings.HasSuffix(faviconLink, `">`) {
		t.Fatalf("faviconLink = %.120s...", faviconLink)
	}
	// The value of an attribute in double quotes: nothing in it may end the attribute or the tag.
	value := strings.TrimSuffix(strings.TrimPrefix(faviconLink, `<link rel="icon" type="image/svg+xml" href="`), `">`)
	if strings.ContainsAny(value, "\"<>#\n\r\t") {
		t.Errorf("the data URI holds a character that HTML or the URL parser would take: %.120s...", value)
	}
	// It is the SVG of the file, and small: it goes into every page.
	var icon svgIcon
	if err := xml.Unmarshal([]byte(decodeDataURI(value)), &icon); err != nil {
		t.Fatalf("the data URI is not an SVG: %v", err)
	}
	if len(icon.Fox.Paths) != 9 || len(value) > 4500 {
		t.Errorf("%d paths in the fox, %d bytes", len(icon.Fox.Paths), len(value))
	}
	if !strings.Contains(faviconSVG, "Copyright 2026 Han Li and contributors") || strings.Contains(value, "Copyright") {
		t.Error("the file carries its license header and the data URI does not")
	}
}

func TestWebappHasItsPlaceholders(t *testing.T) {
	for _, placeholder := range []string{faviconPlaceholder, logoPlaceholder} {
		if bytes.Count(webappHTML, []byte(placeholder)) != 1 {
			t.Fatalf("webapp.html needs exactly one %s", placeholder)
		}
	}
	// The pieces put the page together again, with the favicon in the head and the icon in
	// the src of the brand mark.
	whole := string(webappHead) + faviconPlaceholder + string(webappMiddle) + logoPlaceholder + string(webappTail)
	if whole != string(webappHTML) {
		t.Fatal("the pieces of the page are not the page")
	}
	if !bytes.Contains(webappHead, []byte("<head>")) || !bytes.Contains(webappMiddle, []byte("</head>")) {
		t.Error("the favicon belongs in the head")
	}
	if !bytes.HasSuffix(webappMiddle, []byte(`<img class="mark" src="`)) || !bytes.HasPrefix(webappTail, []byte(`"`)) {
		t.Error("the icon belongs in the src of the brand mark")
	}
	for name, page := range map[string]string{"intro": introTemplate.Tree.Root.String(), "connect": connectTemplate.Tree.Root.String()} {
		if !strings.Contains(page, `rel="icon"`) || strings.Contains(page, faviconPlaceholder) {
			t.Errorf("the %s page has no favicon, or still the placeholder", name)
		}
	}
}

func TestManifest(t *testing.T) {
	var manifest webManifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "vfox+" || manifest.ShortName != "vfox+" || manifest.Display != "standalone" {
		t.Errorf("manifest = %+v", manifest)
	}
	// The start page and the scope are the ones of the web interface, and relative: the server
	// has the address of every machine it runs on.
	if manifest.StartURL != "/app" || manifest.ID != "/app" || manifest.Scope != "/" {
		t.Errorf("start_url %q, id %q, scope %q", manifest.StartURL, manifest.ID, manifest.Scope)
	}
	// What Chromium asks of an installable icon: a square PNG, SVG or WebP of the purpose "any", at
	// least 144 px or of any size.
	if len(manifest.Icons) != 1 {
		t.Fatalf("icons = %+v", manifest.Icons)
	}
	icon := manifest.Icons[0]
	if icon.Type != "image/svg+xml" || icon.Sizes != "any" || icon.Purpose != "any" || icon.Src != faviconURI {
		t.Errorf("icon type %q, sizes %q, purpose %q, the favicon: %t", icon.Type, icon.Sizes, icon.Purpose, icon.Src == faviconURI)
	}
	// The colours are those of the page.
	page := string(webappHTML)
	if !strings.Contains(page, `<meta name="theme-color" content="`+manifest.ThemeColor+`">`) {
		t.Errorf("theme_color %s is not the theme-color of the page", manifest.ThemeColor)
	}
	if !strings.Contains(page, "--m3-background: "+manifest.BackgroundColor+";") {
		t.Errorf("background_color %s is not the light background of the page", manifest.BackgroundColor)
	}
	if !strings.Contains(page, `<link rel="manifest" href="/manifest.webmanifest">`) {
		t.Error("the page does not link its manifest")
	}
}

type svgPath struct {
	ID        string `xml:"id,attr"`
	Fill      string `xml:"fill,attr"`
	D         string `xml:"d,attr"`
	Transform string `xml:"transform,attr"`
}

type svgGradient struct {
	ID    string  `xml:"id,attr"`
	X1    float64 `xml:"x1,attr"`
	Y1    float64 `xml:"y1,attr"`
	X2    float64 `xml:"x2,attr"`
	Y2    float64 `xml:"y2,attr"`
	Stops []struct {
		Offset  float64 `xml:"offset,attr"`
		Color   string  `xml:"stop-color,attr"`
		Opacity string  `xml:"stop-opacity,attr"`
	} `xml:"stop"`
}

type svgIcon struct {
	Gradients []svgGradient `xml:"defs>linearGradient"`
	Paths     []svgPath     `xml:"path"`
	Fox       struct {
		Transform string    `xml:"transform,attr"`
		Paths     []svgPath `xml:"path"`
	} `xml:"g"`
}

// The favicon is the master icon without its halo: the same plate, the same fox
// and the same gradients, where the fox may be placed larger.
func TestFaviconIsTheMasterIcon(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "shared", "packaging", "linux", "dev.vfox.VfoxPlus.svg"))
	if err != nil {
		t.Fatal(err)
	}
	var master, favicon svgIcon
	if err := xml.Unmarshal(data, &master); err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal([]byte(faviconSVG), &favicon); err != nil {
		t.Fatal(err)
	}
	if len(favicon.Paths) != 1 || favicon.Paths[0].ID != "plate-shape" {
		t.Fatalf("the favicon draws the plate and the fox, nothing else: %+v", favicon.Paths)
	}
	if favicon.Paths[0] != master.Paths[0] {
		t.Errorf("the plate differs from the master:\nfavicon %+v\nmaster  %+v", favicon.Paths[0], master.Paths[0])
	}
	if len(favicon.Fox.Paths) != len(master.Fox.Paths) {
		t.Fatalf("the fox has %d parts, the master's %d", len(favicon.Fox.Paths), len(master.Fox.Paths))
	}
	for i, part := range favicon.Fox.Paths {
		if part != master.Fox.Paths[i] {
			t.Errorf("part %s differs from the master:\nfavicon %+v\nmaster  %+v", part.ID, part, master.Fox.Paths[i])
		}
	}
	byID := map[string]svgGradient{}
	for _, g := range master.Gradients {
		byID[g.ID] = g
	}
	for _, g := range favicon.Gradients {
		want, ok := byID[g.ID]
		if !ok || g.X1 != want.X1 || g.Y1 != want.Y1 || g.X2 != want.X2 || g.Y2 != want.Y2 || len(g.Stops) != len(want.Stops) {
			t.Errorf("gradient %s differs from the master: %+v, master %+v", g.ID, g, want)
			continue
		}
		for i, stop := range g.Stops {
			if stop != want.Stops[i] {
				t.Errorf("gradient %s, stop %d differs from the master: %+v, master %+v", g.ID, i, stop, want.Stops[i])
			}
		}
	}
	if len(favicon.Gradients) != 3 {
		t.Errorf("the favicon has the gradients of the plate, the fur and the cheeks, not %d", len(favicon.Gradients))
	}
	// Larger than the master's fox, which is placed to fit its halo.
	if favicon.Fox.Transform == master.Fox.Transform || !strings.HasPrefix(favicon.Fox.Transform, "translate(") {
		t.Errorf("the favicon places the fox with %q, the master with %q", favicon.Fox.Transform, master.Fox.Transform)
	}
}

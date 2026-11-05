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

package gatewaycheck

import (
	"context"
	"errors"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/chrome"
)

func TestTheShotsNeedTheProgramTheDirectoryAndTheApplication(t *testing.T) {
	for name, args := range map[string][]string{
		"nothing":                   {},
		"the application only":      {"app"},
		"a program without the app": {"--binary", "gateway", "--output", "pictures"},
		"no directory":              {"--binary", "gateway", "app"},
		"no program":                {"--output", "pictures", "app"},
	} {
		if err := Shots(args); err == nil || !strings.Contains(err.Error(), "Usage:") {
			t.Errorf("%s: %v, want the usage", name, err)
		}
	}
}

func TestTheShotsNeedTheAddressOfTheDocumentation(t *testing.T) {
	defer func(real func() ([]net.Addr, error)) { interfaceAddrs = real }(interfaceAddrs)
	cases := map[string]struct {
		addrs   []net.Addr
		err     error
		wantErr string // what the error says; empty if the address is there.
	}{
		"the address is there": {addrs: []net.Addr{
			&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
			&net.IPNet{IP: net.ParseIP(documentationAddress), Mask: net.CIDRMask(32, 32)},
		}},
		"another address of the same network": {
			addrs:   []net.Addr{&net.IPNet{IP: net.ParseIP("192.168.1.21"), Mask: net.CIDRMask(24, 32)}},
			wantErr: "no address " + documentationAddress,
		},
		"an address that is no network": {
			addrs:   []net.Addr{&net.IPAddr{IP: net.ParseIP(documentationAddress)}},
			wantErr: "no address " + documentationAddress,
		},
		"no address at all": {wantErr: "unshare -rn"},
		"no list of them":   {err: errors.New("no interfaces"), wantErr: "no interfaces"},
	}
	for name, c := range cases {
		interfaceAddrs = func() ([]net.Addr, error) { return c.addrs, c.err }
		err := requireAddress(documentationAddress)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: %v", name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: %v, want an error that says %q", name, err, c.wantErr)
		}
	}
	// The command stops before it starts anything when the computer lacks the address, and says what to do.
	interfaceAddrs = func() ([]net.Addr, error) { return nil, nil }
	err := Shots([]string{"--binary", "gateway", "--output", t.TempDir(), "app"})
	if err == nil || !strings.Contains(err.Error(), "ip addr add "+documentationAddress) {
		t.Errorf("without the address: %v, want the advice of a network namespace", err)
	}
}

// The steps of the pictures are tried against a small page that does what a page of the gateway does: a field that
// a script reads, a button that changes the page, and a card (the main element) that the picture is as high as.
const shotsPage = `<!doctype html><meta charset="utf-8"><title>test</title>
<style>body{margin:0;padding:24px 16px} main{height:500px} h1{margin:0}</style>
<main><h1>This is the gateway of vfox+</h1>
<input type="text" id="link"><button id="go" type="button"
 onclick="document.getElementById('out').textContent = 'pasted ' + document.getElementById('link').value">Pair</button>
<p id="out"></p></main>`

// tab starts the Chrome of the tests on a page served over HTTPS with a certificate that nobody signed, as the
// gateway serves its own, or ends the test on a machine without a Chrome.
func tab(t *testing.T) *pages {
	t.Helper()
	binary, err := chrome.Find("")
	if err != nil {
		t.Skip("no Chrome to drive: set CHROME_BIN")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(shotsPage))
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	page, closeBrowser := openBrowser(ctx, binary)
	t.Cleanup(closeBrowser)
	return &pages{tab: page, base: server.URL, dir: t.TempDir(), wait: 300 * time.Millisecond}
}

func TestThePicturesAreAsHighAsThePageAndItsMargin(t *testing.T) {
	p := tab(t)
	if err := p.run(
		p.visit("/"), p.until("the page", `document.body.innerText.includes("This is the gateway of vfox+")`), p.shot("page"),
		p.size(appWidth, appHeight), p.window("window"),
	); err != nil {
		t.Fatal(err)
	}
	// 24 pixels of padding and 500 of main: the picture ends a margin under it, and is as wide as a column.
	for name, want := range map[string][2]int{"page": {pageWidth, 24 + 500 + pageMargin}, "window": {appWidth, appHeight}} {
		file, err := os.Open(filepath.Join(p.dir, shotPrefix+name+".png")) //nolint:gosec // A picture of this test.
		if err != nil {
			t.Fatal(err)
		}
		size, err := png.DecodeConfig(file)
		_ = file.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if size.Width != want[0] || size.Height != want[1] {
			t.Errorf("%s is %dx%d, want %dx%d", name, size.Width, size.Height, want[0], want[1])
		}
	}
}

func TestAStepFillsAFieldAndPressesAButton(t *testing.T) {
	p := tab(t)
	if err := p.run(
		p.visit("/"), p.until("the field", `!!document.getElementById("link")`),
		p.fill("#link", "https://192.168.1.20:8369/auth/connect/abc"), p.press("#go"),
		p.until("what the button did", `document.getElementById("out").textContent === "pasted https://192.168.1.20:8369/auth/connect/abc"`),
	); err != nil {
		t.Fatal(err)
	}
}

func TestAStepChoosesTheLanguageThatThePageKeeps(t *testing.T) {
	p := tab(t)
	if err := p.run(
		p.visit("/"), p.until("the page", `!!document.getElementById("link")`),
		p.language("zh"), p.reload(), p.until("the page, loaded again", `localStorage.getItem("vfox-plus-lang") === "zh"`),
	); err != nil {
		t.Fatal(err)
	}
}

func TestAStepSaysWhatIsMissing(t *testing.T) {
	p := tab(t)
	if err := p.run(p.visit("/"), p.until("the field", `!!document.getElementById("link")`)); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		step step
		want string
	}{
		"a field that is not there":  {p.fill("#nothing", "x"), "fill #nothing: the page has no such element"},
		"a button that is not there": {p.press("#nothing"), "press #nothing: the page has no such element"},
		"a page that never says it":  {p.until("a word that is never said", "false"), "timed out waiting for a word that is never said"},
	} {
		if err := c.step(); err == nil || err.Error() != c.want {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	if err := p.run(p.visit("/"), p.fill("#nothing", "x"), p.shot("never")); err == nil {
		t.Error("a run went on after a step that failed")
	}
	if _, err := os.Stat(filepath.Join(p.dir, shotPrefix+"never.png")); err == nil {
		t.Error("a picture was taken after a step that failed")
	}
}

func TestAStepFailsWhenThePageCannotBeOpened(t *testing.T) {
	p := tab(t)
	p.base = "https://127.0.0.1:1"
	if err := p.visit("/")(); err == nil || !strings.Contains(err.Error(), "cannot open /") {
		t.Errorf("an address nothing listens on: %v", err)
	}
}

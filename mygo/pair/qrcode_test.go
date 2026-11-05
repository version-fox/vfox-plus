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
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// internalServer starts a server in the package, with the application state of
// the captured output of vfox.
func internalServer(t *testing.T, options Options) *Server {
	t.Helper()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        testsupport.FixtureRunner(),
		SettingsStore: settings.NewMemorySettingsStore(settings.DefaultSettings()),
		Environment:   map[string]string{"HOME": "/home/u"},
	})
	t.Cleanup(model.Dispose)
	options.Model = model
	server, err := NewServer(options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

func TestQRMatrixIsASquareCodeInsideItsQuietZone(t *testing.T) {
	const link = "https://192.168.1.5:8369/auth/connect/0123456789abcdef0123456789abcdef"
	first, err := QRMatrix(link)
	if err != nil {
		t.Fatal(err)
	}
	second, err := QRMatrix(link)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatal("encoding must be deterministic")
	}
	size := len(first)
	// A QR code is 17+4v modules wide, and the quiet zone adds four on each side.
	if size < 21+8 || (size-8-17)%4 != 0 {
		t.Fatalf("%d modules is no QR code with a quiet zone", size)
	}
	dark := 0
	for y, row := range first {
		if len(row) != size {
			t.Fatalf("the code is not square: row %d has %d modules of %d", y, len(row), size)
		}
		for x, module := range row {
			inside := x >= 4 && x < size-4 && y >= 4 && y < size-4
			if module && !inside {
				t.Fatalf("module %d,%d is dark in the quiet zone", x, y)
			}
			if module {
				dark++
			}
		}
	}
	if dark == 0 {
		t.Fatal("the code is blank")
	}
	// The finder patterns, a 7x7 square of dark, light and dark rings, mark three corners.
	finder := func(left, top int) bool {
		for dy := 0; dy < 7; dy++ {
			for dx := 0; dx < 7; dx++ {
				ring := min(dx, 6-dx, dy, 6-dy)
				if first[top+dy][left+dx] != (ring != 1) {
					return false
				}
			}
		}
		return true
	}
	if !finder(4, 4) || !finder(size-4-7, 4) || !finder(4, size-4-7) {
		t.Error("a corner has no finder pattern")
	}
	if finder(size-4-7, size-4-7) {
		t.Error("the fourth corner has no finder pattern")
	}
	if _, err := QRMatrix(strings.Repeat("x", 4000)); err == nil {
		t.Error("a text that is too long for a code must fail")
	}
}

func TestQRSVGDrawsTheModulesOfTheMatrix(t *testing.T) {
	const link = "https://192.168.1.5:8369/auth/connect/abc123"
	matrix, err := QRMatrix(link)
	if err != nil {
		t.Fatal(err)
	}
	svg, err := QRSVG(link)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
		Rect    struct {
			Fill string `xml:"fill,attr"`
		} `xml:"rect"`
		Path struct {
			Fill string `xml:"fill,attr"`
			D    string `xml:"d,attr"`
		} `xml:"path"`
	}
	if err := xml.Unmarshal([]byte(svg), &document); err != nil {
		t.Fatalf("not an SVG: %v\n%s", err, svg)
	}
	size := len(matrix)
	if want := fmt.Sprintf("0 0 %d %d", size, size); document.ViewBox != want {
		t.Errorf("viewBox %q, want %q", document.ViewBox, want)
	}
	// Black on white, whatever the theme around it: a scanner needs the contrast.
	if document.Rect.Fill != "#fff" || document.Path.Fill != "#000" {
		t.Errorf("ground %q, modules %q", document.Rect.Fill, document.Path.Fill)
	}
	// Every bar of the path is a run of dark modules; together they are the dark modules.
	drawn := make([][]bool, size)
	for y := range drawn {
		drawn[y] = make([]bool, size)
	}
	for _, bar := range strings.Split(strings.TrimSuffix(document.Path.D, "z"), "z") {
		var x, y, w int
		if _, err := fmt.Sscanf(bar, "M%d %dh%dv1h-%d", &x, &y, &w, new(int)); err != nil {
			t.Fatalf("bar %q: %v", bar, err)
		}
		for i := 0; i < w; i++ {
			drawn[y][x+i] = true
		}
	}
	if fmt.Sprint(drawn) != fmt.Sprint(matrix) {
		t.Error("the SVG does not draw the modules of the matrix")
	}
	if _, err := QRSVG(strings.Repeat("x", 4000)); err == nil {
		t.Error("a text that is too long for a code must fail")
	}
}

func TestPairingURLNamesTheAddressAndThePortOfTheServer(t *testing.T) {
	server := internalServer(t, Options{})
	_, port, _ := net.SplitHostPort(server.Addr().String())
	got := server.PairingURL("192.168.1.5", "tok")
	want := "http://192.168.1.5:" + port + "/auth/connect/tok"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := server.PairingURL("2001:db8::1", "tok"); !strings.Contains(got, "[2001:db8::1]") {
		t.Errorf("IPv6 must be bracketed: %q", got)
	}
	if got := server.PairingURL("[2001:db8::1]", "tok"); strings.Contains(got, "[[") {
		t.Errorf("a bracketed address stays as it is: %q", got)
	}
}

func TestNewTokenSweepsExpired(t *testing.T) {
	server := internalServer(t, Options{TokenTTL: time.Nanosecond})
	if _, _, err := server.NewToken(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, _, err := server.NewToken(); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	left := len(server.tokens)
	server.mu.Unlock()
	if left != 1 {
		t.Fatalf("want the expired token swept, %d left", left)
	}
}

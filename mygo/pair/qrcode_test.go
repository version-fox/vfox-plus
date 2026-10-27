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
	"net"
	"strings"
	"testing"
	"time"
)

func TestReachablePairingURL(t *testing.T) {
	const token = "abc123"
	cases := []struct {
		name   string
		scheme string
		port   string
		hosts  []string
		want   string
	}{
		{"ipv4", "https", "8080", []string{"192.168.1.5"}, "https://192.168.1.5:8080/auth/connect/" + token},
		{"ipv4 first", "https", "8080", []string{"2001:db8::1", "192.168.1.5"}, "https://192.168.1.5:8080/auth/connect/" + token},
		{"ipv6 bracketed", "https", "8080", []string{"2001:db8::1"}, "https://[2001:db8::1]:8080/auth/connect/" + token},
		{"loopback skipped", "https", "8080", []string{"127.0.0.1", "10.0.0.2"}, "https://10.0.0.2:8080/auth/connect/" + token},
		{"link local skipped", "https", "8080", []string{"fe80::1", "10.0.0.2"}, "https://10.0.0.2:8080/auth/connect/" + token},
		{"brackets trimmed", "https", "8080", []string{"[192.168.1.5]"}, "https://192.168.1.5:8080/auth/connect/" + token},
		{"no port", "http", "", []string{"192.168.1.5"}, "http://192.168.1.5/auth/connect/" + token},
		{"none", "https", "8080", nil, ""},
		{"loopback only", "https", "8080", []string{"127.0.0.1", "::1"}, ""},
		{"garbage", "https", "8080", []string{"not a host"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReachablePairingURL(tc.scheme, tc.port, token, tc.hosts); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTerminalQRCode(t *testing.T) {
	first, err := TerminalQRCode("https://192.168.1.5:8080/auth/connect/abc123")
	if err != nil {
		t.Fatal(err)
	}
	second, err := TerminalQRCode("https://192.168.1.5:8080/auth/connect/abc123")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("rendering must be deterministic")
	}
	lines := strings.Split(strings.TrimSuffix(first, "\n"), "\n")
	if len(lines) < 10 {
		t.Fatalf("too few lines: %d", len(lines))
	}
	width := -1
	for _, line := range lines {
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Fatalf("line must end with a reset: %q", line)
		}
		cells := strings.Count(line, "█") + strings.Count(line, "▀") +
			strings.Count(line, "▄") + strings.Count(line, " ")
		// Each cell is one module column plus escape sequences; the quiet
		// zone keeps the first and last visual cells blank.
		if width < 0 {
			width = cells
		} else if cells != width {
			t.Fatalf("ragged code: %d vs %d", cells, width)
		}
	}
	// The code starts with quiet lines, then the dark finder pattern.
	// (The encoder already pads the required quiet zone; QRQuietZone adds a
	// little more for terminals with tight line spacing.)
	plain := func(line string) string {
		out := line
		for _, esc := range []string{"\x1b[47m", "\x1b[30m", "\x1b[0m"} {
			out = strings.ReplaceAll(out, esc, "")
		}
		return out
	}
	if strings.TrimSpace(plain(lines[0])) != "" || strings.TrimSpace(plain(lines[1])) != "" {
		t.Fatal("expected quiet lines first")
	}
	dark := -1
	for i, line := range lines {
		if strings.Contains(line, "█") {
			dark = i
			break
		}
	}
	if dark < 2 {
		t.Fatalf("expected quiet zone before the finder, first dark line %d", dark)
	}
	lead := len(plain(lines[dark])) - len(strings.TrimLeft(plain(lines[dark]), " "))
	trail := len(plain(lines[dark])) - len(strings.TrimRight(plain(lines[dark]), " "))
	if lead != trail || lead < 2 {
		t.Fatalf("quiet zone must be symmetric, lead %d trail %d", lead, trail)
	}
	for _, bad := range []string{"\r", "\t"} {
		if strings.Contains(first, bad) {
			t.Fatalf("code contains %q", bad)
		}
	}
}

func TestPairingURL(t *testing.T) {
	server, err := NewServer(Options{AllowNonLoopback: true, Bind: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	_, port, _ := net.SplitHostPort(server.Addr().String())
	got := server.PairingURL("192.168.1.5", "tok")
	want := "http://192.168.1.5:" + port + "/auth/connect/tok"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := server.PairingURL("2001:db8::1", "tok"); !strings.Contains(got, "[2001:db8::1]") {
		t.Errorf("IPv6 must be bracketed: %q", got)
	}
}

func TestNewTokenSweepsExpired(t *testing.T) {
	server, err := NewServer(Options{TokenTTL: time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
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

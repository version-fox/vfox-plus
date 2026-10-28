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

package measureserve

import (
	"strings"
	"testing"
)

func TestPairingURLFromLine(t *testing.T) {
	url, ok := pairingURLFromLine("  http://127.0.0.1:8080/auth/connect/abcdef0123456789abcdef0123456789")
	if !ok || url != "http://127.0.0.1:8080/auth/connect/abcdef0123456789abcdef0123456789" {
		t.Fatalf("got %q, %v", url, ok)
	}
	for _, line := range []string{
		"",
		"Serving vfox-plus on http://0.0.0.0:8080",
		"http://192.168.1.5:8080/auth/connect/abc",
		"http://127.0.0.1:8080/auth/connect/",
		"  http://127.0.0.1:8080/auth/connect/xyz",
	} {
		if _, ok := pairingURLFromLine(line); ok {
			t.Fatalf("got a link from %q", line)
		}
	}
}

func TestSessionCookieValue(t *testing.T) {
	value, ok := sessionCookieValue("vfox-plus-session=abc123; Path=/; Secure; HttpOnly; SameSite=Lax")
	if !ok || value != "abc123" {
		t.Fatalf("got %q, %v", value, ok)
	}
	for _, header := range []string{"", "other=1", "vfox-plus-session=", "vfox-plus-session"} {
		if _, ok := sessionCookieValue(header); ok {
			t.Fatalf("got a cookie from %q", header)
		}
	}
}

func TestMeasureUsage(t *testing.T) {
	if err := Run([]string{}); err == nil || !strings.Contains(err.Error(), "Usage:") {
		t.Fatalf("got %v", err)
	}
	if err := Run([]string{"--settle", "-1", "server"}); err == nil {
		t.Fatal("want an error for a negative settle time")
	}
}

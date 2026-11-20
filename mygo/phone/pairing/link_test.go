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

package pairing_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

// sampleToken has the shape of the tokens that the desktop makes.
const sampleToken = "0123456789abcdef0123456789abcdef"

func TestParseLinkAcceptsPairingLinks(t *testing.T) {
	cases := []struct {
		name, text, base, token string
	}{
		{"https with a port", "https://192.168.1.10:8369/auth/connect/" + sampleToken, "https://192.168.1.10:8369", sampleToken},
		{"surrounding white space", "  https://vfox.example/auth/connect/abc_DEF-123\n", "https://vfox.example", "abc_DEF-123"},
		{"https on the default port", "https://vfox.example:443/auth/connect/tok", "https://vfox.example:443", "tok"},
		{"IPv6 address", "https://[::1]:8369/auth/connect/tok", "https://[::1]:8369", "tok"},
		{"bare address with the pairing path", "192.168.1.10:8369/auth/connect/" + sampleToken, "https://192.168.1.10:8369", sampleToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			link, err := pairing.ParseLink(tc.text)
			if err != nil {
				t.Fatalf("ParseLink(%q): %v", tc.text, err)
			}
			if got := link.Base.String(); got != tc.base {
				t.Errorf("base = %q, want %q", got, tc.base)
			}
			if link.Token != tc.token {
				t.Errorf("token = %q, want %q", link.Token, tc.token)
			}
		})
	}
}

func TestParseLinkRejectsEverythingElse(t *testing.T) {
	cases := map[string]string{
		"empty":                       "",
		"white space only":            "   ",
		"plain text":                  "not a link",
		"bare address without path":   "192.168.1.10:8369",
		"http scheme":                 "http://192.168.1.10:8369/auth/connect/" + sampleToken,
		"other scheme":                "ftp://192.168.1.10/auth/connect/" + sampleToken,
		"app path":                    "https://192.168.1.10:8369/app",
		"root path":                   "https://192.168.1.10:8369/",
		"connect without a token":     "https://192.168.1.10:8369/auth/connect",
		"empty token":                 "https://192.168.1.10:8369/auth/connect/",
		"token with a slash":          "https://192.168.1.10:8369/auth/connect/a/b",
		"token with a trailing slash": "https://192.168.1.10:8369/auth/connect/tok/",
		"encoded slash in token":      "https://192.168.1.10:8369/auth/connect/tok%2Fx",
		"token with white space":      "https://192.168.1.10:8369/auth/connect/tok with space",
		"query":                       "https://192.168.1.10:8369/auth/connect/tok?x=1",
		"fragment":                    "https://192.168.1.10:8369/auth/connect/tok#frag",
		"user information":            "https://user:secret@192.168.1.10:8369/auth/connect/tok",
		"port zero":                   "https://192.168.1.10:0/auth/connect/tok",
		"port too large":              "https://192.168.1.10:70000/auth/connect/tok",
		"colon without a port":        "https://192.168.1.10:/auth/connect/tok",
		"no host":                     "https:///auth/connect/tok",
		"double slash before path":    "https://192.168.1.10:8369//auth/connect/tok",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := pairing.ParseLink(text)
			if !errors.Is(err, pairing.ErrNotPairingLink) {
				t.Fatalf("ParseLink(%q) = %v, want ErrNotPairingLink", text, err)
			}
		})
	}
}

func TestParseLinkErrorsDoNotRepeatTheToken(t *testing.T) {
	_, err := pairing.ParseLink("http://192.168.1.10:8369/auth/connect/" + sampleToken)
	if err == nil {
		t.Fatal("an http link is accepted")
	}
	if strings.Contains(err.Error(), sampleToken) {
		t.Fatalf("the error repeats the token: %v", err)
	}
}

func TestValidToken(t *testing.T) {
	if !pairing.ValidToken(sampleToken) || !pairing.ValidToken("a-_Z") {
		t.Error("a token of letters, digits, '-' and '_' is rejected")
	}
	for _, token := range []string{"", "a/b", "a b", "a?b", strings.Repeat("a", 129)} {
		if pairing.ValidToken(token) {
			t.Errorf("ValidToken(%q) = true", token)
		}
	}
}

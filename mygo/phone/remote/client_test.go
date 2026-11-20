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

package remote

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// tokenOf returns the token of a pairing link.
func tokenOf(link string) string {
	return link[strings.LastIndex(link, "/")+1:]
}

func TestPairingIssuesASession(t *testing.T) {
	s := startServer(t)
	_, link, err := s.server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(s.base, s.fingerprint, "")
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.Pair(t.Context(), tokenOf(link))
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}
	if session == "" {
		t.Fatal("Pair gave no session")
	}
	paired, _ := s.client(t, session)
	if _, err := paired.Inventory(t.Context()); err != nil {
		t.Fatalf("the session that Pair gave is refused: %v", err)
	}
}

func TestPairingRefusesInvalidAndUsedTokens(t *testing.T) {
	s := startServer(t)
	client, err := New(s.base, s.fingerprint, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if _, err := client.Pair(ctx, "0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrPairingRefused) {
		t.Errorf("a token the server never issued: %v, want ErrPairingRefused", err)
	}
	if _, err := client.Pair(ctx, "a/b"); !errors.Is(err, ErrPairingRefused) {
		t.Errorf("a malformed token: %v, want ErrPairingRefused", err)
	}

	_, link, err := s.server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	s.follow(t, link) // uses the token
	_, err = client.Pair(ctx, tokenOf(link))
	if !errors.Is(err, ErrPairingRefused) {
		t.Fatalf("a used token: %v, want ErrPairingRefused", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Errorf("the refusal does not carry the 403 answer: %v", err)
	}
}

func TestCallsWithoutAValidSessionAreUnpaired(t *testing.T) {
	s := startServer(t)
	for name, session := range map[string]string{
		"no session": "",
		"a session that the server does not know": "not-a-session",
	} {
		client, _ := s.client(t, session)
		_, err := client.Inventory(t.Context())
		var apiErr *APIError
		if !errors.Is(err, ErrUnpaired) || !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
			t.Errorf("%s: %v, want ErrUnpaired with status 401", name, err)
		}
	}
}

func TestMissingAndFailingAnswersMapToTheSentinels(t *testing.T) {
	s := startServer(t)
	client, _ := s.paired(t)
	ctx := t.Context()
	if _, err := client.Task(ctx, 987654); !errors.Is(err, ErrNotFound) {
		t.Errorf("Task of an unknown id: %v, want ErrNotFound", err)
	}
	if _, err := client.TaskLog(ctx, 987654); !errors.Is(err, ErrNotFound) {
		t.Errorf("TaskLog of an unknown id: %v, want ErrNotFound", err)
	}
	if _, err := client.SdkInfo(ctx, "no-such-plugin"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("SdkInfo of a plugin that vfox does not know: %v, want ErrUnavailable", err)
	}
	if _, err := client.SdkInfo(ctx, "a/b"); err == nil {
		t.Error("a name with a slash is sent to the server")
	}
}

func TestAPIErrorMatchesTheSentinelOfItsStatus(t *testing.T) {
	cases := map[int][]error{
		http.StatusUnauthorized:        {ErrUnpaired},
		http.StatusNotFound:            {ErrNotFound},
		http.StatusBadGateway:          {ErrUnavailable},
		http.StatusGatewayTimeout:      {ErrUnavailable},
		http.StatusForbidden:           nil, // only the pairing call maps a 403 to ErrPairingRefused
		http.StatusInternalServerError: nil,
	}
	all := []error{ErrUnpaired, ErrNotFound, ErrUnavailable, ErrPairingRefused}
	for status, want := range cases {
		err := error(&APIError{Status: status, Message: "x"})
		for _, sentinel := range all {
			expected := false
			for _, w := range want {
				expected = expected || w == sentinel
			}
			if errors.Is(err, sentinel) != expected {
				t.Errorf("status %d: errors.Is(%v) = %v, want %v", status, sentinel, !expected, expected)
			}
		}
	}
}

func TestTheFingerprintPinsTheCertificate(t *testing.T) {
	s := startServer(t)
	ctx := t.Context()
	session := s.pairRaw(t)

	// Letter case and surrounding white space do not matter; the pinned
	// certificate answers.
	pinned, err := New(s.base, " "+strings.ToLower(s.fingerprint)+" ", session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pinned.Inventory(ctx); err != nil {
		t.Fatalf("the pinned certificate is refused: %v", err)
	}

	// Another fingerprint is a changed certificate: the connection is refused.
	wrong, err := New(s.base, strings.Repeat("00:", 31)+"00", session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Inventory(ctx); !errors.Is(err, ErrCertificateChanged) {
		t.Fatalf("a certificate with another fingerprint: %v, want ErrCertificateChanged", err)
	}

	// Without a fingerprint, the system roots decide, and they do not trust a
	// self-signed certificate.
	system, err := New(s.base, "", session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.Inventory(ctx); err == nil || errors.Is(err, ErrUnpaired) {
		t.Fatalf("a self-signed certificate without a pin: %v, want a certificate error", err)
	}
}

func TestNewValidatesItsArguments(t *testing.T) {
	mustURL := func(text string) *url.URL {
		t.Helper()
		u, err := url.Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	const fingerprint = "00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF"
	bad := []struct {
		name        string
		base        *url.URL
		fingerprint string
	}{
		{"no address", nil, ""},
		{"no host", mustURL("https://"), ""},
		{"unknown scheme", mustURL("ftp://192.168.1.10"), ""},
		{"a path", mustURL("https://192.168.1.10:8369/app"), ""},
		{"a query", mustURL("https://192.168.1.10:8369/?x=1"), ""},
		{"user information", mustURL("https://user@192.168.1.10:8369"), ""},
		{"plain http on the network", mustURL("http://192.168.1.10:8369"), ""},
		{"a fingerprint over plain http", mustURL("http://127.0.0.1:8369"), fingerprint},
		{"a malformed fingerprint", mustURL("https://192.168.1.10:8369"), "00:11"},
	}
	for _, tc := range bad {
		if _, err := New(tc.base, tc.fingerprint, ""); err == nil {
			t.Errorf("%s: New accepted it", tc.name)
		}
	}
	for _, text := range []string{"http://127.0.0.1:8369", "http://localhost:8369", "https://192.168.1.10:8369/"} {
		if _, err := New(mustURL(text), "", ""); err != nil {
			t.Errorf("New(%s): %v", text, err)
		}
	}
}

func TestErrorsDoNotCarryTheTokenOfALink(t *testing.T) {
	// A port that was just closed refuses connections.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	base, err := url.Parse("http://" + address)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(base, "", "")
	if err != nil {
		t.Fatal(err)
	}
	const token = "0123456789abcdef0123456789abcdef"
	_, err = client.Pair(t.Context(), token)
	if err == nil {
		t.Fatal("Pair to a closed port succeeded")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("the error carries the token of the link: %v", err)
	}
}

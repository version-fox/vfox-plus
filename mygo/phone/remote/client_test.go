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
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/phone/internal/phonetest"
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
	client, err := New(s.base, s.certificate, "")
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
	client, err := New(s.base, s.certificate, "")
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
				expected = expected || errors.Is(w, sentinel)
			}
			if errors.Is(err, sentinel) != expected {
				t.Errorf("status %d: errors.Is(%v) = %v, want %v", status, sentinel, !expected, expected)
			}
		}
	}
}

func TestAPinnedCertificateIsTheOnlyRoot(t *testing.T) {
	s := startServer(t)
	other := startServer(t) // Another desktop: a certificate of its own.
	ctx := t.Context()
	session := s.pairRaw(t)

	// The certificate that the user compared is the root, and it answers.
	pinned, err := New(s.base, s.certificate, session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pinned.Inventory(ctx); err != nil {
		t.Fatalf("the pinned certificate is refused: %v", err)
	}

	// Another certificate is not the one that was paired: it is refused, and nothing is
	// trusted in its place. The error says so, and keeps the cause of the refusal.
	wrong, err := New(s.base, other.certificate, session)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrong.Inventory(ctx)
	if !errors.Is(err, ErrCertificateChanged) {
		t.Fatalf("a certificate other than the pinned one: %v, want ErrCertificateChanged", err)
	}
	var unknown x509.UnknownAuthorityError
	if !errors.As(err, &unknown) {
		t.Errorf("the error of a refused certificate lost its cause: %v", err)
	}

	// Without a pin the system roots decide, and they do not vouch for a self-signed
	// certificate: it is not the one that was paired either.
	system, err := New(s.base, nil, session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.Inventory(ctx); !errors.Is(err, ErrCertificateChanged) {
		t.Fatalf("a self-signed certificate without a pin: %v, want ErrCertificateChanged", err)
	}
}

// TestAPinnedCertificateIsVerifiedForTheNameItHas checks that the pinned certificate
// answers when the phone reaches the computer by an address that the certificate does
// not name: it is verified for a name that it has, as the gateway verifies it.
func TestAPinnedCertificateIsVerifiedForTheNameItHas(t *testing.T) {
	cert := phonetest.Valid(t, []string{"desktop.example"}) // Names no address, and the phone dials 127.0.0.1.
	base := serveCertificate(t, cert)
	client, err := New(base, cert.DER, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.call(t.Context(), http.MethodGet, "/api/anything", nil, nil); err != nil {
		t.Fatalf("a pinned certificate that does not name the address is refused: %v", err)
	}
}

// TestAPinnedCertificateThatExpiredFailsAsItself checks that the pinned certificate is
// verified as any is, its dates included: when it expired it is still the one that was
// paired, so the error says that it expired, and not that the certificate changed.
func TestAPinnedCertificateThatExpiredFailsAsItself(t *testing.T) {
	now := time.Now()
	cert := phonetest.Mint(t, now.Add(-2*time.Hour), now.Add(-time.Hour), []string{"localhost"}, net.ParseIP("127.0.0.1"))
	base := serveCertificate(t, cert)
	client, err := New(base, cert.DER, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.call(t.Context(), http.MethodGet, "/api/anything", nil, nil)
	if err == nil {
		t.Fatal("an expired certificate is accepted")
	}
	if errors.Is(err, ErrCertificateChanged) {
		t.Errorf("the pinned certificate itself is reported as changed: %v", err)
	}
	var invalid x509.CertificateInvalidError
	if !errors.As(err, &invalid) || invalid.Reason != x509.Expired {
		t.Errorf("the error does not say that the certificate expired: %v", err)
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
	named := phonetest.Valid(t, []string{"desktop.example"})
	nameless := phonetest.Valid(t, nil)
	bad := []struct {
		name   string
		base   *url.URL
		pinned []byte
	}{
		{"no address", nil, nil},
		{"no host", mustURL("https://"), nil},
		{"unknown scheme", mustURL("ftp://192.168.1.10"), nil},
		{"a path", mustURL("https://192.168.1.10:8369/app"), nil},
		{"a query", mustURL("https://192.168.1.10:8369/?x=1"), nil},
		{"user information", mustURL("https://user@192.168.1.10:8369"), nil},
		{"plain http on the network", mustURL("http://192.168.1.10:8369"), nil},
		{"a certificate over plain http", mustURL("http://127.0.0.1:8369"), named.DER},
		{"a pinned value that is not a certificate", mustURL("https://192.168.1.10:8369"), []byte("not a certificate")},
		{"a certificate that names no host and no address", mustURL("https://192.168.1.10:8369"), nameless.DER},
	}
	for _, tc := range bad {
		if _, err := New(tc.base, tc.pinned, ""); err == nil {
			t.Errorf("%s: New accepted it", tc.name)
		}
	}
	for _, text := range []string{"http://127.0.0.1:8369", "http://localhost:8369", "https://192.168.1.10:8369/"} {
		if _, err := New(mustURL(text), nil, ""); err != nil {
			t.Errorf("New(%s): %v", text, err)
		}
	}
	if _, err := New(mustURL("https://192.168.1.10:8369"), named.DER, ""); err != nil {
		t.Errorf("New with a certificate that names a host: %v", err)
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
	client, err := New(base, nil, "")
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

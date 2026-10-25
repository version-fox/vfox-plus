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

package pair_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

func TestEnsureServerCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(dir, []string{"example.com", "192.0.2.1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fingerprint, ":") || len(strings.ReplaceAll(fingerprint, ":", "")) != 64 {
		t.Errorf("got %q", fingerprint)
	}
	content, err := os.ReadFile(certFile) //nolint:gosec // The test reads the file it just wrote.
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(content)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.NotAfter.Sub(cert.NotBefore) < 9*365*24*time.Hour {
		t.Errorf("validity too short: %v to %v", cert.NotBefore, cert.NotAfter)
	}
	var sans []string
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}
	sans = append(sans, cert.DNSNames...)
	for _, want := range []string{"127.0.0.1", "localhost", "example.com", "192.0.2.1"} {
		found := false
		for _, san := range sans {
			if san == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing SAN %q in %q", want, sans)
		}
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the key must be owner-only, got %o", info.Mode().Perm())
	}
	// A second call reuses the pair instead of minting a new one.
	before, err := os.ReadFile(certFile) //nolint:gosec // The test reads the file it just wrote.
	if err != nil {
		t.Fatal(err)
	}
	_, _, again, err := pair.EnsureServerCert(dir, []string{"other.example"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(certFile) //nolint:gosec // The test reads the file it just wrote.
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || again != fingerprint {
		t.Error("a valid pair must be reused")
	}
	if _, err := pair.CertFingerprint(certFile); err != nil {
		t.Fatal(err)
	}
	if _, err := pair.CertFingerprint(filepath.Join(dir, "missing.pem")); err == nil {
		t.Error("a missing certificate must fail")
	}
}

func tlsClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Tests trust the self-signed certificate they just minted.
	}}
}

func TestTLSServerPairsOverHTTPS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	server, err := pair.NewServer(pair.Options{
		AllowNonLoopback: true,
		TLSCertFile:      certFile,
		TLSKeyFile:       keyFile,
		Fingerprint:      fingerprint,
		Snapshot:         func() pair.Snapshot { return testSnapshot() },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	if !strings.HasPrefix(server.BaseURL(), "https://") {
		t.Errorf("got %q", server.BaseURL())
	}
	client := tlsClient()
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("got %d", response.StatusCode)
	}
	secure := false
	for _, cookie := range response.Cookies() {
		if cookie.Name == "vfox-plus-session" && cookie.Secure {
			secure = true
		}
	}
	if !secure {
		t.Error("the session cookie must be Secure over TLS")
	}
}

func TestPlainServerCookieHasNoSecure(t *testing.T) {
	server := testServer(t, testSnapshot())
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	_, _, cookies := get(t, url, "")
	for _, cookie := range cookies {
		if cookie.Name == "vfox-plus-session" && cookie.Secure {
			t.Error("loopback HTTP must not set Secure")
		}
	}
}

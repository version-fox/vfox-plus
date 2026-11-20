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
	"bytes"
	"crypto/x509"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/internal/phonetest"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

func TestProbeReportsTheFingerprintOfTheDesktopCertificate(t *testing.T) {
	s := startServer(t)
	result, err := Probe(t.Context(), s.base)
	if err != nil {
		t.Fatal(err)
	}
	if result.Fingerprint != s.fingerprint {
		t.Fatalf("Probe fingerprint = %q, the server's is %q", result.Fingerprint, s.fingerprint)
	}
	if !bytes.Equal(result.Certificate, s.certificate) {
		t.Error("Probe does not return the certificate that the server presents: it is what the client pins")
	}
	if result.SystemTrusted {
		t.Error("a self-signed certificate is reported as trusted by the system")
	}
}

func TestProbeOfAnotherTLSServerReportsItsCertificate(t *testing.T) {
	// The server's log is discarded, as the fixtures of the screens discard theirs, so
	// that the output of the tests holds only their own failures.
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Probe(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	if want := pairing.FormatFingerprint(server.Certificate().Raw); result.Fingerprint != want {
		t.Fatalf("Probe fingerprint = %q, want %q", result.Fingerprint, want)
	}
	if !bytes.Equal(result.Certificate, server.Certificate().Raw) {
		t.Error("Probe does not return the certificate that the server presents")
	}
	if result.SystemTrusted {
		t.Error("the test certificate is reported as trusted by the system")
	}
}

// TestProbeSaysThatTheRootsVouchForACertificate checks the one answer that a desktop's own
// certificate never gets: a certificate that the roots verify for the address needs no
// pinning, and the probe says so. The roots are the ones that the test gives, in place of
// the system's.
func TestProbeSaysThatTheRootsVouchForACertificate(t *testing.T) {
	cert := phonetest.Valid(t, []string{"localhost"}, net.ParseIP("127.0.0.1"))
	base := serveCertificate(t, cert)
	result, err := probe(t.Context(), base, rootsOf(t, cert.DER))
	if err != nil {
		t.Fatal(err)
	}
	if !result.SystemTrusted {
		t.Error("a certificate that the roots verify is reported as untrusted")
	}
	if !bytes.Equal(result.Certificate, cert.DER) || result.Fingerprint != pairing.FormatFingerprint(cert.DER) {
		t.Error("the probe of a verified certificate does not describe it")
	}
}

// TestProbeStillShowsACertificateThatDoesNotNameTheAddress checks that a certificate that
// fails the verification for another reason than its authority (here, the address it is
// asked for) is shown to the user as well, as a self-signed one is: the failed connection
// has the certificate that was presented, and nothing is trusted until the user agrees.
func TestProbeStillShowsACertificateThatDoesNotNameTheAddress(t *testing.T) {
	cert := phonetest.Valid(t, []string{"desktop.example"}) // The probe dials 127.0.0.1.
	base := serveCertificate(t, cert)
	for name, roots := range map[string]*x509.CertPool{"the system roots": nil, "roots that hold the certificate": rootsOf(t, cert.DER)} {
		result, err := probe(t.Context(), base, roots)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if result.SystemTrusted {
			t.Errorf("%s: a certificate that does not name the address is reported as trusted", name)
		}
		if !bytes.Equal(result.Certificate, cert.DER) {
			t.Errorf("%s: the probe does not return the certificate that was presented", name)
		}
	}
}

func TestProbeNeedsHTTPSAndAListeningServer(t *testing.T) {
	plain, err := url.Parse("http://127.0.0.1:8369")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(t.Context(), plain); err == nil {
		t.Error("Probe of plain http succeeded")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	closed, err := url.Parse("https://" + address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(t.Context(), closed); err == nil {
		t.Error("Probe of a closed port succeeded")
	}
}

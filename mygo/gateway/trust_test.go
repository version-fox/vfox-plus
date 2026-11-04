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

package gateway

import (
	"crypto/x509"
	"encoding/pem"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

// trust posts what the page that shows the fingerprint posts when the person says that it is the one.
func (b *browser) trust(link, fingerprint string) (*http.Response, string) {
	b.t.Helper()
	return b.post("/gateway/trust", url.Values{"link": {link}, "fingerprint": {fingerprint}})
}

func TestTheFingerprintOfACertificateNobodyVouchesForIsComparedBeforeItIsTrusted(t *testing.T) {
	instance := startSecureInstance(t, t.TempDir(), 0)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	link := instance.link()

	response, body := b.connect(link)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, instance.fingerprint) || !strings.Contains(body, "not signed by an authority") {
		t.Fatalf("the person is shown the fingerprint to compare: %d %.500s", response.StatusCode, body)
	}
	if strings.Contains(body, "not the one you trusted") {
		t.Error("a certificate that was never trusted is warned about as if it had changed")
	}
	if g.Status().Connected || g.Status().Address != "" {
		t.Fatalf("the gateway trusted the certificate before it was compared: %+v", g.Status())
	}
	// The page carries the link and the fingerprint to the next step, and looking at the certificate did not
	// use the link up: it works once.
	if !strings.Contains(body, `name="link" value="`) || !strings.Contains(body, `name="fingerprint" value="`+instance.fingerprint+`"`) {
		t.Errorf("the page does not carry what it needs to the next step: %.800s", body)
	}

	response, body = b.trust(link, instance.fingerprint)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/app" {
		t.Fatalf("trusting the certificate that was compared: %d %.300s", response.StatusCode, body)
	}
	if status := g.Status(); !status.Connected || status.Fingerprint != instance.fingerprint {
		t.Errorf("status = %+v, want the certificate that was compared to be the one trusted", status)
	}
	if status, inventory := b.get("/api/inventory"); status != http.StatusOK || !strings.Contains(inventory, "nodejs") {
		t.Errorf("the interface through HTTPS and the pinned certificate: %d %.200s", status, inventory)
	}
	_, page := b.get("/")
	if !strings.Contains(page, instance.fingerprint) {
		t.Errorf("the page of the gateway does not say which certificate it trusts: %.600s", page)
	}
}

func TestAFingerprintThatIsNotTheOneShownTrustsNothing(t *testing.T) {
	instance := startSecureInstance(t, t.TempDir(), 0)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	link := instance.link()

	wrong := "00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF"
	response, body := b.trust(link, wrong)
	if response.StatusCode != http.StatusConflict || !strings.Contains(body, "Nothing was trusted") {
		t.Errorf("got %d %.300s", response.StatusCode, body)
	}
	if g.Status().Address != "" {
		t.Errorf("the gateway paired anyway: %+v", g.Status())
	}
	// What was refused did not use the link up.
	if response, _ := b.trust(link, instance.fingerprint); response.StatusCode != http.StatusSeeOther {
		t.Errorf("the link was used up by the refusal: %d", response.StatusCode)
	}
}

func TestTheTrustStepNeedsAnHTTPSLinkOfAnInstance(t *testing.T) {
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	for _, link := range []string{"", "nonsense", "http://127.0.0.1:1/auth/connect/abcdef012345"} {
		if response, _ := b.trust(link, "AA"); response.StatusCode != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400", link, response.StatusCode)
		}
	}
}

func TestACertificateThatWasTrustedIsNotAskedAboutAgain(t *testing.T) {
	dir := t.TempDir()
	port := testsupport.FreePort(t)
	instance := startSecureInstance(t, dir, port)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	if response, body := b.trust(instance.link(), instance.fingerprint); response.StatusCode != http.StatusSeeOther {
		t.Fatalf("%d %s", response.StatusCode, body)
	}

	// The instance starts again, as the application does, with the certificate it made and kept, and the
	// sessions it had are gone. Pairing the gateway again does not ask for the comparison a second time.
	_ = instance.server.Close()
	again := startSecureInstance(t, dir, port)
	if again.fingerprint != instance.fingerprint {
		t.Fatal("the application made another certificate")
	}
	if status, _ := b.get("/api/inventory"); status != http.StatusUnauthorized {
		t.Fatalf("the session of the instance that ended: %d, want 401", status)
	}
	response, body := b.connect(again.link())
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/app" {
		t.Errorf("a certificate that was compared before: %d %.300s", response.StatusCode, body)
	}
	if status, _ := b.get("/api/inventory"); status != http.StatusOK {
		t.Errorf("after pairing again: %d", status)
	}
}

func TestACertificateThatChangedIsNotTrustedAndIsAskedAboutWithAWarning(t *testing.T) {
	port := testsupport.FreePort(t)
	first := startSecureInstance(t, t.TempDir(), port)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	if response, body := b.trust(first.link(), first.fingerprint); response.StatusCode != http.StatusSeeOther {
		t.Fatalf("%d %s", response.StatusCode, body)
	}
	if status, _ := b.get("/api/inventory"); status != http.StatusOK {
		t.Fatalf("before: %d", status)
	}

	// Somebody else answers at the address, or the application made a new certificate. The gateway does not
	// talk to it with the session it has: a certificate that is not the one that was compared is refused.
	_ = first.server.Close()
	second := startSecureInstance(t, t.TempDir(), port)
	if second.fingerprint == first.fingerprint {
		t.Fatal("the two instances have the same certificate")
	}
	response := b.send(http.MethodGet, "/api/inventory", nil, nil)
	body := readAll(t, response)
	if response.StatusCode != http.StatusBadGateway || !strings.Contains(body, "is not the one that was trusted") ||
		!strings.Contains(body, second.fingerprint) || !strings.Contains(body, first.fingerprint) {
		t.Errorf("a certificate that is not the trusted one: %d %s", response.StatusCode, body)
	}
	if !g.Status().Connected {
		t.Error("the gateway dropped its session because of a certificate that it did not trust, which tells it nothing of the session")
	}

	// Pairing it with the new instance asks again, and says why.
	response2, page := b.connect(second.link())
	if response2.StatusCode != http.StatusOK || !strings.Contains(page, "not the one you trusted before") || !strings.Contains(page, second.fingerprint) {
		t.Errorf("the page for a certificate that changed: %d %.600s", response2.StatusCode, page)
	}
	if g.Status().Fingerprint != first.fingerprint {
		t.Errorf("the gateway moved to the new certificate before it was compared: %+v", g.Status())
	}
	if response, _ := b.trust(second.link(), second.fingerprint); response.StatusCode != http.StatusSeeOther {
		t.Errorf("trusting the new certificate: %d", response.StatusCode)
	}
	if g.Status().Fingerprint != second.fingerprint {
		t.Errorf("status = %+v", g.Status())
	}
}

func TestACertificateThatTheRootsVouchForNeedsNoComparison(t *testing.T) {
	instance := startSecureInstance(t, t.TempDir(), 0)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)

	// The authority that signed it, here the certificate itself, is among the roots.
	pool := x509.NewCertPool()
	content, err := os.ReadFile(instance.certFile)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(content)
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pool.AddCert(certificate)
	g.roots = pool

	response, body := b.connect(instance.link())
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/app" {
		t.Fatalf("a certificate that verifies is trusted without a comparison: %d %.400s", response.StatusCode, body)
	}
	if status := g.Status(); !status.Connected || status.Fingerprint != "" {
		t.Errorf("status = %+v, want nothing pinned: the roots decide", status)
	}
	if status, _ := b.get("/api/inventory"); status != http.StatusOK {
		t.Errorf("the interface through a certificate that verifies: %d", status)
	}
}

func TestAnInstanceThatSpeaksPlainHTTPOnTheLoopbackAddressNeedsNoCertificate(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	if status := g.Status(); !status.Connected || status.Fingerprint != "" {
		t.Errorf("status = %+v", status)
	}
}

func TestAnInstanceThatCannotBeReachedIsSaidSoWhenTheGatewayIsPaired(t *testing.T) {
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	free := testsupport.FreePort(t)
	for name, link := range map[string]string{
		"plain HTTP": "http://127.0.0.1:" + strconv.Itoa(free) + pairingPath + "abcdef0123456789",
		"HTTPS":      "https://127.0.0.1:" + strconv.Itoa(free) + pairingPath + "abcdef0123456789",
	} {
		response, body := b.connect(link)
		// What the program says is escaped in the page: the + of vfox+ is &#43;.
		message := html.UnescapeString(body)
		if response.StatusCode != http.StatusBadGateway || !strings.Contains(message, "Cannot reach vfox+ at 127.0.0.1:"+strconv.Itoa(free)) {
			t.Errorf("%s: got %d %s", name, response.StatusCode, message[strings.Index(message, `role="alert"`):])
		}
		if !strings.Contains(message, "--network host") {
			t.Errorf("%s: a loopback address in a container is the container itself, and the page does not say so", name)
		}
	}
	if g.Status().Address != "" {
		t.Errorf("the gateway is paired with an instance that is not there: %+v", g.Status())
	}
}

func TestALinkThatTheInstanceRefusesIsSaidSo(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	link := instance.link()
	pairGateway(t, b, link)

	// A link works once.
	response, body := b.connect(link)
	if response.StatusCode != http.StatusForbidden || !strings.Contains(body, "was used already or has expired") {
		t.Errorf("a link used twice: %d %.400s", response.StatusCode, body)
	}
	if !g.Status().Connected {
		t.Error("a link that was refused ended the pairing that worked")
	}
}

func TestWhatIsPastedThatIsNotAPairingLinkIsSaidSoAndNothingIsContacted(t *testing.T) {
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	for _, link := range []string{
		"", "hello", "https://", "ftp://x/auth/connect/abcdef0123", "https://host/other/abcdef0123",
		"https://host:8369/auth/connect/", "https://host:8369/auth/connect/a", "https://host:8369/auth/connect/ab/cd",
		"http://192.168.1.2:8369/auth/connect/abcdef0123", "https://user:password@host:8369/auth/connect/abcdef0123",
	} {
		response, body := b.connect(link)
		if response.StatusCode != http.StatusBadRequest || !strings.Contains(body, `role="alert"`) {
			t.Errorf("%q: got %d, want 400 with the reason", link, response.StatusCode)
		}
	}
}

func readAll(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

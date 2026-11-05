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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

// syncBuffer is what the gateway writes its announcement to while the test reads it.
type syncBuffer struct {
	mu   sync.Mutex
	text strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.text.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.text.String()
}

// running is a gateway that serves HTTPS on a port of its own.
type running struct {
	t       *testing.T
	g       *Gateway
	address string
	banner  *syncBuffer
	stop    context.CancelFunc
	ended   chan error
	once    sync.Once
	err     error
}

// shutDown tells the gateway to stop, waits for it, and returns how it ended. It may be called again.
func (r *running) shutDown() error {
	r.once.Do(func() {
		r.stop()
		r.err = <-r.ended
	})
	return r.err
}

// serve starts Serve on a loopback port and waits for its announcement.
func serve(t *testing.T, dir string) *running {
	t.Helper()
	g, err := New(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	r := &running{t: t, g: g, address: listener.Addr().String(), banner: &syncBuffer{}, stop: stop, ended: make(chan error, 1)}
	go func() { r.ended <- g.Serve(ctx, listener, r.banner) }()
	t.Cleanup(func() { _ = r.shutDown() })
	waitUntil(t, "the announcement of the gateway", func() bool { return strings.Contains(r.banner.String(), pairingPath) })
	return r
}

// link is the pairing link the announcement gives, for the address that reaches the gateway.
func (r *running) link() string {
	r.t.Helper()
	match := regexp.MustCompile(`https://localhost:(\d+)(/auth/connect/\S+)`).FindStringSubmatch(r.banner.String())
	if match == nil {
		r.t.Fatalf("no pairing link in the announcement:\n%s", r.banner.String())
	}
	return "https://" + r.address + match[2]
}

// fingerprint is the fingerprint the announcement gives.
func (r *running) fingerprint() string {
	r.t.Helper()
	match := regexp.MustCompile(`(?m)^  ((?:[0-9A-F]{2}:){31}[0-9A-F]{2})$`).FindStringSubmatch(r.banner.String())
	if match == nil {
		r.t.Fatalf("no fingerprint in the announcement:\n%s", r.banner.String())
	}
	return match[1]
}

// client is a browser that trusts the certificate with the fingerprint, and no other, as a person does who has
// compared it with the one that the gateway printed.
func (r *running) client(fingerprint string) *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		r.t.Fatal(err)
	}
	return &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Verified by fingerprint below.
			VerifyConnection: func(connection tls.ConnectionState) error {
				if got := pair.Fingerprint(connection.PeerCertificates[0].Raw); got != fingerprint {
					return fmt.Errorf("the certificate is %s, want %s", got, fingerprint)
				}
				return nil
			},
		}},
	}
}

func TestTheGatewayServesHTTPSAndSaysHowToPairABrowser(t *testing.T) {
	instance := startFakeInstance(t)
	r := serve(t, t.TempDir())
	banner := r.banner.String()
	for _, want := range []string{"vfox+ gateway", "Serving over HTTPS on port", "Not paired with a vfox+ yet", "expires in 5 minutes", "localhost replaced"} {
		if !strings.Contains(banner, want) {
			t.Errorf("the announcement lacks %q:\n%s", want, banner)
		}
	}
	client := r.client(r.fingerprint())

	// The browser pairs with the link of the announcement, and lands on the page of the gateway.
	response, err := client.Get(r.link())
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(page), "not paired with a vfox+ yet") {
		t.Fatalf("pairing the browser: %d %.300s", response.StatusCode, page)
	}
	// The link is used up.
	if again, err := client.Get(r.link()); err != nil || again.StatusCode != http.StatusForbidden {
		t.Errorf("a link used twice: %v %v", again, err)
	}

	// And the gateway is paired with an instance on that page, and serves its interface over HTTPS.
	request, _ := http.NewRequest(http.MethodPost, "https://"+r.address+"/gateway/connect", strings.NewReader(url.Values{"link": {instance.link()}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://"+r.address)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	page, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(page), "the page of the instance") {
		t.Errorf("after pairing the gateway: %d %.300s", response.StatusCode, page)
	}
	if response.Request.URL.Path != "/app" {
		t.Errorf("the browser ended at %s, want /app", response.Request.URL.Path)
	}
}

func TestTheGatewayKeepsItsCertificateAndItsInstanceBetweenRuns(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	first := serve(t, dir)
	client := first.client(first.fingerprint())
	if response, err := client.Get(first.link()); err != nil {
		t.Fatal(err)
	} else {
		_ = response.Body.Close()
	}
	request, _ := http.NewRequest(http.MethodPost, "https://"+first.address+"/gateway/connect", strings.NewReader(url.Values{"link": {instance.link()}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://"+first.address)
	if response, err := client.Do(request); err != nil {
		t.Fatal(err)
	} else {
		_ = response.Body.Close()
	}
	fingerprint := first.fingerprint()
	if err := first.shutDown(); err != nil {
		t.Fatalf("Serve ended with %v, want nil: it was told to stop", err)
	}

	second := serve(t, dir)
	if second.fingerprint() != fingerprint {
		t.Errorf("another certificate: %s then %s, and a browser that trusted the first would be asked again", fingerprint, second.fingerprint())
	}
	if !strings.Contains(second.banner.String(), "Paired with vfox+ at "+strings.TrimPrefix(instance.server.URL, "http://")) {
		t.Errorf("the announcement does not say that it is paired:\n%s", second.banner.String())
	}
	// The browsers of the first run are not paired: that is not remembered, and the link of this run is needed.
	if response, err := second.client(fingerprint).Get("https://" + second.address + "/app"); err != nil || response.Request.URL.Path != "/" {
		t.Errorf("a browser of the first run: %v %v", response, err)
	}
}

func TestTheAnnouncementSaysThatTheSessionHasEnded(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	g, server := serveGateway(t, Options{Dir: dir})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	instance.endSessions()
	b.get("/api/echo")

	r := serve(t, dir)
	if !strings.Contains(r.banner.String(), "has ended: pair the gateway again on its page") {
		t.Errorf("the announcement does not say that the session ended:\n%s", r.banner.String())
	}
}

func TestTheCertificateOfThePersonIsServedInsteadOfTheOneTheGatewayMakes(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(other, []string{"gateway.example"})
	if err != nil {
		t.Fatal(err)
	}
	for from, to := range map[string]string{certFile: ownCertFile, keyFile: ownKeyFile} {
		content, err := os.ReadFile(from) //nolint:gosec // A file of this test.
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, to), content, 0o600); err != nil { //nolint:gosec // A file of this test.
			t.Fatal(err)
		}
	}
	r := serve(t, dir)
	if got := r.fingerprint(); got != fingerprint {
		t.Errorf("the announcement says %s, want that of the certificate in the directory, %s", got, fingerprint)
	}
	if _, err := r.client(fingerprint).Get("https://" + r.address + "/"); err != nil {
		t.Errorf("the certificate that is served is not the one in the directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "serve-cert.pem")); err == nil {
		t.Error("the gateway made a certificate of its own beside the one that it was given")
	}
}

func TestACertificateOfThePersonThatCannotBeUsedStopsTheGateway(t *testing.T) {
	other := t.TempDir()
	certFile, keyFile, _, err := pair.EnsureServerCert(other, nil)
	if err != nil {
		t.Fatal(err)
	}
	another := t.TempDir()
	_, otherKeyFile, _, err := pair.EnsureServerCert(another, nil)
	if err != nil {
		t.Fatal(err)
	}
	copyFile := func(from, dir, name string) {
		content, err := os.ReadFile(from) //nolint:gosec // A file of this test.
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil { //nolint:gosec // A file of this test.
			t.Fatal(err)
		}
	}

	onlyCertificate := t.TempDir()
	copyFile(certFile, onlyCertificate, ownCertFile)
	if _, _, _, err := certificate(onlyCertificate); err == nil || !strings.Contains(err.Error(), "go together") {
		t.Errorf("a certificate without a key: %v", err)
	}
	onlyKey := t.TempDir()
	copyFile(keyFile, onlyKey, ownKeyFile)
	if _, _, _, err := certificate(onlyKey); err == nil || !strings.Contains(err.Error(), "go together") {
		t.Errorf("a key without a certificate: %v", err)
	}
	mismatched := t.TempDir()
	copyFile(certFile, mismatched, ownCertFile)
	copyFile(otherKeyFile, mismatched, ownKeyFile)
	if _, _, _, err := certificate(mismatched); err == nil || !strings.Contains(err.Error(), "cannot be used") {
		t.Errorf("a key of another certificate: %v", err)
	}

	// Serve says so, and does not serve with another.
	g, err := New(Options{Dir: mismatched})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	if err := g.Serve(context.Background(), listener, io.Discard); err == nil {
		t.Error("the gateway served with a certificate that cannot be used")
	}
}

func TestServeEndsWhenItsListenerIsGone(t *testing.T) {
	g, err := New(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = listener.Close()
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	if err := g.Serve(ctx, listener, io.Discard); err == nil {
		t.Error("Serve ended without an error on a listener that is closed")
	}
}

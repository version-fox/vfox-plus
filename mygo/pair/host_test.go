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
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// hostFixture is a host on the application state of the captured output of vfox,
// with a directory of its own for the certificate.
type hostFixture struct {
	host  *pair.Host
	model *state.AppModel
	dir   string
	quit  atomic.Int32
}

func newHost(t *testing.T) *hostFixture {
	t.Helper()
	fixture := &hostFixture{model: testModel(t, testsupport.FixtureRunner()), dir: t.TempDir()}
	fixture.host = pair.NewHost(fixture.model, pair.HostOptions{SettingsDir: fixture.dir, Quit: func() { fixture.quit.Add(1) }})
	t.Cleanup(fixture.host.Close)
	return fixture
}

// set changes the settings and makes the servers match them.
func (f *hostFixture) set(change func(*settings.AppSettings)) {
	f.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		change(&current)
		return current
	})
	f.host.Apply()
}

func TestTheWebInterfaceIsServedOnlyWhileTheSettingsSayAndNeverToTheNetworkByDefault(t *testing.T) {
	f := newHost(t)
	f.host.Start()
	if status := f.host.Status(); status.Running || status.Remote || status.LocalURL != "" {
		t.Fatalf("nothing is served by default: %v", status)
	}
	if _, err := f.host.LocalLink(); !errors.Is(err, pair.ErrNotServing) {
		t.Errorf("no link without a server, got %v", err)
	}
	if _, err := f.host.RemoteLink(""); !errors.Is(err, pair.ErrNoRemote) {
		t.Errorf("no link for other devices without a server for them, got %v", err)
	}

	f.set(func(s *settings.AppSettings) { s.WebInterface = true })
	status := f.host.Status()
	if !status.Running || !strings.HasPrefix(status.LocalURL, "http://127.0.0.1:") || status.Remote || status.Error != "" {
		t.Fatalf("the web interface listens on the loopback interface only: %v", status)
	}
	if len(status.Addresses) != 0 || status.Fingerprint != "" {
		t.Errorf("a server on this computer has no addresses of the network and no certificate: %+v", status)
	}
	link, err := f.host.LocalLink()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link, status.LocalURL+"/auth/connect/") {
		t.Errorf("got %q", link)
	}
	response := request(t, link, "")
	if response.StatusCode != http.StatusSeeOther || sessionOf(response.Cookies()) == "" {
		t.Fatalf("the link pairs a browser, got %d", response.StatusCode)
	}

	f.set(func(s *settings.AppSettings) { s.WebInterface = false })
	if status := f.host.Status(); status.Running || status.LocalURL != "" {
		t.Errorf("turning the web interface off stops it: %v", status)
	}
	if _, err := http.Get(link); err == nil { //nolint:gosec,noctx // The test asks the server it stopped.
		t.Error("the server still answers after it was turned off")
	}
}

func TestTheHostFollowsTheSettingsAndTellsWhoListens(t *testing.T) {
	f := newHost(t)
	f.host.Start()
	var told atomic.Int32
	stop := f.host.AddListener(func() { told.Add(1) })
	defer stop()

	// Whoever changes the settings, such as the window or the page, does not wait for the host.
	f.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.WebInterface = true
		return current
	})
	deadline := time.Now().Add(5 * time.Second)
	for !f.host.Status().Running {
		if time.Now().After(deadline) {
			t.Fatal("the host did not follow the settings")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if told.Load() == 0 {
		t.Error("the host tells its listeners that it started")
	}
	// A change that is none of its business is none of its business.
	before := told.Load()
	f.set(func(s *settings.AppSettings) { s.RailExpanded = true })
	if told.Load() != before {
		t.Error("the host tells nothing when the servers stay as they are")
	}
	stop()
	f.host.Close()
	if status := f.host.Status(); status.Running {
		t.Errorf("a closed host serves nothing: %v", status)
	}
	f.set(func(s *settings.AppSettings) { s.RailExpanded = false; s.Searches = map[string]string{"java": "tem"} })
	f.host.Apply()
	if status := f.host.Status(); status.Running {
		t.Errorf("a closed host stays closed: %v", status)
	}
}

func TestStartingTheHostWithTheWebInterfaceAlreadyOnServesAtOnce(t *testing.T) {
	f := newHost(t)
	f.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.WebInterface = true
		return current
	})
	f.host.Start()
	if !f.host.Status().Running {
		t.Error("the settings of the last run are served at start")
	}
	f.host.Start() // A second start follows the settings once.
}

func TestOtherDevicesConnectOverHTTPSAndOnlyWhileAllowed(t *testing.T) {
	f := newHost(t)
	port := testsupport.FreePort(t)
	f.set(func(s *settings.AppSettings) { s.WebInterface = true })
	// Allowing other devices is not enough without the web interface: it is off here.
	f.set(func(s *settings.AppSettings) { s.WebInterface, s.WebRemote, s.WebPort = false, true, port })
	if status := f.host.Status(); status.Running || status.Remote {
		t.Fatalf("other devices cannot connect to a web interface that is off: %v", status)
	}
	f.set(func(s *settings.AppSettings) { s.WebInterface = true })
	status := f.host.Status()
	if !status.Running || !status.Remote || status.Port != port || status.RemoteError != "" {
		t.Fatalf("got %v", status)
	}
	if !status.SelfSigned || len(strings.ReplaceAll(status.Fingerprint, ":", "")) != 64 {
		t.Errorf("the application makes a certificate for itself and shows its fingerprint: %+v", status)
	}
	if status.LocalURL == "" || strings.HasPrefix(status.LocalURL, "https") {
		t.Errorf("this computer keeps its own server in the clear: %q", status.LocalURL)
	}
	if _, err := filepath.Glob(filepath.Join(f.dir, "*.pem")); err != nil {
		t.Fatal(err)
	}

	// Whatever address the computer has, the server answers on it over TLS, and the link pairs.
	if len(status.Addresses) == 0 {
		if _, err := f.host.RemoteLink(""); !errors.Is(err, pair.ErrNoAddress) {
			t.Errorf("a computer without an address cannot be reached, got %v", err)
		}
	} else {
		link, err := f.host.RemoteLink("")
		if err != nil {
			t.Fatal(err)
		}
		if want := "https://" + hostPort(status.Addresses[0], port) + "/auth/connect/"; !strings.HasPrefix(link, want) {
			t.Fatalf("got %q, want the prefix %q", link, want)
		}
		response, err := noRedirectTLS().Get(link)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusSeeOther || sessionOf(response.Cookies()) == "" {
			t.Errorf("the link pairs a device over TLS, got %d", response.StatusCode)
		}
		if _, err := f.host.RemoteLink("203.0.113.77"); !errors.Is(err, pair.ErrNoAddress) {
			t.Errorf("an address that the computer does not have is refused, got %v", err)
		}
		if _, err := f.host.RemoteLink(status.Addresses[len(status.Addresses)-1]); err != nil {
			t.Errorf("any address the computer has will do: %v", err)
		}
	}

	// A device on the network is paired with its own server: the links of this computer do not pair it.
	local, err := f.host.LocalLink()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(local, strconv.Itoa(port)) {
		t.Errorf("the link for this computer is for its own server: %q", local)
	}

	// Turning the access off closes the server, with the connections that it holds.
	f.set(func(s *settings.AppSettings) { s.WebRemote = false })
	if status := f.host.Status(); !status.Running || status.Remote || len(status.Addresses) != 0 || status.Fingerprint != "" {
		t.Errorf("only the server for other devices stops: %v", status)
	}
	if _, err := f.host.RemoteLink(""); !errors.Is(err, pair.ErrNoRemote) {
		t.Errorf("got %v", err)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err == nil {
		_ = conn.Close()
		t.Error("the port still answers after the access of other devices was turned off")
	}
}

func hostPort(host string, port int) string { return net.JoinHostPort(host, strconv.Itoa(port)) }

func TestTurningTheAccessOffEndsTheConnectionOfADeviceThatUsesIt(t *testing.T) {
	f := newHost(t)
	port := testsupport.FreePort(t)
	f.set(func(s *settings.AppSettings) { s.WebInterface, s.WebRemote, s.WebPort = true, true, port })
	if !f.host.Status().Remote {
		t.Fatalf("got %v", f.host.Status())
	}
	base := "https://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	// Pair through the server for other devices itself: its tokens are its own.
	link, err := f.host.RemoteLink("")
	if errors.Is(err, pair.ErrNoAddress) {
		t.Skip("this computer has no address on a network")
	}
	if err != nil {
		t.Fatal(err)
	}
	client := noRedirectTLS()
	_, token, _ := strings.Cut(link, "/auth/connect/")
	response, err := client.Get(base + "/auth/connect/" + token)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	session := sessionOf(response.Cookies())
	if session == "" {
		t.Fatalf("pairing gave no session: %d", response.StatusCode)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	events.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: session, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	stream, err := client.Do(events)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stream.Body.Close() }()
	reader := bufio.NewReader(stream.Body)
	if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "event: hello") {
		t.Fatalf("got %q, %v", line, err)
	}

	f.set(func(s *settings.AppSettings) { s.WebRemote = false })
	ended := make(chan error, 1)
	go func() {
		_, err := reader.ReadString('\n')
		for err == nil {
			_, err = reader.ReadString('\n')
		}
		ended <- err
	}()
	select {
	case err := <-ended:
		if err == nil {
			t.Error("a stream that was cut ends with an error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a device that was connected stays connected after the access was turned off")
	}
	// Coming back is a new pairing: the old session is gone with its server.
	f.set(func(s *settings.AppSettings) { s.WebRemote = true })
	again, err := client.Get(base + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	_ = again.Body.Close()
	if again.StatusCode != http.StatusUnauthorized {
		t.Errorf("a device is unpaired once the access was turned off, got %d", again.StatusCode)
	}
}

func TestOtherDevicesCannotConnectWhenThePortIsTakenOrTheCertificateIsWrong(t *testing.T) {
	f := newHost(t)
	taken, err := net.Listen("tcp", "0.0.0.0:0") //nolint:gosec // The test takes the port that the server for other devices wants.
	if err != nil {
		t.Skipf("no interface to take a port on: %v", err)
	}
	defer func() { _ = taken.Close() }()
	port := taken.Addr().(*net.TCPAddr).Port
	if !settings.ValidWebPort(port) {
		t.Skipf("the system gave the port %d, which the settings do not accept", port)
	}
	f.set(func(s *settings.AppSettings) { s.WebInterface, s.WebRemote, s.WebPort = true, true, port })
	status := f.host.Status()
	if !status.Running || status.Remote || status.RemoteError == "" {
		t.Fatalf("a port in use is no reason to stop the web interface for this computer, but it is a reason to say so: %v", status)
	}
	if _, err := f.host.RemoteLink(""); !errors.Is(err, pair.ErrNoRemote) {
		t.Errorf("got %v", err)
	}
	// Another port is another try.
	f.set(func(s *settings.AppSettings) { s.WebPort = testsupport.FreePort(t) })
	if status := f.host.Status(); !status.Remote || status.RemoteError != "" {
		t.Errorf("got %v", status)
	}
	// A certificate that is not there, and a certificate without its key, say why.
	for name, change := range map[string]func(*settings.AppSettings){
		"a certificate that is not there": func(s *settings.AppSettings) {
			s.WebCertificate, s.WebKey = filepath.Join(f.dir, "none.pem"), filepath.Join(f.dir, "none-key.pem")
		},
		"a certificate without its key": func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = filepath.Join(f.dir, "serve-cert.pem"), "" },
		"a key without its certificate": func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = "", filepath.Join(f.dir, "serve-key.pem") },
	} {
		f.set(func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = "", "" })
		f.set(change)
		if status := f.host.Status(); !status.Running || status.Remote || status.RemoteError == "" {
			t.Errorf("%s: %v", name, status)
		}
	}
	f.set(func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = "", "" })
	if status := f.host.Status(); !status.Remote {
		t.Errorf("the generated certificate comes back: %v", status)
	}
}

func TestTheCertificateOfTheUserIsTheOneThatOtherDevicesGet(t *testing.T) {
	f := newHost(t)
	certDir := t.TempDir()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(certDir, []string{"vfox.example"})
	if err != nil {
		t.Fatal(err)
	}
	f.set(func(s *settings.AppSettings) {
		s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(t)
		s.WebCertificate, s.WebKey = certFile, keyFile
	})
	status := f.host.Status()
	if !status.Remote || status.SelfSigned || status.Fingerprint != fingerprint {
		t.Fatalf("the user's certificate is not self-signed by the application, and its fingerprint is the one to compare: %+v", status)
	}
	matches, err := filepath.Glob(filepath.Join(f.dir, "*.pem"))
	if err != nil || len(matches) != 0 {
		t.Errorf("the application makes no certificate of its own when it is given one: %v %v", matches, err)
	}
}

func TestTheHostAsksTheApplicationToQuitOnlyWhenToldTo(t *testing.T) {
	f := newHost(t)
	f.host.Quit()
	if f.quit.Load() != 1 {
		t.Error("quitting asks the application")
	}
	pair.NewHost(f.model, pair.HostOptions{}).Quit() // Without a way to quit, it does nothing.
}

func TestNetworkAddressesAreIPv4FirstAndNeverLoopback(t *testing.T) {
	seen := map[string]bool{}
	sawV6 := false
	for _, address := range pair.NetworkAddresses() {
		ip := net.ParseIP(address)
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			t.Errorf("%q is no address to connect to", address)
		}
		if seen[address] {
			t.Errorf("%q is listed twice", address)
		}
		seen[address] = true
		if ip.To4() == nil {
			sawV6 = true
		} else if sawV6 {
			t.Errorf("IPv4 comes first, but %q follows an IPv6 address", address)
		}
	}
}

// noRedirectTLS is a client of the server for other devices, that trusts the
// certificate the application made for itself and shows the answers, redirects
// included.
func noRedirectTLS() *http.Client {
	client := tlsClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client
}

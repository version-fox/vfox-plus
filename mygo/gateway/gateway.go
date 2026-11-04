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
	"crypto/x509"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultRenewEvery is how often the gateway renews its session at the instance: a quarter of the
	// day that a session lasts, so that the instance has to be out of reach for most of a day for the
	// session to end.
	DefaultRenewEvery = 6 * time.Hour
	// retryRenewAfter is how soon the gateway tries to renew its session again when the instance could not
	// be reached.
	retryRenewAfter = 5 * time.Minute
)

// maxForm is the most that a form of the gateway may send.
const maxForm = 8 << 10

// Options configures a Gateway.
type Options struct {
	// Dir is the directory that the gateway remembers in: the instance it is paired with and its
	// session (private), and the certificate that it serves with.
	Dir string
	// RenewEvery is how often the session at the instance is renewed, DefaultRenewEvery when zero.
	RenewEvery time.Duration

	// clock is the time that the pairing links and the sessions of the browsers go by, time.Now when nil. The tests
	// move it; a link works for DefaultTokenTTL and a session lasts DefaultSessionTTL.
	clock func() time.Time
}

// Status says to which instance the gateway is paired.
type Status struct {
	// Address is the host and port of the instance, empty for a gateway that was never paired or
	// that forgot its instance.
	Address string
	// Connected says that the gateway holds a session at the instance, and so serves its interface.
	Connected bool
	// Fingerprint is the SHA-256 fingerprint of the certificate of the instance that was trusted, in the
	// form that the settings of the application show, empty when the roots of the system vouch for it or
	// the instance speaks HTTP on a loopback address.
	Fingerprint string
	// RenewedAt is when the session was made, or renewed last.
	RenewedAt time.Time
}

// Gateway is the interface of one instance, served to the browsers that are paired with it.
type Gateway struct {
	options     Options
	browsers    *browsers
	crossOrigin *http.CrossOriginProtection
	proxy       *httputil.ReverseProxy
	// fingerprint is the certificate that the gateway serves with, which its intro page shows for the
	// comparison with what it printed. Set before it serves.
	fingerprint string
	// roots are the authorities that vouch for the certificate of an instance; nil, the system's. The
	// tests put their own.
	roots *x509.CertPool

	// mu guards up, and the file that remembers it.
	mu sync.Mutex
	up *upstream
}

// New makes a gateway that remembers in options.Dir, and is paired with the instance that it was paired
// with when it last ran, if any.
func New(options Options) (*Gateway, error) {
	if options.Dir == "" {
		return nil, errors.New("gateway: a directory to remember in is needed")
	}
	if options.RenewEvery <= 0 {
		options.RenewEvery = DefaultRenewEvery
	}
	g := &Gateway{
		options:     options,
		browsers:    newBrowsers(DefaultTokenTTL, DefaultSessionTTL, options.clock),
		crossOrigin: http.NewCrossOriginProtection(),
	}
	g.proxy = g.newProxy()
	if remembered := loadState(options.Dir); remembered.Upstream != "" {
		if base, err := url.Parse(remembered.Upstream); err == nil && (base.Scheme == "https" || (base.Scheme == "http" && isLoopback(base.Hostname()))) && base.Host != "" {
			// A certificate that cannot be read is no certificate that was compared: the gateway is paired with nothing.
			var pinned *x509.Certificate
			if len(remembered.Certificate) > 0 {
				pinned, err = x509.ParseCertificate(remembered.Certificate)
			}
			if err == nil {
				g.up = newUpstream(&url.URL{Scheme: base.Scheme, Host: base.Host}, pinned, g.roots).withSession(remembered.Session, remembered.RenewedAt)
			}
		}
	}
	return g, nil
}

// NewToken makes a one-time token that pairs a browser with the gateway: its link is
// /auth/connect/<token> on the address that reaches the gateway.
func (g *Gateway) NewToken() string { return g.browsers.newToken() }

// Status says to which instance the gateway is paired.
func (g *Gateway) Status() Status {
	up := g.current()
	if up == nil {
		return Status{}
	}
	return Status{Address: up.base.Host, Connected: up.connected(), Fingerprint: up.fingerprint(), RenewedAt: up.renewedAt}
}

// Handler is the gateway: what it serves itself, and the instance's interface that it forwards.
func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	// What the gateway serves itself.
	mux.HandleFunc("GET /{$}", g.serveHome)
	mux.HandleFunc("GET "+pairingPath, g.serveBrowserPairing)
	mux.HandleFunc("/api/token", g.serveToken)
	mux.HandleFunc("POST /gateway/connect", g.serveConnect)
	mux.HandleFunc("POST /gateway/trust", g.serveTrust)
	mux.HandleFunc("POST /gateway/invite", g.serveInvite)
	mux.HandleFunc("POST /gateway/forget", g.serveForget)
	// What is the instance's.
	mux.HandleFunc("/app", g.serveInstance)
	mux.HandleFunc("/api/", g.serveInstance)
	mux.HandleFunc("/manifest.webmanifest", g.serveManifest)
	return mux
}

// current is the instance that the gateway is paired with, if any.
func (g *Gateway) current() *upstream {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.up
}

// replace pairs the gateway with next (nil: with none) and remembers it. The connections to the instance
// that it leaves are closed, unless next is the same instance with another session.
func (g *Gateway) replace(next *upstream) {
	g.mu.Lock()
	previous := g.up
	g.up = next
	g.rememberLocked()
	g.mu.Unlock()
	if previous != nil && (next == nil || next.transport != previous.transport) {
		previous.transport.CloseIdleConnections()
	}
}

// swap is replace, for when previous is still the instance that the gateway is paired with: a renewal that
// finishes after the gateway was paired with another instance leaves that one alone.
func (g *Gateway) swap(previous, next *upstream) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.up != previous {
		return false
	}
	g.up = next
	g.rememberLocked()
	return true
}

// invalidate is what the gateway does when the instance says that it does not know the session any more: it
// keeps the instance, to offer it to be paired again, and drops the session.
func (g *Gateway) invalidate(up *upstream) {
	if !up.connected() {
		return
	}
	if g.swap(up, up.withSession("", time.Time{})) {
		log.Printf("gateway: the session at %s has ended; pair the gateway again on its page", up.base.Host)
	}
}

// rememberLocked writes what the gateway remembers. A failure is reported and no more: the gateway works
// until it ends.
func (g *Gateway) rememberLocked() {
	var state saved
	if up := g.up; up != nil {
		state = saved{Upstream: up.base.String(), Session: up.session, RenewedAt: up.renewedAt}
		if up.pinned != nil {
			state.Certificate = up.pinned.Raw
		}
	}
	if err := saveState(g.options.Dir, state); err != nil {
		log.Printf("gateway: cannot remember the instance in %s: %v", g.options.Dir, err)
	}
}

// trusted is the fingerprint that was trusted for the instance at base, if the gateway is paired with that
// one.
func (g *Gateway) trusted(base *url.URL) string {
	if up := g.current(); up != nil && up.base.Host == base.Host {
		return up.fingerprint()
	}
	return ""
}

// renew pairs the gateway with the instance again, with a link that the instance gives its session: the
// session that results lasts a day from now. It is what keeps the gateway paired for as long as the
// instance runs.
func (g *Gateway) renew(ctx context.Context) error {
	up := g.current()
	if !up.connected() {
		return errNotConnected
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	token, err := up.mint(ctx)
	if errors.Is(err, errSessionEnded) {
		g.invalidate(up)
		return err
	}
	if err != nil {
		return err
	}
	session, err := up.pairWith(ctx, token)
	if err != nil {
		return err
	}
	g.swap(up, up.withSession(session, time.Now()))
	return nil
}

// keepPaired renews the session at the instance now, to see that it is good, and then as often as the options say,
// until ctx is done.
func (g *Gateway) keepPaired(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		wait := g.options.RenewEvery
		if err := g.renew(ctx); err != nil && !errors.Is(err, errNotConnected) && !errors.Is(err, errSessionEnded) && ctx.Err() == nil {
			log.Printf("gateway: cannot renew the session at the instance, will try again: %v", err)
			wait = min(wait, retryRenewAfter)
		}
		timer.Reset(wait)
	}
}

// serveHome is the page of the gateway: the intro for a browser that is not paired with it, and for one that is,
// the instance it is paired with and the form to pair it with another.
func (g *Gateway) serveHome(writer http.ResponseWriter, request *http.Request) {
	if !g.browsers.paired(request) {
		render(writer, http.StatusOK, view{Page: pageIntro, Fingerprint: g.fingerprint})
		return
	}
	g.renderHome(writer, http.StatusOK, "")
}

// renderHome shows the page of a paired browser, with what went wrong with the last thing it asked.
func (g *Gateway) renderHome(writer http.ResponseWriter, status int, problem string) {
	page := view{Page: pageHome, Error: problem}
	if up := g.current(); up != nil {
		page.Address, page.Connected, page.Trusted = up.base.String(), up.connected(), up.fingerprint()
	}
	render(writer, status, page)
}

// serveBrowserPairing pairs a browser holding a one-time link, and takes it to the interface of the instance, or
// to the page of the gateway when it is paired with none.
func (g *Gateway) serveBrowserPairing(writer http.ResponseWriter, request *http.Request) {
	token := strings.TrimPrefix(request.URL.Path, pairingPath)
	if token == "" || strings.Contains(token, "/") || !g.browsers.takeToken(token) {
		http.Error(writer, "This pairing link is invalid, already used or expired. Restart the gateway for a new one, or make one on the page of a browser that is paired.", http.StatusForbidden)
		return
	}
	// Secure unconditionally, as the servers of the application have it: the gateway serves HTTPS, and
	// the browsers send a Secure cookie back over plain HTTP to a loopback address.
	http.SetCookie(writer, &http.Cookie{
		Name:     cookieName,
		Value:    g.browsers.newSession(),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	if g.current().connected() {
		http.Redirect(writer, request, "/app", http.StatusSeeOther)
		return
	}
	http.Redirect(writer, request, "/", http.StatusSeeOther)
}

// serveToken mints a one-time link for a second browser, as the servers of the application do for a paired
// device, in the address that the caller used.
func (g *Gateway) serveToken(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	if !g.browsers.paired(request) {
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(map[string]string{"url": g.inviteLink(request)})
}

// inviteLink makes the pairing link of a new browser, with the host that the caller used.
func (g *Gateway) inviteLink(request *http.Request) string {
	scheme := "https"
	if request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + request.Host + pairingPath + g.browsers.newToken()
}

// authorize checks a form that changes what the gateway is paired with: a browser that is paired, a request that
// comes from the page of the gateway and not from another site, and a form of reasonable size.
func (g *Gateway) authorize(writer http.ResponseWriter, request *http.Request) bool {
	if !g.browsers.paired(request) {
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
		return false
	}
	if err := g.crossOrigin.Check(request); err != nil {
		http.Error(writer, "Cross-site requests are not allowed.", http.StatusForbidden)
		return false
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxForm)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "Invalid request.", http.StatusBadRequest)
		return false
	}
	return true
}

// serveInvite shows the link of another browser.
func (g *Gateway) serveInvite(writer http.ResponseWriter, request *http.Request) {
	if !g.authorize(writer, request) {
		return
	}
	render(writer, http.StatusOK, view{Page: pageInvite, Link: g.inviteLink(request)})
}

// serveForget makes the gateway forget its instance and the session it holds there.
func (g *Gateway) serveForget(writer http.ResponseWriter, request *http.Request) {
	if !g.authorize(writer, request) {
		return
	}
	g.replace(nil)
	http.Redirect(writer, request, "/", http.StatusSeeOther)
}

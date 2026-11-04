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
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

// instanceCookie is the session cookie of the servers of the application.
const instanceCookie = "vfox-plus-session"

// What the gateway says to the person who pairs it, when it cannot.
var (
	// errNotConnected is a gateway that is paired with no instance, or whose session has ended.
	errNotConnected = errors.New("the gateway is not paired with a vfox+")
	// errSessionEnded is the answer of an instance that does not know the session of the gateway any more:
	// it was ended by the instance (they last a day, and end with the instance).
	errSessionEnded = errors.New("the session at vfox+ has ended")
	// errRefused is the answer of an instance to a pairing link that does not work.
	errRefused = errors.New("vfox+ refused the pairing link: it is not valid, was used already or has expired (they work once, for five minutes); " +
		"make a new one in the settings of vfox+")
)

// tokenPattern is what the token of a pairing link looks like: the application makes 32 hexadecimal
// digits, and the pattern is wider than that so that it is not what a change of that decides.
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

// pairingPath is where the servers of the application take a pairing link.
const pairingPath = "/auth/connect/"

// requestTimeout is how long the gateway waits for the instance when it pairs with it and when it renews
// its session. The routes it forwards have no limit of their own: a search of vfox can take minutes.
const requestTimeout = 20 * time.Second

// upstream is the instance that the gateway is paired with, and the session that it holds there. It is
// never changed: a gateway that is paired again, or whose session was renewed, holds another value, so
// that a request that is on its way finishes with the instance and the session it began with.
type upstream struct {
	// base is the address of the instance, without a path.
	base *url.URL
	// pin is the fingerprint of the certificate of the instance that was trusted. Empty: the certificate has
	// to verify with the roots of the system, as it does for an authority, or there is none (HTTP).
	pin string
	// session is the value of the session cookie at the instance. Empty: the instance ended it.
	session   string
	renewedAt time.Time

	transport *http.Transport
	// client does what is not forwarded (pairing and renewing): it follows no redirect, for the redirect
	// of the instance is its answer to a link, and it has a limit of time.
	client *http.Client
}

// newUpstream is the instance at base, whose certificate is trusted by pin or by the roots (the system's, when
// they are nil), with no session yet.
func newUpstream(base *url.URL, pin string, roots *x509.CertPool) *upstream {
	transport := &http.Transport{
		// The instance is on the network of the gateway: the proxy of the environment is not for it.
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	if base.Scheme == "https" {
		transport.TLSClientConfig = tlsConfig(base.Hostname(), pin, roots)
	}
	return &upstream{
		base:      base,
		pin:       pin,
		transport: transport,
		client: &http.Client{
			Transport:     transport,
			Timeout:       requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// withSession is the same instance with another session: the connections to it are kept.
func (u *upstream) withSession(session string, at time.Time) *upstream {
	next := *u
	next.session, next.renewedAt = session, at
	return &next
}

// connected says whether the gateway holds a session at the instance.
func (u *upstream) connected() bool { return u != nil && u.session != "" }

// tlsConfig trusts the certificate that has the fingerprint, whoever signed it, and none other; or, with no
// fingerprint, the certificates that verify with the roots.
func tlsConfig(host, pin string, roots *x509.CertPool) *tls.Config {
	if pin == "" {
		return &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, RootCAs: roots}
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// The chain is not verified because the certificate of an instance is its own, made by the application
		// and signed by itself; the certificate is the one whose fingerprint the person compared, which is
		// checked below, and which is more than a chain would say.
		InsecureSkipVerify: true, //nolint:gosec // Verified against the pinned fingerprint just below.
		// VerifyConnection and not VerifyPeerCertificate: it runs for every connection, a resumed session
		// included, and the fingerprint is checked on all of them.
		VerifyConnection: func(connection tls.ConnectionState) error {
			if len(connection.PeerCertificates) == 0 {
				return errors.New("vfox+ presented no certificate")
			}
			if got := pair.Fingerprint(connection.PeerCertificates[0].Raw); got != pin {
				return fmt.Errorf("the certificate of vfox+ is not the one that was trusted: its fingerprint is %s and %s was trusted", got, pin)
			}
			return nil
		},
	}
}

// parseLink takes a pairing link of an instance apart: the address of the instance, and the token. The
// link is what a person pasted, so it is checked for what it may be used for: a pairing link of the
// application, over HTTPS or, on a loopback address (where the application serves it), HTTP.
func parseLink(link string) (base *url.URL, token string, err error) {
	link = strings.TrimSpace(link)
	parsed, err := url.Parse(link)
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" {
		return nil, "", errors.New("that is not a pairing link of vfox+: it looks like https://192.168.1.20:8369/auth/connect/…")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", errors.New("a pairing link of vfox+ begins with https:// (or http:// for the computer's own address)")
	}
	token, found := strings.CutPrefix(parsed.Path, pairingPath)
	if !found || !tokenPattern.MatchString(token) || parsed.User != nil {
		return nil, "", errors.New("that is not a pairing link of vfox+: it looks like https://192.168.1.20:8369/auth/connect/…")
	}
	if parsed.Scheme == "http" && !isLoopback(parsed.Hostname()) {
		return nil, "", errors.New("vfox+ serves plain HTTP only on the loopback address of its own computer; " +
			"for another computer, use the pairing link for other devices, which is https")
	}
	return &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}, token, nil
}

// isLoopback says whether host is the loopback address of the computer that asks.
func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// certificateOf looks at the certificate that an HTTPS instance presents, and says what its fingerprint is
// and whether the roots vouch for it (then it needs no one to compare anything).
func certificateOf(ctx context.Context, base *url.URL, roots *x509.CertPool) (fingerprint string, verified bool, err error) {
	host := base.Host
	if base.Port() == "" {
		host = net.JoinHostPort(base.Hostname(), "443")
	}
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		// The certificate is looked at, not trusted: it is verified below, to see whether it could be.
		Config: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true, ServerName: base.Hostname()}, //nolint:gosec // Looked at, not trusted.
	}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = conn.Close() }()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", false, errors.New("no TLS connection")
	}
	certificates := tlsConn.ConnectionState().PeerCertificates
	if len(certificates) == 0 {
		return "", false, errors.New("vfox+ presented no certificate")
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	_, verifyErr := certificates[0].Verify(x509.VerifyOptions{DNSName: base.Hostname(), Intermediates: intermediates, Roots: roots})
	return pair.Fingerprint(certificates[0].Raw), verifyErr == nil, nil
}

// unreachable is the error of an instance that cannot be reached, with what a person can do about the one
// way that goes wrong in a container.
func (u *upstream) unreachable(err error) error {
	message := fmt.Sprintf("cannot reach vfox+ at %s: %v", u.base.Host, err)
	if isLoopback(u.base.Hostname()) {
		message += " (inside a container, 127.0.0.1 is the container itself: run it with --network host, or use the pairing link for other devices)"
	}
	return errors.New(message)
}

// pairWith pairs the gateway with the instance by a token of a pairing link, and returns the session that
// the instance gives it.
func (u *upstream) pairWith(ctx context.Context, token string) (string, error) {
	// The address is the one that the person who is paired with the gateway gave it, to pair with it, and what
	// it can be is narrow: HTTP or HTTPS, the path of a pairing link with a token of its pattern, no redirect
	// followed, and nothing of the answer shown but the cookie it sets.
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.base.String()+pairingPath+token, nil) //nolint:gosec // See above.
	if err != nil {
		return "", err
	}
	response, err := u.client.Do(request) //nolint:gosec // See above.
	if err != nil {
		return "", u.unreachable(err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	switch {
	case response.StatusCode == http.StatusForbidden:
		return "", errRefused
	case response.StatusCode != http.StatusSeeOther:
		return "", fmt.Errorf("vfox+ answered %s to the pairing link, which is not what an instance of vfox+ does", response.Status)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == instanceCookie && cookie.Value != "" {
			return cookie.Value, nil
		}
	}
	return "", errors.New("vfox+ accepted the pairing link and gave the gateway no session")
}

// mint asks the instance, with the session, for a one-time link for a second device, and returns its token.
// A paired device is let do that (GET /api/token), and it is how the gateway pairs itself again before the
// session ends.
func (u *upstream) mint(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.base.String()+"/api/token", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Cookie", instanceCookie+"="+u.session)
	response, err := u.client.Do(request)
	if err != nil {
		return "", u.unreachable(err)
	}
	defer func() { _ = response.Body.Close() }()
	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return "", errSessionEnded
	case response.StatusCode != http.StatusOK:
		return "", fmt.Errorf("vfox+ answered %s when the gateway asked for a link", response.Status)
	}
	var minted struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<10)).Decode(&minted); err != nil {
		return "", fmt.Errorf("the link that vfox+ made cannot be read: %w", err)
	}
	// Only the token is taken of it: the address in the link is the one that this request asked, and what
	// the gateway pairs with is the address it was given.
	parsed, err := url.Parse(minted.URL)
	if err != nil {
		return "", fmt.Errorf("the link that vfox+ made cannot be read: %w", err)
	}
	token, found := strings.CutPrefix(parsed.Path, pairingPath)
	if !found || !tokenPattern.MatchString(token) {
		return "", errors.New("the link that vfox+ made is not a pairing link")
	}
	return token, nil
}

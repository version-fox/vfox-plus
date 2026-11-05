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
	"strconv"
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

// hostPattern is what the address of an instance looks like: a host name, or an IP address (an IPv6 one in square
// brackets), and a port or none. It is stricter than the host that a URL may have, which the dialer of the
// gateway would read as it can:
//
//   - A name is ASCII, and its last label begins with a letter, so that no number that a resolver reads as an
//     address of a form of its own (127.1, 2130706433, 0x7f.1) passes for a name. An address is four numbers in
//     decimal, or IPv6 without a zone, and checkAddress sees that they are addresses.
//   - There is no user, no escape and none of the characters that a URL allows in a host and no host name has.
//
// It does not say which hosts a person may name: any that the container reaches, as the instance may be anywhere.
var hostPattern = regexp.MustCompile(`^(?:` +
	`(?:[0-9]{1,3}\.){3}[0-9]{1,3}` + // an IPv4 address,
	`|\[[0-9A-Fa-f:.]+\]` + // an IPv6 address,
	`|(?:[A-Za-z0-9_](?:[A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?\.)*[A-Za-z_](?:[A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?` + // a name.
	`)(?::[0-9]{1,5})?$`)

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
	// pinned is the certificate of the instance that a person compared and the gateway trusts, and no other.
	// Nil: the certificate has to verify with the roots (those of the system), as it does for an authority, or
	// there is none (HTTP).
	pinned *x509.Certificate
	// session is the value of the session cookie at the instance. Empty: the instance ended it.
	session   string
	renewedAt time.Time

	transport *http.Transport
	// client does what is not forwarded (pairing and renewing): it follows no redirect, for the redirect
	// of the instance is its answer to a link, and it has a limit of time.
	client *http.Client
}

// newUpstream is the instance at base, whose certificate is the one that was pinned or one that the roots (the
// system's, when they are nil) vouch for, with no session yet.
func newUpstream(base *url.URL, pinned *x509.Certificate, roots *x509.CertPool) *upstream {
	transport := &http.Transport{
		// The instance is on the network of the gateway: the proxy of the environment is not for it.
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	if base.Scheme == "https" {
		transport.TLSClientConfig = tlsConfig(base.Hostname(), pinned, roots)
	}
	return &upstream{
		base:      base,
		pinned:    pinned,
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

// fingerprint is the SHA-256 fingerprint of the certificate that was pinned, empty when none was.
func (u *upstream) fingerprint() string {
	if u == nil || u.pinned == nil {
		return ""
	}
	return pair.Fingerprint(u.pinned.Raw)
}

// tlsConfig verifies the certificate of the instance as any is verified (its chain, its dates, what it may be used
// for, its name) against the roots. When a person compared a certificate that nobody signed, which the one that
// the application makes for itself is, that certificate is the only root: it is the one that was compared, and it
// verifies, and no other does. The name that it is verified for is one that it has, since the application names
// the addresses of its computer in it and the gateway may reach the computer by another (a container does).
func tlsConfig(host string, pinned *x509.Certificate, roots *x509.CertPool) *tls.Config {
	if pinned == nil {
		return &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, RootCAs: roots}
	}
	only := x509.NewCertPool()
	only.AddCert(pinned)
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: only, ServerName: nameOf(pinned)}
}

// nameOf is a name that the certificate is made for, a DNS name or an address, "" when it has none.
func nameOf(certificate *x509.Certificate) string {
	if len(certificate.DNSNames) > 0 {
		return certificate.DNSNames[0]
	}
	if len(certificate.IPAddresses) > 0 {
		return certificate.IPAddresses[0].String()
	}
	return ""
}

// parseLink takes a pairing link of an instance apart: the address of the instance, and the token. The
// link is what a person pasted, so it is checked for what it may be used for: a pairing link of the
// application, over HTTPS or, on a loopback address (where the application serves it), HTTP, whose
// address has the form of one (hostPattern, checkAddress) and whose token has the form of one.
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
	if !hostPattern.MatchString(parsed.Host) {
		return nil, "", fmt.Errorf("that is not a pairing link of vfox+: %q is not a host name or an IP address, with a port or without one", parsed.Host)
	}
	if err := checkAddress(parsed.Hostname(), parsed.Port()); err != nil {
		return nil, "", err
	}
	if parsed.Scheme == "http" && !isLoopback(parsed.Hostname()) {
		return nil, "", errors.New("vfox+ serves plain HTTP only on the loopback address of its own computer; " +
			"for another computer, use the pairing link for other devices, which is https")
	}
	return &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}, token, nil
}

// checkAddress sees what hostPattern cannot: that what has the form of an IP address is one (999.1.1.1 is not, and
// neither is 010.0.0.1, which some resolvers read as octal), and that a port is one of the ports of a computer.
func checkAddress(hostname, port string) error {
	if (strings.Trim(hostname, "0123456789.") == "" || strings.Contains(hostname, ":")) && net.ParseIP(hostname) == nil {
		return fmt.Errorf("that is not a pairing link of vfox+: %q is not an IP address", hostname)
	}
	if port != "" {
		if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("that is not a pairing link of vfox+: %q is not a port, which is a number from 1 to 65535", port)
		}
	}
	return nil
}

// isLoopback says whether host is the loopback address of the computer that asks.
func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// certificateOf looks at the certificate that an HTTPS instance presents, and says whether the roots vouch for it
// (then it needs no one to compare anything). The connection is verified as any is, and a certificate that nobody
// vouches for fails it: the error has the certificate that was presented, which is what a person is asked to
// compare, and no connection that verifies nothing is made.
func certificateOf(ctx context.Context, base *url.URL, roots *x509.CertPool) (certificate *x509.Certificate, verified bool, err error) {
	host := base.Host
	if base.Port() == "" {
		host = net.JoinHostPort(base.Hostname(), "443")
	}
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		Config:    &tls.Config{MinVersion: tls.VersionTLS12, ServerName: base.Hostname(), RootCAs: roots},
	}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		var failed *tls.CertificateVerificationError
		if errors.As(err, &failed) && len(failed.UnverifiedCertificates) > 0 {
			return failed.UnverifiedCertificates[0], false, nil
		}
		return nil, false, err
	}
	defer func() { _ = conn.Close() }()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil, false, errors.New("no TLS connection")
	}
	certificates := tlsConn.ConnectionState().PeerCertificates
	if len(certificates) == 0 {
		return nil, false, errors.New("vfox+ presented no certificate")
	}
	return certificates[0], true, nil
}

// unreachable is the error of an instance that cannot be reached, with what a person can do about the one
// way that goes wrong in a container.
func (u *upstream) unreachable(err error) error {
	// A certificate that is not the one that was trusted is said to be so, with both fingerprints: that is what a
	// person has to decide about. The same certificate that fails (it expired, say) says why in the error.
	var failed *tls.CertificateVerificationError
	if u.pinned != nil && errors.As(err, &failed) && len(failed.UnverifiedCertificates) > 0 {
		if got := pair.Fingerprint(failed.UnverifiedCertificates[0].Raw); got != u.fingerprint() {
			return fmt.Errorf("cannot reach vfox+ at %s: the certificate of vfox+ is not the one that was trusted: its fingerprint is %s and %s was trusted",
				u.base.Host, got, u.fingerprint())
		}
	}
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

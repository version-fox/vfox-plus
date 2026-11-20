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

// Package remote is the client of the remote API of a desktop server (see
// mygo/pair): the pairing of the phone and the calls that it makes once paired.
// A client pins the certificate of its server by fingerprint when the user has
// confirmed one; see New.
//
// The package does not depend on the desktop code. Its tests do, through the
// server of mygo/pair.
package remote

import (
	"bytes"
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
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

const (
	// sessionCookie is the name of the cookie that carries the paired session.
	sessionCookie = "vfox-plus-session"
	// requestTimeout bounds one call. The server waits up to two minutes for
	// vfox before it answers 504, so the bound is longer than that.
	requestTimeout = 150 * time.Second
	// dialTimeout bounds connecting and the TLS handshake.
	dialTimeout = 10 * time.Second
	// maxAnswerBytes bounds the answer to one call. The full log of a task is the
	// largest answer.
	maxAnswerBytes = 64 << 20
)

// Client calls the remote API of one server. It is safe for concurrent use.
type Client struct {
	// base is the address of the server: scheme and host, with no path.
	base *url.URL
	// session is the value of the paired session cookie, or empty before pairing.
	session string
	// client makes the calls. It has a timeout, and it does not follow redirects.
	client *http.Client
	// streamer opens the event stream, which lasts as long as its context. It has
	// no timeout.
	streamer *http.Client
}

// New returns a client for the server at base: https://host:port. For tests, a
// plain http address of the loopback interface (127.0.0.1, ::1 or localhost) is
// accepted too.
//
// fingerprint, when it is not empty, is the SHA-256 fingerprint of the server's
// certificate that the user confirmed, in the form pairing.FormatFingerprint
// gives. The client then accepts only a server whose leaf certificate has that
// fingerprint, and it does not consult the system roots: the confirmation is the
// trust. When fingerprint is empty, the certificate must chain to the system
// roots. A fingerprint needs https.
//
// session is the value of the session cookie of an earlier pairing, or empty.
func New(base *url.URL, fingerprint, session string) (*Client, error) {
	address, err := normalizeBase(base)
	if err != nil {
		return nil, err
	}
	pinned := ""
	if fingerprint != "" {
		if address.Scheme != "https" {
			return nil, errors.New("remote: a pinned fingerprint needs an https server")
		}
		if pinned, err = pairing.NormalizeFingerprint(fingerprint); err != nil {
			return nil, err
		}
	}
	transport := newTransport(pinned)
	return &Client{
		base:     address,
		session:  session,
		client:   &http.Client{Transport: transport, Timeout: requestTimeout, CheckRedirect: noRedirects},
		streamer: &http.Client{Transport: transport, CheckRedirect: noRedirects},
	}, nil
}

// normalizeBase checks an address of a server and returns the parts that a
// client uses: its scheme and host.
func normalizeBase(base *url.URL) (*url.URL, error) {
	if base == nil {
		return nil, errors.New("remote: there is no server address")
	}
	if base.Hostname() == "" {
		return nil, errors.New("remote: the server address names no host")
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return nil, errors.New("remote: a server address is a scheme, a host and a port")
	}
	switch base.Scheme {
	case "https":
	case "http":
		if !isLoopback(base.Hostname()) {
			return nil, errors.New("remote: plain http is only accepted for the loopback interface")
		}
	default:
		return nil, errors.New("remote: a server address starts with https://")
	}
	return &url.URL{Scheme: base.Scheme, Host: base.Host}, nil
}

// isLoopback reports whether host names the loopback interface.
func isLoopback(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// newTransport returns the transport of a client. When pinned is empty, the
// certificates are checked against the system roots. Otherwise only the leaf
// certificate with that fingerprint is accepted. VerifyPeerCertificate and
// VerifyConnection run even when the standard verification is skipped, which is
// why the skip is safe here.
//
// The transport uses no proxy: the desktop is on the local network, and a proxy
// of the system would replace its certificate.
func newTransport(pinned string) *http.Transport {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if pinned != "" {
		tlsConfig.InsecureSkipVerify = true //nolint:gosec // The leaf certificate is checked against the pinned fingerprint below.
		tlsConfig.VerifyPeerCertificate = func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 || pairing.FormatFingerprint(raw[0]) != pinned {
				return ErrCertificateChanged
			}
			return nil
		}
		// crypto/tls does not call VerifyPeerCertificate when it resumes a session, so
		// a resumed connection would skip the check above. VerifyConnection runs on every
		// connection, resumed or not, and makes the same check on the leaf certificate
		// of that connection. Without a ClientSessionCache, crypto/tls never resumes, as
		// here; SessionTicketsDisabled keeps it that way if a cache is added later.
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 || pairing.FormatFingerprint(state.PeerCertificates[0].Raw) != pinned {
				return ErrCertificateChanged
			}
			return nil
		}
		tlsConfig.SessionTicketsDisabled = true
	}
	dialer := &net.Dialer{Timeout: dialTimeout}
	return &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: dialTimeout,
		IdleConnTimeout:     90 * time.Second,
	}
}

// noRedirects returns the first answer as it is. The server does not redirect
// the calls; a redirect of a pairing link is an answer to read, not to follow.
func noRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// endpoint returns the address of path with query. path is not escaped: its
// segments are names that validSegment has checked.
func (c *Client) endpoint(path string, query url.Values) string {
	address := *c.base
	address.Path = path
	address.RawQuery = query.Encode()
	return address.String()
}

// authorize adds the session cookie of a paired client to request. The client only
// sends the cookie back, so the Cookie header is written as the server reads it,
// name=value. The session is a value that pair.go read from the server's Set-Cookie,
// which holds only bytes that a cookie value may have, so nothing needs quoting. The
// request is new and has no other cookie, so Set writes the whole header.
func (c *Client) authorize(request *http.Request) {
	if c.session != "" {
		request.Header.Set("Cookie", sessionCookie+"="+c.session)
	}
}

// call makes one request and returns the answer of a 2xx status. A payload that
// is not nil is sent as JSON. Any other status is an *APIError.
func (c *Client) call(ctx context.Context, method, path string, query url.Values, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("remote: encode the request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.endpoint(path, query), body)
	if err != nil {
		return nil, fmt.Errorf("remote: make the request: %w", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	c.authorize(request)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, requestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	answer, err := readAnswer(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, newAPIError(response.StatusCode, answer)
	}
	return answer, nil
}

// readAnswer reads the body of an answer, up to maxAnswerBytes.
func readAnswer(body io.Reader) ([]byte, error) {
	answer, err := io.ReadAll(io.LimitReader(body, maxAnswerBytes+1))
	if err != nil {
		return nil, fmt.Errorf("remote: the answer was interrupted: %w", err)
	}
	if len(answer) > maxAnswerBytes {
		return nil, errors.New("remote: the answer is too large")
	}
	return answer, nil
}

// exchange makes a call and decodes its answer into out, when out is not nil.
func (c *Client) exchange(ctx context.Context, method, path string, query url.Values, payload, out any) error {
	answer, err := c.call(ctx, method, path, query, payload)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(answer, out); err != nil {
		return fmt.Errorf("remote: the answer is not what this app expects: %w", err)
	}
	return nil
}

// result makes a call and returns its answer as T.
func result[T any](ctx context.Context, c *Client, method, path string, query url.Values, payload any) (T, error) {
	var out T
	if err := c.exchange(ctx, method, path, query, payload, &out); err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

// validSegment reports whether name can be one segment of a path: it is not
// empty, has no surrounding white space and no character that would change the
// path.
func validSegment(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "/\\?#%")
}

// sdkPath returns the path of the SDK name with suffix, such as "/info".
func sdkPath(name, suffix string) (string, error) {
	if !validSegment(name) {
		return "", fmt.Errorf("remote: %q is not the name of a plugin", name)
	}
	return "/api/sdks/" + name + suffix, nil
}

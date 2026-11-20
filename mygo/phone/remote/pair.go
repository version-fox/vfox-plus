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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

// ProbeResult is what Probe learned about the certificate of a server.
//
// The type is not called Probe, which is the name of the function: a Go package
// cannot have both.
type ProbeResult struct {
	// Fingerprint is the SHA-256 fingerprint of the leaf certificate, in the form
	// that the desktop application shows it (pairing.FormatFingerprint).
	Fingerprint string
	// Certificate is the DER encoding of the leaf certificate: what New pins once
	// the user has compared the fingerprint.
	Certificate []byte
	// SystemTrusted reports whether the system roots verify the certificate for
	// the host of the address. The self-signed certificate of a desktop server is
	// not, so the user confirms its fingerprint instead; a certificate that the
	// roots vouch for needs no pinning, and a renewal of it needs no pairing.
	SystemTrusted bool
}

// Probe connects once to the server at base over TLS and reports the certificate
// that it presents. The connection is verified as any is, with the system roots,
// and a certificate that they do not vouch for fails it: the error of that failure
// has the certificate that was presented, which is what the user is asked to
// compare, and no connection that verifies nothing is made. Nothing is trusted yet:
// the user compares the fingerprint with the one that the desktop application shows,
// and only then is the certificate pinned with New and Pair. base must be https.
func Probe(ctx context.Context, base *url.URL) (ProbeResult, error) {
	return probe(ctx, base, nil)
}

// probe is Probe with the roots that verify the connection: the system's when roots
// is nil, which is what Probe uses. The tests give their own.
func probe(ctx context.Context, base *url.URL, roots *x509.CertPool) (ProbeResult, error) {
	address, err := normalizeBase(base)
	if err != nil {
		return ProbeResult{}, err
	}
	if address.Scheme != "https" {
		return ProbeResult{}, errors.New("remote: only an https server has a certificate to probe")
	}
	host := address.Hostname()
	port := address.Port()
	if port == "" {
		port = "443"
	}
	// The dial and the handshake are bounded; the connection that is made is not
	// affected once the context ends.
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: dialTimeout},
		Config:    &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, RootCAs: roots},
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		var failed *tls.CertificateVerificationError
		if errors.As(err, &failed) && len(failed.UnverifiedCertificates) > 0 {
			return probeResult(failed.UnverifiedCertificates[0], false), nil
		}
		return ProbeResult{}, requestError(err)
	}
	defer func() { _ = conn.Close() }()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return ProbeResult{}, errors.New("remote: the connection is not TLS")
	}
	chain := tlsConn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return ProbeResult{}, errors.New("remote: the server sent no certificate")
	}
	return probeResult(chain[0], true), nil
}

// probeResult describes the leaf certificate that a server presented.
func probeResult(leaf *x509.Certificate, trusted bool) ProbeResult {
	return ProbeResult{
		Fingerprint:   pairing.FormatFingerprint(leaf.Raw),
		Certificate:   leaf.Raw,
		SystemTrusted: trusted,
	}
}

// Pair exchanges the one-time token of a pairing link for a session, over the
// connection that the client pins: the session belongs to the server whose
// certificate the user confirmed. The client should have been made with New and
// that certificate (none when the system roots vouch for the server), and without a
// session.
//
// The server answers 204 with the session cookie. A refused token (invalid,
// already used or expired) is an error that matches ErrPairingRefused. Pair does
// not follow redirects, so an answer other than 204 or a refusal is an error too.
func (c *Client) Pair(ctx context.Context, token string) (string, error) {
	if !pairing.ValidToken(token) {
		return "", fmt.Errorf("%w: the token has the wrong shape", ErrPairingRefused)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/auth/connect/"+token, nil), nil)
	if err != nil {
		return "", fmt.Errorf("remote: make the request: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return "", c.requestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	answer, err := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	if err != nil {
		return "", fmt.Errorf("remote: the answer was interrupted: %w", err)
	}
	switch response.StatusCode {
	case http.StatusNoContent:
		for _, cookie := range response.Cookies() {
			if cookie.Name == sessionCookie && cookie.Value != "" {
				return cookie.Value, nil
			}
		}
		return "", errors.New("remote: the server paired the phone but sent no session")
	case http.StatusForbidden:
		return "", fmt.Errorf("%w: %w", ErrPairingRefused, newAPIError(response.StatusCode, answer))
	default:
		return "", fmt.Errorf("remote: unexpected answer to the pairing link: %w", newAPIError(response.StatusCode, answer))
	}
}

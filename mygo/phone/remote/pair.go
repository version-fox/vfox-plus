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
	// SystemTrusted reports whether the system roots verify the certificate for
	// the host of the address. The self-signed certificate of a desktop server is
	// not, so the user confirms its fingerprint instead.
	SystemTrusted bool
}

// Probe connects once to the server at base over TLS, without checking the
// certificate, and reports its fingerprint. Nothing is trusted yet: the user
// compares the fingerprint with the one that the desktop application shows, and
// only then is it pinned with New and Pair. base must be https.
func Probe(ctx context.Context, base *url.URL) (ProbeResult, error) {
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
		Config: &tls.Config{
			// The certificate is judged by the user, by its fingerprint, and not by
			// the chain: that is what Probe reports.
			InsecureSkipVerify: true, //nolint:gosec // See the comment above.
			MinVersion:         tls.VersionTLS12,
			ServerName:         host,
		},
	}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
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
	return ProbeResult{
		Fingerprint:   pairing.FormatFingerprint(chain[0].Raw),
		SystemTrusted: systemTrusted(chain, host),
	}, nil
}

// systemTrusted reports whether the system roots verify the chain of a server
// for host.
func systemTrusted(chain []*x509.Certificate, host string) bool {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return false
	}
	intermediates := x509.NewCertPool()
	for _, cert := range chain[1:] {
		intermediates.AddCert(cert)
	}
	_, err = chain[0].Verify(x509.VerifyOptions{DNSName: host, Roots: roots, Intermediates: intermediates})
	return err == nil
}

// Pair exchanges the one-time token of a pairing link for a session, over the
// connection that the client pins: the session belongs to the server whose
// fingerprint the user confirmed. The client should have been made with New and
// the fingerprint, and without a session.
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
		return "", requestError(err)
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

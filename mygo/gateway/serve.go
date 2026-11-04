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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

// The certificate that the gateway serves with is the one that the person puts in its directory, a pair of
// PEM files with these names (for a certificate of an authority, which browsers trust), or else one that the
// gateway makes for itself, which browsers ask to be trusted and whose fingerprint it prints.
const (
	ownCertFile = "cert.pem"
	ownKeyFile  = "key.pem"
)

// certificate returns the certificate and key to serve with, and the fingerprint of the certificate.
func certificate(dir string) (certFile, keyFile, fingerprint string, err error) {
	own, ownKey := filepath.Join(dir, ownCertFile), filepath.Join(dir, ownKeyFile)
	_, certErr := os.Stat(own)
	_, keyErr := os.Stat(ownKey)
	switch {
	case certErr == nil && keyErr == nil:
		if _, err := tls.LoadX509KeyPair(own, ownKey); err != nil {
			return "", "", "", fmt.Errorf("the certificate in %s cannot be used: %w", dir, err)
		}
		fingerprint, err := pair.CertFingerprint(own)
		return own, ownKey, fingerprint, err
	case certErr == nil || keyErr == nil:
		return "", "", "", fmt.Errorf("%s and %s go together: %s has only one of them", ownCertFile, ownKeyFile, dir)
	}
	return pair.EnsureServerCert(dir, pair.LocalAddresses())
}

// shutdownGrace is how long the gateway lets the requests that are on their way finish when it ends. The
// events of the web interface never do: a browser with the interface open holds one, and `docker stop`
// would wait the whole time for it. What the gateway does is written before it answers, so that an answer
// that is cut off loses nothing.
const shutdownGrace = time.Second

// Serve serves the gateway on listener, over HTTPS, until ctx is done. It keeps the session at the instance
// renewed, and says on out how to pair a browser with it: the one-time link, and the fingerprint of the
// certificate to compare with the one that the browser shows.
func (g *Gateway) Serve(ctx context.Context, listener net.Listener, out io.Writer) error {
	certFile, keyFile, fingerprint, err := certificate(g.options.Dir)
	if err != nil {
		return err
	}
	g.fingerprint = fingerprint
	g.announce(out, listener, fingerprint)

	go g.keepPaired(ctx)

	server := &http.Server{
		Handler:           g.Handler(),
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second,
	}
	ended := make(chan error, 1)
	go func() { ended <- server.ServeTLS(listener, certFile, keyFile) }()
	select {
	case err := <-ended:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
	}
	if err := <-ended; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// announce says what a person needs to use the gateway: where it is, which certificate it has, whether it
// is paired, and the link that pairs a browser.
func (g *Gateway) announce(out io.Writer, listener net.Listener, fingerprint string) {
	port := "8080"
	if _, p, err := net.SplitHostPort(listener.Addr().String()); err == nil {
		port = p
	}
	status := g.Status()
	_, _ = fmt.Fprintf(out, "vfox+ gateway\n")
	_, _ = fmt.Fprintf(out, "Serving over HTTPS on port %s. Certificate fingerprint, to compare with what the browser shows:\n  %s\n", port, fingerprint)
	switch {
	case status.Connected:
		_, _ = fmt.Fprintf(out, "Paired with vfox+ at %s.\n", status.Address)
	case status.Address != "":
		_, _ = fmt.Fprintf(out, "The session at vfox+ at %s has ended: pair the gateway again on its page.\n", status.Address)
	default:
		_, _ = fmt.Fprintf(out, "Not paired with a vfox+ yet: pair it on its page, with a pairing link from the settings of vfox+.\n")
	}
	_, _ = fmt.Fprintf(out, "Pair a browser with this link (it works once and expires in %d minutes), with localhost replaced by the address that reaches the gateway:\n  https://localhost:%s%s%s\n",
		int(DefaultTokenTTL.Minutes()), port, pairingPath, g.NewToken())
}

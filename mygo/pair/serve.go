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

package pair

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
)

// ServeMain runs `vfox-plus serve`: the pairing server as a persistent
// service for containers and remote instances. It prints one pairing link
// at start; paired browsers mint more with `GET /api/token`. The output
// stays English on purpose: it must survive any terminal encoding on bash
// and on PowerShell.
func ServeMain(args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	bind := flags.String("bind", DefaultBind, "interface to listen on")
	port := flags.Int("port", 0, "port to listen on (0 picks a free one)")
	tokenTTL := flags.Duration("token-ttl", DefaultTokenTTL, "how long a pairing link works")
	sessionTTL := flags.Duration("session-ttl", DefaultSessionTTL, "how long a paired browser stays paired (0 lives as long as the server)")
	tlsCert := flags.String("tls-cert", "", "TLS certificate file (enables HTTPS)")
	tlsKey := flags.String("tls-key", "", "TLS key file (enables HTTPS)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus serve [--bind 127.0.0.1] [--port 0] [--token-ttl 5m] [--session-ttl 24h] [--tls-cert cert.pem --tls-key key.pem]")
		return 2
	}
	if (*tlsCert == "") != (*tlsKey == "") {
		fmt.Fprintln(os.Stderr, "serve: --tls-cert and --tls-key go together")
		return 2
	}
	loopback := true
	if ip := net.ParseIP(*bind); ip == nil || !ip.IsLoopback() {
		loopback = false
	}
	certFile, keyFile, fingerprint := *tlsCert, *tlsKey, ""
	selfSigned := false
	switch {
	case certFile != "":
		print, err := CertFingerprint(certFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "serve:", err)
			return 1
		}
		fingerprint = print
	case !loopback:
		// Non-loopback binds always speak TLS: a self-signed pair is
		// generated next to the settings unless the user brings one.
		hosts := LocalAddresses()
		if parsed := net.ParseIP(strings.Trim(*bind, "[]")); parsed != nil && !parsed.IsUnspecified() {
			hosts = append(hosts, *bind)
		}
		generated, generatedKey, print, err := EnsureServerCert(LocalSettingsDir(), hosts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "serve:", err)
			return 1
		}
		certFile, keyFile, fingerprint, selfSigned = generated, generatedKey, print, true
	}
	snapshot := LocalSnapshot()
	_ = snapshot()
	model := NewLocalModel()
	defer model.Dispose()
	server, err := NewServer(Options{
		Bind:             *bind,
		AllowNonLoopback: true,
		Port:             *port,
		TokenTTL:         *tokenTTL,
		SessionTTL:       *sessionTTL,
		Snapshot:         func() Snapshot { return SnapshotOf(model.Cli) },
		Model:            model,
		TLSCertFile:      certFile,
		TLSKeyFile:       keyFile,
		Fingerprint:      fingerprint,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		return 1
	}
	defer func() { _ = server.Close() }()
	token, _, err := server.NewToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		return 1
	}
	url := server.DisplayBaseURL() + "/auth/connect/" + token
	fmt.Println()
	fmt.Println("  Serving vfox-plus on " + server.DisplayBaseURL())
	if !loopback {
		fmt.Println("  Listening on non-loopback " + *bind + "; replace the address below")
		fmt.Println("  with the reachable one.")
		if selfSigned {
			fmt.Println("  Using a self-signed certificate: verify the fingerprint below")
			fmt.Println("  in the browser before trusting it.")
		}
	}
	if fingerprint != "" {
		fmt.Println("  Certificate fingerprint: " + fingerprint)
	}
	fmt.Println("  Open a link to connect. Links work once and expire in " + formatTTL(server.ttl) + ".")
	fmt.Println()
	fmt.Println("  " + url)
	fmt.Println()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
	fmt.Println()
	return 0
}

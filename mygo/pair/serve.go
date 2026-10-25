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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
)

// Main of `vfox-plus serve`: the pairing server as a persistent service,
// for containers and remote instances. It prints one pairing link at start;
// paired browsers mint more with `GET /api/token`. The output stays English
// on purpose: it must survive any terminal encoding on bash and on
// PowerShell.
func ServeMain(args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	bind := flags.String("bind", DefaultBind, "interface to listen on")
	port := flags.Int("port", 0, "port to listen on (0 picks a free one)")
	tokenTTL := flags.Duration("token-ttl", DefaultTokenTTL, "how long a pairing link works")
	sessionTTL := flags.Duration("session-ttl", DefaultSessionTTL, "how long a paired browser stays paired (0 lives as long as the server)")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus serve [--bind 127.0.0.1] [--port 0] [--token-ttl 5m] [--session-ttl 24h]")
		return 2
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
	if ip := net.ParseIP(*bind); ip == nil || !ip.IsLoopback() {
		fmt.Println("  WARNING: listening on non-loopback " + *bind + "; anyone who can")
		fmt.Println("  reach it and holds a link can pair. Prefer an encrypted tunnel")
		fmt.Println("  and replace the address below with the reachable one.")
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

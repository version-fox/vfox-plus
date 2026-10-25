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
	"os"
	"os/signal"
	"time"
)

// Main runs `vfox-plus pair`. It returns the process exit code. The output
// stays English on purpose: it must survive any terminal encoding on bash
// and on PowerShell, and it follows the wording of `opencode pair`.
func Main(args []string) int {
	flags := flag.NewFlagSet("pair", flag.ContinueOnError)
	bind := flags.String("bind", DefaultBind, "interface to listen on (loopback only)")
	port := flags.Int("port", 0, "port to listen on (0 picks a free one)")
	tokenTTL := flags.Duration("token-ttl", DefaultTokenTTL, "how long the pairing link works")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus pair [--bind 127.0.0.1] [--port 0] [--token-ttl 5m]")
		return 2
	}
	snapshot := LocalSnapshot()
	// Probe now, so that a missing vfox fails fast with a clear message
	// instead of an empty pairing page.
	_ = snapshot()
	server, err := NewServer(Options{Bind: *bind, Port: *port, TokenTTL: *tokenTTL, Snapshot: snapshot})
	if err != nil {
		fmt.Fprintln(os.Stderr, "pair:", err)
		return 1
	}
	defer func() { _ = server.Close() }()
	_, url, err := server.NewToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pair:", err)
		return 1
	}
	fmt.Println()
	fmt.Println("  Open a link to connect. Links work once and expire in", formatTTL(server.ttl)+".")
	fmt.Println()
	fmt.Println("  " + url)
	fmt.Println()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
	fmt.Println()
	return 0
}

// formatTTL renders a duration the way the pairing message words it.
func formatTTL(ttl time.Duration) string {
	if ttl >= time.Minute && ttl%time.Minute == 0 {
		minutes := int(ttl.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	return ttl.String()
}

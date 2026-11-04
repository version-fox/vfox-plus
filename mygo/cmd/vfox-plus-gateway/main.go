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

// Command vfox-plus-gateway is the program of the Docker image of the same name: the web interface of
// one vfox+, served from another place (see package gateway). It is a client of vfox+, as the Android
// app is, and not vfox+ in another mode: it has no vfox, no window and no state of vfox.
//
// It takes no arguments and reads no environment switch. It serves HTTPS on port 8080, prints the link
// that pairs a browser with it, and is paired with a vfox+ on its web page. What it remembers is in the
// configuration directory of its user (HOME, in a container: mount a volume there).
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/version-fox/vfox-plus/mygo/gateway"
)

// address is where the gateway listens. Port 8080 is the one the image exposes; map it to another
// on the host.
const address = ":8080"

func main() {
	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "vfox-plus-gateway takes no arguments: it is paired with a vfox+ on its web page.")
		os.Exit(2)
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "vfox-plus-gateway: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("no directory to remember in: %w", err)
	}
	g, err := gateway.New(gateway.Options{Dir: filepath.Join(config, "vfox-plus-gateway")})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address) //nolint:gosec // The gateway is there to be reached from other devices: it serves HTTPS, and pairs the browsers.
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return g.Serve(ctx, listener, os.Stdout)
}

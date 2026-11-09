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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ErrNoCloudflared means that the cloudflared program is nowhere to be found.
var ErrNoCloudflared = errors.New("the tunnel program is missing")

// TunnelURL finds the public address that a cloudflared quick tunnel printed.
var tunnelURL = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// Tunnel is a quick tunnel of Cloudflare in front of the server for other
// devices: a random https address, no account, for as long as it runs.
type Tunnel struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	mu  sync.Mutex
	url string
}

// PublishTunnel starts cloudflared for the port that other devices connect to,
// and waits until it prints the address where it serves it.
func PublishTunnel(ctx context.Context, port int) (*Tunnel, error) {
	binary := CloudflaredPath()
	if binary == "" {
		return nil, ErrNoCloudflared
	}
	runCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(runCtx, binary, "tunnel", "--no-autoupdate", //nolint:gosec // The binary is the tunnel program found above.
		"--url", fmt.Sprintf("https://127.0.0.1:%d", port), "--no-tls-verify")
	// The address arrives in the log, on the error stream of cloudflared.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	tunnel := &Tunnel{cmd: cmd, cancel: cancel}
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if address := tunnelURL.FindString(scanner.Text()); address != "" {
				tunnel.mu.Lock()
				if tunnel.url == "" {
					tunnel.url = address
				}
				tunnel.mu.Unlock()
			}
		}
	}()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		tunnel.mu.Lock()
		address := tunnel.url
		tunnel.mu.Unlock()
		if address != "" {
			return tunnel, nil
		}
		select {
		case <-runCtx.Done():
			return nil, errors.New("the tunnel stopped before it had an address")
		case <-time.After(200 * time.Millisecond):
		}
	}
	tunnel.Stop()
	return nil, errors.New("the tunnel did not answer in time")
}

// URL is the address that the tunnel serves, like https://one-two-three.trycloudflare.com.
func (t *Tunnel) URL() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.url
}

// Stop ends the tunnel, and the public address with it.
func (t *Tunnel) Stop() {
	if t == nil {
		return
	}
	t.cancel()
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	_ = t.cmd.Wait()
}

// isTunnelAddress reports whether address is one that a quick tunnel serves.
func isTunnelAddress(address string) bool {
	return strings.HasSuffix(strings.TrimRight(address, "/"), ".trycloudflare.com")
}

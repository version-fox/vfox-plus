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

import "testing"

// The address of a quick tunnel arrives in a box of the log of cloudflared.
func TestTheTunnelAddressIsFoundInTheLog(t *testing.T) {
	for _, line := range []string{
		"2026-11-09T12:56:12Z INF |  https://charms-bosnia-participation-belongs.trycloudflare.com  |",
		"INF Your quick Tunnel has been created! Visit it at https://one-two-three.trycloudflare.com",
	} {
		if got := tunnelURL.FindString(line); got == "" || !isTunnelAddress(got) {
			t.Errorf("%q: got %q", line, got)
		}
	}
	for _, line := range []string{
		"INF Requesting new quick Tunnel on trycloudflare.com...",
		"ERR failed to connect to an ideally positioned edge: https://example.com",
	} {
		if got := tunnelURL.FindString(line); got != "" {
			t.Errorf("%q has no tunnel address, got %q", line, got)
		}
	}
	if !isTunnelAddress("https://a-b-c.trycloudflare.com/") {
		t.Error("a trailing slash is still a tunnel address")
	}
	if isTunnelAddress("https://example.com") {
		t.Error("another host is no tunnel")
	}
}

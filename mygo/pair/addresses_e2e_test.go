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

//go:build e2e

package pair

import (
	"slices"
	"testing"
)

func TestTheEndToEndBuildCanBeToldTheAddressesOfTheComputer(t *testing.T) {
	real := localAddresses
	t.Cleanup(func() { localAddresses = real })

	useFixedAddresses("")
	useFixedAddresses(" , ")
	if got := LocalAddresses(); !slices.Equal(got, real()) {
		t.Errorf("no address named: LocalAddresses() = %v, want the addresses of the interfaces %v", got, real())
	}

	useFixedAddresses("192.168.1.20, fd00::1 ,,192.168.1.20")
	want := []string{"192.168.1.20", "fd00::1", "192.168.1.20"}
	if got := LocalAddresses(); !slices.Equal(got, want) {
		t.Errorf("LocalAddresses() = %v, want %v", got, want)
	}
	// What a caller does to the list is not what the next one gets.
	LocalAddresses()[0] = "10.0.0.1"
	if got := LocalAddresses(); got[0] != "192.168.1.20" {
		t.Errorf("a caller changed the addresses: %v", got)
	}
	// The addresses that other devices connect to are these, the IPv4 ones first and none twice.
	if got := NetworkAddresses(); !slices.Equal(got, []string{"192.168.1.20", "fd00::1"}) {
		t.Errorf("NetworkAddresses() = %v", got)
	}
}

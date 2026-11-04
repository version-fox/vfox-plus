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
	"os"
	"slices"
	"strings"
)

// The end-to-end build can be told the addresses of the computer, with VFOX_PLUS_E2E_ADDRESSES (separated
// by commas), so that what shows them does not depend on the machine that runs it: the screenshots of
// the documentation show an address that is the same at every run and belongs to no one, and a sandbox
// with no network, which has no address of its own, still has one to pair a phone with.
func init() { useFixedAddresses(os.Getenv("VFOX_PLUS_E2E_ADDRESSES")) }

// useFixedAddresses makes LocalAddresses answer with the addresses in value, a list separated by commas.
// An empty value leaves the addresses of the interfaces.
func useFixedAddresses(value string) {
	var fixed []string
	for _, address := range strings.Split(value, ",") {
		if address = strings.TrimSpace(address); address != "" {
			fixed = append(fixed, address)
		}
	}
	if len(fixed) == 0 {
		return
	}
	localAddresses = func() []string { return slices.Clone(fixed) }
}

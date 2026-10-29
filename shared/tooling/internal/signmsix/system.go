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

package signmsix

import "crypto/x509"

// system is what depends on the operating system: the privileges, the store of
// the certificates that the machine trusts, and the installation of a package.
// Windows has the implementation that does something; elsewhere only the
// signing works. The tests use one of their own.
type system interface {
	// elevated reports whether this process has administrator rights (where
	// there is no such thing, it does).
	elevated() bool
	// relaunchElevated runs `sign-msix` with args in a copy of this program with
	// administrator rights, which may ask the user to allow it, waits for the
	// copy and returns its exit code.
	relaunchElevated(args []string) (int, error)
	// addToStore trusts a certificate (DER) for the local machine, and removes
	// from the store the certificates for which replaces is true: the ones that
	// an earlier run added.
	addToStore(der []byte, replaces func(*x509.Certificate) bool) error
	// install installs a bundle for all users.
	install(bundle string) error
}

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

//go:build windows

package msixinstall

import (
	"crypto/sha1" //nolint:gosec // The thumbprint of a certificate is its SHA-1, as Windows names it.
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The certificates of the machine are read with the same calls that sign-msix uses to
// put its certificate there: a PowerShell query of the store does not show what the
// store holds on the runners, and this one does.

// trustedCertificate is a certificate of the TrustedPeople store, with a copy of the
// context that Windows gives it. The caller frees the copy, or deletes it.
type trustedCertificate struct {
	context     *windows.CertContext
	certificate Certificate
}

// withTrusted calls act with the certificates of the TrustedPeople store of the
// machine that keep reports, while the store is open.
func withTrusted(keep func(Certificate) bool, act func([]trustedCertificate) error) error {
	name, err := windows.UTF16PtrFromString("TrustedPeople")
	if err != nil {
		return err
	}
	store, err := windows.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(name)))
	runtime.KeepAlive(name)
	if err != nil {
		return fmt.Errorf("cannot open the TrustedPeople store of this machine: %w", err)
	}
	defer func() { _ = windows.CertCloseStore(store, 0) }()

	// The enumeration frees the context it is given, so each one that is kept is copied.
	var found []trustedCertificate
	var previous *windows.CertContext
	for {
		current, err := windows.CertEnumCertificatesInStore(store, previous)
		if err != nil || current == nil {
			break
		}
		der := append([]byte(nil), unsafe.Slice(current.EncodedCert, current.Length)...)
		if parsed, err := x509.ParseCertificate(der); err == nil {
			sum := sha1.Sum(der)
			cert := Certificate{Thumbprint: hex.EncodeToString(sum[:]), Subject: parsed.Subject.String()}
			if keep(cert) {
				found = append(found, trustedCertificate{context: windows.CertDuplicateCertificateContext(current), certificate: cert})
			}
		}
		previous = current
	}
	return act(found)
}

// Certificates lists the certificates of the application in the TrustedPeople store.
func (windowsMachine) Certificates() ([]Certificate, error) {
	var certs []Certificate
	err := withTrusted(func(c Certificate) bool { return c.Subject == certSubject }, func(found []trustedCertificate) error {
		for _, f := range found {
			certs = append(certs, f.certificate)
			_ = windows.CertFreeCertificateContext(f.context)
		}
		return nil
	})
	return certs, err
}

// RemoveCertificate removes the certificate with the thumbprint from the TrustedPeople store.
func (windowsMachine) RemoveCertificate(thumbprint string) error {
	return withTrusted(func(c Certificate) bool { return strings.EqualFold(c.Thumbprint, thumbprint) }, func(found []trustedCertificate) error {
		if len(found) == 0 {
			return fmt.Errorf("no certificate with the thumbprint %s in the TrustedPeople store", thumbprint)
		}
		for _, f := range found {
			// The call frees the context, whether it works or not.
			if err := windows.CertDeleteCertificateFromStore(f.context); err != nil {
				return fmt.Errorf("cannot remove the certificate %s: %w", thumbprint, err)
			}
		}
		return nil
	})
}

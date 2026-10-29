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

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"
)

// serialMarker starts the serial number of every certificate that this tool
// makes, so that a later run knows its own certificates in the store of the
// machine and replaces them instead of adding more. The rest of the serial is
// random.
var serialMarker = []byte("vfxtool.")

// signer is what one run signs with: a self-signed code-signing certificate for
// the publisher, and the private key that belongs to it.
type signer struct {
	der      []byte // the certificate, as it goes into a store
	chainPEM []byte // the same, for osslsigncode
	keyPEM   []byte // the private key, PKCS#8, for osslsigncode
}

// newSigner makes a certificate for CN=commonName that is valid for five years
// and for code signing only, with a new 2048-bit RSA key.
func newSigner(commonName string, now time.Time) (*signer, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serial := make([]byte, 16)
	copy(serial, serialMarker)
	if _, err := rand.Read(serial[len(serialMarker):]); err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetBytes(serial),
		Subject:      pkix.Name{CommonName: commonName},
		// A few minutes back, for a clock that runs a little behind.
		NotBefore:             now.Add(-10 * time.Minute),
		NotAfter:              now.AddDate(5, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true, // A leaf: CA:FALSE.
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		return nil, err
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, err
	}
	return &signer{
		der:      der,
		chainPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		keyPEM:   pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}),
	}, nil
}

// replaces reports whether c is a certificate that an earlier run made for the
// same publisher, which the certificate of this run takes the place of: self-
// signed, for code signing, with the same common name and the marker of this
// tool in its serial number. Nothing else qualifies, so a certificate that
// someone else put into the store stays.
func (s *signer) replaces(c *x509.Certificate) bool {
	mine, err := x509.ParseCertificate(s.der)
	if err != nil || bytes.Equal(c.Raw, s.der) {
		return false
	}
	return c.Subject.CommonName == mine.Subject.CommonName &&
		bytes.Equal(c.RawIssuer, c.RawSubject) &&
		bytes.HasPrefix(c.SerialNumber.Bytes(), serialMarker) &&
		slices.Contains(c.ExtKeyUsage, x509.ExtKeyUsageCodeSigning)
}

// signAll signs every package in dir with osslsigncode: the .msix files first,
// then the bundles, which hold them. The key is written to a temporary file
// for osslsigncode to read, and deleted when the signing is done.
func signAll(tool string, s *signer, dir string) error {
	packages, err := packagesIn(dir)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "vfxtool-sign-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	chain := filepath.Join(work, "signer.pem")
	private := filepath.Join(work, "signer.key")
	if err := os.WriteFile(chain, s.chainPEM, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(private, s.keyPEM, 0o600); err != nil {
		return err
	}
	for _, file := range packages {
		name := filepath.Base(file)
		fmt.Printf("Signing %s...\n", name)
		signed := file + ".signed-tmp"
		command := exec.Command(tool, "sign", "-certs", chain, "-key", private, "-in", file, "-out", signed) //nolint:gosec // The tool is the pinned download or the one that the caller named.
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			_ = os.Remove(signed)
			return fmt.Errorf("osslsigncode failed for %s: %w", name, err)
		}
		if err := os.Rename(signed, file); err != nil {
			return err
		}
	}
	return nil
}

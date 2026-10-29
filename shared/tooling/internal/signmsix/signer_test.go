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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestNewSigner(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	s, err := newSigner("vfox-plus", now)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(s.der)
	if err != nil {
		t.Fatal(err)
	}
	// The subject is what the manifest of a package calls its publisher.
	if cert.Subject.String() != "CN=vfox-plus" || !bytes.Equal(cert.RawIssuer, cert.RawSubject) {
		t.Errorf("subject %s, issuer %s", cert.Subject, cert.Issuer)
	}
	if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		t.Errorf("not self-signed: %v", err)
	}
	if !slices.Equal(cert.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}) || cert.KeyUsage != x509.KeyUsageDigitalSignature {
		t.Errorf("usage %v, %v", cert.ExtKeyUsage, cert.KeyUsage)
	}
	if cert.IsCA || !cert.BasicConstraintsValid {
		t.Errorf("a leaf is CA:FALSE: IsCA %t, valid %t", cert.IsCA, cert.BasicConstraintsValid)
	}
	if !cert.NotBefore.Equal(now.Add(-10*time.Minute)) || !cert.NotAfter.Equal(time.Date(2031, 10, 29, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("valid from %s to %s", cert.NotBefore, cert.NotAfter)
	}
	key, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok || key.N.BitLen() != 2048 || cert.SignatureAlgorithm != x509.SHA256WithRSA {
		t.Errorf("key %T, algorithm %s", cert.PublicKey, cert.SignatureAlgorithm)
	}
	if !bytes.HasPrefix(cert.SerialNumber.Bytes(), serialMarker) || len(cert.SerialNumber.Bytes()) != 16 || cert.SerialNumber.Sign() <= 0 {
		t.Errorf("serial %x", cert.SerialNumber.Bytes())
	}
	// What osslsigncode reads: a PEM certificate, and a PKCS#8 PEM key of it.
	if block, _ := pem.Decode(s.chainPEM); block == nil || block.Type != "CERTIFICATE" || !bytes.Equal(block.Bytes, s.der) {
		t.Error("the certificate PEM is not the certificate")
	}
	if block, _ := pem.Decode(s.keyPEM); block == nil || block.Type != "PRIVATE KEY" {
		t.Error("the key is not PKCS#8 PEM")
	}
	if _, err := tls.X509KeyPair(s.chainPEM, s.keyPEM); err != nil {
		t.Errorf("the certificate and the key do not belong together: %v", err)
	}
	other, err := newSigner("vfox-plus", now)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(other.der, s.der) || bytes.Equal(other.keyPEM, s.keyPEM) {
		t.Error("two runs made the same certificate")
	}
}

// issue makes a certificate like those of the tool, changed by mutate, signed
// by its own key or, with parent, by the key of another certificate.
func issue(t *testing.T, mutate func(*x509.Certificate), parent *x509.Certificate, parentKey *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial := make([]byte, 16)
	copy(serial, serialMarker)
	_, _ = rand.Read(serial[len(serialMarker):])
	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetBytes(serial),
		Subject:      pkix.Name{CommonName: "vfox-plus"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	if mutate != nil {
		mutate(template)
	}
	signer, signingKey := template, key
	if parent != nil {
		signer, signingKey = parent, parentKey
	}
	der, err := x509.CreateCertificate(rand.Reader, template, signer, &key.PublicKey, signingKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestReplacesOnlyEarlierCertificatesOfThisTool(t *testing.T) {
	s, err := newSigner("vfox-plus", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	own, _ := x509.ParseCertificate(s.der)

	// Someone else's authority, that signs a certificate which is like ours.
	issuerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Issuer"}, IsCA: true, BasicConstraintsValid: true,
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageCertSign,
	}

	for name, tc := range map[string]struct {
		cert *x509.Certificate
		want bool
	}{
		"an earlier certificate of the tool": {issue(t, nil, nil, nil), true},
		"the certificate itself":             {own, false},
		"another publisher": {issue(t, func(c *x509.Certificate) { c.Subject = pkix.Name{CommonName: "other"} }, nil, nil),
			false},
		"no marker in the serial": {issue(t, func(c *x509.Certificate) { c.SerialNumber = big.NewInt(12345) }, nil, nil), false},
		"another purpose": {issue(t, func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth} }, nil, nil),
			false},
		"signed by someone else": {issue(t, nil, issuer, issuerKey), false},
	} {
		if got := s.replaces(tc.cert); got != tc.want {
			t.Errorf("%s: got %t, want %t", name, got, tc.want)
		}
	}
}

// The real osslsigncode takes the PEM files of the tool and signs a real
// package that vfxtool built, and verifies the signature. Where there is no
// osslsigncode the test is skipped; CI runs the whole of it on Windows.
func TestSignWithOsslsigncode(t *testing.T) {
	tool, err := exec.LookPath("osslsigncode")
	if err != nil {
		t.Skip("osslsigncode is not installed")
	}
	directory := artifact(t, "", "x64", "arm64")
	signed := filepath.Join(directory, "signed")
	o := options{directory: directory, publisherCN: "vfox-plus", pkg: "vfox-plus"}
	if err := rebuild(o, "0.1.0", []string{"x64", "arm64"}, signed); err != nil {
		t.Fatal(err)
	}
	s, err := newSigner("vfox-plus", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := signAll(tool, s, signed); err != nil {
		t.Fatal(err)
	}
	trust := filepath.Join(t.TempDir(), "trust.pem")
	if err := os.WriteFile(trust, s.chainPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	packages, err := packagesIn(signed)
	if err != nil || len(packages) != 3 {
		t.Fatalf("%v, %v", packages, err)
	}
	for _, file := range packages {
		output, err := exec.Command(tool, "verify", "-in", file, "-CAfile", trust).CombinedOutput() //nolint:gosec // The tool of the test.
		if err != nil {
			t.Errorf("%s does not verify: %v\n%s", filepath.Base(file), err, output)
		}
	}
}

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

// Package phonetest holds what the tests of the phone app share: the certificates that
// stand for the servers they talk to. The tests of the app never skip the verification
// of a certificate, so a server that they make presents a certificate that they made,
// and the client under test is given that certificate to pin, or another one to refuse.
package phonetest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// Certificate is a self-signed certificate of a test server, as the desktop's is.
type Certificate struct {
	// DER is the DER encoding of the certificate: what a client pins.
	DER []byte
	// Pair is the certificate with its key, for a server that presents it.
	Pair tls.Certificate
}

// Mint makes a self-signed certificate valid from notBefore to notAfter, for the DNS
// names and the addresses that it is given, which may be none.
func Mint(t testing.TB, notBefore, notAfter time.Time, dnsNames []string, addresses ...net.IP) Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "vfox-plus test"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
		IPAddresses:  addresses,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return Certificate{DER: der, Pair: tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}}
}

// Valid makes a certificate that is valid at the time of the test (an hour either way),
// for the names that it is given and no other.
func Valid(t testing.TB, dnsNames []string, addresses ...net.IP) Certificate {
	t.Helper()
	return Mint(t, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), dnsNames, addresses...)
}

// Another makes a certificate that is not the one of any server of the tests: a
// certificate that a client pinned to it refuses a server for. It names example.com, which
// the client needs of a pinned certificate.
func Another(t testing.TB) []byte {
	t.Helper()
	return Valid(t, []string{"example.com"}).DER
}

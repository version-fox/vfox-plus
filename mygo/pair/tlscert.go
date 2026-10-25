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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// certFileName and keyFileName are the self-signed server certificate of
// `vfox-plus serve`, next to the GUI settings.
const (
	certFileName = "serve-cert.pem"
	keyFileName  = "serve-key.pem"
)

// certValidity is how long a generated certificate lasts. Rotation is
// manual: deleting the files mints a fresh pair on the next start.
const certValidity = 10 * 365 * 24 * time.Hour

// EnsureServerCert returns the certificate and key in dir, generating a
// self-signed pair when absent, expired or mismatched. hosts names the
// subject alternative names besides loopback: DNS names and IP addresses,
// for example the bind address and the machine's interface addresses.
// It also returns the SHA-256 fingerprint of the certificate, in the
// OpenSSL colon-separated form the browser shows for comparison.
func EnsureServerCert(dir string, hosts []string) (certFile, keyFile, fingerprint string, err error) {
	certFile = filepath.Join(dir, certFileName)
	keyFile = filepath.Join(dir, keyFileName)
	if fingerprint, ok := validPair(certFile, keyFile); ok {
		return certFile, keyFile, fingerprint, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", "", err
	}
	certDER, keyDER, err := generateCert(hosts)
	if err != nil {
		return "", "", "", err
	}
	if err := writeFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return "", "", "", err
	}
	if err := writeFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o644); err != nil {
		return "", "", "", err
	}
	return certFile, keyFile, Fingerprint(certDER), nil
}

// validPair reports whether certFile and keyFile hold a matching,
// unexpired pair, with its fingerprint.
func validPair(certFile, keyFile string) (string, bool) {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil || len(pair.Certificate) == 0 {
		return "", false
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return "", false
	}
	now := time.Now()
	if now.Before(cert.NotBefore.Add(-time.Hour)) || now.After(cert.NotAfter) {
		return "", false
	}
	return Fingerprint(pair.Certificate[0]), true
}

// generateCert mints a self-signed ECDSA certificate for hosts.
func generateCert(hosts []string) (certDER, keyDER []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "vfox-plus serve"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(certValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	seen := map[string]bool{}
	addIP := func(ip net.IP) {
		if ip == nil || seen[ip.String()] {
			return
		}
		seen[ip.String()] = true
		template.IPAddresses = append(template.IPAddresses, ip)
	}
	addDNS := func(name string) {
		name = strings.ToLower(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		template.DNSNames = append(template.DNSNames, name)
	}
	addIP(net.ParseIP("127.0.0.1"))
	addIP(net.ParseIP("::1"))
	addDNS("localhost")
	for _, host := range hosts {
		if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
			addIP(ip)
		} else if host != "" {
			addDNS(host)
		}
	}
	certDER, err = x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err = x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return certDER, keyDER, nil
}

func writeFile(path string, content []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm) //nolint:gosec // Permissions are set explicitly here.
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// Fingerprint renders the SHA-256 fingerprint of a DER certificate the way
// browsers show it: upper-case hex pairs separated by colons.
func Fingerprint(certDER []byte) string {
	sum := sha256.Sum256(certDER)
	hexed := strings.ToUpper(hex.EncodeToString(sum[:]))
	pairs := make([]string, 0, len(hexed)/2)
	for i := 0; i < len(hexed); i += 2 {
		pairs = append(pairs, hexed[i:i+2])
	}
	return strings.Join(pairs, ":")
}

// CertFingerprint returns the SHA-256 fingerprint of the first certificate
// in certFile.
func CertFingerprint(certFile string) (string, error) {
	content, err := os.ReadFile(certFile) //nolint:gosec // The user chose the certificate file.
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(content)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("pair: %s holds no certificate", certFile)
	}
	return Fingerprint(block.Bytes), nil
}

// LocalAddresses lists the machine's non-loopback IP addresses, for the
// certificate of a server behind a port mapping.
func LocalAddresses() []string {
	var addresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch typed := addr.(type) {
			case *net.IPNet:
				ip = typed.IP
			case *net.IPAddr:
				ip = typed.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			addresses = append(addresses, ip.String())
		}
	}
	return addresses
}

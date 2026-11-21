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

package pairing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// ErrBadFingerprint reports text that is not a fingerprint: 32 pairs of
// hexadecimal digits separated by colons.
var ErrBadFingerprint = errors.New("the certificate fingerprint is not 32 hexadecimal pairs separated by colons")

// FormatFingerprint returns the SHA-256 fingerprint of a DER-encoded certificate
// in the form that the desktop application shows and compares: upper-case
// hexadecimal pairs separated by colons. It is the same text as pair.Fingerprint
// of the desktop, which is what the user compares on both screens.
func FormatFingerprint(certDER []byte) string {
	sum := sha256.Sum256(certDER)
	return colonPairs(sum[:])
}

// NormalizeFingerprint returns fingerprint in the form of FormatFingerprint.
// Letter case and surrounding white space do not matter. Anything else that is
// not such a fingerprint is ErrBadFingerprint.
func NormalizeFingerprint(fingerprint string) (string, error) {
	text := strings.ToUpper(strings.TrimSpace(fingerprint))
	parts := strings.Split(text, ":")
	if len(parts) != sha256.Size {
		return "", ErrBadFingerprint
	}
	for _, part := range parts {
		if len(part) != 2 {
			return "", ErrBadFingerprint
		}
		if _, err := hex.DecodeString(part); err != nil {
			return "", ErrBadFingerprint
		}
	}
	return strings.Join(parts, ":"), nil
}

// colonPairs renders sum as upper-case hexadecimal pairs separated by colons.
func colonPairs(sum []byte) string {
	hexed := strings.ToUpper(hex.EncodeToString(sum))
	parts := make([]string, 0, len(sum))
	for i := 0; i < len(hexed); i += 2 {
		parts = append(parts, hexed[i:i+2])
	}
	return strings.Join(parts, ":")
}

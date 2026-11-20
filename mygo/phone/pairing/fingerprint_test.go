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

package pairing_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

func TestFormatFingerprintIsTheDesktopFingerprint(t *testing.T) {
	for _, der := range [][]byte{nil, []byte("abc"), []byte("a certificate in DER form")} {
		if got, want := pairing.FormatFingerprint(der), pair.Fingerprint(der); got != want {
			t.Errorf("FormatFingerprint = %q, the desktop gives %q", got, want)
		}
	}
	// SHA-256("abc") is ba7816bf…; the desktop shows it in upper case, in pairs.
	const want = "BA:78:16:BF:8F:01:CF:EA:41:41:40:DE:5D:AE:22:23:B0:03:61:A3:96:17:7A:9C:B4:10:FF:61:F2:00:15:AD"
	if got := pairing.FormatFingerprint([]byte("abc")); got != want {
		t.Errorf("FormatFingerprint(abc) = %q, want %q", got, want)
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	want := pairing.FormatFingerprint([]byte("abc"))
	lower := strings.ToLower(want)
	for _, text := range []string{want, lower, "  " + lower + "\n"} {
		got, err := pairing.NormalizeFingerprint(text)
		if err != nil || got != want {
			t.Errorf("NormalizeFingerprint(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
	bad := []string{
		"",
		"BA",
		strings.ReplaceAll(want, ":", ""),
		want[:len(want)-3],
		want + ":00",
		strings.Replace(want, "BA", "BG", 1),
		strings.Replace(want, "BA:", "BAA:", 1),
	}
	for _, text := range bad {
		if _, err := pairing.NormalizeFingerprint(text); !errors.Is(err, pairing.ErrBadFingerprint) {
			t.Errorf("NormalizeFingerprint(%q) = %v, want ErrBadFingerprint", text, err)
		}
	}
}

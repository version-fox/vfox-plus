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

package screens

import (
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

func TestThePairingAddressesStartWithThePublicOne(t *testing.T) {
	web := remote.WebState{
		PublicAddress: "https://pub.example:8443",
		Addresses:     []string{"https://10.0.0.2:8443", "https://pub.example:8443", "10.0.0.3"},
	}
	if got := settingsInitialHost(web); got != "https://pub.example:8443" {
		t.Errorf("the link names %q first, want the public address", got)
	}
	choices := settingsHostChoices(web)
	var values, labels []string
	for _, choice := range choices {
		values = append(values, choice.Value)
		labels = append(labels, choice.Label)
	}
	wantValues := "https://pub.example:8443 https://10.0.0.2:8443 10.0.0.3"
	wantLabels := "pub.example:8443 10.0.0.2:8443 10.0.0.3"
	if got := strings.Join(values, " "); got != wantValues {
		t.Errorf("the addresses are %q, want %q", got, wantValues)
	}
	if got := strings.Join(labels, " "); got != wantLabels {
		t.Errorf("the labels are %q, want %q (without the scheme, as the web writes them)", got, wantLabels)
	}
}

func TestWithoutAPublicAddressTheFirstOneIsTheDefault(t *testing.T) {
	web := remote.WebState{Addresses: []string{"192.168.1.5:8443"}}
	if got := settingsInitialHost(web); got != "192.168.1.5:8443" {
		t.Errorf("the link names %q, want the first address", got)
	}
	if got := settingsInitialHost(remote.WebState{}); got != "" {
		t.Errorf("a computer with no address names %q, want none (the computer chooses)", got)
	}
}

func TestThePairingCodeIsASquareWithAQuietZone(t *testing.T) {
	matrix, err := settingsQRMatrix("https://192.168.1.5:8443/auth/connect/0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix) < 21 {
		t.Fatalf("the code has %d rows, want at least 21 (version 1)", len(matrix))
	}
	for y, row := range matrix {
		if len(row) != len(matrix) {
			t.Fatalf("row %d has %d modules, want %d (a square)", y, len(row), len(matrix))
		}
	}
	for x := range matrix[0] {
		if matrix[0][x] {
			t.Fatal("the first row is dark: the code has no quiet zone")
		}
	}
}

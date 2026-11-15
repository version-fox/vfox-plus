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

package msixinstall

import (
	"strings"
	"testing"
)

func TestParseListReadsTheLastListAfterPowerShellMessages(t *testing.T) {
	out := "WARNING: something\n" +
		`[{"PackageFullName":"dev.vfox.VfoxPlus_1.0.0.0_x64__abc","Architecture":"X64","SignatureKind":"Developer"}]` + "\n"
	var pkgs []Package
	if err := parseList(out, &pkgs); err != nil {
		t.Fatalf("parseList: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Architecture != "X64" || pkgs[0].FullName != "dev.vfox.VfoxPlus_1.0.0.0_x64__abc" {
		t.Errorf("parseList = %+v", pkgs)
	}
}

func TestParseListOfNothing(t *testing.T) {
	var certs []Certificate
	if err := parseList("[]\n", &certs); err != nil || len(certs) != 0 {
		t.Errorf("parseList of an empty list = %v, %v; want none and no error", certs, err)
	}
}

func TestParseListRefusesOutputWithoutAList(t *testing.T) {
	var certs []Certificate
	err := parseList("the cmdlet printed text\n", &certs)
	if err == nil || !strings.Contains(err.Error(), "printed no list") {
		t.Errorf("parseList = %v, want an error that says no list was printed", err)
	}
}

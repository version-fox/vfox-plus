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

package measureapk

import "testing"

func TestApplicationID(t *testing.T) {
	id, ok := ApplicationID("android {\n    defaultConfig {\n        applicationId = \"dev.vfox.vfoxplus\"\n        minSdk = 26\n    }\n}")
	if !ok || id != "dev.vfox.vfoxplus" {
		t.Fatalf("got %q, %v", id, ok)
	}
	if _, ok := ApplicationID("namespace = \"dev.vfox.vfoxplus\""); ok {
		t.Fatal("got an id without applicationId")
	}
}

func TestParseMeminfo(t *testing.T) {
	report := `Applications Memory Usage (in Kilobytes):
Uptime: 100 Realtime: 100

** MEMINFO in pid 4321 [dev.vfox.vfoxplus] **
                   Pss  Private  Private  SwapPss      Rss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty    Total     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------   ------
  Native Heap     5000     4900        0        0     6000    10000     6000     4000
        TOTAL    81920    60000    12000        0   130048    20000     9000     8000

 App Summary
                       Pss(KB)                        Rss(KB)
 TOTAL PSS:    81920            TOTAL RSS:   130048       TOTAL SWAP PSS:        0
`
	pss, rss, ok := ParseMeminfo(report)
	if !ok || pss != 81920 {
		t.Fatalf("pss = %d, %v", pss, ok)
	}
	if rss != 130048 {
		t.Fatalf("rss = %d", rss)
	}
	if _, _, ok := ParseMeminfo("No process found for: dev.vfox.vfoxplus"); ok {
		t.Fatal("parsed an empty report")
	}
}

func TestLine(t *testing.T) {
	if got := Line(81920, 130048); got != "memory (apk-cold): RSS 127 MB, PSS 80 MB in 1 processes" {
		t.Fatalf("got %q", got)
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run([]string{}); err == nil {
		t.Fatal("want a usage error")
	}
}

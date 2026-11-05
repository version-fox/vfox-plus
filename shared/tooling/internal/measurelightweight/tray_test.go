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

package measurelightweight

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
)

func TestTheIconMayCostALittleAndNotTheFallbackOfTheLibrary(t *testing.T) {
	without := measure.Usage{RSS: 64000, PSS: 49000, Private: 15600, Processes: 1}
	cases := map[string]struct {
		with measure.Usage
		fail string
	}{
		"what it costs":              {measure.Usage{RSS: 64440, Private: 15680}, ""},
		"less, as a run varies":      {measure.Usage{RSS: 63500, Private: 15400}, ""},
		"as much as it may":          {measure.Usage{RSS: 64000 + maxIconResident, Private: 15600 + maxIconPrivate}, ""},
		"the fallback, 7 MB":         {measure.Usage{RSS: 71400, Private: 17400}, fmt.Sprintf("KB of resident memory, at most %d", maxIconResident)},
		"only private memory grows":  {measure.Usage{RSS: 64100, Private: 15600 + maxIconPrivate + 1}, fmt.Sprintf("KB of private memory, at most %d", maxIconPrivate)},
		"a little over the resident": {measure.Usage{RSS: 64000 + maxIconResident + 1, Private: 15600}, "KB of resident memory"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkIconCost(without, c.with)
			if c.fail == "" {
				if err != nil {
					t.Fatalf("checkIconCost = %v, want none", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.fail) || !strings.Contains(err.Error(), "costs too much memory") {
				t.Fatalf("checkIconCost = %v, want %q", err, c.fail)
			}
		})
	}
}

// Without the session bus of a desktop there is no host to stand in for, and the tool says so before
// it starts the application.
func TestTheIconIsMeasuredOnADesktopOnly(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "no-bus"))
	err := measureTray([]string{"/nonexistent/application"}, t.TempDir(), "", time.Second)
	if err == nil || !strings.Contains(err.Error(), "session bus") {
		t.Fatalf("measureTray = %v, want the demand for the session bus of a desktop", err)
	}
}

func TestTheTrayFlagIsKnown(t *testing.T) {
	// The flag is parsed before anything starts: a run that fails after it has been parsed,
	// as this one does for lack of an application, was not refused for the flag.
	err := Run([]string{"--tray", "--settle", "-1", "application"})
	if err == nil || !strings.Contains(err.Error(), "--settle must not be negative") {
		t.Fatalf("Run(--tray) = %v", err)
	}
}

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

package e2erun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

// The scenario on Windows reads the log of the application for what the end-to-end build writes there, which no
// compiler checks: a word that drifted in one of the two would make the scenario wait for what never comes.
func TestTheScenarioReadsWhatTheApplicationSaysAboutItsIcon(t *testing.T) {
	root, err := appsession.FindRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, "mygo", "ui", "e2e.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, words := range []string{iconPlaced, iconNoPlace} {
		// The application writes them through log.Printf and log.Print, with the numbers of the place after the first.
		if !strings.Contains(string(source), words) {
			t.Errorf("mygo/ui/e2e.go does not say %q, which the scenario waits for", words)
		}
	}
}

func TestTheLogOfTheApplicationTellsWhereTheIconIs(t *testing.T) {
	log := "2026/11/04 15:00:01 e2e: the icon is in the tray at 1880,1040, 24 by 24\r\n" +
		"2026/11/04 15:00:09 something else\r\n" +
		"2026/11/04 15:00:12 e2e: the icon is in the tray at 1880,1040, 24 by 24\r\n"
	if got := placedCount(log); got != 2 {
		t.Errorf("placed %d times, want 2", got)
	}
	if gaveNoPlace(log) {
		t.Error("the desktop gave a place, and the log is read as if it did not")
	}
	if !gaveNoPlace(log + "2026/11/04 15:00:30 e2e: the icon has no place in the tray\n") {
		t.Error("the desktop gave none, and the log is not read so")
	}
	if placedCount("") != 0 || gaveNoPlace("") {
		t.Error("an empty log says nothing about the icon")
	}
}

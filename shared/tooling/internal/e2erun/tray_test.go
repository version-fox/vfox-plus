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
	"errors"
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

func TestTheLogOfTheApplicationTellsWhatTheDesktopAnsweredForEachIcon(t *testing.T) {
	log := "2026/11/04 15:00:01 e2e: the icon is in the tray at 1880,1040, 24 by 24\r\n" +
		"2026/11/04 15:00:09 something else\r\n" +
		"2026/11/04 15:00:12 e2e: the icon has no place in the tray\r\n" +
		"2026/11/04 15:00:20 e2e: the icon is in the tray at 1880,1040, 24 by 24\r\n"
	got := iconOutcomes(log)
	if len(got) != 3 || !got[0].Placed || got[1].Placed || !got[2].Placed {
		t.Fatalf("outcomes = %+v, want a place, none, a place", got)
	}
	if want := "2026/11/04 15:00:01 e2e: the icon is in the tray at 1880,1040, 24 by 24"; got[0].Line != want {
		t.Errorf("the line is %q, want %q, with the line end taken off", got[0].Line, want)
	}
	if got := iconOutcomes(""); len(got) != 0 {
		t.Errorf("an empty log says something about the icons: %+v", got)
	}
	if got := iconOutcomes("2026/11/04 15:00:01 nothing of the kind\n"); len(got) != 0 {
		t.Errorf("a log with no word of the icons says something about them: %+v", got)
	}
}

func TestTheRecordOfAnIconIsFoundByTheNameOfItsProgram(t *testing.T) {
	records := []shellRecord{
		{Key: "1", Executable: `C:\Windows\explorer.exe`, ToolTip: "Network", Promoted: 1},
		{Key: "2", Executable: `C:\a\vfox-plus\vfox-plus\e2e\arm64\vfox-plus-e2e.exe`, ToolTip: "vfox+ is running in the background", Promoted: 0},
		{Key: "3", Executable: "", ToolTip: "Without a program"},
	}
	for name, executable := range map[string]string{
		"the path as the shell wrote it":            `C:\a\vfox-plus\vfox-plus\e2e\arm64\vfox-plus-e2e.exe`,
		"the path with the slashes of the scenario": `C:\a\vfox-plus\vfox-plus\e2e/arm64/vfox-plus-e2e.exe`,
		"another case":   `c:\A\VFOX-PLUS\e2e\ARM64\VFOX-PLUS-E2E.EXE`,
		"the name alone": "vfox-plus-e2e.exe",
	} {
		record, found := recordOf(records, executable)
		if !found || record.Key != "2" {
			t.Errorf("%s: recordOf(%q) = %+v, %t, want the record 2", name, executable, record, found)
		}
	}
	if record, found := recordOf(records, `C:\other\application.exe`); found {
		t.Errorf("the record of another program was found: %+v", record)
	}
	if record, found := recordOf(records, ""); found {
		t.Errorf("a record without a program was found for no program: %+v", record)
	}
	if _, found := recordOf(nil, "vfox-plus-e2e.exe"); found {
		t.Error("a record was found among none")
	}
}

func TestTheRecordsOfTheShellAreListedForTheOneWhoReadsAFailure(t *testing.T) {
	records := []shellRecord{{Key: "7", Executable: `C:\x\a.exe`, ToolTip: "A", Promoted: 1}, {Key: "8", Executable: `C:\x\b.exe`, ToolTip: "B", Promoted: -1}}
	want := "the shell has 2 records of icons:\n  7: C:\\x\\a.exe, tool tip \"A\", promoted 1\n  8: C:\\x\\b.exe, tool tip \"B\", promoted -1"
	if got := describeRecords(records, nil); got != want {
		t.Errorf("describeRecords =\n%s\nwant\n%s", got, want)
	}
	if got := describeRecords(nil, nil); got != "the shell has no records of icons" {
		t.Errorf("no records: %q", got)
	}
	if got := describeRecords(nil, errors.New("the key is not there")); got != "the records of the shell cannot be read: the key is not there" {
		t.Errorf("a failure: %q", got)
	}
}

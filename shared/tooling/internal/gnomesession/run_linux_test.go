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

//go:build linux

package gnomesession

import (
	"os"
	"strings"
	"testing"
)

func TestTheSessionIsNotStartedWithoutACommandOrWithoutTheRightToStartIt(t *testing.T) {
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "Usage: gnome-session") {
		t.Errorf("Run(nil) = %v", err)
	}
	if os.Geteuid() != 0 {
		// Not root and not in the session: the system bus of the container is not this
		// tool's to start, and the message says what to do.
		if err := Run([]string{"true"}); err == nil || !strings.Contains(err.Error(), "takes root") {
			t.Errorf("Run(true) as an ordinary user = %v", err)
		}
	}
}

func TestACommandThatFailsIsReportedWithItsStatus(t *testing.T) {
	if err := exitOf(nil); err != nil {
		t.Errorf("exitOf(nil) = %v", err)
	}
}

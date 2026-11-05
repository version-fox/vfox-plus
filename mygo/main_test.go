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

package main

import (
	"runtime"
	"testing"
)

func TestTheSchedulerUsesAFewProcessorsUnlessTheUserSaysOtherwise(t *testing.T) {
	previous := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	// A machine with many processors: the application takes a few of them.
	runtime.GOMAXPROCS(maxProcessors + 12)
	t.Setenv("GOMAXPROCS", "")
	limitProcessors()
	if got := runtime.GOMAXPROCS(0); got != maxProcessors {
		t.Errorf("GOMAXPROCS = %d, want %d", got, maxProcessors)
	}

	// One with fewer: it takes what there is.
	runtime.GOMAXPROCS(2)
	limitProcessors()
	if got := runtime.GOMAXPROCS(0); got != 2 {
		t.Errorf("GOMAXPROCS = %d, want 2: a limit does not add processors", got)
	}

	// The user who sets the number has chosen it.
	runtime.GOMAXPROCS(maxProcessors + 12)
	t.Setenv("GOMAXPROCS", "12")
	limitProcessors()
	if got := runtime.GOMAXPROCS(0); got != maxProcessors+12 {
		t.Errorf("GOMAXPROCS = %d, want the user's choice to stand", got)
	}
}

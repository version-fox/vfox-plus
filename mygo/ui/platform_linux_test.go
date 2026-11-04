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

package ui

import "testing"

// Where the session bus cannot be reached there is no host for tray icons, and asking for
// one says so without an error that stops the application. (That a host that is there is
// found is shown by driving the application against one, which needs a desktop: see
// docs/en/dev/architecture.md, "The tray".)
func TestADesktopWhoseSessionBusCannotBeReachedHasNoTrayHost(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+t.TempDir()+"/no-bus")
	if trayHostAvailable() {
		t.Error("a host for tray icons on a bus that is not there")
	}
}

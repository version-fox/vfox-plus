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
	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

// fakeScenario is the application against the fake vfox: the Home page, the
// versions of Node.js, using one and installing another.
func fakeScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("home")
	// The executable of the settings is the one that the Home page names.
	s.WaitForText("vfox 1.0.13 · " + s.Vfox)

	s.Navigate(appsession.SDKs)
	s.ClickItem("nodejs")
	s.WaitForText("Node.js runtime environment.")
	s.Step("detail")

	// Select 22.21.0 as the global default.
	s.ClickButton("Use")
	s.WaitForTask("Use nodejs@22.21.0")

	// Install a version from the available tab.
	s.ClickButton("Available")
	s.WaitForText("872 versions")
	s.Fill("Filter versions", "24.11.0", false)
	s.WaitForText("1 version")
	s.ClickButtonNear("24.11.0", "Install")
	s.WaitForTask("Install nodejs@24.11.0")

	// What vfox printed is kept with the task, under its row.
	s.Navigate(appsession.Tasks)
	s.ClickItem("Install nodejs@24.11.0")
	s.WaitForTextContaining("Install nodejs@24.11.0 success!")
}

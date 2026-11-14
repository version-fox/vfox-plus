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

// runPython runs the application against the fake vfox of a machine that has Python
// installed next to the others of the profiles (VFOX_PLUS_FAKE_INVENTORY=python).
func runPython(command []string) error {
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	config := appsession.Config{
		Command:  command,
		Fixtures: appsession.FixturesDir(root),
		Env:      []string{"VFOX_PLUS_FAKE_INVENTORY=python"},
	}
	return appsession.RunScenario(config, "python", pythonScenario)
}

// pythonScenario is the application on that machine: the options of Python, where the
// install method is set to uv-build only, and the versions, where the filter of the GraalPy
// builds narrows the list to the GraalPy Community releases and one of them is installed.
func pythonScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("home")

	s.Navigate(appsession.SDKs)
	s.ClickItem("python")
	s.WaitForText("Python language support for CPython and GraalPy, https://www.python.org")
	s.Step("detail")

	// The downloads and the version files of Python, which the plugin reads. The row of the
	// install method shows the preset that was saved, with its value.
	s.ClickButton("Options")
	s.WaitForText("python.org mirror")
	// The version files of the plugin come after the configuration is loaded; the name of the
	// file is a tag of the switch, which the accessibility tree does not name.
	s.WaitForText("Version used from legacy files")
	// The rows of the options are not items of their own in the accessibility tree: the page is one
	// item, which holds the titles of all of them. ClickButtonNear would press the first Change of the
	// page, the one of the mirror, so the button is named by the key of its variable instead.
	s.PressKey("change:VFOX_PYTHON_USE_UV_BUILD")
	s.ClickButton("uv-build only")
	s.ClickButton("Save")
	s.WaitForText("uv-build only · 1")
	s.Step("options")

	// The filter of the GraalPy builds: 18 of the 999 versions that the plugin lists. The one that is
	// installed is found by its name in the filter, as the fake scenario does, so that one row is left.
	s.ClickButton("Available")
	s.ClickButton("GraalPy Community")
	s.WaitForText("18 versions")
	s.Fill("Filter versions", "graalpy3.13-community-25.5.5", false)
	s.WaitForText("1 version")
	s.ClickButtonNear("graalpy3.13-community-25.5.5", "Install")
	s.WaitForTask("Install python@graalpy3.13-community-25.5.5")
	s.Step("installed")
}

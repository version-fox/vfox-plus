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

// runPlugins runs the application against the fake vfox of a machine that has Erlang/OTP, clang and etcd
// installed, and whose plugins are the ones of the profiles (VFOX_PLUS_FAKE_INVENTORY=plugins).
func runPlugins(command []string) error {
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	config := appsession.Config{
		Command:  command,
		Fixtures: appsession.FixturesDir(root),
		Env:      []string{"VFOX_PLUS_FAKE_INVENTORY=plugins"},
	}
	return appsession.RunScenario(config, "plugins", pluginsScenario)
}

// profilePlugins are the plugins whose pages the scenario opens, with the description that their captured plugin
// information gives. The description is on the page of the plugin once it is open.
var profilePlugins = []struct{ name, description string }{
	{"etcd", "etcd vfox plugin, support for managing multiple etcd & etcdctl & etcdutl versions."},
	{"erlang", "Erlang/OTP vfox plugin, support for managing multiple Erlang/OTP versions in Windows & Unix-like systems."},
	{"deno", "Deno plugin, https://deno.com/"},
	{"groovy", "Groovy plugin, https://groovy.apache.org/"},
	{"scala", "Scala plugin, https://www.scala-lang.org/"},
	{"zig", "Zig runtime environment."},
	{"clang", "Clang plugin, https://clang.llvm.org/"},
	{"elixir", "Elixir vfox plugin, support for managing multiple Elixir language versions in Windows & Unix-like system."},
}

// pluginsScenario is the application on that machine: the page of each plugin of the profiles, then the
// options of Erlang/OTP, where the prebuilt builds of Ubuntu 24.04 are chosen and saved, and the install of a
// zig from its list of versions, which is filtered to the one installed.
func pluginsScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("home")

	for _, p := range profilePlugins {
		s.Navigate(appsession.SDKs)
		s.ClickItem(p.name)
		s.WaitForText(p.description)
		s.Step(p.name)
	}

	// The prebuilt builds of Erlang/OTP: the choice of the release is a preset of the variable of the plugin.
	s.Navigate(appsession.SDKs)
	s.ClickItem("erlang")
	s.ClickButton("Options")
	s.WaitForText("Prebuilt build")
	s.PressKey("change:USE_PREBUILT_OTP")
	s.ClickButton("Ubuntu 24.04")
	s.ClickButton("Save")
	s.WaitForText("Ubuntu 24.04 · ubuntu-24.04")
	s.Step("options")

	// The install of zig 0.17.1: the list of its versions is filtered to that one name, which is installed.
	s.Navigate(appsession.SDKs)
	s.ClickItem("zig")
	s.ClickButton("Available")
	s.Fill("Filter versions", "0.17.1", false)
	s.WaitForText("1 version")
	s.ClickButtonNear("0.17.1", "Install")
	s.WaitForTask("Install zig@0.17.1")
	s.Step("installed")
}

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

package ui

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// A plugin that the application has no profile of brings its own, with a switch
// and a value of its own, and no way to find installs that are on the machine.
func TestAPluginThatDescribesItself(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬ruby\n  └──3.4.1\n"}).
		On("current", core.ProcessOutput{Stdout: "ruby -> v3.4.1\n"}).
		On("info ruby --format "+vfox.PluginInfoTemplate, core.ProcessOutput{Stdout: "ruby\x1f0.1.0\x1fhttps://example.com/ruby\x1f/home/u/.vfox/cache/ruby\x1fRuby"}).
		On("alias list ruby", testsupport.FixtureOutput("alias-list", 0)).
		On("search ruby", core.ProcessOutput{Stdout: "Available versions:\n - 3.4.1\n - 3.3.7\n"})
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	for name, content := range map[string]string{
		"hooks/available.lua": ``,
		"lib/vfox-plus.json": `{"variables": [
			{"key": "VFOX_RUBY_QUIET", "label": "Quiet builds", "help": "Say little.", "kind": "toggle"},
			{"key": "VFOX_RUBY_CONFIGURE", "label": "Build options", "help": "Given to configure.", "kind": "text"}]}`,
	} {
		file := filepath.Join(ta.app.home(), ".vfox", "plugin", "ruby", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ta.start()
	ta.openAvailable("ruby")
	ta.until("the files of the plugin", func() bool { return ta.app.detailFor("ruby").controller.Files.HasValue() })
	ta.click(kit.RoleButton, "More ways to add")
	if _, offered := ta.find(kit.RoleMenuItem, "Detect and link"); offered {
		t.Error("a plugin without the Detect hook cannot find installs")
	}
	ta.click(kit.RoleMenuItem, "Install by name…")
	ta.button("Cancel")

	ta.openTab("Options")
	ta.mustText("VFOX_RUBY_QUIET")
	ta.snapshot("options-plugin-profile")
	// A value of the profile of the plugin is asked about before it is set.
	ta.click(kit.RoleSwitch, "Quiet builds")
	ta.mustText("Use this value?")
	ta.snapshot("options-plugin-confirm")
	ta.button("Cancel")
	if got := ta.app.model.PluginVariable("VFOX_RUBY_QUIET"); got != "" {
		t.Fatalf("a question that is refused sets nothing: %q", got)
	}
	ta.click(kit.RoleSwitch, "Quiet builds")
	ta.button("Use value")
	if got := ta.app.model.PluginVariable("VFOX_RUBY_QUIET"); got != "1" {
		t.Fatalf("stored %q", got)
	}
	// Switching it off takes it away, which needs no question.
	ta.click(kit.RoleSwitch, "Quiet builds")
	if got := ta.app.model.PluginVariable("VFOX_RUBY_QUIET"); got != "" || ta.text("Use this value?") {
		t.Fatalf("stored %q", got)
	}

	// A text that the user types is theirs: no question.
	ta.button("Change")
	field, ok := ta.find(kit.RoleTextField, "Value")
	if !ok {
		t.Fatalf("no field for the value: %q", ta.texts())
	}
	ta.fill(field, "--prefix=/opt/ruby --disable-install-doc", true)
	if got := ta.app.model.PluginVariable("VFOX_RUBY_CONFIGURE"); got != "--prefix=/opt/ruby --disable-install-doc" {
		t.Errorf("stored %q", got)
	}
}

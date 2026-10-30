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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/testserver"
)

// runReal runs the application against the real vfox (VFOX_PLUS_E2E_VFOX), with the
// offline test plugin that a local server serves.
func runReal(command []string) error {
	vfox := os.Getenv("VFOX_PLUS_E2E_VFOX")
	if vfox == "" {
		return errors.New("Set VFOX_PLUS_E2E_VFOX to the vfox executable.")
	}
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	plugin := filepath.Join(root, "shared", "test-plugin", "testsdk.lua")
	if _, err := os.Stat(plugin); err != nil {
		return fmt.Errorf("test plugin not found at %s: %w", plugin, err)
	}
	// The test server runs in this process: the plugin of vfox downloads from it.
	server, stop, err := testserver.Start(plugin)
	if err != nil {
		return fmt.Errorf("start the test server: %w", err)
	}
	defer stop()
	config := appsession.Config{Command: command, Vfox: vfox, Env: []string{"VFOX_PLUS_TEST_SERVER=" + server}}
	return appsession.RunScenario(config, "real", func(s *appsession.Session) { realScenario(s, server, plugin) })
}

// realScenario is the application against the real vfox: the plugin registry, the
// installation, the cancellation, the use and the removal of what the offline test
// plugin offers.
func realScenario(s *appsession.Session, server, pluginSource string) {
	home := s.Home
	control := func(action string) {
		response, err := http.Get(server + "/control/" + action) //nolint:gosec,noctx // The server of the test.
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}
	// vfox moves local .lua files away, so the application must hand it a copy.
	localPlugin := filepath.Join(s.Sandbox, "my-plugin.lua")
	source, err := os.ReadFile(pluginSource) //nolint:gosec // The test plugin of the repository.
	s.Check(err == nil, "the test plugin can be read")
	s.Check(os.WriteFile(localPlugin, source, 0o644) == nil, "the copy of the test plugin can be written")

	s.WaitForText("Versions in use")
	installed, _ := os.ReadDir(filepath.Join(home, ".vfox", "plugin"))
	s.Check(len(installed) == 0, "no plugin is installed yet")

	// Point vfox to the local registry through the vfox configuration editor.
	s.Step("configure the registry")
	s.Navigate(appsession.Settings)
	s.Fill("Plugin registry", server+"/registry", true)
	s.WaitFor("the registry to be saved", 30*time.Second, func() bool {
		return strings.TrimSpace(s.Vfoxcmd("config", "registry.address")) == server+"/registry"
	})

	// Add the plugin from the registry.
	s.Step("add a plugin from the registry")
	s.Navigate(appsession.Plugins)
	s.WaitForText("testsdk")
	s.ClickButton("Add")
	s.WaitForTask("Add plugin testsdk")
	s.WaitForText("Added")

	// Add a local copy of the plugin under another name.
	s.Step("add a plugin from a local file")
	s.ClickButton("Add from source…")
	s.WaitForText("Add a plugin from source")
	s.Fill("Source", localPlugin, false)
	s.Fill("Name (optional)", "localsdk", false)
	s.TapDialog(true)
	s.WaitForTask("Add plugin localsdk")
	_, err = os.Stat(localPlugin)
	s.Check(err == nil, "the source file survives vfox add --source")

	// Search and install both versions.
	s.Step("install")
	s.Navigate(appsession.SDKs)
	s.ClickItem("testsdk")
	s.WaitForText("An offline SDK for the vfox-plus end-to-end tests.")
	s.ClickButton("Available")
	s.WaitForText("2 versions")
	s.WaitForText("LTS")
	s.WaitForText("testtool 2.0")
	s.Check(s.Tree().Count("LTS") == 2, "the note is shown once on the version, and once on the chip that filters by it")

	// The plugin takes an argument, which the real vfox hands to its Available hook:
	// the long-term support releases only. A chip of the notes narrows the list in the
	// same way, without asking vfox again.
	s.Step("choose what to list")
	s.ClickButton("LTS")
	s.WaitForText("1 version")
	s.ClickButton("Clear filters")
	s.WaitForText("2 versions")
	s.ClickButton("Channel: Every version")
	s.WaitForText("Long-term support")
	s.ClickItem("Long-term support")
	s.WaitForText("Channel: Long-term support")
	s.WaitUntil("the list of the long-term support releases", 30*time.Second, func(t automation.Tree) bool {
		return t.Shows("1 version") && t.RowSays("2.0.0", "LTS") && !t.Shows("1.0.0")
	})
	lts := s.Vfoxcmd("search", "testsdk", "lts")
	s.Check(strings.Contains(lts, "2.0.0") && !strings.Contains(lts, "1.0.0"), "vfox itself lists the same")
	s.ClickButton("Channel: Long-term support")
	s.ClickItem("Every version")
	s.WaitForText("2 versions")

	s.ClickButtonNear("2.0.0", "Install")
	s.WaitForTask("Install testsdk@2.0.0")

	// Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
	// leaves the partial installation behind, but must neither list it nor
	// refuse to install the version again.
	s.Step("cancel an installation")
	control("hold")
	s.ClickButtonNear("1.0.0", "Install")
	s.WaitUntil("the download to start", 30*time.Second, func(t automation.Tree) bool {
		percent, _ := t.Downloading()
		return percent > 0
	})
	s.ClickButton("Cancel")
	s.WaitUntil("vfox to exit", 30*time.Second, func(t automation.Tree) bool { return t.RowSays("1.0.0", "Install") })
	s.CheckNoTaskFailed()
	s.Check(!strings.Contains(s.Vfoxcmd("list", "testsdk"), "1.0.0"), "vfox does not list the partial version")
	control("release")
	s.Pause(500 * time.Millisecond)
	// Install it again by its name, which vfox looks up while it installs.
	s.Step("install by name")
	s.ClickButton("Install by name…")
	s.WaitForText("Install testsdk by name")
	s.Fill("Version name", "1.0.0", false)
	s.TapDialog(true)
	s.WaitForTask("Install testsdk@1.0.0")

	// Make 2.0.0 the global default.
	s.Step("use")
	s.Navigate(appsession.SDKs)
	s.ClickButton("Installed (2)")
	s.WaitForText("1.0.0")
	s.ClickButtonNear("2.0.0", "Use")
	s.WaitForTask("Use testsdk@2.0.0")
	s.WaitForText("In use: 2.0.0")
	toml, _ := os.ReadFile(filepath.Join(home, ".vfox", ".vfox.toml")) //nolint:gosec // A file of the sandbox.
	s.Check(strings.Contains(string(toml), `testsdk = "2.0.0"`), "the global .vfox.toml pins 2.0.0")
	_, err = os.Lstat(filepath.Join(home, ".vfox", "sdks", "testsdk"))
	s.Check(err == nil, "the SDK is linked")

	// Uninstall 1.0.0.
	s.Step("uninstall")
	s.ChooseMenuItem("More actions for 1.0.0", "Uninstall")
	s.WaitForText("Uninstall testsdk 1.0.0?")
	s.TapDialog(true)
	s.WaitForTask("Uninstall testsdk@1.0.0")
	s.WaitForText("Installed (1)")

	// Remove the second plugin.
	s.Step("remove")
	s.ClickItem("localsdk")
	s.WaitForText("localsdk")
	s.ChooseMenuItem("More actions", "Remove plugin")
	s.WaitForText("Remove localsdk?")
	s.TapDialog(true)
	s.WaitForTask("Remove plugin localsdk")
	s.WaitForText("Select an SDK to manage its versions.")
	s.WaitUntil("the removed plugin to disappear from the list", 30*time.Second, func(t automation.Tree) bool {
		_, listed := t.Find(func(n automation.Node) bool { return n.KeyText() == "sdk:localsdk" })
		return !listed
	})

	// Every task succeeded (the waits above fail on one that did not), the cancelled
	// one apart, and vfox agrees with what the application shows.
	s.CheckNoTaskFailed()
	s.Check(s.Vfoxcmd("current") == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0")
	s.Check(strings.Contains(s.Vfoxcmd("list", "testsdk"), "2.0.0"), "vfox lists testsdk 2.0.0")
	// The copy of the local plugin is deleted once its task has finished.
	entries, _ := os.ReadDir(filepath.Join(home, ".vfox", "tmp"))
	var leftovers []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vfox-plus-plugin-") {
			leftovers = append(leftovers, entry.Name())
		}
	}
	s.Check(len(leftovers) == 0, fmt.Sprintf("the copy of the local plugin is deleted: %v", leftovers))
}

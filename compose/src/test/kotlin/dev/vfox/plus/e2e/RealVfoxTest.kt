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

package dev.vfox.plus.e2e

import androidx.compose.ui.test.hasText
import dev.vfox.plus.state.TaskStatus
import org.junit.jupiter.api.Assumptions.assumeTrue
import org.junit.jupiter.api.Tag
import java.nio.file.Files
import kotlin.io.path.exists
import kotlin.io.path.name
import kotlin.io.path.readText
import kotlin.test.Test

/**
 * Every operation against the real vfox of VFOX_PLUS_E2E_VFOX and the offline
 * test plugin of shared/test-plugin, in a throw-away home directory. Skipped
 * when no vfox is named; see docs/vfox-cli-contract.md#testing-against-the-real-vfox.
 */
@Tag("e2e")
class RealVfoxTest {
    @Test
    fun `every operation against the real vfox`() {
        assumeTrue(realVfox != null, "VFOX_PLUS_E2E_VFOX names no vfox executable")
        Harness(Vfox.REAL).use { harness ->
            val server = requireNotNull(harness.server)
            runApp(harness) {
                step("scenario real started")
                waitForText("Versions in use")
                waitFor("the installed plugins to load") { model.installed.hasValue }
                check(model.installed.value?.isEmpty() == true, "no plugin is installed yet")

                // Point vfox to the local registry through the vfox configuration editor.
                step("configure the registry")
                clickDestination("Settings")
                fill("Plugin registry", "${server.url}/registry", submit = true)
                waitForText("Saved")
                check(harness.vfox("config", "registry.address").trim() == "${server.url}/registry", "vfox stored the registry")

                // Add the plugin from the registry.
                step("add a plugin from the registry")
                clickDestination("Plugins")
                waitForText("testsdk")
                clickButtonIn("plugin:testsdk", "Add")
                waitForTask("Add plugin testsdk")
                waitForText("Added")

                // Add a local copy of the plugin under another name. vfox moves
                // local .lua files away, so the application must hand it a copy.
                step("add a plugin from a local file")
                clickButton("Add from source…")
                waitForText("Add a plugin from source")
                fill("Source", harness.localPlugin.toString())
                fill("Name (optional)", "localsdk")
                clickInDialog("Add plugin")
                waitForTask("Add plugin localsdk")
                check(harness.localPlugin.exists(), "the source file survives vfox add --source")

                // Search and install both versions.
                step("install")
                clickDestination("SDKs")
                clickTag("sdk:testsdk")
                waitForText("An offline SDK for the vfox-plus end-to-end tests.", substring = true)
                clickButton("Available")
                waitForText("2 versions")
                waitForText("testtool 2.0")
                check(count(hasText("LTS")) == 1, "the note is shown once")
                clickButtonIn("available:2.0.0", "Install")
                waitForTask("Install testsdk@2.0.0")
                check(
                    model.tasks.tasks
                        .first()
                        .progress == 1.0,
                    "vfox reports the download progress on standard error",
                )

                // Cancel the installation of 1.0.0 during the download. From 1.0.13
                // vfox leaves the partial installation behind, but must neither
                // list it nor refuse to install the version again.
                step("cancel an installation")
                server.hold()
                clickButtonIn("available:1.0.0", "Install")
                waitFor("the download to start") {
                    (
                        model.tasks.tasks
                            .first()
                            .progress ?: 0.0
                    ) > 0.0
                }
                clickButtonIn("available:1.0.0", "Cancel")
                waitFor("vfox to exit") {
                    model.tasks.tasks
                        .first()
                        .finishedAt != null
                }
                check(
                    model.tasks.tasks
                        .first()
                        .status == TaskStatus.CANCELLED,
                    "the task is cancelled",
                )
                check("1.0.0" !in harness.vfox("list", "testsdk"), "vfox does not list the partial version")
                server.release()
                Thread.sleep(500)
                clickButtonIn("available:1.0.0", "Install")
                waitForTask("Install testsdk@1.0.0")

                // Make 2.0.0 the global default.
                step("use")
                clickButton("Installed (2)")
                waitForText("1.0.0")
                clickButtonIn("installed:2.0.0", "Use")
                waitForTask("Use testsdk@2.0.0")
                waitForText("In use: 2.0.0")
                check(
                    harness.home
                        .resolve(".vfox/.vfox.toml")
                        .readText()
                        .contains("testsdk = \"2.0.0\""),
                    "the global .vfox.toml pins 2.0.0",
                )
                check(harness.home.resolve(".vfox/sdks/testsdk").exists(), "the SDK is linked")

                // Uninstall 1.0.0.
                step("uninstall")
                clickButton("More actions for 1.0.0")
                clickInMenu("Uninstall")
                waitForText("Uninstall testsdk 1.0.0?")
                clickInDialog("Uninstall")
                waitForTask("Uninstall testsdk@1.0.0")
                waitForText("Installed (1)")

                // Remove the second plugin.
                step("remove")
                clickTag("sdk:localsdk")
                waitFor("the detail of localsdk") { navigation.shownDetail?.let { it.name == "localsdk" && it.info.hasValue } == true }
                clickButton("More actions")
                clickInMenu("Remove plugin")
                waitForText("Remove localsdk?")
                clickInDialog("Remove")
                waitForTask("Remove plugin localsdk")
                waitForText("Select an SDK to manage its versions.")
                check(model.installed.value?.none { it.name == "localsdk" } == true, "the removed plugin is no longer listed")

                // Every task but the cancelled one succeeded, and vfox agrees
                // with what the application shows.
                val statuses =
                    model.tasks.tasks
                        .map { it.status }
                        .filter { it != TaskStatus.SUCCEEDED }
                check(statuses == listOf(TaskStatus.CANCELLED), "only the cancelled task did not succeed: $statuses")
                check(harness.vfox("current") == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0")
                check("2.0.0" in harness.vfox("list", "testsdk"), "vfox lists testsdk 2.0.0")
                // The copy of the local plugin is deleted once its task has finished.
                val copies =
                    Files.list(harness.home.resolve(".vfox/tmp")).use { entries ->
                        entries.filter { it.name.startsWith("vfox-plus-plugin-") }.toList()
                    }
                check(copies.isEmpty(), "the copy of the local plugin is deleted: $copies")
                step("scenario real passed")
            }
        }
    }
}

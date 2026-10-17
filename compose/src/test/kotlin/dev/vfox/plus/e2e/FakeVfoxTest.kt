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

import dev.vfox.plus.state.TaskStatus
import dev.vfox.plus.ui.Links
import dev.vfox.plus.ui.pages.ACTIVATION_COMMAND
import org.junit.jupiter.api.Tag
import kotlin.test.Test

/**
 * The application against shared/fixtures/fake-vfox.sh, which replays the
 * captured output of vfox and fails when the application breaks the CLI
 * contract: the application is started as if from a terminal in which vfox
 * is activated, whose session it must not leak.
 */
@Tag("e2e")
class FakeVfoxTest {
    @Test
    fun `the main flows against the captured output of vfox`() =
        Harness(Vfox.FAKE).use { harness ->
            runApp(harness) {
                step("scenario fake started")
                waitForText("Versions in use")
                step("home")
                waitForText("vfox 1.0.13 · ${harness.executable}")
                check(model.vfox.executable == harness.executable, "the executable from the settings is used")
                clickButton("Copy")
                waitFor("the activation line to be copied") { services.copied == listOf(ACTIVATION_COMMAND) }
                clickButton("vfox documentation")
                check(services.opened == listOf(Links.VFOX_DOCS), "the documentation opens in the browser")

                clickDestination("SDKs")
                clickTag("sdk:nodejs")
                waitForText("Node.js runtime environment.", substring = true)
                step("detail")

                // Select 22.21.0 as the global default.
                clickButtonIn("installed:22.21.0", "Use")
                waitForTask("Use nodejs@22.21.0")

                // Install a version from the available tab.
                clickButton("Available")
                waitForText("872 versions")
                fill("Filter versions", "24.11.0")
                waitForText("1 version")
                clickButtonIn("available:24.11.0", "Install")
                waitForTask("Install nodejs@24.11.0")

                val tasks = model.tasks.tasks
                check(tasks.all { it.status == TaskStatus.SUCCEEDED }, "every task succeeded")
                check(tasks.first().progress == 1.0, "the download progress reached 100 %")
                check("Install nodejs@24.11.0 success!" in tasks.first().log, "the output of vfox is kept")

                // The task list shows both, with the output of vfox.
                clickDestination("Tasks")
                waitForText("Use nodejs@22.21.0")
                clickButtonIn("task:${tasks.first().id}", "Show output")
                waitForText("Install nodejs@24.11.0 success!", substring = true)
                step("scenario fake passed")
            }
        }
}

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

import dev.vfox.plus.core.LanguagePreference
import dev.vfox.plus.core.ThemePreference
import org.junit.jupiter.api.Tag
import java.io.File
import kotlin.test.Test

/**
 * Screenshots of every page against the fake vfox, for reviews of the user
 * interface and for comparing the implementations: `gradle screenshots`
 * writes them to build/screenshots.
 */
@Tag("screenshots")
class ScreenshotsTest {
    private val directory = File(System.getProperty("vfox.plus.screenshots") ?: "build/screenshots")

    @Test
    fun `the pages in light mode`() =
        Harness(Vfox.FAKE).use { harness ->
            runApp(harness) {
                waitForText("Versions in use")
                screenshot(directory, "home")

                clickDestination("SDKs")
                clickTag("sdk:nodejs")
                waitForText("Node.js runtime environment.", substring = true)
                screenshot(directory, "sdks")
                clickButton("More actions for 22.21.0")
                waitForText("Copy installation path")
                screenshot(directory, "menu")
                clickInMenu("Copy installation path")
                waitForText("Copied to the clipboard")

                clickButton("Available")
                waitForText("872 versions")
                screenshot(directory, "available")
                fill("Filter versions", "24.11.0")
                waitForText("1 version")
                clickButtonIn("available:24.11.0", "Install")
                waitForTask("Install nodejs@24.11.0")

                clickDestination("Tasks")
                clickButton("Show output")
                waitForText("Install nodejs@24.11.0 success!", substring = true)
                screenshot(directory, "tasks")

                clickDestination("Plugins")
                waitForText("dotnet")
                screenshot(directory, "plugins")
                clickButton("Add from source…")
                waitForText("Add a plugin from source")
                screenshot(directory, "dialog")
                clickInDialog("Cancel")

                clickDestination("Settings")
                waitForText("Plugin registry")
                screenshot(directory, "settings")
            }
        }

    @Test
    fun `an installation in progress`() =
        Harness(Vfox.FAKE, extraEnvironment = mapOf("VFOX_PLUS_FAKE_DELAY" to "4")).use { harness ->
            runApp(harness) {
                waitForText("Versions in use")
                clickDestination("SDKs")
                clickTag("sdk:nodejs")
                clickButton("Available")
                waitForText("872 versions")
                clickButtonIn("available:26.12.0", "Install")
                waitFor("the download to start") {
                    (
                        model.tasks.tasks
                            .firstOrNull()
                            ?.progress ?: 0.0
                    ) > 0.0
                }
                screenshot(directory, "installing")
            }
        }

    @Test
    fun `dark mode with the expanded rail`() =
        Harness(Vfox.FAKE, settings = { it.copy(theme = ThemePreference.DARK, railExpanded = true) }).use { harness ->
            runApp(harness) {
                waitForText("Versions in use")
                screenshot(directory, "dark-expanded")
                clickDestination("SDKs")
                clickTag("sdk:maven")
                waitForText("3.8.9")
                screenshot(directory, "dark-sdks")
            }
        }

    @Test
    fun `Simplified Chinese`() =
        Harness(Vfox.FAKE, settings = { it.copy(language = LanguagePreference.SIMPLIFIED_CHINESE) }).use { harness ->
            runApp(harness) {
                waitForText("正在使用的版本")
                screenshot(directory, "zh-home")
                clickDestination("设置")
                waitForText("插件仓库")
                screenshot(directory, "zh-settings")
            }
        }

    @Test
    fun `a compact window`() =
        Harness(Vfox.FAKE).use { harness ->
            runApp(harness, width = 412, height = 860) {
                waitForText("Versions in use")
                screenshot(directory, "compact-home")
                clickDestination("SDKs")
                clickTag("sdk:nodejs")
                waitForText("Node.js runtime environment.", substring = true)
                screenshot(directory, "compact-detail")
            }
        }
}

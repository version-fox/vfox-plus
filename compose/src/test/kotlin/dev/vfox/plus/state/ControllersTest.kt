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

package dev.vfox.plus.state

import dev.vfox.plus.core.AppSettings
import dev.vfox.plus.core.MemorySettingsStore
import dev.vfox.plus.core.ProcessOutput
import dev.vfox.plus.core.ThemePreference
import dev.vfox.plus.core.VfoxCli
import dev.vfox.plus.core.VfoxException
import dev.vfox.plus.core.VfoxLaunchConfig
import dev.vfox.plus.core.VfoxLocator
import dev.vfox.plus.support.FakeProcessRunner
import dev.vfox.plus.support.fixtureOutput
import dev.vfox.plus.support.fixtureRunner
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runTest
import java.nio.file.Files
import kotlin.io.path.createTempDirectory
import kotlin.io.path.exists
import kotlin.io.path.writeText
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertNull
import kotlin.test.assertTrue

class ControllersTest {
    @Test
    fun `a resource keeps its value while reloading and records errors`() =
        runTest {
            var calls = 0
            val resource =
                AsyncResource(backgroundScope) {
                    calls++
                    if (calls == 2) error("boom")
                    calls
                }
            resource.load().join()
            assertEquals(1, resource.value)
            assertTrue(resource.hasValue)
            val reload = resource.load()
            assertTrue(resource.isLoading)
            assertEquals(1, resource.value)
            reload.join()
            assertEquals("boom", resource.error?.message)
            assertEquals(1, resource.value)
            assertFalse(resource.isLoading)
        }

    @Test
    fun `the latest of overlapping loads wins and ensureLoaded loads once`() =
        runTest {
            val answers = mutableListOf<CompletableDeferred<String>>()
            val resource = AsyncResource(backgroundScope) { CompletableDeferred<String>().also { answers += it }.await() }
            val first = resource.load()
            val second = resource.load()
            testScheduler.runCurrent()
            answers[1].complete("new")
            answers[0].complete("old")
            first.join()
            second.join()
            assertEquals("new", resource.value)

            var loads = 0
            val counted = AsyncResource(backgroundScope) { ++loads }
            counted.ensureLoaded()
            counted.ensureLoaded()?.join()
            counted.ensureLoaded()
            assertEquals(1, loads)
            counted.replace(9)
            assertEquals(9, counted.value)
        }

    @Test
    fun `settings are persisted and no-ops skipped`() =
        runTest {
            val store = MemorySettingsStore(AppSettings(theme = ThemePreference.DARK))
            val controller = SettingsController(store)
            val changes = mutableListOf<Pair<AppSettings, AppSettings>>()
            controller.onChange { previous, current -> changes += previous to current }
            controller.load()
            assertEquals(ThemePreference.DARK, controller.settings.theme)
            controller.update { it.copy(theme = ThemePreference.DARK) }
            assertEquals(0, store.saveCount)
            controller.update { it.copy(theme = ThemePreference.LIGHT) }
            assertEquals(1, store.saveCount)
            assertEquals(ThemePreference.LIGHT, store.load().theme)
            assertEquals(ThemePreference.LIGHT, changes.single().second.theme)
        }

    private fun TestScope.vfox(
        runner: FakeProcessRunner,
        found: String? = null,
    ): Pair<VfoxCli, VfoxController> {
        val cli = VfoxCli(runner, VfoxLaunchConfig(executable = "vfox"))
        val locator = VfoxLocator(emptyMap()) { it == found }
        return cli to VfoxController(cli, locator, mapOf("HOME" to "/home/u"), StandardTestDispatcher(testScheduler))
    }

    @Test
    fun `vfox is detected or chosen, and its version read`() =
        runTest {
            val runner = fixtureRunner()
            val (cli, controller) = vfox(runner, "/usr/local/bin/vfox")
            controller.apply(AppSettings(extraEnvironment = mapOf("A" to "1")))
            assertEquals(VfoxAvailability.READY, controller.availability)
            assertEquals("1.0.13", controller.version)
            assertEquals("/usr/local/bin/vfox", controller.executable)
            assertFalse(controller.isCustomExecutable)
            assertNull(controller.error)
            assertEquals(mapOf("A" to "1"), cli.launchConfig.extraEnvironment)
            controller.apply(AppSettings(vfoxExecutable = "/opt/vfox"))
            assertEquals("/opt/vfox", controller.executable)
            assertTrue(controller.isCustomExecutable)
        }

    @Test
    fun `a missing and a broken vfox are told apart`() =
        runTest {
            val runner = fixtureRunner()
            val (_, controller) = vfox(runner)
            runner.missingExecutable = true
            controller.apply(AppSettings())
            assertEquals(VfoxAvailability.NOT_FOUND, controller.availability)
            assertNull(controller.executable)
            runner.missingExecutable = false
            runner.on("--version", ProcessOutput(0, "hello", ""))
            controller.check()
            assertEquals(VfoxAvailability.BROKEN, controller.availability)
            assertIs<VfoxException>(controller.error)
        }

    @Test
    fun `available versions are filtered by text, note and installation`() =
        runTest {
            val controller = SdkDetailController(VfoxCli(fixtureRunner()), backgroundScope, "nodejs")
            controller.showTab(SdkTab.AVAILABLE)
            testScheduler.runCurrent()
            val installed = setOf("24.11.1")
            assertEquals(872, controller.visibleVersions(installed).size)
            controller.filter = VersionFilter.WITH_NOTE
            assertTrue(controller.visibleVersions(installed).all { it.note == "LTS" })
            controller.filter = VersionFilter.ALL
            controller.query = "24.11."
            assertEquals(listOf("24.11.1", "24.11.0"), controller.visibleVersions(installed).map { it.version })
            controller.query = "lts"
            assertTrue(controller.visibleVersions(installed).size > 10)
            controller.query = ""
            controller.filter = VersionFilter.INSTALLED
            assertEquals(listOf("24.11.1"), controller.visibleVersions(installed).map { it.version })
        }

    @Test
    fun `the vfox configuration is saved and reloaded`() =
        runTest {
            val runner = fixtureRunner().on("config gitignore.enable false", ProcessOutput(0, "", ""))
            val controller = VfoxConfigController(VfoxCli(runner), backgroundScope)
            controller.entries.load().join()
            assertEquals("true", controller.valueFor("gitignore.enable"))
            assertNull(controller.valueFor("missing"))
            controller.set("gitignore.enable", "false")
            assertFalse(controller.isSaving("gitignore.enable"))
            assertEquals(2, runner.calls("config --list").size)
            runner.on("config cache.availableHookDuration x", fixtureOutput("config-set-invalid-duration", 1))
            assertFailsWith<VfoxException> { controller.set("cache.availableHookDuration", "x") }
            assertTrue(controller.entries.hasValue)
        }

    private fun TestScope.model(
        runner: FakeProcessRunner,
        environment: Map<String, String>,
    ) = AppModel(
        runner,
        MemorySettingsStore(),
        environment,
        backgroundScope,
        VfoxLocator(emptyMap()) { it == "/usr/bin/vfox" },
        io = StandardTestDispatcher(testScheduler),
    )

    @Test
    fun `the model probes vfox, loads the SDKs and reloads them after tasks`() =
        runTest {
            val runner = fixtureRunner()
            val model = model(runner, mapOf("HOME" to "/home/u"))
            model.start()
            testScheduler.runCurrent()
            assertEquals(VfoxAvailability.READY, model.vfox.availability)
            assertEquals(listOf("maven", "mvn2", "nodejs"), model.installed.value?.map { it.name })
            assertEquals("24.11.1", model.installedSdk("nodejs")?.current)
            assertNull(model.installedSdk("python"))
            val lists = runner.calls("list").size
            model.run(model.cli.use("nodejs", "22.21.0"))
            testScheduler.runCurrent()
            assertEquals(lists + 1, runner.calls("list").size)
            assertEquals(
                TaskStatus.SUCCEEDED,
                model.tasks.tasks
                    .first()
                    .status,
            )

            // A task that never ran changes nothing.
            val running = model.run(model.cli.updatePlugin("a"))
            val queued = model.run(model.cli.updatePlugin("b"))
            model.tasks.cancel(queued)
            testScheduler.runCurrent()
            assertEquals(TaskStatus.SUCCEEDED, running.status)
            assertEquals(lists + 2, runner.calls("list").size)
            model.dispose()
        }

    @Test
    fun `changing the executable or the environment applies them again`() =
        runTest {
            val runner = fixtureRunner()
            val model = model(runner, mapOf("HOME" to "/home/u"))
            model.start()
            testScheduler.runCurrent()
            model.registry.load()
            model.vfoxConfig.entries.load()
            testScheduler.runCurrent()
            model.updateSettings { it.copy(extraEnvironment = mapOf("X" to "1")) }
            testScheduler.runCurrent()
            assertEquals(2, runner.calls("available").size)
            assertEquals(2, runner.calls("config --list").size)
            assertEquals("1", runner.invocations.last().environment["X"])
            model.updateSettings { it.copy(vfoxExecutable = "/opt/vfox") }
            testScheduler.runCurrent()
            assertEquals("/opt/vfox", model.vfox.executable)
            assertEquals("/opt/vfox", runner.invocations.last().executable)
            // Other settings do not touch vfox.
            val count = runner.invocations.size
            model.updateSettings { it.copy(theme = ThemePreference.DARK) }
            testScheduler.runCurrent()
            assertEquals(count, runner.invocations.size)
        }

    @Test
    fun `a local lua plugin is handed to vfox as a copy that is deleted afterwards`() =
        runTest {
            val home = createTempDirectory("vfox-plus-home")
            val original = home.resolve("mine.lua").also { it.writeText("PLUGIN = {}") }
            val runner = fixtureRunner()
            val model = model(runner, mapOf("HOME" to home.toString()))
            model.start()
            testScheduler.runCurrent()
            val task = model.addPluginFromSource(original.toString(), "mine")
            testScheduler.runCurrent()
            val arguments = runner.calls("add").single().arguments
            val copy = arguments[arguments.indexOf("--source") + 1]
            // vfox renames the file into its temporary directory, which only
            // works on the same filesystem.
            assertTrue(copy.startsWith("$home/.vfox/tmp/vfox-plus-plugin-"), copy)
            assertEquals(listOf("--alias", "mine"), arguments.takeLast(2))
            testScheduler.runCurrent()
            assertEquals(TaskStatus.SUCCEEDED, task.status)
            assertTrue(original.exists())
            assertFalse(
                java.nio.file.Path
                    .of(copy)
                    .exists(),
            )

            model.addPluginFromSource("https://example.com/p.zip", null)
            testScheduler.runCurrent()
            assertEquals(listOf("add", "--source", "https://example.com/p.zip"), runner.calls("add").last().arguments)
            Files.walk(home).sorted(Comparator.reverseOrder()).forEach(Files::delete)
        }
}

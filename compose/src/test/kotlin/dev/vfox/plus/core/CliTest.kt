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

package dev.vfox.plus.core

import dev.vfox.plus.support.FakeProcessRunner
import dev.vfox.plus.support.fixtureOutput
import dev.vfox.plus.support.fixtureRunner
import kotlinx.coroutines.test.runTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class CliTest {
    private lateinit var runner: FakeProcessRunner
    private lateinit var cli: VfoxCli
    private val listener =
        object : ProcessListener {
            override fun stdout(chunk: ByteArray) {}

            override fun stderr(chunk: ByteArray) {}
        }

    @BeforeTest
    fun setUp() {
        runner = fixtureRunner()
        cli =
            VfoxCli(
                runner,
                VfoxLaunchConfig(
                    executable = "/usr/local/bin/vfox",
                    baseEnvironment =
                        mapOf(
                            "HOME" to "/home/user",
                            "PATH" to "/usr/bin",
                            "__VFOX_SHELL" to "bash",
                            "__VFOX_PID" to "42",
                            "__VFOX_CURTMPPATH" to "/home/user/.vfox/tmp/x",
                        ),
                    extraEnvironment = mapOf("VFOX_NODEJS_MIRROR" to "https://mirror", "PATH" to "/opt/bin"),
                ),
            )
    }

    @Test
    fun `the environment drops the shell session, disables colours and lets user variables win`() {
        val environment = cli.launchConfig.environment()
        SESSION_VARIABLES.forEach { assertFalse(it in environment, it) }
        assertEquals("1", environment["NO_COLOR"])
        assertEquals("/home/user", environment["HOME"])
        assertEquals("/opt/bin", environment["PATH"])
        assertEquals("https://mirror", environment["VFOX_NODEJS_MIRROR"])
        assertEquals("bash", cli.launchConfig.environment(hookShell = true)["__VFOX_SHELL"])
    }

    @Test
    fun `queries run in the neutral directory with the configured executable`() =
        runTest {
            assertEquals("1.0.13", cli.version())
            val call = runner.invocations.single()
            assertEquals("/usr/local/bin/vfox", call.executable)
            assertEquals("/", call.workingDirectory)
            assertEquals("1", call.environment["NO_COLOR"])
        }

    @Test
    fun `an executable that prints no version is rejected`() =
        runTest {
            runner.on("--version", ProcessOutput(0, "hello\n", ""))
            assertEquals("Unrecognised version output", assertFailsWith<VfoxException> { cli.version() }.message)
        }

    @Test
    fun `installed SDKs merge list and current`() =
        runTest {
            assertEquals(
                listOf(
                    InstalledSdk("maven", listOf("3.9.16", "3.8.9"), "3.9.16"),
                    InstalledSdk("mvn2", emptyList(), null),
                    InstalledSdk("nodejs", listOf("24.11.1", "22.21.0"), "24.11.1"),
                ),
                cli.installedSdks(),
            )
            runner.on("list", fixtureOutput("list-empty", 1)).on("current", fixtureOutput("current-empty"))
            assertEquals(emptyList(), cli.installedSdks())
        }

    @Test
    fun `a failing list is reported when plugins are installed`() =
        runTest {
            runner.on("list", ProcessOutput(1, "boom\n", ""))
            val error = assertFailsWith<VfoxException> { cli.installedSdks() }
            assertEquals("boom", error.message)
            assertEquals(listOf("list"), error.arguments)
            assertEquals(1, error.exitCode)
        }

    @Test
    fun `plugin information, versions, registry and install paths`() =
        runTest {
            assertEquals("0.5.1", cli.pluginInfo("nodejs").version)
            assertEquals(
                listOf("info", "nodejs", "--format"),
                runner.invocations
                    .last()
                    .arguments
                    .take(3),
            )
            assertFailsWith<VfoxException> { cli.pluginInfo("maven") }
            assertTrue(cli.searchVersions("maven").any { it.version == "3.9.16" })
            assertEquals(43, cli.availablePlugins().size)
            runner
                .on("info nodejs@24.11.1", fixtureOutput("info-version-path"))
                .on("info nodejs@1.0.0", fixtureOutput("info-version-not-installed"))
            assertEquals("/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1", cli.installPath("nodejs", "24.11.1"))
            assertNull(cli.installPath("nodejs", "1.0.0"))
        }

    @Test
    fun `the configuration is listed, set and reset`() =
        runTest {
            runner
                .on("config proxy.url http://proxy:8080", ProcessOutput(0, "", ""))
                .on("config --unset proxy.url", ProcessOutput(0, "", ""))
            assertEquals(10, cli.configEntries().size)
            cli.setConfig("proxy.url", "http://proxy:8080")
            cli.unsetConfig("proxy.url")
            assertEquals(
                listOf("config --list", "config proxy.url http://proxy:8080", "config --unset proxy.url"),
                runner.calls("config").map { it.arguments.joinToString(" ") },
            )
            runner.on("config cache.availableHookDuration bad", fixtureOutput("config-set-invalid-duration", 1))
            assertEquals(
                "time: invalid duration \"notaduration\"",
                assertFailsWith<VfoxException> { cli.setConfig("cache.availableHookDuration", "bad") }.message,
            )
            runner.on("config --unset proxy.url", ProcessOutput(3, "", ""))
            assertEquals("vfox exited with code 3", assertFailsWith<VfoxException> { cli.unsetConfig("proxy.url") }.message)
            assertFailsWith<IllegalArgumentException> { cli.setConfig("proxy..url", "x") }
        }

    @Test
    fun `a missing executable becomes a VfoxNotFoundException`() =
        runTest {
            runner.missingExecutable = true
            assertFailsWith<VfoxNotFoundException> { cli.version() }
            assertFailsWith<VfoxNotFoundException> { cli.start(cli.updateAllPlugins(), listener) }
        }

    @Test
    fun `commands`() {
        val install = cli.install("nodejs", "24.11.1")
        assertEquals(listOf("install", "--yes", "nodejs@24.11.1"), install.arguments)
        assertEquals("nodejs@24.11.1", install.subject)
        assertEquals("vfox install --yes nodejs@24.11.1", install.commandLine)
        assertNull(install.workingDirectory)

        val use = cli.use("nodejs", "24.11.1")
        assertEquals(listOf("use", "--global", "nodejs@24.11.1"), use.arguments)
        assertTrue(use.requiresHookEnvironment)
        val project = cli.use("java", "21", UseScope.PROJECT, "/work/app")
        assertEquals(listOf("use", "--project", "java@21"), project.arguments)
        assertEquals("/work/app", project.workingDirectory)
        assertFailsWith<IllegalArgumentException> { cli.use("java", "21", UseScope.PROJECT) }
        assertFailsWith<IllegalArgumentException> { cli.unuse("java", UseScope.PROJECT, " ") }

        assertEquals(listOf("unuse", "--global", "java"), cli.unuse("java").arguments)
        assertEquals("/p", cli.unuse("java", UseScope.PROJECT, "/p").workingDirectory)
        assertEquals(listOf("uninstall", "java@21"), cli.uninstall("java", "21").arguments)
        assertEquals(listOf("remove", "--yes", "java"), cli.removePlugin("java").arguments)
        assertEquals(listOf("update", "java"), cli.updatePlugin("java").arguments)
        assertEquals(listOf("update", "--all"), cli.updateAllPlugins().arguments)
        assertNull(cli.updateAllPlugins().subject)
    }

    @Test
    fun `plugins are added from the registry or from a source`() {
        assertEquals(listOf("add", "nodejs"), cli.addPlugin("nodejs").arguments)
        val custom = cli.addPlugin("", source = " https://x/p.zip ", alias = "mine")
        assertEquals(listOf("add", "--source", "https://x/p.zip", "--alias", "mine"), custom.arguments)
        assertEquals("mine", custom.subject)
        assertNull(cli.addPlugin("", source = "/p.lua").sdk)
        assertFailsWith<IllegalArgumentException> { cli.addPlugin("  ") }
    }

    @Test
    fun `names and versions that look like options are rejected`() {
        assertFailsWith<IllegalArgumentException> { cli.install("--all", "1") }
        assertFailsWith<IllegalArgumentException> { cli.install("java", "-1") }
        assertFailsWith<IllegalArgumentException> { cli.install("java", "1 2") }
        assertFailsWith<IllegalArgumentException> { cli.install("java", "a@b") }
        assertFailsWith<IllegalArgumentException> { cli.install("java", "") }
        assertTrue(isValidPluginName("my-sdk.2"))
        assertFalse(isValidPluginName("-x"))
    }

    @Test
    fun `start launches commands with the right environment`() {
        cli.start(cli.use("java", "21", UseScope.PROJECT, "/work/app"), listener)
        val call = runner.invocations.single()
        assertTrue(call.streamed)
        assertEquals("/work/app", call.workingDirectory)
        assertEquals("bash", call.environment["__VFOX_SHELL"])
    }

    @Test
    fun `the locator searches PATH first, then the usual places`() {
        val locator = VfoxLocator(mapOf("PATH" to "/opt/bin::/usr/bin", "HOME" to "/home/me")) { false }
        assertEquals(
            listOf(
                "/opt/bin",
                "/usr/bin",
                "/usr/local/bin",
                "/home/me/.local/bin",
                "/home/me/bin",
                "/home/me/.linuxbrew/bin",
                "/home/linuxbrew/.linuxbrew/bin",
                "/snap/bin",
            ),
            locator.searchDirectories,
        )
        assertEquals(
            "/home/me/.local/bin/vfox",
            VfoxLocator(mapOf("HOME" to "/home/me")) { it == "/home/me/.local/bin/vfox" || it == "/snap/bin/vfox" }.locate(),
        )
        assertNull(VfoxLocator(emptyMap()) { false }.locate())
    }
}

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

@file:OptIn(ExperimentalTestApi::class)

package dev.vfox.plus.e2e

import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.graphics.toAwtImage
import androidx.compose.ui.test.DesktopComposeUiTest
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.isPopup
import androidx.compose.ui.test.isSelectable
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextReplacement
import androidx.compose.ui.test.v2.runDesktopComposeUiTest
import dev.vfox.plus.core.AppSettings
import dev.vfox.plus.core.JsonFileSettingsStore
import dev.vfox.plus.core.JvmProcessRunner
import dev.vfox.plus.core.LanguagePreference
import dev.vfox.plus.core.settingsPath
import dev.vfox.plus.state.AppModel
import dev.vfox.plus.state.TaskStatus
import dev.vfox.plus.support.SHARED
import dev.vfox.plus.ui.Navigation
import dev.vfox.plus.ui.PlatformServices
import dev.vfox.plus.ui.VfoxPlusApp
import dev.vfox.plus.ui.taskTitle
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.TimeUnit
import javax.imageio.ImageIO
import kotlin.io.path.createTempDirectory

/** Which vfox the application talks to. */
enum class Vfox {
    /** shared/fixtures/fake-vfox.sh, which replays the captured output of vfox. */
    FAKE,

    /** The executable named by VFOX_PLUS_E2E_VFOX, with the offline plugin of shared/test-plugin. */
    REAL,
}

/** The real vfox of VFOX_PLUS_E2E_VFOX, if one is named. */
val realVfox: String? get() = System.getenv("VFOX_PLUS_E2E_VFOX")?.ifBlank { null }

/**
 * A throw-away home directory with a vfox for the application: nothing of the
 * machine's vfox installation, configuration or home directory is used. For
 * the real vfox, the local server of the offline test plugin runs as well.
 */
class Harness(
    val vfox: Vfox,
    settings: (AppSettings) -> AppSettings = { it },
    /** Variables for the fake vfox, such as VFOX_PLUS_FAKE_DELAY. */
    extraEnvironment: Map<String, String> = emptyMap(),
) : AutoCloseable {
    val root: Path = createTempDirectory("vfox-plus-e2e")
    val home: Path = Files.createDirectories(root.resolve("home"))

    /** The vfox executable that the settings name. */
    val executable: String

    /** The server of the offline test plugin, for the real vfox. */
    val server: TestServer?

    /** A copy of the offline test plugin to add from a local file. */
    val localPlugin: Path = root.resolve("my-plugin.lua")

    /** The environment of the application. */
    val environment: Map<String, String>

    init {
        val base =
            mapOf(
                "HOME" to home.toString(),
                "PATH" to (System.getenv("PATH") ?: "/usr/bin:/bin"),
                "LANG" to "C.UTF-8",
            ) + extraEnvironment
        when (vfox) {
            Vfox.FAKE -> {
                val fake = root.resolve("vfox")
                Files.copy(SHARED.resolve("fixtures/fake-vfox.sh"), fake)
                fake.toFile().setExecutable(true)
                executable = fake.toString()
                server = null
                // As if the application was started from a terminal in which vfox is activated.
                environment =
                    base +
                    mapOf(
                        "VFOX_PLUS_FIXTURES" to SHARED.resolve("fixtures/vfox-1.0.13").toString(),
                        "__VFOX_PID" to "1",
                        "__VFOX_SHELL" to "bash",
                    )
            }

            Vfox.REAL -> {
                executable = requireNotNull(realVfox) { "Set VFOX_PLUS_E2E_VFOX to the vfox executable" }
                server = TestServer(Files.createDirectories(root.resolve("server")))
                environment = base + ("VFOX_PLUS_TEST_SERVER" to server.url)
            }
        }
        Files.copy(SHARED.resolve("test-plugin/testsdk.lua"), localPlugin)
        runBlocking {
            JsonFileSettingsStore(settingsPath(environment))
                .save(settings(AppSettings(language = LanguagePreference.ENGLISH, vfoxExecutable = executable)))
        }
    }

    /** Runs vfox itself, as a user would in a terminal of the test home, and returns its standard output. */
    fun vfox(vararg arguments: String): String {
        val process =
            ProcessBuilder(listOf(executable) + arguments)
                .directory(File("/"))
                .redirectInput(ProcessBuilder.Redirect.from(File("/dev/null")))
                .redirectError(ProcessBuilder.Redirect.DISCARD)
        process.environment().apply {
            clear()
            putAll(environment - setOf("__VFOX_PID", "__VFOX_SHELL"))
            put("NO_COLOR", "1")
        }
        val started = process.start()
        val output = started.inputStream.readAllBytes().decodeToString()
        started.waitFor(60, TimeUnit.SECONDS)
        return output
    }

    override fun close() {
        server?.close()
        root.toFile().deleteRecursively()
    }
}

/** Desktop integration that records what the application asks for, and answers file choosers as told. */
class RecordingServices : PlatformServices {
    val opened = mutableListOf<String>()
    val copied = mutableListOf<String>()

    /** What the next file chooser returns. */
    var nextFile: String? = null

    /** What the next directory chooser returns. */
    var nextDirectory: String? = null

    override suspend fun accentColor(): Long? = null

    override suspend fun prefersDark(): Boolean = false

    override fun openUrl(url: String) {
        opened += url
    }

    override suspend fun pickDirectory(initial: String?): String? = nextDirectory

    override suspend fun pickFile(
        initial: String?,
        label: String?,
        extensions: List<String>,
    ): String? = nextFile

    override fun copyText(text: String) {
        copied += text
    }
}

/** Drives the application in a test, as a user would: by the texts and names on the screen. */
class AppDriver(
    val test: DesktopComposeUiTest,
    val model: AppModel,
    val navigation: Navigation,
    val services: RecordingServices,
    val harness: Harness,
) {
    /** Reports progress, like the end-to-end runners of the other implementations. */
    fun step(name: String) = println("e2e: $name")

    fun check(
        condition: Boolean,
        message: String,
    ) {
        if (!condition) throw AssertionError(message)
    }

    /** Waits until [condition] holds, while the application runs. */
    fun waitFor(
        description: String,
        timeoutMillis: Long = 30_000,
        condition: () -> Boolean,
    ) = test.waitUntil(description, timeoutMillis, condition)

    /** How many nodes match [matcher]. */
    fun count(matcher: SemanticsMatcher): Int = test.onAllNodes(matcher).fetchSemanticsNodes().size

    private fun exists(matcher: SemanticsMatcher): Boolean = count(matcher) > 0

    /** Waits until a node with [text] is shown. */
    fun waitForText(
        text: String,
        substring: Boolean = false,
        timeoutMillis: Long = 30_000,
    ) = waitFor("\"$text\" to be shown", timeoutMillis) { exists(hasText(text, substring)) }

    /** Waits until no node with [text] is shown. */
    fun waitForNoText(text: String) = waitFor("\"$text\" to disappear") { !exists(hasText(text)) }

    /** Waits until a node matching [matcher] is shown and clicks the first one. */
    fun click(
        description: String,
        matcher: SemanticsMatcher,
    ) {
        waitFor(description) { exists(matcher) }
        test.onAllNodes(matcher).onFirst().performClick()
    }

    private fun button(label: String) = (hasText(label) or hasContentDescription(label)) and hasClickAction()

    /** Clicks the button named [label]. */
    fun clickButton(label: String) = click("a button \"$label\"", button(label))

    /** Goes to the destination named [label] with the navigation rail or bar. */
    fun clickDestination(label: String) = click("the destination \"$label\"", button(label) and isSelectable())

    /** Clicks the button named [label] of the dialog that is shown. */
    fun clickInDialog(label: String) = click("a button \"$label\" of the dialog", button(label) and hasAnyAncestor(isDialog()))

    /** Clicks the item named [label] of the menu that is open. */
    fun clickInMenu(label: String) = click("the menu item \"$label\"", button(label) and hasAnyAncestor(isPopup()))

    /** Clicks the button named [label] inside the element tagged [tag], such as the row of a version. */
    fun clickButtonIn(
        tag: String,
        label: String,
    ) = click("a button \"$label\" in $tag", button(label) and hasAnyAncestor(hasTestTag(tag)))

    /** Clicks the element tagged [tag], such as an SDK of the list. */
    fun clickTag(tag: String) = click("the element $tag", hasTestTag(tag))

    /** Replaces the text of the field named [label] and, with [submit], presses Enter. */
    fun fill(
        label: String,
        text: String,
        submit: Boolean = false,
    ) {
        val matcher = hasSetTextAction() and (hasText(label) or hasContentDescription(label))
        waitFor("a text field \"$label\"") { exists(matcher) }
        val field = test.onAllNodes(matcher).onFirst()
        field.performTextReplacement(text)
        if (submit) field.performImeAction()
    }

    /** Waits until the task titled [title] has succeeded; fails as soon as it fails. */
    fun waitForTask(
        title: String,
        timeoutMillis: Long = 60_000,
    ) = waitFor("the task \"$title\" to finish", timeoutMillis) {
        val task = model.tasks.tasks.firstOrNull { taskTitle(it) == title }
        if (task?.status == TaskStatus.FAILED) throw AssertionError("$title failed: ${task.errorMessage}")
        task?.status == TaskStatus.SUCCEEDED
    }

    private fun taskTitle(task: dev.vfox.plus.state.VfoxTask): String =
        dev.vfox.plus.core.i18n.Localizations
            .forPreference(LanguagePreference.ENGLISH, emptyList())
            .taskTitle(task.command)

    /**
     * Saves what the window shows as `<directory>/<name>.png`, after 5 s of the
     * clock of the user interface: animations settle and snack bars go.
     */
    fun screenshot(
        directory: File,
        name: String,
    ) {
        directory.mkdirs()
        test.mainClock.advanceTimeBy(5_000)
        test.waitForIdle()
        ImageIO.write(test.captureToImage().toAwtImage(), "png", directory.resolve("$name.png"))
    }
}

/**
 * Runs the application of [harness] in a window of [width] × [height] and
 * [block] against it. The state of the application changes on the test
 * dispatcher of the user interface, while vfox runs as a real process.
 */
fun runApp(
    harness: Harness,
    width: Int = 1280,
    height: Int = 800,
    block: AppDriver.() -> Unit,
) = runDesktopComposeUiTest(width, height) {
    val services = RecordingServices()
    val navigation = Navigation()
    var created: AppModel? = null
    setContent {
        val scope = rememberCoroutineScope()
        val model =
            remember {
                AppModel(JvmProcessRunner(), JsonFileSettingsStore(settingsPath(harness.environment)), harness.environment, scope)
            }
        created = model
        LaunchedEffect(model) { model.start() }
        VfoxPlusApp(model, navigation, services)
    }
    val model = requireNotNull(created)
    try {
        AppDriver(this, model, navigation, services, harness).block()
    } finally {
        model.dispose()
    }
}

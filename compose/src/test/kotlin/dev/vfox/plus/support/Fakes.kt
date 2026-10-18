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

package dev.vfox.plus.support

import dev.vfox.plus.core.PLUGIN_INFO_TEMPLATE
import dev.vfox.plus.core.ProcessListener
import dev.vfox.plus.core.ProcessOptions
import dev.vfox.plus.core.ProcessOutput
import dev.vfox.plus.core.ProcessRunner
import dev.vfox.plus.core.ProcessStartException
import dev.vfox.plus.core.RunningProcess
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Deferred
import java.nio.file.Files
import java.nio.file.Path

/** The shared resources of the repository (set by the build). */
val SHARED: Path = Path.of(System.getProperty("vfox.plus.shared") ?: "../shared")

/** The captured vfox outputs that the tests use by default. */
const val FIXTURES_VERSION = "1.0.13"

private val captureEscape = Regex("""\\(?:x([0-9a-f]{2})|\\)""")

/** Reads a capture of shared/fixtures, in which control characters are `\xHH` and backslashes `\\`. */
fun readCapture(path: Path): String =
    captureEscape.replace(Files.readString(path)) { match ->
        val hex = match.groupValues[1]
        if (hex.isEmpty()) "\\" else hex.toInt(16).toChar().toString()
    }

/** The result of the fixture [name], with [exitCode]. */
fun fixtureOutput(
    name: String,
    exitCode: Int = 0,
): ProcessOutput {
    val base = SHARED.resolve("fixtures/vfox-$FIXTURES_VERSION")
    return ProcessOutput(exitCode, readCapture(base.resolve("$name.stdout")), readCapture(base.resolve("$name.stderr")))
}

/** One recorded invocation of a [FakeProcessRunner]. */
data class ProcessCall(
    val executable: String,
    val arguments: List<String>,
    val workingDirectory: String,
    val environment: Map<String, String>,
    val streamed: Boolean,
)

/** A [RunningProcess] whose output and exit the test drives. */
class FakeRunningProcess : RunningProcess {
    private val exit = CompletableDeferred<Int>()
    override val exitCode: Deferred<Int> = exit
    var killed = false
        private set
    private var listener: ProcessListener? = null

    fun attach(listener: ProcessListener) {
        this.listener = listener
    }

    fun writeStdout(text: String) {
        if (text.isNotEmpty()) listener?.stdout(text.encodeToByteArray())
    }

    fun writeStderr(text: String) {
        if (text.isNotEmpty()) listener?.stderr(text.encodeToByteArray())
    }

    fun exit(code: Int) {
        exit.complete(code)
    }

    override fun kill() {
        killed = true
        exit(143)
    }
}

/** Prints its output as soon as a listener is attached, then exits. */
class FakeRunningProcessWithOutput(
    private val output: ProcessOutput,
) : RunningProcess {
    private val process = FakeRunningProcess()
    override val exitCode: Deferred<Int> get() = process.exitCode

    override fun kill() = process.kill()

    fun attach(listener: ProcessListener) {
        process.attach(listener)
        process.writeStdout(output.stdout)
        process.writeStderr(output.stderr)
        process.exit(output.exitCode)
    }
}

/**
 * A scripted [ProcessRunner]: responses are matched by the joined argument
 * list, so tests read like the command lines they expect. Unmatched queries
 * fail with exit code 127.
 */
class FakeProcessRunner : ProcessRunner {
    val invocations = mutableListOf<ProcessCall>()

    /** When set, every invocation fails as if the executable did not exist. */
    var missingExecutable = false
    private val responses = mutableMapOf<String, () -> ProcessOutput>()
    private val streams = mutableMapOf<String, () -> FakeRunningProcess>()

    fun on(
        arguments: String,
        output: ProcessOutput,
    ): FakeProcessRunner = apply { responses[arguments] = { output } }

    fun onStart(
        arguments: String,
        create: () -> FakeRunningProcess,
    ): FakeProcessRunner = apply { streams[arguments] = create }

    /** The calls whose arguments start with [prefix]. */
    fun calls(prefix: String): List<ProcessCall> = invocations.filter { it.arguments.joinToString(" ").startsWith(prefix) }

    override suspend fun run(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
    ): ProcessOutput {
        record(executable, arguments, options, false)
        if (missingExecutable) throw ProcessStartException(executable, "No such file or directory")
        return responses[arguments.joinToString(" ")]?.invoke() ?: ProcessOutput(127, "", "unexpected command")
    }

    override fun start(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
        listener: ProcessListener,
    ): RunningProcess {
        record(executable, arguments, options, true)
        if (missingExecutable) throw ProcessStartException(executable, "No such file or directory")
        val key = arguments.joinToString(" ")
        streams[key]?.let { create ->
            return create().also { it.attach(listener) }
        }
        val output = responses[key]?.invoke() ?: ProcessOutput(0, "", "")
        return FakeRunningProcessWithOutput(output).also { it.attach(listener) }
    }

    private fun record(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
        streamed: Boolean,
    ) {
        invocations += ProcessCall(executable, arguments, options.workingDirectory, options.environment, streamed)
    }
}

/**
 * A runner that answers the read-only commands like the fixtures: maven 3.9.16
 * (current) and 3.8.9, mvn2 without versions and nodejs 24.11.1 (current) and
 * 22.21.0.
 */
fun fixtureRunner(): FakeProcessRunner =
    FakeProcessRunner()
        .on("--version", fixtureOutput("version"))
        .on("list", fixtureOutput("list"))
        .on("current", fixtureOutput("current"))
        .on("available", fixtureOutput("available"))
        .on("config --list", fixtureOutput("config-list"))
        .on("search maven", fixtureOutput("search-maven-installed"))
        .on("search nodejs", fixtureOutput("search-nodejs"))
        .on("info nodejs --format $PLUGIN_INFO_TEMPLATE", fixtureOutput("info"))

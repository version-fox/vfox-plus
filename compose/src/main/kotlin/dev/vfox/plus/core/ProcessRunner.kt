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

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Deferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.nio.file.Files
import java.nio.file.Path
import kotlin.concurrent.thread

/** Everything a finished process wrote, decoded as UTF-8, and how it exited. */
data class ProcessOutput(
    val exitCode: Int,
    val stdout: String,
    val stderr: String,
)

/** Where and how a child process runs. */
data class ProcessOptions(
    val workingDirectory: String,
    /**
     * The complete environment of the child: the environment of the application
     * is not inherited, so that callers can remove variables as well as add them.
     */
    val environment: Map<String, String>,
)

/** Receives the output of a started process while it runs. */
interface ProcessListener {
    /** A chunk of raw standard output. */
    fun stdout(chunk: ByteArray)

    /** A chunk of raw standard error. */
    fun stderr(chunk: ByteArray)
}

/** A child process whose output is consumed while it runs. */
interface RunningProcess {
    /** Completes with the exit code after every chunk of output was delivered. */
    val exitCode: Deferred<Int>

    /** Asks the process to terminate with `SIGTERM`. */
    fun kill()
}

/** An executable could not be started, for example because it is missing. */
class ProcessStartException(
    val executable: String,
    message: String,
) : Exception(message)

/**
 * Spawns child processes. The application never spawns processes directly,
 * which keeps the vfox client testable with a fake runner. Standard input is
 * closed immediately: an open stdin would make an unexpected prompt wait
 * forever.
 */
interface ProcessRunner {
    /** Runs [executable] to completion; throws [ProcessStartException] if it cannot start. */
    suspend fun run(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
    ): ProcessOutput

    /** Starts [executable]; throws [ProcessStartException] if it cannot start. */
    fun start(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
        listener: ProcessListener,
    ): RunningProcess
}

/** Runs processes on the JVM, as docs/vfox-cli-contract.md asks. */
class JvmProcessRunner : ProcessRunner {
    override suspend fun run(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
    ): ProcessOutput =
        withContext(Dispatchers.IO) {
            val process = launch(executable, arguments, options)
            var stderr = ByteArray(0)
            val errorReader = thread(name = "vfox-plus stderr") { stderr = process.errorStream.readAllBytes() }
            val stdout = process.inputStream.readAllBytes()
            errorReader.join()
            ProcessOutput(process.waitFor(), stdout.decodeToString(), stderr.decodeToString())
        }

    override fun start(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
        listener: ProcessListener,
    ): RunningProcess {
        val process = launch(executable, arguments, options)
        val exitCode = CompletableDeferred<Int>()
        val readers =
            listOf(
                forward(process.inputStream, listener::stdout),
                forward(process.errorStream, listener::stderr),
            )
        thread(name = "vfox-plus wait", isDaemon = true) {
            // The exit is only reported once both streams are drained.
            readers.forEach { it.join() }
            exitCode.complete(process.waitFor())
        }
        return object : RunningProcess {
            override val exitCode: Deferred<Int> = exitCode

            // Process.destroy sends SIGTERM on Unix, as the CLI contract asks.
            override fun kill() = process.destroy()
        }
    }

    private fun forward(
        stream: InputStream,
        deliver: (ByteArray) -> Unit,
    ): Thread =
        thread(name = "vfox-plus output", isDaemon = true) {
            val buffer = ByteArray(16 * 1024)
            try {
                while (true) {
                    val read = stream.read(buffer)
                    if (read < 0) break
                    if (read > 0) deliver(buffer.copyOf(read))
                }
            } catch (_: IOException) {
                // The process is gone.
            }
        }

    private fun launch(
        executable: String,
        arguments: List<String>,
        options: ProcessOptions,
    ): Process {
        val path = lookPath(executable, options.environment["PATH"].orEmpty())
        val builder =
            ProcessBuilder(listOf(path) + arguments)
                .directory(File(options.workingDirectory))
                .redirectInput(ProcessBuilder.Redirect.from(File("/dev/null")))
        builder.environment().apply {
            clear()
            putAll(options.environment)
        }
        return try {
            builder.start()
        } catch (error: IOException) {
            throw ProcessStartException(executable, error.message ?: "Cannot start $executable")
        }
    }

    /**
     * Finds [name] in the directories of [path], the PATH of the child, as a
     * shell would; the JVM would search the PATH of the application instead.
     */
    private fun lookPath(
        name: String,
        path: String,
    ): String {
        if (name.contains('/')) return name
        for (directory in path.split(':')) {
            if (directory.isEmpty()) continue
            val candidate = Path.of(directory, name)
            if (Files.isRegularFile(candidate) && Files.isExecutable(candidate)) return candidate.toString()
        }
        throw ProcessStartException(name, "$name: executable file not found in PATH")
    }
}

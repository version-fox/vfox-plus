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

import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import java.io.ByteArrayOutputStream
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class ProcessRunnerTest {
    private val runner = JvmProcessRunner()
    private val options = ProcessOptions("/", mapOf("ONLY" to "this", "PATH" to "/usr/bin:/bin"))

    private fun sh(
        script: String,
        environment: Map<String, String> = options.environment,
    ): ProcessOutput = runBlocking { runner.run("sh", listOf("-c", script), options.copy(environment = environment)) }

    @Test
    fun `runs with exactly the given environment and working directory`() {
        assertEquals(ProcessOutput(3, "this::/\n", "err\n"), sh("echo \"\$ONLY:\$HOME:\$(pwd)\"; echo err >&2; exit 3"))
    }

    @Test
    fun `closes standard input and replaces malformed output`() {
        assertEquals("done\n", sh("cat; echo done").stdout)
        assertEquals("a�b", sh("printf 'a\\377b'").stdout)
    }

    @Test
    fun `looks executables up in the PATH of the child`() {
        assertFailsWith<ProcessStartException> { sh("true", mapOf("PATH" to "/nonexistent")) }
        assertFailsWith<ProcessStartException> {
            runBlocking { runner.run("/nonexistent/vfox", emptyList(), options) }
        }
        assertFailsWith<ProcessStartException> {
            runner.start("/nonexistent/vfox", emptyList(), options, Collector())
        }
    }

    private class Collector : ProcessListener {
        val out = ByteArrayOutputStream()
        val err = ByteArrayOutputStream()

        override fun stdout(chunk: ByteArray) = synchronized(this) { out.writeBytes(chunk) }

        override fun stderr(chunk: ByteArray) = synchronized(this) { err.writeBytes(chunk) }
    }

    @Test
    fun `streams the output and exits after it`() =
        runBlocking {
            val collector = Collector()
            val process = runner.start("sh", listOf("-c", "echo out; echo err >&2; exit 4"), options, collector)
            assertEquals(4, withTimeout(10_000) { process.exitCode.await() })
            assertEquals("out\n", collector.out.toString())
            assertEquals("err\n", collector.err.toString())
        }

    @Test
    fun `kill sends SIGTERM and signals become 128 plus the signal`() =
        runBlocking {
            val ready = Files.createTempFile("vfox-plus", ".ready")
            Files.delete(ready)
            val process =
                runner.start(
                    "sh",
                    listOf("-c", "trap 'exit 42' TERM; touch '$ready'; while :; do sleep 0.05; done"),
                    options,
                    Collector(),
                )
            withTimeout(10_000) { while (!Files.exists(ready)) Thread.sleep(10) }
            process.kill()
            assertEquals(42, withTimeout(10_000) { process.exitCode.await() })
            Files.delete(ready)
            val killed = runner.start("sh", listOf("-c", "kill -KILL $$"), options, Collector())
            assertEquals(128 + 9, withTimeout(10_000) { killed.exitCode.await() })
            assertTrue(isExecutableFile("/bin/sh"))
        }
}

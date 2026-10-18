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

import dev.vfox.plus.core.ProcessOutput
import dev.vfox.plus.core.VfoxCli
import dev.vfox.plus.core.VfoxLaunchConfig
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.support.FakeProcessRunner
import dev.vfox.plus.support.FakeRunningProcess
import dev.vfox.plus.support.fixtureOutput
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertSame
import kotlin.test.assertTrue

class TaskQueueTest {
    private val runner = FakeProcessRunner()
    private val cli = VfoxCli(runner, VfoxLaunchConfig(executable = "vfox"))

    private fun TestScope.queue(finished: MutableList<VfoxTask>): TaskQueue =
        TaskQueue(cli, backgroundScope, maxLogLines = 3, maxFinishedTasks = 2).also { it.onFinished { task -> finished += task } }

    @Test
    fun `runs one command at a time, in order`() =
        runTest {
            val finished = mutableListOf<VfoxTask>()
            val queue = queue(finished)
            val first = FakeRunningProcess()
            val second = FakeRunningProcess()
            runner.onStart("install --yes nodejs@24") { first }.onStart("install --yes java@21") { second }
            val a = queue.enqueue(cli.install("nodejs", "24"))
            val b = queue.enqueue(cli.install("java", "21"))
            testScheduler.runCurrent()
            assertEquals(TaskStatus.RUNNING, a.status)
            assertEquals(TaskStatus.QUEUED, b.status)
            assertEquals(2, queue.activeCount)
            assertEquals(1, runner.invocations.size)
            first.exit(0)
            testScheduler.runCurrent()
            assertEquals(TaskStatus.SUCCEEDED, a.status)
            assertEquals(TaskStatus.RUNNING, b.status)
            second.exit(0)
            testScheduler.runCurrent()
            assertEquals(listOf(a, b), finished)
            assertEquals(listOf(b, a), queue.tasks)
            assertEquals(0, queue.activeCount)
            assertNotNull(a.startedAt)
            assertNotNull(a.finishedAt)
        }

    @Test
    fun `an identical active command is not queued twice`() =
        runTest {
            val queue = queue(mutableListOf())
            runner.onStart("update --all") { FakeRunningProcess() }
            val a = queue.enqueue(cli.updateAllPlugins())
            assertSame(a, queue.enqueue(cli.updateAllPlugins()))
            assertEquals(1, queue.tasks.size)
        }

    @Test
    fun `progress bars become progress and the log is bounded`() =
        runTest {
            val queue = queue(mutableListOf())
            val process = FakeRunningProcess()
            runner.onStart("install --yes maven@3.9.16") { process }
            val task = queue.enqueue(cli.install("maven", "3.9.16"))
            testScheduler.runCurrent()
            process.writeStdout("Preinstalling maven@3.9.16...\n")
            process.writeStderr("\rDownloading...  42% [===>  ] (1 MB/s)\r")
            testScheduler.runCurrent()
            assertEquals(0.42, task.progress)
            assertEquals(listOf("Preinstalling maven@3.9.16..."), task.log)
            process.writeStdout("one\ntwo\nthree")
            process.exit(0)
            testScheduler.runCurrent()
            assertEquals(listOf("one", "two", "three"), task.log)
            assertEquals("three", task.lastLine)
            assertEquals(1, task.droppedLines)
            assertEquals(TaskStatus.SUCCEEDED, task.status)
            assertEquals(0, task.exitCode)
        }

    @Test
    fun `failures are summarised with the message of vfox`() =
        runTest {
            val finished = mutableListOf<VfoxTask>()
            val queue = queue(finished)
            runner.on("install --yes maven@3.9.16", fixtureOutput("install-failure", 1))
            val task = queue.enqueue(cli.install("maven", "3.9.16"))
            testScheduler.runCurrent()
            assertEquals(TaskStatus.FAILED, task.status)
            assertEquals(
                "mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: permission denied",
                task.errorMessage,
            )
            assertEquals(listOf(task), finished)
            runner.on("update java", ProcessOutput(4, "", ""))
            val silent = queue.enqueue(cli.updatePlugin("java"))
            testScheduler.runCurrent()
            assertEquals("vfox exited with code 4", silent.errorMessage)
            runner.missingExecutable = true
            val missing = queue.enqueue(cli.updatePlugin("go"))
            testScheduler.runCurrent()
            assertEquals(TaskStatus.FAILED, missing.status)
            assertTrue(missing.errorMessage!!.isNotEmpty())
        }

    @Test
    fun `cancelling drops queued tasks and kills running ones`() =
        runTest {
            val finished = mutableListOf<VfoxTask>()
            val queue = queue(finished)
            val process = FakeRunningProcess()
            runner.onStart("install --yes go@1") { process }
            val running = queue.enqueue(cli.install("go", "1"))
            val queued = queue.enqueue(cli.updatePlugin("java"))
            testScheduler.runCurrent()
            queue.cancel(queued)
            assertEquals(TaskStatus.CANCELLED, queued.status)
            queue.cancel(running)
            testScheduler.runCurrent()
            assertTrue(process.killed)
            assertEquals(TaskStatus.CANCELLED, running.status)
            assertEquals(listOf(queued, running), finished)
            assertTrue(runner.calls("update java").isEmpty())
            // A finished task is not cancelled again.
            queue.cancel(running)
            assertEquals(2, finished.size)
        }

    @Test
    fun `a failed task is retried, finished tasks are cleared and the oldest forgotten`() =
        runTest {
            val queue = queue(mutableListOf())
            runner.on("update java", ProcessOutput(1, "java plugin not installed\n", ""))
            val failed = queue.enqueue(cli.updatePlugin("java"))
            testScheduler.runCurrent()
            assertEquals(TaskStatus.FAILED, failed.status)
            runner.on("update java", fixtureOutput("update"))
            val retried = queue.retry(failed)
            testScheduler.runCurrent()
            assertNotEquals(failed, retried)
            assertEquals(TaskStatus.SUCCEEDED, retried.status)
            assertTrue(queue.hasFinished)
            queue.clearFinished()
            assertEquals(emptyList(), queue.tasks)
            for (name in listOf("a", "b", "c", "d")) {
                queue.enqueue(cli.updatePlugin(name))
                testScheduler.runCurrent()
            }
            assertEquals(listOf("d", "c", "b"), queue.tasks.map { it.command.sdk })
        }

    @Test
    fun `the active task of a version is found`() =
        runTest {
            val queue = queue(mutableListOf())
            runner.onStart("install --yes go@1") { FakeRunningProcess() }
            val task = queue.enqueue(cli.install("go", "1"))
            assertSame(task, queue.activeTask(VfoxOperation.INSTALL, "go", "1"))
            assertNull(queue.activeTask(VfoxOperation.UNINSTALL, "go"))
            queue.dispose()
        }
}

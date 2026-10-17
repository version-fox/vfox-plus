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

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import dev.vfox.plus.core.ProcessListener
import dev.vfox.plus.core.RunningProcess
import dev.vfox.plus.core.TerminalLineSplitter
import dev.vfox.plus.core.VfoxCli
import dev.vfox.plus.core.VfoxCommand
import dev.vfox.plus.core.VfoxException
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.core.parseDownloadProgress
import dev.vfox.plus.core.summarizeError
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/** The life cycle of a [VfoxTask]. */
enum class TaskStatus {
    /** Waiting for earlier tasks to finish. */
    QUEUED,

    /** The vfox process runs. */
    RUNNING,

    /** vfox exited with code zero. */
    SUCCEEDED,

    /** vfox could not be started or exited with a non-zero code. */
    FAILED,

    /** The user cancelled the task. */
    CANCELLED,

    ;

    /** Whether the task has not finished yet. */
    val isActive: Boolean get() = this == QUEUED || this == RUNNING
}

/** One execution of a state-changing [VfoxCommand]. Its properties are snapshot state. */
class VfoxTask internal constructor(
    /** Sequential identifier, unique within a [TaskQueue]. */
    val id: Int,
    /** The command that is executed. */
    val command: VfoxCommand,
    /** When the task was queued, in milliseconds since the epoch. */
    val createdAt: Long,
    private val maxLogLines: Int,
) {
    var status: TaskStatus by mutableStateOf(TaskStatus.QUEUED)
        internal set

    /** Download progress between 0 and 1, if vfox reported any. */
    var progress: Double? by mutableStateOf(null)
        private set

    /** When the process started. */
    var startedAt: Long? by mutableStateOf(null)
        internal set

    /** When the task finished. */
    var finishedAt: Long? by mutableStateOf(null)
        internal set

    /** The exit code of vfox, if it ran to completion. */
    var exitCode: Int? by mutableStateOf(null)
        internal set

    /** A short description of the failure, if the task failed. */
    var errorMessage: String? by mutableStateOf(null)
        internal set

    /** How many old lines were dropped from [log]. */
    var droppedLines: Int by mutableIntStateOf(0)
        private set

    private val lines = mutableStateListOf<String>()
    private val stdoutLines = mutableListOf<String>()
    private val stderrLines = mutableListOf<String>()

    /** Output lines (standard output and error interleaved), without progress redraws; the most recent only. */
    val log: List<String> get() = lines

    /** The last line of the output. */
    val lastLine: String? get() = lines.lastOrNull()

    internal var process: RunningProcess? = null

    internal fun append(
        line: String,
        isError: Boolean,
    ) {
        val lineProgress = parseDownloadProgress(line)
        if (lineProgress != null) {
            progress = lineProgress
            return
        }
        (if (isError) stderrLines else stdoutLines) += line
        lines += line
        if (lines.size > maxLogLines) {
            lines.removeAt(0)
            droppedLines++
        }
    }

    internal fun summary(): String = summarizeError(stdoutLines.joinToString("\n"), stderrLines.joinToString("\n"))
}

/**
 * Runs state-changing vfox commands one after another. vfox does not lock its
 * data directory, so concurrent installs, uses and removals could corrupt
 * `.vfox.toml` or the symlinks; the queue runs at most one command at a time,
 * in the order they were queued.
 */
class TaskQueue(
    private val cli: VfoxCli,
    private val scope: CoroutineScope,
    /** How many log lines each task keeps. */
    val maxLogLines: Int = 2000,
    /** How many finished tasks are kept before the oldest are forgotten. */
    val maxFinishedTasks: Int = 50,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    private val all = mutableStateListOf<VfoxTask>()
    private val finishedListeners = mutableListOf<(VfoxTask) -> Unit>()
    private var nextId = 1
    private var running: VfoxTask? = null

    /** All known tasks, newest first. */
    val tasks: List<VfoxTask> get() = all.asReversed()

    /** The number of queued and running tasks. */
    val activeCount: Int get() = all.count { it.status.isActive }

    /** Whether any finished task can be cleared. */
    val hasFinished: Boolean get() = all.any { !it.status.isActive }

    /** Calls [listener] with every task once it has finished, failed or been cancelled. */
    fun onFinished(listener: (VfoxTask) -> Unit): () -> Unit {
        finishedListeners += listener
        return { finishedListeners -= listener }
    }

    /**
     * Queues [command] and returns its task. If an identical command is queued
     * or running already, its task is returned instead.
     */
    fun enqueue(command: VfoxCommand): VfoxTask {
        all.firstOrNull { it.status.isActive && sameCommand(it.command, command) }?.let { return it }
        val task = VfoxTask(nextId++, command, clock(), maxLogLines)
        all += task
        trimFinished()
        pump()
        return task
    }

    /** Queues the command of [task] again. */
    fun retry(task: VfoxTask): VfoxTask = enqueue(task.command)

    /** Cancels [task]: a queued task is dropped, a running process is killed. */
    fun cancel(task: VfoxTask) {
        when (task.status) {
            TaskStatus.QUEUED -> {
                finish(task, TaskStatus.CANCELLED)
            }

            TaskStatus.RUNNING -> {
                task.process?.kill()
                // The task is settled once the process has exited.
                task.errorMessage = null
                task.status = TaskStatus.CANCELLED
            }

            else -> {}
        }
    }

    /** Forgets every finished task. */
    fun clearFinished() {
        all.removeAll { !it.status.isActive }
    }

    /** The active task matching the filter, if any. */
    fun activeTask(
        operation: VfoxOperation? = null,
        sdk: String? = null,
        version: String? = null,
    ): VfoxTask? =
        all.firstOrNull {
            it.status.isActive &&
                (operation == null || it.command.operation == operation) &&
                (sdk == null || it.command.sdk == sdk) &&
                (version == null || it.command.version == version)
        }

    private fun pump() {
        if (running != null) return
        val next = all.firstOrNull { it.status == TaskStatus.QUEUED } ?: return
        running = next
        scope.launch { execute(next) }
    }

    private suspend fun execute(task: VfoxTask) {
        task.startedAt = clock()
        task.status = TaskStatus.RUNNING
        val stdout = TerminalLineSplitter()
        val stderr = TerminalLineSplitter()
        try {
            // The output arrives on threads of the process runner and is
            // handed to the dispatcher of the queue, in order.
            val process =
                cli.start(
                    task.command,
                    object : ProcessListener {
                        override fun stdout(chunk: ByteArray) {
                            val lines = synchronized(stdout) { stdout.push(chunk) }
                            if (lines.isNotEmpty()) scope.launch { lines.forEach { task.append(it, false) } }
                        }

                        override fun stderr(chunk: ByteArray) {
                            val lines = synchronized(stderr) { stderr.push(chunk) }
                            if (lines.isNotEmpty()) scope.launch { lines.forEach { task.append(it, true) } }
                        }
                    },
                )
            task.process = process
            // Cancelled while the process was being spawned.
            if (task.status == TaskStatus.CANCELLED) process.kill()
            val exitCode = process.exitCode.await()
            // Let the output that was handed over before the exit land first.
            scope
                .launch {
                    synchronized(stdout) { stdout.close() }.forEach { task.append(it, false) }
                    synchronized(stderr) { stderr.close() }.forEach { task.append(it, true) }
                    settle(task, exitCode)
                }.join()
        } catch (error: VfoxException) {
            task.errorMessage = error.message
            finish(task, TaskStatus.FAILED)
        } finally {
            task.process = null
            running = null
            pump()
        }
    }

    private fun settle(
        task: VfoxTask,
        exitCode: Int,
    ) {
        task.exitCode = exitCode
        when {
            task.status == TaskStatus.CANCELLED -> {
                finish(task, TaskStatus.CANCELLED)
            }

            exitCode == 0 -> {
                finish(task, TaskStatus.SUCCEEDED)
            }

            else -> {
                task.errorMessage = task.summary().ifEmpty { "vfox exited with code $exitCode" }
                finish(task, TaskStatus.FAILED)
            }
        }
    }

    private fun finish(
        task: VfoxTask,
        status: TaskStatus,
    ) {
        task.finishedAt = clock()
        task.status = status
        finishedListeners.toList().forEach { it(task) }
    }

    private fun trimFinished() {
        val finished = all.filter { !it.status.isActive }
        val excess = finished.size - maxFinishedTasks
        if (excess > 0) all.removeAll(finished.take(excess).toSet())
    }

    /** Kills the processes that still run, when the application quits. */
    fun dispose() {
        all.forEach { it.process?.kill() }
        finishedListeners.clear()
    }

    private fun sameCommand(
        a: VfoxCommand,
        b: VfoxCommand,
    ) = a.workingDirectory == b.workingDirectory && a.arguments == b.arguments
}

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
import dev.vfox.plus.core.InstalledSdk
import dev.vfox.plus.core.PreparedPluginSource
import dev.vfox.plus.core.ProcessRunner
import dev.vfox.plus.core.SettingsStore
import dev.vfox.plus.core.VfoxCli
import dev.vfox.plus.core.VfoxCommand
import dev.vfox.plus.core.VfoxLaunchConfig
import dev.vfox.plus.core.VfoxLocator
import dev.vfox.plus.core.preparePluginSource
import dev.vfox.plus.core.vfoxTemporaryDirectory
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * The application state: every controller, wired together. The model owns the
 * [VfoxCli] and keeps the derived data consistent: the installed SDKs are
 * reloaded whenever vfox becomes available and whenever a task that changes
 * them finishes.
 */
class AppModel(
    runner: ProcessRunner,
    settingsStore: SettingsStore,
    /** The environment of the application. */
    val environment: Map<String, String>,
    /** Where the state changes: the main thread of the application. */
    private val scope: CoroutineScope,
    locator: VfoxLocator = VfoxLocator(environment),
    /** Where files are read and written. */
    private val io: CoroutineDispatcher = Dispatchers.IO,
) {
    /** The vfox client that every controller shares. */
    val cli = VfoxCli(runner, VfoxLaunchConfig(baseEnvironment = environment))

    /** Application preferences. */
    val settings = SettingsController(settingsStore)

    /** Availability of the vfox executable. */
    val vfox = VfoxController(cli, locator, environment, io)

    /** Installed plugins and their versions. */
    val installed = AsyncResource(scope) { cli.installedSdks() }

    /** Plugins available in the registry. */
    val registry = AsyncResource(scope) { cli.availablePlugins() }

    /** The vfox configuration. */
    val vfoxConfig = VfoxConfigController(cli, scope)

    /** State-changing operations. */
    val tasks = TaskQueue(cli, scope)

    private val pluginCopies = mutableMapOf<VfoxTask, PreparedPluginSource>()

    init {
        tasks.onFinished(::onTaskFinished)
        settings.onChange { previous, current ->
            if (previous.vfoxExecutable != current.vfoxExecutable || previous.extraEnvironment != current.extraEnvironment) {
                scope.launch { applySettings() }
            }
        }
    }

    /** Loads the settings and probes vfox. */
    fun start(): Job =
        scope.launch {
            settings.load()
            applySettings()
        }

    /** Points vfox at the executable and environment of the settings and reloads what depends on it. */
    suspend fun applySettings() {
        vfox.apply(settings.settings)
        if (vfox.availability == VfoxAvailability.READY) {
            installed.load()
            if (registry.hasValue || registry.error != null) registry.load()
            if (vfoxConfig.entries.hasValue || vfoxConfig.entries.error != null) vfoxConfig.entries.load()
        }
    }

    /** Changes the settings and saves them. */
    fun updateSettings(change: (AppSettings) -> AppSettings): Job = scope.launch { settings.update(change) }

    /** The installed SDK called [name], if it is installed. */
    fun installedSdk(name: String): InstalledSdk? = installed.value?.firstOrNull { it.name == name }

    /** Creates the state of the detail view of [name]. */
    fun sdkDetail(name: String) = SdkDetailController(cli, scope, name)

    /** Queues [command] and returns its task. */
    fun run(command: VfoxCommand): VfoxTask = tasks.enqueue(command)

    /**
     * Adds a plugin from [source], a URL or a local `.zip` or `.lua` file,
     * optionally under [alias]. vfox moves a local `.lua` file away instead of
     * reading it, so such a file is copied into the temporary directory of vfox
     * and vfox gets the copy, which is deleted once the task has finished.
     */
    suspend fun addPluginFromSource(
        source: String,
        alias: String?,
    ): VfoxTask {
        val prepared =
            withContext(io) {
                preparePluginSource(source, vfoxTemporaryDirectory(cli.launchConfig.environment()))
            }
        val task = run(cli.addPlugin("", source = prepared.path, alias = alias))
        if (prepared.isCopy) pluginCopies[task] = prepared
        return task
    }

    private fun onTaskFinished(task: VfoxTask) {
        pluginCopies.remove(task)?.dispose()
        // A task that never ran changed nothing.
        if (task.status == TaskStatus.CANCELLED && task.startedAt == null) return
        installed.load()
    }

    /** Releases every resource when the application quits. */
    fun dispose() {
        tasks.dispose()
        pluginCopies.values.forEach { it.dispose() }
        pluginCopies.clear()
    }
}

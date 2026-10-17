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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import dev.vfox.plus.core.AppSettings
import dev.vfox.plus.core.AvailableVersion
import dev.vfox.plus.core.PluginInfo
import dev.vfox.plus.core.SettingsStore
import dev.vfox.plus.core.VFOX_EXECUTABLE_NAME
import dev.vfox.plus.core.VfoxCli
import dev.vfox.plus.core.VfoxException
import dev.vfox.plus.core.VfoxLocator
import dev.vfox.plus.core.VfoxNotFoundException
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/** Holds the [AppSettings] and persists every change. */
class SettingsController(
    private val store: SettingsStore,
) {
    /** The current settings. */
    var settings: AppSettings by mutableStateOf(AppSettings())
        private set

    private val listeners = mutableListOf<(previous: AppSettings, current: AppSettings) -> Unit>()

    /** Calls [listener] with the previous and the new settings after every change. */
    fun onChange(listener: (previous: AppSettings, current: AppSettings) -> Unit) {
        listeners += listener
    }

    /** Loads the persisted settings. */
    suspend fun load() {
        settings = store.load()
    }

    /** Applies [change] to the settings and saves the result. */
    suspend fun update(change: (AppSettings) -> AppSettings) {
        val previous = settings
        val updated = change(previous)
        if (updated == previous) return
        settings = updated
        listeners.toList().forEach { it(previous, updated) }
        store.save(updated)
    }
}

/** Whether vfox can be used. */
enum class VfoxAvailability {
    /** Not checked yet. */
    UNKNOWN,

    /** The executable is being probed. */
    CHECKING,

    /** vfox runs and reported its version. */
    READY,

    /** No vfox executable was found. */
    NOT_FOUND,

    /** An executable was found but did not behave like vfox. */
    BROKEN,
}

/** Locates vfox, configures the [VfoxCli] and checks that vfox works. */
class VfoxController(
    private val cli: VfoxCli,
    private val locator: VfoxLocator,
    private val environment: Map<String, String>,
    /** Where the file system is searched for vfox. */
    private val io: CoroutineDispatcher = Dispatchers.IO,
) {
    var availability: VfoxAvailability by mutableStateOf(VfoxAvailability.UNKNOWN)
        private set

    /** The executable in use, if one was found. */
    var executable: String? by mutableStateOf(null)
        private set

    /** Whether [executable] was chosen by the user rather than detected. */
    var isCustomExecutable: Boolean by mutableStateOf(false)
        private set

    /** The vfox version, once vfox is ready. */
    var version: String? by mutableStateOf(null)
        private set

    /** Why vfox is not available, if it is not. */
    var error: Throwable? by mutableStateOf(null)
        private set

    /** Points the CLI at the executable and the environment of [settings] and checks the result. */
    suspend fun apply(settings: AppSettings) {
        val custom = settings.vfoxExecutable
        isCustomExecutable = custom != null
        executable = custom ?: withContext(io) { locator.locate() }
        cli.launchConfig =
            cli.launchConfig.copy(
                executable = executable ?: VFOX_EXECUTABLE_NAME,
                baseEnvironment = environment,
                extraEnvironment = settings.extraEnvironment,
            )
        check()
    }

    /** Runs `vfox --version` to check that vfox works. */
    suspend fun check() {
        availability = VfoxAvailability.CHECKING
        try {
            version = cli.version()
            error = null
            availability = VfoxAvailability.READY
        } catch (failure: VfoxException) {
            version = null
            error = failure
            availability = if (failure is VfoxNotFoundException) VfoxAvailability.NOT_FOUND else VfoxAvailability.BROKEN
        }
    }
}

/** The vfox configuration (`vfox config`), editable key by key. */
class VfoxConfigController(
    private val cli: VfoxCli,
    scope: CoroutineScope,
) {
    /** The configuration entries. */
    val entries = AsyncResource(scope) { cli.configEntries() }

    private var saving: Set<String> by mutableStateOf(emptySet())

    /** The value of [key], or `null` if vfox does not know the key. */
    fun valueFor(key: String): String? = entries.value?.firstOrNull { it.key == key }?.value

    /** Whether [key] is being saved. */
    fun isSaving(key: String): Boolean = key in saving

    /** Sets [key] to [value] and reloads; throws [VfoxException] if vfox rejects the value. */
    suspend fun set(
        key: String,
        value: String,
    ) = save(key) { cli.setConfig(key, value) }

    /** Resets [key] to its default and reloads. */
    suspend fun reset(key: String) = save(key) { cli.unsetConfig(key) }

    private suspend fun save(
        key: String,
        action: suspend () -> Unit,
    ) {
        saving = saving + key
        try {
            action()
        } finally {
            saving = saving - key
            entries.load().join()
        }
    }
}

/** Which versions the available tab lists. */
enum class VersionFilter {
    ALL,

    /** Versions with a note such as `LTS`. */
    WITH_NOTE,

    /** Versions that are installed already. */
    INSTALLED,
}

/** The two tabs of the detail of an SDK. */
enum class SdkTab { INSTALLED, AVAILABLE }

/** State of the detail view of one installed plugin. */
class SdkDetailController(
    cli: VfoxCli,
    scope: CoroutineScope,
    /** The plugin name. */
    val name: String,
) {
    /** Plugin metadata. */
    val info: AsyncResource<PluginInfo> = AsyncResource(scope) { cli.pluginInfo(name) }

    /** Installable versions, loaded when they are first shown because the search can be slow. */
    val versions: AsyncResource<List<AvailableVersion>> = AsyncResource(scope) { cli.searchVersions(name) }

    /** The visible tab. */
    var tab: SdkTab by mutableStateOf(SdkTab.INSTALLED)
        private set

    /** The text the available versions are filtered by. */
    var query: String by mutableStateOf("")

    /** Which available versions are shown. */
    var filter: VersionFilter by mutableStateOf(VersionFilter.ALL)

    /** Shows [newTab], loading the available versions on first use. */
    fun showTab(newTab: SdkTab) {
        tab = newTab
        if (newTab == SdkTab.AVAILABLE) versions.ensureLoaded()
    }

    /**
     * The available versions matching [query] and [filter]. [installed] holds
     * the installed versions; it is passed in rather than taken from the
     * search output, so that it stays current after installations.
     */
    fun visibleVersions(installed: Set<String>): List<AvailableVersion> {
        val needle = query.trim().lowercase()
        return versions.value.orEmpty().filter { version ->
            val matchesFilter =
                when (filter) {
                    VersionFilter.ALL -> true
                    VersionFilter.WITH_NOTE -> version.note != null
                    VersionFilter.INSTALLED -> version.version in installed
                }
            matchesFilter &&
                (
                    needle.isEmpty() ||
                        version.version.lowercase().contains(needle) ||
                        version.note?.lowercase()?.contains(needle) == true
                )
        }
    }
}

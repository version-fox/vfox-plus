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

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption

/** Loads and saves [AppSettings]. */
interface SettingsStore {
    /** The stored settings, or the defaults if none are stored. */
    suspend fun load(): AppSettings

    /** Persists [settings]. */
    suspend fun save(settings: AppSettings)
}

/**
 * The settings file for [environment]: `$XDG_CONFIG_HOME/vfox-plus/settings.json`,
 * where `XDG_CONFIG_HOME` defaults to `~/.config` as the XDG Base Directory
 * Specification defines.
 */
fun settingsPath(environment: Map<String, String>): Path {
    val xdgConfigHome = environment["XDG_CONFIG_HOME"]
    val base =
        if (xdgConfigHome != null && xdgConfigHome.startsWith("/")) {
            Path.of(xdgConfigHome)
        } else {
            Path.of(environment["HOME"] ?: System.getProperty("java.io.tmpdir"), ".config")
        }
    return base.resolve("vfox-plus").resolve("settings.json")
}

@OptIn(ExperimentalSerializationApi::class)
private val prettyJson =
    Json {
        prettyPrint = true
        prettyPrintIndent = "  "
    }

/**
 * Stores settings as JSON in a file. Corrupted files are ignored rather than
 * fatal: the application then starts with the defaults.
 */
class JsonFileSettingsStore(
    val path: Path,
) : SettingsStore {
    override suspend fun load(): AppSettings =
        withContext(Dispatchers.IO) {
            runCatching { AppSettings.fromJson(Json.parseToJsonElement(Files.readString(path))) }.getOrDefault(AppSettings())
        }

    override suspend fun save(settings: AppSettings) =
        withContext(Dispatchers.IO) {
            // A temporary file first, so that a crash never leaves a truncated
            // settings file behind.
            Files.createDirectories(path.parent)
            val temporary = path.resolveSibling("${path.fileName}.tmp")
            Files.writeString(temporary, prettyJson.encodeToString(settings.toJson()) + "\n")
            Files.move(temporary, path, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE)
            Unit
        }
}

/** Keeps settings in memory, for tests. */
class MemorySettingsStore(
    private var settings: AppSettings = AppSettings(),
) : SettingsStore {
    /** How many times [save] was called. */
    var saveCount = 0
        private set

    override suspend fun load(): AppSettings = settings

    override suspend fun save(settings: AppSettings) {
        saveCount++
        this.settings = settings
    }
}

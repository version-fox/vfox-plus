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

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.longOrNull

// Preferences of the application itself, not of vfox. The JSON keys are part
// of the shared specification in docs/architecture.md#settings, so that every
// implementation of the GUI reads the same file.

/** Light or dark appearance; [SYSTEM] follows the desktop. */
enum class ThemePreference(
    val json: String,
) {
    SYSTEM("system"),
    LIGHT("light"),
    DARK("dark"),
}

/** The language of the user interface; [SYSTEM] follows the desktop. */
enum class LanguagePreference(
    val json: String,
    /** The language code of the catalogue, or `null` to follow the desktop. */
    val code: String?,
) {
    SYSTEM("system", null),
    ENGLISH("english", "en"),
    SIMPLIFIED_CHINESE("simplifiedChinese", "zh"),
}

/** The brand colour when the desktop accent colour is not used: the orange of the vfox logo. */
const val DEFAULT_SEED_COLOR: Long = 0xFFF28C1B

/** Preferences of the application, persisted as JSON. */
data class AppSettings(
    val theme: ThemePreference = ThemePreference.SYSTEM,
    /** Whether to derive the colour scheme from the desktop accent colour. */
    val useSystemColor: Boolean = true,
    /** ARGB seed of the colour scheme when [useSystemColor] is off. */
    val seedColor: Long = DEFAULT_SEED_COLOR,
    val language: LanguagePreference = LanguagePreference.SYSTEM,
    /** Path of the vfox executable chosen by the user, or `null` to detect it. */
    val vfoxExecutable: String? = null,
    /** Additional environment variables for vfox, such as mirror settings. */
    val extraEnvironment: Map<String, String> = emptyMap(),
    /** Whether the navigation rail shows labels next to the icons. */
    val railExpanded: Boolean = false,
    /** Keys this version does not know, kept so that settings of other implementations survive. */
    val unknown: Map<String, JsonElement> = emptyMap(),
) {
    /** The settings as JSON, with the keys in the order of the specification. */
    fun toJson(): JsonObject =
        JsonObject(
            buildMap {
                put("theme", JsonPrimitive(theme.json))
                put("useSystemColor", JsonPrimitive(useSystemColor))
                put("seedColor", JsonPrimitive(seedColor))
                put("language", JsonPrimitive(language.json))
                vfoxExecutable?.let { put("vfoxExecutable", JsonPrimitive(it)) }
                put("extraEnvironment", JsonObject(extraEnvironment.mapValues { JsonPrimitive(it.value) }))
                put("railExpanded", JsonPrimitive(railExpanded))
                putAll(unknown)
            },
        )

    companion object {
        private val knownKeys =
            setOf("theme", "useSystemColor", "seedColor", "language", "vfoxExecutable", "extraEnvironment", "railExpanded")

        /** Restores settings from [json], ignoring malformed values. */
        fun fromJson(json: JsonElement): AppSettings {
            val map = json as? JsonObject ?: return AppSettings()

            fun string(key: String) = (map[key] as? JsonPrimitive)?.takeIf { it.isString }?.contentOrNull
            val environment =
                (map["extraEnvironment"] as? JsonObject)
                    ?.mapNotNull { (key, value) -> (value as? JsonPrimitive)?.takeIf { it.isString }?.let { key to it.content } }
                    ?.toMap()
                    .orEmpty()
            return AppSettings(
                theme = ThemePreference.entries.firstOrNull { it.json == string("theme") } ?: ThemePreference.SYSTEM,
                useSystemColor = (map["useSystemColor"] as? JsonPrimitive)?.takeUnless { it.isString }?.booleanOrNull ?: true,
                seedColor = (map["seedColor"] as? JsonPrimitive)?.takeUnless { it.isString }?.longOrNull ?: DEFAULT_SEED_COLOR,
                language = LanguagePreference.entries.firstOrNull { it.json == string("language") } ?: LanguagePreference.SYSTEM,
                vfoxExecutable = string("vfoxExecutable")?.ifEmpty { null },
                extraEnvironment = environment,
                railExpanded = (map["railExpanded"] as? JsonPrimitive)?.takeUnless { it.isString }?.booleanOrNull ?: false,
                unknown = map.filterKeys { it !in knownKeys },
            )
        }
    }
}

/** The variables of the valid lines of [parseEnvironmentLines], and the 1-based numbers of the others. */
data class ParsedEnvironment(
    val variables: Map<String, String>,
    val invalidLines: List<Int>,
)

private val environmentKey = Regex("^[A-Za-z_][A-Za-z0-9_]*$")

/**
 * Parses `KEY=VALUE` lines into environment variables. Blank lines and lines
 * starting with `#` are ignored; keys must be valid POSIX variable names.
 */
fun parseEnvironmentLines(text: String): ParsedEnvironment {
    val variables = LinkedHashMap<String, String>()
    val invalid = mutableListOf<Int>()
    text.split('\n').forEachIndexed { index, rawLine ->
        val line = rawLine.trim()
        if (line.isEmpty() || line.startsWith("#")) return@forEachIndexed
        val equals = line.indexOf('=')
        val key = if (equals < 0) line else line.substring(0, equals).trim()
        if (equals < 0 || !environmentKey.matches(key)) {
            invalid += index + 1
        } else {
            variables[key] = line.substring(equals + 1).trim()
        }
    }
    return ParsedEnvironment(variables, invalid)
}

/** Formats environment variables as `KEY=VALUE` lines, sorted by key. */
fun formatEnvironmentLines(variables: Map<String, String>): String = variables.keys.sorted().joinToString("\n") { "$it=${variables[it]}" }

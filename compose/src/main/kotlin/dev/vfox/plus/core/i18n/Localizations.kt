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

package dev.vfox.plus.core.i18n

import dev.vfox.plus.core.LanguagePreference
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import java.util.concurrent.ConcurrentHashMap

/** The languages with a catalogue in shared/i18n. */
val SUPPORTED_LANGUAGES = listOf("en", "zh")

/**
 * Picks the catalogue for [preference]. With the system preference, the locales
 * of the desktop are matched against the catalogues, most preferred first;
 * every Chinese locale gets Simplified Chinese, and English is the fallback.
 */
fun resolveLanguage(
    preference: LanguagePreference,
    systemLocales: List<String>,
): String {
    preference.code?.let { return it }
    for (locale in systemLocales) {
        when (locale.lowercase().split('-', '_', '.', '@').first()) {
            "zh" -> return "zh"
            "en" -> return "en"
        }
    }
    return "en"
}

/**
 * The locales of the desktop, most preferred first, as GLib reads them from
 * [environment], then the default locale of the JVM.
 */
fun systemLocales(environment: Map<String, String>): List<String> =
    (
        environment["LANGUAGE"].orEmpty().split(':') +
            listOf(environment["LC_ALL"], environment["LC_MESSAGES"], environment["LANG"]).map { it.orEmpty() }
    ).filter { it.isNotEmpty() && it != "C" && it != "POSIX" && !it.startsWith("C.") } +
        java.util.Locale
            .getDefault()
            .toLanguageTag()

/** An ARB catalogue: messages and their `@`-prefixed metadata. */
class Catalogue(
    val entries: JsonObject,
) {
    /** The message keys, without metadata. */
    val keys: List<String> get() = entries.keys.filter { !it.startsWith("@") }

    /** The message [key], or `null`. */
    fun message(key: String): String? = (entries[key] as? JsonPrimitive)?.takeIf { it.isString }?.content

    companion object {
        /** Parses the ARB catalogue [text]. */
        fun parse(text: String): Catalogue = Catalogue(Json.parseToJsonElement(text).jsonObject)

        /** Loads the catalogue of [language] from the resources (generated from shared/i18n). */
        fun load(language: String): Catalogue {
            val stream =
                Catalogue::class.java.getResourceAsStream("/i18n/app_$language.arb")
                    ?: error("The catalogue of $language is missing from the resources")
            return parse(stream.use { it.readBytes().decodeToString() })
        }
    }
}

/** The messages of one language, with English as the fallback. */
class Localizations(
    /** The language code of [messages]. */
    val language: String,
    private val messages: Catalogue,
    private val fallback: Catalogue,
) {
    private val cache = ConcurrentHashMap<String, List<MessagePart>>()

    /** The BCP 47 tag of the language. */
    val languageTag: String get() = if (language == "zh") "zh-Hans" else "en"

    /**
     * The message [key] formatted with [arguments]. A key that the language
     * lacks falls back to English; a key that no catalogue has is returned as
     * it is, which the tests catch.
     */
    fun t(
        key: String,
        vararg arguments: Pair<String, Any>,
    ): String {
        val parts =
            cache.getOrPut(key) {
                val source = messages.message(key) ?: fallback.message(key)
                if (source == null) listOf(MessagePart.Text(key)) else parseMessage(source)
            }
        return formatMessage(parts, arguments.toMap(), language)
    }

    companion object {
        /** The translations for [preference] and the desktop locales [systemLocales]. */
        fun forPreference(
            preference: LanguagePreference,
            systemLocales: List<String>,
        ): Localizations {
            val language = resolveLanguage(preference, systemLocales)
            val english = Catalogue.load("en")
            return Localizations(language, if (language == "en") english else Catalogue.load(language), english)
        }
    }
}

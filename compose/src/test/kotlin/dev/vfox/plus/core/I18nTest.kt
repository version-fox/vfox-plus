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

import dev.vfox.plus.core.i18n.Catalogue
import dev.vfox.plus.core.i18n.Localizations
import dev.vfox.plus.core.i18n.MessageFormatException
import dev.vfox.plus.core.i18n.MessagePart
import dev.vfox.plus.core.i18n.formatMessage
import dev.vfox.plus.core.i18n.parseMessage
import dev.vfox.plus.core.i18n.resolveLanguage
import dev.vfox.plus.core.i18n.systemLocales
import dev.vfox.plus.support.SHARED
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class I18nTest {
    private val en = Catalogue.parse(Files.readString(SHARED.resolve("i18n/app_en.arb")))
    private val zh = Catalogue.parse(Files.readString(SHARED.resolve("i18n/app_zh.arb")))

    private fun placeholders(parts: List<MessagePart>): Set<String> =
        parts
            .flatMap { part ->
                when (part) {
                    is MessagePart.Text -> emptyList()
                    is MessagePart.Argument -> listOf(part.name)
                    is MessagePart.Plural -> listOf(part.name) + part.cases.values.flatMap { placeholders(it) }
                }
            }.toSet()

    @Test
    fun `placeholders and plurals with exact matches first`() {
        val parts = parseMessage("{count, plural, =0{none} =1{one {name}} other{{count} of {name}}}!")
        assertEquals("none!", formatMessage(parts, mapOf("count" to 0, "name" to "x"), "en"))
        assertEquals("one x!", formatMessage(parts, mapOf("count" to 1, "name" to "x"), "en"))
        assertEquals("7 of x!", formatMessage(parts, mapOf("count" to 7, "name" to "x"), "en"))
        val categories = parseMessage("{n, plural, one{single} other{many}}")
        assertEquals("single", formatMessage(categories, mapOf("n" to 1), "en"))
        assertEquals("many", formatMessage(categories, mapOf("n" to 1), "zh"))
        assertEquals("don't {x}", formatMessage(parseMessage("don't {x}"), emptyMap(), "en"))
    }

    @Test
    fun `malformed messages are rejected`() {
        assertFailsWith<MessageFormatException> { parseMessage("{a, select, x{y} other{z}}") }
        assertFailsWith<MessageFormatException> { parseMessage("{a, plural, =1{y}}") }
        assertFailsWith<MessageFormatException> { parseMessage("a } b") }
        assertFailsWith<MessageFormatException> { parseMessage("{1}") }
    }

    @Test
    fun `the catalogues have the same messages, descriptions and placeholders`() {
        assertEquals(
            listOf("app_en.arb", "app_zh.arb"),
            Files.list(SHARED.resolve("i18n")).use { s ->
                s.map { it.fileName.toString() }.sorted().toList()
            },
        )
        assertEquals(JsonPrimitive("en"), en.entries["@@locale"])
        assertEquals(JsonPrimitive("zh"), zh.entries["@@locale"])
        assertEquals(en.keys.sorted(), zh.keys.sorted())
        for (key in en.keys) {
            val metadata = en.entries["@$key"] as? JsonObject
            assertNotNull((metadata?.get("description") as? JsonPrimitive)?.content, key)
            val template = placeholders(parseMessage(en.message(key)!!))
            val translated = placeholders(parseMessage(zh.message(key)!!))
            assertTrue(translated.containsAll(template), "$key lacks placeholders in app_zh.arb")
            assertEquals((metadata["placeholders"] as? JsonObject)?.keys.orEmpty(), template, key)
        }
    }

    @Test
    fun `localizations format each language and fall back to English`() {
        val english = Localizations("en", en, en)
        val chinese = Localizations("zh", zh, en)
        assertEquals("2 versions installed", english.t("sdksVersionCount", "count" to 2))
        assertEquals("1 version installed", english.t("sdksVersionCount", "count" to 1))
        assertEquals("已安装 2 个版本", chinese.t("sdksVersionCount", "count" to 2))
        assertEquals("安装 maven@3.9.16", chinese.t("taskInstall", "subject" to "maven@3.9.16"))
        assertEquals("SDKs", Localizations("zh", Catalogue(JsonObject(mapOf("navHome" to JsonPrimitive("首页")))), en).t("navSdks"))
        assertEquals("noSuchKey", english.t("noSuchKey"))
        assertEquals("zh-Hans", chinese.languageTag)
        // The catalogues of the resources are those of shared/i18n.
        assertEquals(
            "已安装 2 个版本",
            Localizations.forPreference(LanguagePreference.SIMPLIFIED_CHINESE, emptyList()).t(
                "sdksVersionCount",
                "count" to 2,
            ),
        )
    }

    @Test
    fun `the system language`() {
        assertEquals("zh", resolveLanguage(LanguagePreference.SYSTEM, listOf("zh-TW")))
        assertEquals("zh", resolveLanguage(LanguagePreference.SYSTEM, listOf("zh_CN.UTF-8")))
        assertEquals("en", resolveLanguage(LanguagePreference.SYSTEM, listOf("de-DE", "en-US")))
        assertEquals("en", resolveLanguage(LanguagePreference.SYSTEM, listOf("fr")))
        assertEquals("en", resolveLanguage(LanguagePreference.ENGLISH, listOf("zh-CN")))
        assertEquals("zh", resolveLanguage(LanguagePreference.SIMPLIFIED_CHINESE, listOf("en")))
        assertEquals(
            listOf("zh_CN", "en_US", "de_DE.UTF-8"),
            systemLocales(mapOf("LANGUAGE" to "zh_CN:en_US", "LC_ALL" to "", "LANG" to "de_DE.UTF-8")).dropLast(1),
        )
        assertEquals(emptyList(), systemLocales(mapOf("LANG" to "C.UTF-8")).dropLast(1))
    }
}

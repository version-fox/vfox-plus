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

import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import java.nio.file.Files
import java.nio.file.Path
import kotlin.io.path.createTempDirectory
import kotlin.io.path.exists
import kotlin.io.path.readText
import kotlin.io.path.writeText
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class SettingsTest {
    @Test
    fun `settings round-trip through JSON with the keys of the specification in order`() {
        val settings =
            AppSettings(
                theme = ThemePreference.DARK,
                useSystemColor = false,
                seedColor = 0xFF123456,
                language = LanguagePreference.SIMPLIFIED_CHINESE,
                vfoxExecutable = "/opt/vfox",
                extraEnvironment = mapOf("A" to "1"),
                railExpanded = true,
            )
        assertEquals(settings, AppSettings.fromJson(Json.parseToJsonElement(settings.toJson().toString())))
        assertEquals(
            listOf("theme", "useSystemColor", "seedColor", "language", "extraEnvironment", "railExpanded"),
            AppSettings().toJson().keys.toList(),
        )
        assertEquals(DEFAULT_SEED_COLOR, (AppSettings().toJson()["seedColor"] as JsonPrimitive).content.toLong())
    }

    @Test
    fun `malformed values are ignored and unknown keys kept`() {
        val settings =
            AppSettings.fromJson(
                Json.parseToJsonElement(
                    """{"theme":"purple","useSystemColor":"yes","seedColor":"#fff","language":42,"vfoxExecutable":"",
                       "extraEnvironment":{"A":"1","B":2},"railExpanded":"true","future":true}""",
                ),
            )
        assertEquals(AppSettings(extraEnvironment = mapOf("A" to "1"), unknown = mapOf("future" to JsonPrimitive(true))), settings)
        assertEquals(JsonPrimitive(true), settings.toJson()["future"])
        assertEquals(AppSettings(), AppSettings.fromJson(Json.parseToJsonElement("[1, 2]")))
        assertEquals(emptyMap(), AppSettings.fromJson(Json.parseToJsonElement("""{"extraEnvironment":["x"]}""")).extraEnvironment)
    }

    @Test
    fun `environment lines`() {
        val parsed = parseEnvironmentLines("# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n")
        assertEquals(mapOf("A" to "1", "B" to "x=y"), parsed.variables)
        assertEquals(listOf(5, 6), parsed.invalidLines)
        assertEquals("A=1\nB=2", formatEnvironmentLines(mapOf("B" to "2", "A" to "1")))
    }

    @Test
    fun `the settings file follows XDG and is written atomically`() =
        runTest {
            assertEquals(Path.of("/x/vfox-plus/settings.json"), settingsPath(mapOf("XDG_CONFIG_HOME" to "/x", "HOME" to "/h")))
            assertEquals(
                Path.of("/h/.config/vfox-plus/settings.json"),
                settingsPath(mapOf("XDG_CONFIG_HOME" to "relative", "HOME" to "/h")),
            )
            val directory = createTempDirectory("vfox-plus-settings")
            val store = JsonFileSettingsStore(directory.resolve("nested/settings.json"))
            assertEquals(AppSettings(), store.load())
            val settings = AppSettings(theme = ThemePreference.LIGHT)
            store.save(settings)
            assertEquals(settings, store.load())
            assertTrue(store.path.readText().endsWith("}\n"))
            assertTrue(store.path.readText().contains("\n  \"theme\": \"light\""))
            assertFalse(directory.resolve("nested/settings.json.tmp").exists())
            store.path.writeText("{not json")
            assertEquals(AppSettings(), store.load())
            directory.toFile().deleteRecursively()
        }

    @Test
    fun `settings written by another implementation keep their unknown keys`() =
        runTest {
            val directory = createTempDirectory("vfox-plus-settings")
            val path = directory.resolve("settings.json")
            path.writeText("""{"theme": "dark", "windowSize": {"width": 1}}""")
            val store = JsonFileSettingsStore(path)
            store.save(store.load().copy(railExpanded = true))
            val saved = Json.parseToJsonElement(path.readText()).jsonObject
            assertEquals(JsonPrimitive("dark"), saved["theme"])
            assertTrue("windowSize" in saved)
            directory.toFile().deleteRecursively()
        }

    @Test
    fun `plugin sources`() {
        assertTrue(vfoxMovesPluginSource("/home/u/plugin.lua"))
        assertTrue(vfoxMovesPluginSource("/home/u/PLUGIN.LUA"))
        assertFalse(vfoxMovesPluginSource("/home/u/plugin.zip"))
        assertFalse(vfoxMovesPluginSource("https://example.com/plugin.lua"))
        assertTrue(isRemotePluginSource("http://127.0.0.1:8080/p.zip"))

        val home = createTempDirectory("vfox-plus-home")
        assertEquals(home.resolve(".vfox/tmp"), vfoxTemporaryDirectory(mapOf("HOME" to home.toString())))
        Files.createDirectories(home.resolve(".version-fox"))
        assertEquals(home.resolve(".version-fox/tmp"), vfoxTemporaryDirectory(mapOf("HOME" to home.toString())))

        for (source in listOf("https://example.com/p.lua", "$home/p.zip")) {
            val prepared = preparePluginSource(source, home.resolve(".vfox/tmp"))
            assertEquals(source, prepared.path)
            assertFalse(prepared.isCopy)
        }

        val original = home.resolve("mine.lua")
        original.writeText("PLUGIN = {}")
        val prepared = preparePluginSource(original.toString(), home.resolve(".vfox/tmp"))
        assertTrue(prepared.isCopy)
        assertTrue(prepared.path.startsWith("$home/.vfox/tmp/vfox-plus-plugin-"))
        assertTrue(prepared.path.endsWith("/mine.lua"))
        assertEquals("PLUGIN = {}", Path.of(prepared.path).readText())
        prepared.dispose()
        assertFalse(Path.of(prepared.path).exists())
        assertTrue(original.exists())
        assertEquals(0, Files.list(home.resolve(".vfox/tmp")).use { it.count() })

        val missing = preparePluginSource("$home/missing.lua", home.resolve("tmp"))
        assertEquals("$home/missing.lua", missing.path)
        assertFalse(missing.isCopy)
        assertEquals(0, Files.list(home.resolve("tmp")).use { it.count() })
        home.toFile().deleteRecursively()
    }
}

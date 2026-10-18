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

package dev.vfox.plus

import dev.vfox.plus.support.SHARED
import dev.vfox.plus.ui.APP_VERSION
import dev.vfox.plus.ui.AppIcons
import dev.vfox.plus.ui.Destination
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import java.nio.file.Files
import java.nio.file.Path
import kotlin.io.path.extension
import kotlin.io.path.readText
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Checks that the sources, the shared resources and the build agree. */
class ProjectTest {
    private val sources: List<Pair<Path, String>> =
        Files.walk(Path.of("src/main/kotlin")).use { paths ->
            paths.filter { it.extension == "kt" }.map { it to it.readText() }.toList()
        }

    private fun literals(pattern: Regex): Set<String> =
        sources.flatMap { (_, text) -> pattern.findAll(text).map { it.groupValues[1] } }.toSet()

    @Test
    fun `every message the user interface shows is in the catalogues`() {
        val catalogue = Json.parseToJsonElement(Files.readString(SHARED.resolve("i18n/app_en.arb"))).jsonObject.keys
        val used = literals(Regex("""\bt\(\s*"([A-Za-z0-9]+)"""")) + Destination.entries.map { it.label }
        assertTrue(used.size > 150, "the message keys are found: ${used.size}")
        assertEquals(emptySet(), used - catalogue, "keys that the catalogues lack")
    }

    @Test
    fun `every icon is in shared icons json and in the generated font`() {
        val manifest =
            Json
                .parseToJsonElement(Files.readString(SHARED.resolve("icons.json")))
                .jsonObject
                .getValue("icons")
                .jsonObject.keys
        val used =
            literals(Regex("""\b(?:Icon|TooltipIconButton|IconLabel|StatusMessage|Shortcut)\(\s*"([A-Za-z]+)"""")) +
                literals(Regex("""\bicon = "([A-Za-z]+)"""")) +
                // The third argument of a choice; the second is its label.
                literals(Regex("""\bChoice\(.*,\s*"([a-z][A-Za-z]*)"\)""")) +
                Destination.entries.map { it.icon }
        assertTrue(used.size > 25, "the icons are found: ${used.size}")
        assertEquals(emptySet(), used - manifest, "icons that shared/icons.json lacks")
        assertEquals(manifest, AppIcons.codepoints.keys, "the font has every icon of the manifest")
    }

    @Test
    fun `the version of the application is the one of the build`() {
        val build = Path.of("build.gradle.kts").readText()
        assertTrue(build.contains("version = \"$APP_VERSION\""), "build.gradle.kts has version $APP_VERSION")
    }
}

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

import dev.vfox.plus.support.SHARED
import dev.vfox.plus.support.readCapture
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.jupiter.api.DynamicTest
import org.junit.jupiter.api.TestFactory
import java.nio.file.Files
import kotlin.io.path.exists
import kotlin.io.path.name
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

// Runs the parsers against the shared vfox output fixtures. The expected
// results live in shared/fixtures/vfox-*/cases.json and are shared by every
// implementation; see shared/fixtures/README.md.

/** A JSON value as plain Kotlin values, with every number as a Double. */
private fun canonical(element: JsonElement): Any? =
    when (element) {
        is JsonNull -> null
        is JsonObject -> element.mapValues { canonical(it.value) }
        is JsonArray -> element.map { canonical(it) }
        is JsonPrimitive -> if (element.isString) element.content else element.booleanOrNull ?: element.doubleOrNull
    }

private fun indexed(items: List<Map<String, Any?>>): Map<String, Any?> =
    mapOf("count" to items.size.toDouble(), "items" to items.withIndex().associate { (index, item) -> index.toString() to item })

private fun version(version: AvailableVersion): Map<String, Any?> =
    mapOf(
        "version" to version.version,
        "note" to version.note,
        "additions" to version.additions.map { mapOf("name" to it.name, "version" to it.version) },
        "installed" to version.installed,
    )

/** Turns a parser result into the shape of cases.json. */
private val checks: Map<String, (String, String) -> Any?> =
    mapOf(
        "version" to { stdout, _ -> parseVfoxVersion(stdout) },
        "current" to { stdout, _ -> parseCurrent(stdout) },
        "list" to { stdout, _ -> parseInstalledTree(stdout) },
        "search" to { stdout, _ -> indexed(parseSearch(stdout).map(::version)) },
        "available" to { stdout, _ ->
            indexed(parseAvailablePlugins(stdout).map { mapOf("name" to it.name, "homepage" to it.homepage, "official" to it.official) })
        },
        "pluginInfo" to { stdout, _ ->
            parsePluginInfo(stdout)?.let {
                mapOf(
                    "name" to it.name,
                    "version" to it.version,
                    "homepage" to it.homepage,
                    "installPath" to it.installPath,
                    "description" to it.description,
                )
            }
        },
        "installPath" to { stdout, _ -> parseInstallPath(stdout) },
        "config" to { stdout, _ -> parseConfigList(stdout).map { mapOf("key" to it.key, "value" to it.value) } },
        "finalProgress" to { _, stderr ->
            splitTerminalLines(stripAnsi(stderr)).fold(null as Double?) { last, line -> parseDownloadProgress(line) ?: last }
        },
        "error" to { stdout, stderr -> summarizeError(stdout, stderr) },
    )

/** For search and available, only the listed items are compared. */
@Suppress("UNCHECKED_CAST")
private fun select(
    check: String,
    actual: Any?,
    expected: Any?,
): Any? {
    if (check != "search" && check != "available") return actual
    val actualMap = actual as Map<String, Any?>
    val actualItems = actualMap["items"] as Map<String, Any?>
    val expectedItems = (expected as Map<String, Any?>)["items"] as Map<String, Any?>
    return mapOf("count" to actualMap["count"], "items" to expectedItems.keys.associateWith { actualItems[it] })
}

class OutputContractTest {
    private val fixtures = SHARED.resolve("fixtures")
    private val versions =
        Files.list(fixtures).use { entries ->
            entries
                .filter { it.resolve("cases.json").exists() }
                .map { it.name }
                .sorted()
                .toList()
        }

    @Test
    fun `finds the fixtures`() {
        assertTrue(versions.isNotEmpty())
    }

    @TestFactory
    fun `every case of every captured vfox version`(): List<DynamicTest> =
        versions.flatMap { version ->
            val directory = fixtures.resolve(version)
            val manifest = Json.parseToJsonElement(Files.readString(directory.resolve("cases.json"))).jsonObject
            manifest.getValue("cases").jsonArray.flatMap { case ->
                val name =
                    case.jsonObject
                        .getValue("name")
                        .jsonPrimitive.content
                val stdout = readCapture(directory.resolve("$name.stdout"))
                val stderr = readCapture(directory.resolve("$name.stderr"))
                case.jsonObject.getValue("expect").jsonObject.map { (check, expected) ->
                    DynamicTest.dynamicTest("$version $name: $check") {
                        val parse = checks[check] ?: error("Unknown parser \"$check\" in cases.json")
                        val expectedValue = canonical(expected)
                        assertEquals(expectedValue, canonicalize(select(check, parse(stdout, stderr), expectedValue)))
                    }
                }
            }
        }

    /** Numbers as Doubles, maps and lists recursively, as [canonical] does for JSON. */
    private fun canonicalize(value: Any?): Any? =
        when (value) {
            is Map<*, *> -> value.entries.associate { it.key.toString() to canonicalize(it.value) }
            is List<*> -> value.map { canonicalize(it) }
            is Number -> value.toDouble()
            else -> value
        }
}

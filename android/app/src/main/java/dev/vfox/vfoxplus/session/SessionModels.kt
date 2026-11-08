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

package dev.vfox.vfoxplus.session

import org.json.JSONArray
import org.json.JSONObject

// The JSON of the paired server, parsed by hand with org.json (in the
// platform, no dependency): the shapes of mygo/pair/webapi.go. Only what the
// native screens show is carried; the web page keeps the rest.
object SessionModels {
    data class Vfox(val availability: String, val version: String?, val error: String?)
    data class InstalledSdk(val name: String, val versions: List<String>, val current: String?, val hasCurrent: Boolean)
    data class AvailablePlugin(val name: String, val homepage: String, val official: Boolean)
    data class ConfigEntry(val key: String, val value: String)
    data class Inventory(
        val vfox: Vfox,
        val installed: List<InstalledSdk>,
        val installedLoading: Boolean,
        val registry: List<AvailablePlugin>,
        val registryLoading: Boolean,
        val config: List<ConfigEntry>,
        val complete: Boolean,
    )

    data class PluginInfo(
        val name: String,
        val version: String,
        val homepage: String,
        val installPath: String,
        val description: String,
    )

    data class VersionAddition(val name: String, val version: String)
    data class VersionEntry(
        val version: String,
        val note: String,
        val hasNote: Boolean,
        val additions: List<VersionAddition>,
        val installed: Boolean,
    )

    data class ChoiceName(val en: String, val zh: String)
    data class SearchChoice(
        val value: String,
        val name: ChoiceName,
        val maker: ChoiceName,
        val aliases: List<String>,
        val popular: Boolean,
    )

    data class SearchSpec(
        val labelEn: String,
        val labelZh: String,
        val default: String,
        val current: String,
        val choices: List<SearchChoice>,
    )

    data class Shorthand(val name: String, val meaningEn: String, val meaningZh: String)
    data class VariablePreset(val value: String, val name: String)
    data class Variable(
        val key: String,
        val label: String,
        val help: String,
        val kind: String,
        val presets: List<VariablePreset>,
        val example: String,
        val advanced: Boolean,
        val value: String,
    )

    data class Profile(
        val plugin: String,
        val naming: String,
        val search: SearchSpec?,
        val shorthands: List<Shorthand>,
        val variables: List<Variable>,
        val legacyFiles: List<String>,
        val legacyStrategy: Boolean,
        val guideEn: String,
        val guideZh: String,
        val docs: List<Pair<String, String>>,
    )

    data class Task(
        val id: String,
        val operation: String,
        val sdk: String?,
        val version: String?,
        val commandLine: String,
        val status: String,
        val progress: String?,
        val errorMessage: String?,
        val lastLine: String?,
        val log: List<String>,
        val droppedLines: Int,
        val hasFullLog: Boolean,
    )

    data class Settings(
        val vfoxExecutable: String,
        val extraEnvironment: String,
        val language: String,
        val updateMirror: String,
    )

    data class Web(
        val remote: Boolean,
        val port: Int,
        val lightweight: Boolean,
        val listening: Boolean,
        val addresses: List<String>,
        val fingerprint: String,
        val selfSigned: Boolean,
        val error: String,
    )

    private fun JSONObject.str(key: String) = if (isNull(key)) "" else optString(key, "")
    private fun JSONObject.list(key: String): List<JSONObject> {
        val array = optJSONArray(key) ?: JSONArray()
        return (0 until array.length()).mapNotNull { array.optJSONObject(it) }
    }

    fun inventory(root: JSONObject): Inventory {
        val vfox = root.optJSONObject("vfox") ?: JSONObject()
        val installed = root.optJSONObject("installed") ?: JSONObject()
        val registry = root.optJSONObject("registry") ?: JSONObject()
        val config = root.optJSONObject("config") ?: JSONObject()
        return Inventory(
            vfox = Vfox(vfox.str("availability"), vfox.optString("version", null), vfox.optString("error", null)),
            installed = installed.list("value").map {
                InstalledSdk(
                    it.str("Name"),
                    (0 until (it.optJSONArray("Versions")?.length() ?: 0)).map { i ->
                        it.optJSONArray("Versions")?.optString(i).orEmpty()
                    },
                    it.optString("Current", null)?.ifEmpty { null },
                    it.optBoolean("HasCurrent", false),
                )
            },
            installedLoading = installed.optBoolean("loading", false),
            registry = registry.list("value").map {
                AvailablePlugin(it.str("Name"), it.str("Homepage"), it.optBoolean("Official", false))
            },
            registryLoading = registry.optBoolean("loading", false),
            config = config.list("value").map { ConfigEntry(it.str("Key"), it.str("Value")) },
            complete = root.optBoolean("complete", false),
        )
    }

    fun pluginInfo(root: JSONObject) = PluginInfo(
        root.str("Name"), root.str("Version"), root.str("Homepage"),
        root.str("InstallPath"), root.str("Description"),
    )

    fun versions(root: JSONArray): List<VersionEntry> {
        return (0 until root.length()).mapNotNull { root.optJSONObject(it) }.map {
            VersionEntry(
                it.str("Version"), it.str("Note"), it.optBoolean("HasNote", false),
                it.list("Additions").map { a -> VersionAddition(a.str("Name"), a.str("Version")) },
                it.optBoolean("Installed", false),
            )
        }
    }

    private fun choiceName(root: JSONObject?) = ChoiceName(root?.str("en").orEmpty(), root?.str("zh").orEmpty())

    fun profile(root: JSONObject): Profile {
        val search = root.optJSONObject("search")
        return Profile(
            plugin = root.str("plugin"),
            naming = root.str("naming"),
            search = search?.let {
                SearchSpec(
                    it.optJSONObject("label")?.str("en").orEmpty(),
                    it.optJSONObject("label")?.str("zh").orEmpty(),
                    it.str("default"), it.str("current"),
                    it.list("choices").map { c ->
                        SearchChoice(
                            c.str("value"), choiceName(c.optJSONObject("name")),
                            choiceName(c.optJSONObject("maker")),
                            (0 until (c.optJSONArray("aliases")?.length() ?: 0)).map { i ->
                                c.optJSONArray("aliases")?.optString(i).orEmpty()
                            },
                            c.optBoolean("popular", false),
                        )
                    },
                )
            },
            shorthands = root.list("shorthands").map {
                Shorthand(it.str("name"), it.optJSONObject("meaning")?.str("en").orEmpty(), it.optJSONObject("meaning")?.str("zh").orEmpty())
            },
            variables = root.list("variables").map {
                Variable(
                    it.str("key"), it.str("label"), it.str("help"), it.str("kind"),
                    it.list("presets").map { p -> VariablePreset(p.str("value"), p.str("name")) },
                    it.str("example"), it.optBoolean("advanced", false), it.str("value"),
                )
            },
            legacyFiles = (0 until (root.optJSONArray("legacyFiles")?.length() ?: 0)).map { i ->
                root.optJSONArray("legacyFiles")?.optString(i).orEmpty()
            },
            legacyStrategy = root.optBoolean("legacyStrategy", false),
            guideEn = root.optJSONObject("guide")?.str("en").orEmpty(),
            guideZh = root.optJSONObject("guide")?.str("zh").orEmpty(),
            docs = root.list("docs").map { d -> d.str("title") to d.str("url") },
        )
    }

    fun task(root: JSONObject) = Task(
        root.str("id"), root.str("operation"),
        root.optString("sdk", null)?.ifEmpty { null }, root.optString("version", null)?.ifEmpty { null },
        root.str("commandLine"), root.str("status"),
        root.optString("progress", null)?.ifEmpty { null },
        root.optString("errorMessage", null)?.ifEmpty { null },
        root.optString("lastLine", null)?.ifEmpty { null },
        (0 until (root.optJSONArray("log")?.length() ?: 0)).map { i -> root.optJSONArray("log")?.optString(i).orEmpty() },
        root.optInt("droppedLines", 0), root.optBoolean("hasFullLog", false),
    )

    fun tasks(root: JSONArray): List<Task> =
        (0 until root.length()).mapNotNull { root.optJSONObject(it) }.map { task(it) }

    fun settings(root: JSONObject) = Settings(
        root.str("vfoxExecutable"), root.str("extraEnvironment"),
        root.str("language"), root.str("updateMirror"),
    )

    fun web(root: JSONObject) = Web(
        root.optBoolean("remote", false), root.optInt("port", 8369),
        root.optBoolean("lightweight", false), root.optBoolean("listening", false),
        (0 until (root.optJSONArray("addresses")?.length() ?: 0)).map { i ->
            root.optJSONArray("addresses")?.optString(i).orEmpty()
        },
        root.str("fingerprint"), root.optBoolean("selfSigned", false), root.str("error"),
    )
}

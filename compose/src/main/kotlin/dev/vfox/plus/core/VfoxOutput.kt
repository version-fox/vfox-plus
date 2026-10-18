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

// Parsers for the human-oriented output of the vfox CLI. vfox has no
// machine-readable output mode for most commands, so every format handled here
// is documented in docs/vfox-cli-contract.md and covered by the captured
// outputs in shared/fixtures. The parsers are pure functions: they never throw
// on unexpected lines, which they skip.

/** The ASCII unit separator between the fields of `vfox info --format`. */
const val FIELD_SEPARATOR = "\u001f"

/**
 * The Go template passed to `vfox info <name> --format`. The fields are joined
 * with [FIELD_SEPARATOR], which cannot appear in any of them, so descriptions
 * may contain new lines and any punctuation.
 */
const val PLUGIN_INFO_TEMPLATE =
    "{{.Name}}{{\"\\x1f\"}}{{.Version}}{{\"\\x1f\"}}{{.Homepage}}" +
        "{{\"\\x1f\"}}{{.InstallPath}}{{\"\\x1f\"}}{{.Description}}"

private fun lines(output: String): List<String> = splitTerminalLines(stripAnsi(output))

private val versionPattern = Regex("""(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)""")

/** Parses `vfox --version`, for example `vfox version 1.0.12`, or returns `null`. */
fun parseVfoxVersion(output: String): String? = versionPattern.find(stripAnsi(output))?.groupValues?.get(1)

private fun stripVPrefix(version: String): String = if (version.startsWith("v") && version.length > 1) version.substring(1) else version

/**
 * Parses `vfox current` without arguments: lines `<name> -> v<version>`, or
 * `<name> -> N/A` when no version is in effect. The `v` prefix that vfox adds
 * is removed again. The map keeps the order of the output.
 */
fun parseCurrent(output: String): Map<String, String?> {
    val result = LinkedHashMap<String, String?>()
    for (line in lines(output)) {
        val arrow = line.indexOf(" -> ")
        if (arrow <= 0) continue
        val name = line.substring(0, arrow).trim()
        val value = line.substring(arrow + 4).trim()
        if (name.isEmpty() || name.contains(' ')) continue
        result[name] = if (value.isEmpty() || value == "N/A") null else stripVPrefix(value)
    }
    return result
}

private val treeItem = Regex("""^([│ ]*)[├└]─[─┬](.+)$""")

/**
 * Parses the tree printed by `vfox list` without arguments:
 *
 * ```text
 * All installed sdk versions
 * ├─┬java
 * │ ├──25.4.4.1-graalce
 * │ └──25.3.4.1-graalce
 * └──maven
 * ```
 *
 * A plugin without installed versions is drawn as a leaf. The map and the
 * version lists keep the order of the output.
 */
fun parseInstalledTree(output: String): Map<String, List<String>> {
    val result = LinkedHashMap<String, MutableList<String>>()
    var currentSdk: String? = null
    for (line in lines(output)) {
        val match = treeItem.find(line) ?: continue
        // The tree characters are single UTF-16 code units.
        val level = match.groupValues[1].length / 2
        val text = match.groupValues[2].trim()
        if (text.isEmpty()) continue
        if (level == 0) {
            currentSdk = text
            result.getOrPut(text) { mutableListOf() }
        } else if (currentSdk != null) {
            result.getValue(currentSdk) += text
        }
    }
    return result
}

/**
 * Combines the outputs of `vfox list` and `vfox current` into SDKs. Plugins
 * that only appear in one of them are still included, so that the result
 * degrades gracefully if one command fails.
 */
fun mergeInstalledSdks(
    tree: Map<String, List<String>>,
    current: Map<String, String?>,
): List<InstalledSdk> =
    (tree.keys + current.keys).map { name ->
        InstalledSdk(name = name, versions = tree[name].orEmpty(), current = current[name])
    }

/**
 * Parses `vfox search <name>` in a non-interactive terminal:
 *
 * ```text
 * Available versions:
 *  - 24.22.0 (LTS) [npm 11.19.1]
 *  - 3.9.16 (installed)
 * ```
 *
 * Every entry is `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
 */
fun parseSearch(output: String): List<AvailableVersion> =
    lines(output).mapNotNull { rawLine ->
        val line = rawLine.trimEnd()
        if (line.startsWith(" - ")) parseSearchEntry(line.substring(3).trim()) else null
    }

private fun parseSearchEntry(entry: String): AvailableVersion? {
    if (entry.isEmpty()) return null
    val firstSpace = entry.indexOf(' ')
    if (firstSpace < 0) return AvailableVersion(entry, null, emptyList(), false)
    val version = entry.substring(0, firstSpace)
    var rest = entry.substring(firstSpace + 1).trim()

    var installed = false
    val installedSuffix = "(installed)"
    if (rest.endsWith(installedSuffix)) {
        installed = true
        rest = rest.dropLast(installedSuffix.length).trim()
    }

    var additions = emptyList<VersionAddition>()
    if (rest.endsWith("]")) {
        val open = rest.lastIndexOf('[')
        if (open >= 0) {
            additions = parseAdditions(rest.substring(open + 1, rest.length - 1))
            rest = rest.substring(0, open).trim()
        }
    }

    val note =
        when {
            rest.startsWith("(") && rest.endsWith(")") -> rest.substring(1, rest.length - 1).trim().ifEmpty { null }
            rest.isNotEmpty() -> rest
            else -> null
        }
    return AvailableVersion(version, note, additions, installed)
}

private fun parseAdditions(text: String): List<VersionAddition> =
    text.split(',').map { it.trim() }.filter { it.isNotEmpty() }.map { item ->
        val space = item.indexOf(' ')
        if (space < 0) VersionAddition(item, "") else VersionAddition(item.substring(0, space), item.substring(space + 1).trim())
    }

private val availablePlugin = Regex("""^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$""")

/**
 * Parses `vfox available`:
 *
 * ```text
 * AVAILABLE PLUGINS
 *
 *   nodejs              ✓  https://github.com/version-fox/vfox-nodejs
 *   bun                 ✗  https://github.com/arfaWong/vfox-bun
 * ```
 */
fun parseAvailablePlugins(output: String): List<AvailablePlugin> =
    lines(output).mapNotNull { line ->
        availablePlugin.find(line)?.let {
            AvailablePlugin(name = it.groupValues[1], homepage = it.groupValues[3], official = it.groupValues[2] == "✓")
        }
    }

/**
 * Parses the output of `vfox info <name> --format` with
 * [PLUGIN_INFO_TEMPLATE], or returns `null` if it lacks fields.
 */
fun parsePluginInfo(output: String): PluginInfo? {
    val fields = stripAnsi(output).split(FIELD_SEPARATOR)
    if (fields.size < 5) return null
    return PluginInfo(
        name = fields[0].trim(),
        version = fields[1].trim(),
        homepage = fields[2].trim(),
        installPath = fields[3].trim(),
        // The description is the last field and may contain separators itself.
        description = fields.drop(4).joinToString(FIELD_SEPARATOR).trim(),
    )
}

/**
 * Parses `vfox info <name>@<version>`, which prints the directory of an
 * installed version, or `notfound`, for which it returns `null`.
 */
fun parseInstallPath(output: String): String? = stripAnsi(output).trim().takeUnless { it.isEmpty() || it == "notfound" }

/** Parses `vfox config --list`, which prints one `<key> = <value>` per line. */
fun parseConfigList(output: String): List<ConfigEntry> {
    val result = mutableListOf<ConfigEntry>()
    for (line in lines(output)) {
        val equals = line.indexOf(" = ")
        if (equals <= 0) {
            // `key =` with an empty value loses its trailing space when trimmed.
            val trimmed = line.trimEnd()
            if (trimmed.endsWith(" =")) result += ConfigEntry(trimmed.dropLast(2).trim(), "")
            continue
        }
        val key = line.substring(0, equals).trim()
        if (key.isEmpty() || key.contains(' ')) continue
        result += ConfigEntry(key, line.substring(equals + 3).trim())
    }
    return result
}

private val downloadProgress = Regex("""Downloading\.\.\.\s+(\d{1,3})%""")

/**
 * Extracts the download progress from a line of `vfox install` output: the
 * progress bar on standard error, `Downloading...  42% [=====>    ]`. Returns a
 * fraction between 0 and 1, or `null` for other lines.
 */
fun parseDownloadProgress(line: String): Double? {
    val percent =
        downloadProgress
            .find(line)
            ?.groupValues
            ?.get(1)
            ?.toInt() ?: return null
    return percent.coerceIn(0, 100) / 100.0
}

private val installTrailer = Regex("""^failed to install\b""")

/**
 * Picks a short, human readable error message from a failed command. Usage
 * errors go to standard error. The error a command returns goes to standard
 * output up to vfox 1.0.12 and to standard error from 1.0.13, while what a
 * command prints before failing stays on standard output. So the lines of
 * standard output are read first and those of standard error last; progress
 * bar redraws and blank lines are ignored, and the last meaningful line wins,
 * unless it is the generic `failed to install <name>` trailer, in which case
 * the more specific line before it is used.
 */
fun summarizeError(
    stdout: String,
    stderr: String,
): String {
    val all = meaningfulLines(stdout) + meaningfulLines(stderr)
    val last = all.lastOrNull() ?: return ""
    if (all.size > 1 && installTrailer.containsMatchIn(last)) return all[all.size - 2]
    return last
}

private fun meaningfulLines(text: String): List<String> =
    lines(text).map { it.trim() }.filter {
        it.isNotEmpty() &&
            parseDownloadProgress(it) == null
    }

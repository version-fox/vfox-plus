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

import java.io.IOException
import java.nio.file.Files
import java.nio.file.Path
import kotlin.io.path.deleteRecursively
import kotlin.io.path.exists

// Sources passed to `vfox add --source`.

/** Whether [source] is a URL rather than a local path. */
fun isRemotePluginSource(source: String): Boolean = source.startsWith("http://") || source.startsWith("https://")

/**
 * Whether vfox would move [source] instead of reading it. vfox (still in
 * 1.0.13) renames a local `.lua` plugin into its temporary directory, which
 * deletes the user's file and fails when the file is on another filesystem;
 * local `.zip` archives and URLs are only read. See docs/vfox-cli-contract.md.
 */
fun vfoxMovesPluginSource(source: String): Boolean = !isRemotePluginSource(source) && source.lowercase().endsWith(".lua")

/**
 * The temporary directory of vfox when it runs with [environment]:
 * `~/.version-fox/tmp` if the legacy `~/.version-fox` exists, `~/.vfox/tmp`
 * otherwise. Without `HOME`, where vfox does not run at all, the system
 * temporary directory.
 */
fun vfoxTemporaryDirectory(environment: Map<String, String>): Path {
    val home = environment["HOME"]?.ifEmpty { null } ?: return Path.of(System.getProperty("java.io.tmpdir"))
    val legacy = Path.of(home, ".version-fox")
    return (if (legacy.exists()) legacy else Path.of(home, ".vfox")).resolve("tmp")
}

/** A plugin source that is safe to hand to vfox. */
class PreparedPluginSource(
    /** The path or URL to pass to `vfox add --source`. */
    val path: String,
    private val copyDirectory: Path?,
) {
    /** Whether [path] is a private copy of the original source. */
    val isCopy: Boolean get() = copyDirectory != null

    /** Deletes the private copy, if one was made. */
    @OptIn(kotlin.io.path.ExperimentalPathApi::class)
    fun dispose() {
        copyDirectory?.let { runCatching { it.deleteRecursively() } }
    }
}

/**
 * Prepares [source] for `vfox add --source`. A local `.lua` file is copied into
 * a new directory under [temporaryRoot], so that vfox moves the copy and the
 * user's file stays; pass the [vfoxTemporaryDirectory], because vfox moves the
 * file with a rename, which only works on the same filesystem. Other sources
 * are returned as they are. If the copy cannot be made, the original path is
 * returned and vfox reports the problem.
 */
@OptIn(kotlin.io.path.ExperimentalPathApi::class)
fun preparePluginSource(
    source: String,
    temporaryRoot: Path,
): PreparedPluginSource {
    if (!vfoxMovesPluginSource(source)) return PreparedPluginSource(source, null)
    var directory: Path? = null
    return try {
        Files.createDirectories(temporaryRoot)
        val created = Files.createTempDirectory(temporaryRoot, "vfox-plus-plugin-")
        directory = created
        val copy = created.resolve(Path.of(source).fileName)
        Files.copy(Path.of(source), copy)
        PreparedPluginSource(copy.toString(), created)
    } catch (_: IOException) {
        directory?.let { runCatching { it.deleteRecursively() } }
        PreparedPluginSource(source, null)
    }
}

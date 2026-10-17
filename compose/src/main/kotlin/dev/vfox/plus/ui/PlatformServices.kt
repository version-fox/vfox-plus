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

package dev.vfox.plus.ui

import dev.vfox.plus.core.isExecutableFile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.awt.Toolkit
import java.awt.datatransfer.StringSelection
import java.io.File
import java.util.concurrent.TimeUnit
import javax.swing.JFileChooser
import javax.swing.SwingUtilities
import javax.swing.filechooser.FileNameExtensionFilter

/** Desktop integration that the user interface needs. */
interface PlatformServices {
    /** The accent colour of the desktop as ARGB, if it exposes one. */
    suspend fun accentColor(): Long?

    /** Whether the desktop prefers a dark appearance, or `null` if it has no preference. */
    suspend fun prefersDark(): Boolean?

    /** Opens [url] in the default browser. */
    fun openUrl(url: String)

    /** Lets the user choose a directory; `null` if they cancel. */
    suspend fun pickDirectory(initial: String? = null): String?

    /** Lets the user choose a file, optionally with one of [extensions]; `null` if they cancel. */
    suspend fun pickFile(
        initial: String? = null,
        label: String? = null,
        extensions: List<String> = emptyList(),
    ): String?

    /** Copies [text] to the clipboard. */
    fun copyText(text: String)
}

/**
 * The integration with a Linux desktop. The JVM has no portal API, so the
 * tools of the desktop do the work: `gdbus` reads the accent colour from the
 * XDG desktop portal, `xdg-open` opens links and `zenity` shows the file
 * choosers of GTK. Without zenity, the file choosers of Swing are used.
 */
class DesktopPlatformServices(
    private val environment: Map<String, String> = System.getenv(),
) : PlatformServices {
    private fun tool(name: String): String? =
        environment["PATH"]
            .orEmpty()
            .split(':')
            .filter { it.isNotEmpty() }
            .map { "$it/$name" }
            .firstOrNull(::isExecutableFile)

    /** Runs [command] and returns its standard output if it succeeds within [seconds]. */
    private fun output(
        command: List<String>,
        seconds: Long,
    ): String? =
        runCatching {
            val process = ProcessBuilder(command).redirectInput(ProcessBuilder.Redirect.from(File("/dev/null"))).start()
            val text = process.inputStream.readAllBytes().decodeToString()
            if (process.waitFor(seconds, TimeUnit.SECONDS) && process.exitValue() == 0) text else null
        }.getOrNull()

    /** Reads a setting of `org.freedesktop.appearance` from the XDG desktop portal with gdbus. */
    private suspend fun appearance(key: String): String? =
        withContext(Dispatchers.IO) {
            val gdbus = tool("gdbus") ?: return@withContext null
            output(
                listOf(
                    gdbus,
                    "call",
                    "--session",
                    "--timeout",
                    "2",
                    "--dest",
                    "org.freedesktop.portal.Desktop",
                    "--object-path",
                    "/org/freedesktop/portal/desktop",
                    "--method",
                    "org.freedesktop.portal.Settings.ReadOne",
                    "org.freedesktop.appearance",
                    key,
                ),
                5,
            )
        }

    override suspend fun accentColor(): Long? = appearance("accent-color")?.let(::parsePortalColor)

    // 1 prefers dark, 2 prefers light and 0 has no preference.
    override suspend fun prefersDark(): Boolean? =
        when (appearance("color-scheme")?.let { Regex("""uint32 (\d)""").find(it)?.groupValues?.get(1) }) {
            "1" -> true
            "2" -> false
            else -> null
        }

    override fun openUrl(url: String) {
        val opener = tool("xdg-open") ?: return
        runCatching { ProcessBuilder(opener, url).redirectInput(ProcessBuilder.Redirect.from(File("/dev/null"))).start() }
    }

    override suspend fun pickDirectory(initial: String?): String? =
        choose(
            zenityArguments = listOf("--file-selection", "--directory") + (initial?.let { listOf("--filename=$it/") } ?: emptyList()),
        ) {
            JFileChooser(initial).apply { fileSelectionMode = JFileChooser.DIRECTORIES_ONLY }
        }

    override suspend fun pickFile(
        initial: String?,
        label: String?,
        extensions: List<String>,
    ): String? {
        val filter =
            if (extensions.isEmpty()) {
                emptyList()
            } else {
                listOf(
                    "--file-filter=${label ?: ""} | ${extensions.joinToString(" ") { "*.$it" }}",
                )
            }
        return choose(listOf("--file-selection") + (initial?.let { listOf("--filename=$it/") } ?: emptyList()) + filter) {
            JFileChooser(initial).apply {
                if (extensions.isNotEmpty()) fileFilter = FileNameExtensionFilter(label ?: "", *extensions.toTypedArray())
            }
        }
    }

    private suspend fun choose(
        zenityArguments: List<String>,
        swing: () -> JFileChooser,
    ): String? {
        val zenity = tool("zenity")
        if (zenity != null) {
            return withContext(Dispatchers.IO) {
                // zenity exits with 1 when the user cancels.
                output(listOf(zenity) + zenityArguments, TimeUnit.HOURS.toSeconds(1))?.trim()?.ifEmpty { null }
            }
        }
        return withContext(Dispatchers.IO) {
            var chosen: String? = null
            SwingUtilities.invokeAndWait {
                val chooser = swing()
                if (chooser.showOpenDialog(null) == JFileChooser.APPROVE_OPTION) chosen = chooser.selectedFile?.absolutePath
            }
            chosen
        }
    }

    override fun copyText(text: String) {
        Toolkit.getDefaultToolkit().systemClipboard.setContents(StringSelection(text), null)
    }
}

private val portalColor = Regex("""\(\s*([-\d.eE+]+)\s*,\s*([-\d.eE+]+)\s*,\s*([-\d.eE+]+)\s*\)""")

/**
 * The ARGB colour in the output of `gdbus` for the portal setting, such as
 * `(<<(0.2, 0.4, 0.8)>>,)`. The portal reports components outside of 0–1 when
 * no accent colour is set.
 */
fun parsePortalColor(output: String): Long? {
    val components =
        portalColor
            .find(output)
            ?.groupValues
            ?.drop(1)
            ?.map { it.toDoubleOrNull() ?: return null } ?: return null
    if (components.any { it < 0.0 || it > 1.0 }) return null
    val (red, green, blue) = components.map { Math.round(it * 255) }
    return 0xFF000000 or (red shl 16) or (green shl 8) or blue
}

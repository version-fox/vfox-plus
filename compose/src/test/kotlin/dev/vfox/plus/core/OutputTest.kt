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

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class OutputTest {
    @Test
    fun `reads release and pre-release versions`() {
        assertEquals("1.0.12", parseVfoxVersion("vfox version 1.0.12\n"))
        assertEquals("2.0.0-beta.1", parseVfoxVersion("vfox version 2.0.0-beta.1"))
        assertNull(parseVfoxVersion("command not found"))
    }

    @Test
    fun `current strips the v prefix once and ignores colours and other lines`() {
        assertEquals(mapOf("go" to "v1.2"), parseCurrent("go -> vv1.2\n"))
        assertEquals(
            mapOf("java" to "21", "maven" to null),
            parseCurrent("Warning: something\njava -> \u001B[92mv21\u001B[0m\nmaven -> N/A \n"),
        )
    }

    @Test
    fun `the tree handles leaves, headers and blank lines`() {
        assertEquals(mapOf("maven" to emptyList<String>()), parseInstalledTree("All installed sdk versions\n└──maven\n"))
        assertEquals(mapOf("nodejs" to listOf("24.11.1")), parseInstalledTree("All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n"))
    }

    @Test
    fun `merging keeps plugins that only one command reported`() {
        assertEquals(
            listOf(InstalledSdk("java", listOf("21"), "21"), InstalledSdk("go", emptyList(), null)),
            mergeInstalledSdks(mapOf("java" to listOf("21")), mapOf("java" to "21", "go" to null)),
        )
    }

    @Test
    fun `search reads notes, additions and the installed marker`() {
        val versions =
            parseSearch(
                "Available versions:\n" +
                    " - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n" +
                    " - 1.0 (release candidate)\n" +
                    " - 0.1.14 [npm ]\n" +
                    " - 2.0\n" +
                    " - 1.0 preview\n" +
                    "Please select a version\n -\n",
            )
        assertEquals(
            listOf(
                AvailableVersion("24.22.0", "LTS", listOf(VersionAddition("npm", "11.19.1"), VersionAddition("corepack", "0.34.0")), true),
                AvailableVersion("1.0", "release candidate", emptyList(), false),
                AvailableVersion("0.1.14", null, listOf(VersionAddition("npm", "")), false),
                AvailableVersion("2.0", null, emptyList(), false),
                AvailableVersion("1.0", "preview", emptyList(), false),
            ),
            versions,
        )
        assertEquals(listOf("npm 11.19.1", "corepack 0.34.0"), versions[0].additions.map { it.label })
        assertEquals("npm", versions[2].additions.single().label)
    }

    @Test
    fun `plugin information keeps separators in the description and rejects incomplete output`() {
        val info = parsePluginInfo("a\u001F1.0\u001Fhttps://x\u001F/p\u001Fone\u001Ftwo")
        assertEquals("one\u001Ftwo", info?.description)
        assertEquals("https://x", info?.homepage)
        assertNull(parsePluginInfo("nodejs not supported"))
        assertNull(parseInstallPath("notfound\n"))
        assertEquals("/opt/sdk/v-1", parseInstallPath("/opt/sdk/v-1\n"))
    }

    @Test
    fun `configuration keeps empty values and values with spaces`() {
        assertEquals(
            listOf(ConfigEntry("proxy.url", ""), ConfigEntry("foo.bar", "a = b"), ConfigEntry("registry.address", "")),
            parseConfigList("proxy.url = \nfoo.bar = a = b\nnot a setting\nregistry.address =\n"),
        )
    }

    @Test
    fun `download progress`() {
        assertEquals(0.42, parseDownloadProgress("Downloading...  42% [===>   ] (1 MB/s) [1s:2s]"))
        assertEquals(1.0, parseDownloadProgress("Downloading... 100% [======]"))
        assertNull(parseDownloadProgress("Preinstalling maven@3.9.16..."))
    }

    @Test
    fun `errors prefer the last meaningful line and skip the install trailer`() {
        assertEquals("Use the -y flag", summarizeError("Removing...\n", "Use the -y flag\n"))
        assertEquals(
            "failed to download x@1 file, err:EOF",
            summarizeError(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n",
                "\rDownloading...  3% [>   ]\r      \r",
            ),
        )
        // vfox 1.0.13 prints the error a command returns to standard error.
        assertEquals(
            "failed to download x@1 file, err:EOF",
            summarizeError(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n",
                "\rDownloading...  3% [>   ]\r      \rfailed to install x\n",
            ),
        )
        assertEquals("", summarizeError("", "  \n"))
    }

    @Test
    fun `terminal text`() {
        assertEquals("ok link x", stripAnsi("\u001B[1;32mok\u001B[0m \u001B]8;;https://x\u0007link\u001B]8;;\u0007 \u001BMx"))
        assertEquals(listOf("a", "b", "c", "d"), splitTerminalLines("a\r\nb\rc\nd"))
    }

    @Test
    fun `the line splitter keeps characters split between chunks and replaces malformed bytes`() {
        val splitter = TerminalLineSplitter()
        assertEquals(listOf("one"), splitter.push("one\r\u001B[32mtw".encodeToByteArray()))
        val euro = "€".encodeToByteArray()
        assertEquals(emptyList(), splitter.push("o ".encodeToByteArray() + euro[0]))
        assertEquals(listOf("two €"), splitter.push(byteArrayOf(euro[1], euro[2], 0x0a, 0x20, 0x0a)))
        assertEquals(emptyList(), splitter.push("last".encodeToByteArray()))
        assertEquals(listOf("last"), splitter.close())
        val malformed = TerminalLineSplitter()
        assertEquals(listOf("a�b"), malformed.push(byteArrayOf(0x61, 0xff.toByte(), 0x62, 0x0a)))
        assertEquals(listOf("�"), TerminalLineSplitter().apply { push(byteArrayOf(0xe2.toByte())) }.close())
    }
}

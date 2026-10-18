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

import java.nio.ByteBuffer
import java.nio.CharBuffer
import java.nio.charset.CodingErrorAction

/**
 * ANSI escape sequences: CSI sequences such as colours and cursor movement, OSC
 * sequences such as hyperlinks, and lone two-byte escapes.
 */
private val ansiEscape = Regex("\u001B(?:\\[[0-?]*[ -/]*[@-~]|\\][^\u0007\u001B]*(?:\u0007|\u001B\\\\)|[@-Z\\\\-_])")

/**
 * Removes ANSI escape sequences from [text]. vfox honours `NO_COLOR`, but its
 * download progress bar does not, so output is always sanitised before it is
 * parsed or shown.
 */
fun stripAnsi(text: String): String = text.replace(ansiEscape, "")

/**
 * Splits terminal output into lines. Both `\n` and a lone `\r` end a line:
 * progress bars redraw themselves by returning the carriage, and every redraw
 * is treated as a new line.
 */
fun splitTerminalLines(text: String): List<String> = text.replace("\r\n", "\n").split('\r', '\n')

/**
 * Turns chunks of raw process output into plain-text lines while the process
 * runs: bytes are decoded as UTF-8 (malformed bytes are replaced, and a
 * character split between two chunks is kept together), lines end at `\n` and
 * at `\r`, ANSI escape sequences are removed and blank lines are dropped. The
 * trailing partial line is returned by [close].
 */
class TerminalLineSplitter {
    private val decoder =
        Charsets.UTF_8
            .newDecoder()
            .onMalformedInput(CodingErrorAction.REPLACE)
            .onUnmappableCharacter(CodingErrorAction.REPLACE)
    private var leftover = ByteBuffer.allocate(0)
    private val pending = StringBuilder()

    /** Feeds a chunk of output and returns the lines it completed. */
    fun push(chunk: ByteArray): List<String> {
        val input = ByteBuffer.allocate(leftover.remaining() + chunk.size)
        input.put(leftover).put(chunk).flip()
        val output = CharBuffer.allocate(input.remaining() + 1)
        decoder.decode(input, output, false)
        leftover = ByteBuffer.allocate(input.remaining()).put(input).flip()
        output.flip()
        return split(output.toString())
    }

    /** Ends the output and returns its last, unterminated line, if any. */
    fun close(): List<String> {
        val output = CharBuffer.allocate(leftover.remaining() * 2 + 2)
        decoder.decode(leftover, output, true)
        decoder.flush(output)
        leftover = ByteBuffer.allocate(0)
        output.flip()
        val lines = split(output.toString())
        val rest = pending.toString()
        pending.clear()
        return lines + clean(rest)
    }

    private fun split(text: String): List<String> {
        val lines = mutableListOf<String>()
        for (char in text) {
            if (char == '\n' || char == '\r') {
                lines += clean(pending.toString())
                pending.clear()
            } else {
                pending.append(char)
            }
        }
        return lines
    }

    private fun clean(line: String): List<String> {
        val text = stripAnsi(line)
        return if (text.isBlank()) emptyList() else listOf(text)
    }
}

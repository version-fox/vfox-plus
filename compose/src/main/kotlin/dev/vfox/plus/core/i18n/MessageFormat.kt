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

package dev.vfox.plus.core.i18n

// The subset of ICU MessageFormat that the catalogues of shared/i18n use:
// placeholders (`{name}`) and plurals with exact matches and plural categories
// (`{count, plural, =0{…} =1{…} other{…}}`), which may contain placeholders
// themselves. Apostrophes are literal, as in Flutter's gen-l10n without
// use-escaping. See docs/i18n.md.

/** A part of a parsed message. */
sealed interface MessagePart {
    data class Text(
        val text: String,
    ) : MessagePart

    data class Argument(
        val name: String,
    ) : MessagePart

    data class Plural(
        val name: String,
        val cases: Map<String, List<MessagePart>>,
    ) : MessagePart
}

/** A message could not be parsed. */
class MessageFormatException(
    message: String,
    source: String,
) : IllegalArgumentException("$message in \"$source\"")

/** Parses [source] into parts. */
fun parseMessage(source: String): List<MessagePart> {
    val parser = MessageParser(source)
    val parts = parser.parseParts()
    if (!parser.atEnd) throw MessageFormatException("Unbalanced \"}\"", source)
    return parts
}

/** The plural category of [count] in [language]: `one` or `other` (the catalogues only need en and zh). */
fun pluralCategory(
    language: String,
    count: Number,
): String =
    when (language) {
        // Chinese has no grammatical number.
        "zh" -> "other"

        else -> if (count.toDouble() == 1.0) "one" else "other"
    }

/** Formats parsed [parts] with [arguments], choosing plural cases by the rules of [language]. */
fun formatMessage(
    parts: List<MessagePart>,
    arguments: Map<String, Any>,
    language: String,
): String =
    buildString {
        for (part in parts) {
            when (part) {
                is MessagePart.Text -> {
                    append(part.text)
                }

                is MessagePart.Argument -> {
                    append(arguments[part.name]?.toString() ?: "{${part.name}}")
                }

                is MessagePart.Plural -> {
                    val value = (arguments[part.name] as? Number) ?: 0
                    val exact = if (value.toDouble() % 1.0 == 0.0) "=${value.toLong()}" else "=$value"
                    val chosen = part.cases[exact] ?: part.cases[pluralCategory(language, value)] ?: part.cases["other"].orEmpty()
                    append(formatMessage(chosen, arguments, language))
                }
            }
        }
    }

private class MessageParser(
    private val source: String,
) {
    private var position = 0

    val atEnd: Boolean get() = position >= source.length

    /** Parses parts until the end or an unmatched `}`, which is not consumed. */
    fun parseParts(): List<MessagePart> {
        val parts = mutableListOf<MessagePart>()
        val text = StringBuilder()
        while (!atEnd) {
            val char = source[position]
            if (char == '}') break
            if (char == '{') {
                if (text.isNotEmpty()) {
                    parts += MessagePart.Text(text.toString())
                    text.clear()
                }
                parts += parsePlaceholder()
            } else {
                text.append(char)
                position++
            }
        }
        if (text.isNotEmpty()) parts += MessagePart.Text(text.toString())
        return parts
    }

    private fun parsePlaceholder(): MessagePart {
        expect('{')
        val name = readIdentifier()
        skipSpaces()
        if (peek() == '}') {
            position++
            return MessagePart.Argument(name)
        }
        expect(',')
        val type = readIdentifier()
        if (type != "plural") throw MessageFormatException("Unsupported argument type \"$type\"", source)
        skipSpaces()
        expect(',')
        val cases = LinkedHashMap<String, List<MessagePart>>()
        while (true) {
            skipSpaces()
            if (peek() == '}') {
                position++
                break
            }
            val selector = readSelector()
            skipSpaces()
            expect('{')
            cases[selector] = parseParts()
            expect('}')
        }
        if ("other" !in cases) throw MessageFormatException("A plural needs an \"other\" case", source)
        return MessagePart.Plural(name, cases)
    }

    private fun readIdentifier(): String {
        skipSpaces()
        val match = identifier.find(source, position)?.takeIf { it.range.first == position }
        if (match == null) throw MessageFormatException("Expected a name", source)
        position = match.range.last + 1
        return match.value
    }

    private fun readSelector(): String {
        val match = selector.find(source, position)?.takeIf { it.range.first == position }
        if (match == null) throw MessageFormatException("Expected a plural case", source)
        position = match.range.last + 1
        return match.value
    }

    private fun skipSpaces() {
        while (!atEnd && source[position].isWhitespace()) position++
    }

    private fun peek(): Char? = source.getOrNull(position)

    private fun expect(char: Char) {
        if (peek() != char) throw MessageFormatException("Expected \"$char\" at $position", source)
        position++
    }

    companion object {
        private val identifier = Regex("[A-Za-z_][A-Za-z0-9_]*")
        private val selector = Regex("=\\d+|[a-z]+")
    }
}

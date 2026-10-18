# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""The subset of ICU MessageFormat that the catalogues in `shared/i18n` use:
placeholders (`{name}`) and plurals with exact matches and plural categories
(`{count, plural, =0{…} =1{…} other{…}}`), which may contain placeholders
themselves. Apostrophes are literal, as in Flutter's `gen-l10n` without
`use-escaping`. See `docs/i18n.md`.
"""

import re
from collections.abc import Mapping, Sequence
from dataclasses import dataclass

type MessageArguments = Mapping[str, str | int | float]
"""Arguments of a message, by placeholder name."""


@dataclass(frozen=True, slots=True)
class Text:
    """Literal text."""

    text: str


@dataclass(frozen=True, slots=True)
class Argument:
    """A placeholder: `{name}`."""

    name: str


@dataclass(frozen=True, slots=True)
class Plural:
    """A plural: `{name, plural, =0{…} other{…}}`."""

    name: str
    cases: Mapping[str, Sequence[MessagePart]]


type MessagePart = Text | Argument | Plural
"""A part of a parsed message."""


class MessageFormatError(ValueError):
    """A message could not be parsed."""

    def __init__(self, message: str, source: str) -> None:
        super().__init__(f'{message} in "{source}"')


def parse_message(source: str) -> list[MessagePart]:
    """Parses `source` into parts."""
    parser = _Parser(source)
    parts = parser.parse_parts()
    if not parser.at_end():
        raise MessageFormatError('Unbalanced "}"', source)
    return parts


def plural_category(value: float, language: str) -> str:
    """The CLDR plural category of `value` in `language`: English has `one`
    and `other`, Chinese only `other`.
    """
    if language == "en" and value == 1:
        return "one"
    return "other"


def format_message(parts: Sequence[MessagePart], args: MessageArguments, language: str) -> str:
    """Formats parsed `parts` with `args`, choosing plural cases with the
    plural rules of `language`.
    """
    result: list[str] = []
    for part in parts:
        match part:
            case Text(text):
                result.append(text)
            case Argument(name):
                value = args.get(name)
                result.append(f"{{{name}}}" if value is None else _format_value(value))
            case Plural(name, cases):
                number = _number(args.get(name, 0))
                selectors = [f"={int(number)}"] if number.is_integer() else []
                selectors += [plural_category(number, language), "other"]
                chosen = next((cases[selector] for selector in selectors if selector in cases), [])
                result.append(format_message(chosen, args, language))
    return "".join(result)


def _format_value(value: str | int | float) -> str:
    if isinstance(value, float) and value.is_integer():
        return str(int(value))
    return str(value)


def _number(value: str | int | float) -> float:
    try:
        return float(value)
    except ValueError:
        return 0.0


_IDENTIFIER = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
_SELECTOR = re.compile(r"=\d+|[a-z]+")


class _Parser:
    def __init__(self, source: str) -> None:
        self._source = source
        self._position = 0

    def at_end(self) -> bool:
        return self._position >= len(self._source)

    def parse_parts(self) -> list[MessagePart]:
        """Parses parts until the end or an unmatched `}`, which is not
        consumed.
        """
        parts: list[MessagePart] = []
        text: list[str] = []
        while not self.at_end():
            char = self._source[self._position]
            if char == "}":
                break
            if char == "{":
                if text:
                    parts.append(Text("".join(text)))
                    text = []
                parts.append(self._parse_placeholder())
            else:
                text.append(char)
                self._position += 1
        if text:
            parts.append(Text("".join(text)))
        return parts

    def _parse_placeholder(self) -> MessagePart:
        self._expect("{")
        name = self._read(_IDENTIFIER, "Expected a name")
        self._skip_spaces()
        if self._peek() == "}":
            self._position += 1
            return Argument(name)
        self._expect(",")
        kind = self._read(_IDENTIFIER, "Expected a name")
        if kind != "plural":
            raise MessageFormatError(f'Unsupported argument type "{kind}"', self._source)
        self._skip_spaces()
        self._expect(",")
        cases: dict[str, list[MessagePart]] = {}
        while True:
            self._skip_spaces()
            if self._peek() == "}":
                self._position += 1
                break
            selector = self._read(_SELECTOR, "Expected a plural case", skip_spaces=False)
            self._skip_spaces()
            self._expect("{")
            cases[selector] = self.parse_parts()
            self._expect("}")
        if "other" not in cases:
            raise MessageFormatError('A plural needs an "other" case', self._source)
        return Plural(name, cases)

    def _read(self, pattern: re.Pattern[str], error: str, *, skip_spaces: bool = True) -> str:
        if skip_spaces:
            self._skip_spaces()
        match = pattern.match(self._source, self._position)
        if match is None:
            raise MessageFormatError(error, self._source)
        self._position = match.end()
        return match.group(0)

    def _skip_spaces(self) -> None:
        while not self.at_end() and self._source[self._position].isspace():
            self._position += 1

    def _peek(self) -> str | None:
        return None if self.at_end() else self._source[self._position]

    def _expect(self, char: str) -> None:
        if self._peek() != char:
            raise MessageFormatError(f'Expected "{char}" at {self._position}', self._source)
        self._position += 1

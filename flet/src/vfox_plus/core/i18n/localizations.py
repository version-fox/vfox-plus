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

"""Translations read from the ARB catalogues in `shared/i18n`.

The catalogues are bundled as they are; nothing is converted. English is the
template and the fallback.
"""

import json
import re
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import Any, Literal

from vfox_plus.core.i18n.message_format import MessageArguments, MessagePart, Text, format_message, parse_message
from vfox_plus.core.settings.app_settings import LanguagePreference

type ArbCatalogue = Mapping[str, Any]
"""An ARB catalogue: messages and their `@`-prefixed metadata."""

type SupportedLanguage = Literal["en", "zh"]
"""A language code with a catalogue."""

SUPPORTED_LANGUAGES: tuple[SupportedLanguage, ...] = ("en", "zh")
"""The languages with a catalogue."""

_LOCALE_SEPARATOR = re.compile(r"[-_.@]")


def system_locales(environment: Mapping[str, str]) -> list[str]:
    """The locales of the desktop, most preferred first, as GLib reads them
    from `environment`.
    """
    candidates = [
        *environment.get("LANGUAGE", "").split(":"),
        environment.get("LC_ALL", ""),
        environment.get("LC_MESSAGES", ""),
        environment.get("LANG", ""),
    ]
    return [locale for locale in candidates if locale not in ("", "C", "POSIX") and not locale.startswith("C.")]


def resolve_language(preference: LanguagePreference, locales: Sequence[str]) -> SupportedLanguage:
    """Picks the catalogue for `preference`. With `system`, the desktop
    locales are matched against the catalogues; every Chinese locale gets
    Simplified Chinese and English is the fallback.
    """
    match preference.language_code:
        case "zh":
            return "zh"
        case "en":
            return "en"
    for locale in locales:
        language = _LOCALE_SEPARATOR.split(locale.lower())[0]
        if language == "zh":
            return "zh"
        if language == "en":
            return "en"
    return "en"


def load_catalogue(directory: Path, language: SupportedLanguage) -> ArbCatalogue:
    """Reads the catalogue `app_<language>.arb` from `directory`."""
    with (directory / f"app_{language}.arb").open(encoding="utf-8") as file:
        catalogue: ArbCatalogue = json.load(file)
    return catalogue


def message_keys(catalogue: ArbCatalogue) -> list[str]:
    """The message keys of a catalogue, without metadata."""
    return [key for key in catalogue if not key.startswith("@")]


class Localizations:
    """The messages of one language."""

    def __init__(self, language: SupportedLanguage, messages: ArbCatalogue, fallback: ArbCatalogue) -> None:
        self.language = language
        """The language code of the messages."""
        self._messages = messages
        self._fallback = fallback
        self._cache: dict[str, list[MessagePart]] = {}

    def t(self, key: str, **args: str | int | float) -> str:
        """The message `key` formatted with `args`.

        A key that the language lacks falls back to English; a key that no
        catalogue has is returned as it is, which the tests catch.
        """
        return self.format(key, args)

    def format(self, key: str, args: MessageArguments) -> str:
        """The message `key` formatted with the mapping `args`."""
        parts = self._cache.get(key)
        if parts is None:
            source = self._messages.get(key, self._fallback.get(key))
            parts = parse_message(source) if isinstance(source, str) else [Text(key)]
            self._cache[key] = parts
        return format_message(parts, args, self.language)

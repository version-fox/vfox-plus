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

"""Preferences of the application itself (not of vfox).

The JSON keys are part of the shared specification in
`docs/architecture.md#settings`, so that every implementation of the GUI can
read the same file.
"""

import json
import re
from collections.abc import Mapping
from dataclasses import dataclass, field
from enum import StrEnum
from types import MappingProxyType
from typing import Any


class ThemePreference(StrEnum):
    """Light or dark appearance; `system` follows the desktop."""

    SYSTEM = "system"
    LIGHT = "light"
    DARK = "dark"


class LanguagePreference(StrEnum):
    """The user's language choice; `system` follows the desktop locale."""

    SYSTEM = "system"
    ENGLISH = "english"
    SIMPLIFIED_CHINESE = "simplifiedChinese"

    @property
    def language_code(self) -> str | None:
        """The language code of the translation catalogue, or `None` for
        `system`.
        """
        return {"english": "en", "simplifiedChinese": "zh"}.get(self.value)


DEFAULT_SEED_COLOR = 0xFFF28C1B
"""The brand colour used when the desktop accent colour is not used or not
available: the orange of the vfox logo."""

_EMPTY: Mapping[str, Any] = MappingProxyType({})


@dataclass(frozen=True, slots=True)
class AppSettings:
    """Preferences of the application, persisted as JSON."""

    theme: ThemePreference = ThemePreference.SYSTEM
    use_system_color: bool = True
    """Whether to derive the colour scheme from the desktop accent colour."""
    seed_color: int = DEFAULT_SEED_COLOR
    """ARGB seed of the colour scheme when :attr:`use_system_color` is off."""
    language: LanguagePreference = LanguagePreference.SYSTEM
    vfox_executable: str | None = None
    """Path of the vfox executable chosen by the user, or `None` to detect
    it."""
    extra_environment: Mapping[str, str] = field(default=_EMPTY)
    """Additional environment variables for vfox, such as mirror settings."""
    rail_expanded: bool = False
    """Whether the navigation rail shows labels next to the icons."""
    unknown: Mapping[str, Any] = field(default=_EMPTY)
    """Keys of the settings file that this version does not know, kept as
    they are so that settings written by another implementation survive."""

    def __eq__(self, other: object) -> bool:
        return isinstance(other, AppSettings) and settings_to_json(self) == settings_to_json(other)

    def __hash__(self) -> int:
        return hash(json.dumps(settings_to_json(self), sort_keys=True))


DEFAULT_SETTINGS = AppSettings()
"""The settings of a fresh installation."""

_KNOWN_KEYS = frozenset(
    {"theme", "useSystemColor", "seedColor", "language", "vfoxExecutable", "extraEnvironment", "railExpanded"}
)


def settings_from_json(data: object) -> AppSettings:
    """Restores settings from decoded JSON, ignoring malformed values."""
    if not isinstance(data, dict):
        return DEFAULT_SETTINGS
    raw_environment = data.get("extraEnvironment")
    environment = (
        {key: value for key, value in raw_environment.items() if isinstance(value, str)}
        if isinstance(raw_environment, dict)
        else {}
    )
    executable = data.get("vfoxExecutable")
    seed = data.get("seedColor")
    use_system_color = data.get("useSystemColor")
    return AppSettings(
        theme=_one_of(ThemePreference, data.get("theme"), ThemePreference.SYSTEM),
        use_system_color=use_system_color if isinstance(use_system_color, bool) else True,
        # JSON booleans are integers in Python; they are not colours.
        seed_color=seed if isinstance(seed, int) and not isinstance(seed, bool) else DEFAULT_SEED_COLOR,
        language=_one_of(LanguagePreference, data.get("language"), LanguagePreference.SYSTEM),
        vfox_executable=executable if isinstance(executable, str) and executable != "" else None,
        extra_environment=MappingProxyType(environment),
        rail_expanded=data.get("railExpanded") is True,
        unknown=MappingProxyType({key: value for key, value in data.items() if key not in _KNOWN_KEYS}),
    )


def settings_to_json(settings: AppSettings) -> dict[str, Any]:
    """Serialises the settings, with the keys in the order of the
    specification.
    """
    result: dict[str, Any] = {
        "theme": settings.theme.value,
        "useSystemColor": settings.use_system_color,
        "seedColor": settings.seed_color,
        "language": settings.language.value,
    }
    if settings.vfox_executable is not None:
        result["vfoxExecutable"] = settings.vfox_executable
    result["extraEnvironment"] = dict(settings.extra_environment)
    result["railExpanded"] = settings.rail_expanded
    result.update(settings.unknown)
    return result


@dataclass(frozen=True, slots=True)
class ParsedEnvironment:
    """The result of :func:`parse_environment_lines`."""

    variables: dict[str, str]
    """The variables of the valid lines."""
    invalid_lines: list[int]
    """The 1-based numbers of the lines that could not be parsed."""


_ENVIRONMENT_KEY = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")


def parse_environment_lines(text: str) -> ParsedEnvironment:
    """Parses `KEY=VALUE` lines into environment variables.

    Blank lines and lines starting with `#` are ignored. Keys must be valid
    POSIX variable names.
    """
    variables: dict[str, str] = {}
    invalid_lines: list[int] = []
    for index, raw_line in enumerate(text.split("\n")):
        line = raw_line.strip()
        if line == "" or line.startswith("#"):
            continue
        key, equals, value = line.partition("=")
        key = key.strip()
        if not equals or _ENVIRONMENT_KEY.fullmatch(key) is None:
            invalid_lines.append(index + 1)
            continue
        variables[key] = value.strip()
    return ParsedEnvironment(variables, invalid_lines)


def format_environment_lines(variables: Mapping[str, str]) -> str:
    """Formats environment variables as `KEY=VALUE` lines, sorted by key."""
    return "\n".join(f"{key}={variables[key]}" for key in sorted(variables))


def _one_of[E: StrEnum](choices: type[E], value: object, fallback: E) -> E:
    for choice in choices:
        if choice.value == value:
            return choice
    return fallback

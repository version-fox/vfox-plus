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

"""The resources the application reads at run time.

`tools/prepare.py` puts them into `vfox_plus/resources`, which `flet build`
packages with the code: the translation catalogues of `shared/i18n` as they
are, the code points of the icons of `shared/icons.json` and the licenses of
the bundled third-party code. None of them is committed.
"""

import json
from functools import cache
from pathlib import Path

from vfox_plus.core.i18n.localizations import ArbCatalogue, SupportedLanguage, load_catalogue

RESOURCES = Path(__file__).resolve().parent.parent / "resources"
"""The generated resources, inside the package."""

_SHARED = Path(__file__).resolve().parents[4] / "shared"


def i18n_directory() -> Path:
    """The directory of the translation catalogues: the bundled copy, or
    `shared/i18n` of the repository when running from the sources without
    `tools/prepare.py`.
    """
    bundled = RESOURCES / "i18n"
    return bundled if bundled.is_dir() else _SHARED / "i18n"


@cache
def catalogue(language: SupportedLanguage) -> ArbCatalogue:
    """The translation catalogue of `language`."""
    return load_catalogue(i18n_directory(), language)


@cache
def icon_codepoints() -> dict[str, int]:
    """The code point of every icon of `shared/icons.json`, by meaning."""
    path = RESOURCES / "icons.json"
    if not path.is_file():
        raise FileNotFoundError(f"{path} is missing; run tools/prepare.py first.")
    icons: dict[str, dict[str, int]] = json.loads(path.read_text(encoding="utf-8"))["icons"]
    return {meaning: entry["codepoint"] for meaning, entry in icons.items()}

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

"""The icons of the user interface: the Material Symbols Rounded of
`shared/icons.json`, by meaning (see `docs/design-system.md#icons`).

The extension bundles the subset of the font that `tools/prepare.py`
generates; `tests/test_project.py` checks that every meaning used in the code
exists.
"""

import flet as ft

from vfox_plus.ui.resources import icon_codepoints
from vfox_plus_m3e import Symbol


def icon(meaning: str) -> Symbol:
    """The icon for `meaning`, such as `home` or `sdks`."""
    return Symbol.of(icon_codepoints()[meaning])


def icon_control(
    meaning: str, *, size: float | None = None, color: ft.ColorValue | None = None, filled: bool = False
) -> ft.Icon:
    """An :class:`ft.Icon` for `meaning`, outlined or `filled`."""
    return ft.Icon(icon(meaning), size=size, color=color, fill=1 if filled else 0)

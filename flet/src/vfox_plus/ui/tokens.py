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

"""Design tokens of the Material 3 Expressive system used by the application.

The values are documented in `docs/design-system.md`, which every
implementation of the GUI follows.
"""

from enum import IntEnum

from vfox_plus_m3e import FULL


class Corners:
    """The Material 3 Expressive corner radius scale, in logical pixels.

    See https://m3.material.io/styles/shape/corner-radius-scale.
    """

    NONE = 0
    EXTRA_SMALL = 4
    SMALL = 8
    MEDIUM = 12
    LARGE = 16
    LARGE_INCREASED = 20
    EXTRA_LARGE = 28
    EXTRA_LARGE_INCREASED = 32
    EXTRA_EXTRA_LARGE = 48
    FULL = FULL
    """Fully rounded: half of the shorter side."""


class Gaps:
    """Spacing between elements, on a 4 dp grid."""

    HAIRLINE = 2
    """Between the segments of connected groups."""
    XS = 4
    S = 8
    M = 12
    L = 16
    XL = 24
    XXL = 32


class WindowSizeClass(IntEnum):
    """Material 3 window size classes, by window width.

    See https://m3.material.io/foundations/layout/applying-layout/window-size-classes.
    """

    COMPACT = 0
    """Narrower than 600 dp."""
    MEDIUM = 1
    """600 dp to 839 dp."""
    EXPANDED = 2
    """840 dp to 1199 dp."""
    LARGE = 3
    """1200 dp to 1599 dp."""
    EXTRA_LARGE = 4
    """1600 dp and wider."""

    @classmethod
    def of(cls, width: float) -> WindowSizeClass:
        """The size class of a window `width` logical pixels wide."""
        for limit, size in ((600, cls.COMPACT), (840, cls.MEDIUM), (1200, cls.EXPANDED), (1600, cls.LARGE)):
            if width < limit:
                return size
        return cls.EXTRA_LARGE

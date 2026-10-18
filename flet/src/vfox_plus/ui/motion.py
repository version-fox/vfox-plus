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

"""The Material 3 Expressive motion system: physics-based springs.

The springs that move components run in the Flutter extension (`m3e/`).
Python needs them where Flet only accepts a duration, such as the shape
animation of buttons: :meth:`Spring.settle_duration` is how long a spring
takes to settle, as `Spring.settleDuration` of the Flutter implementation.
See `docs/design-system.md#motion`.
"""

import math
from dataclasses import dataclass
from functools import cached_property


@dataclass(frozen=True)
class Spring:
    """A critically or under-damped spring with unit mass."""

    damping_ratio: float
    stiffness: float

    def position(self, time: float) -> float:
        """Where the spring is after `time` seconds, moving from 0 to 1."""
        omega = math.sqrt(self.stiffness)
        if self.damping_ratio >= 1:
            return 1 - math.exp(-omega * time) * (1 + omega * time)
        damped = omega * math.sqrt(1 - self.damping_ratio**2)
        decay = math.exp(-self.damping_ratio * omega * time)
        return 1 - decay * (math.cos(damped * time) + self.damping_ratio * omega / damped * math.sin(damped * time))

    def velocity(self, time: float) -> float:
        """The velocity of the spring after `time` seconds, moving from 0 to
        1.
        """
        omega = math.sqrt(self.stiffness)
        if self.damping_ratio >= 1:
            return omega * omega * time * math.exp(-omega * time)
        damped = omega * math.sqrt(1 - self.damping_ratio**2)
        decay = math.exp(-self.damping_ratio * omega * time)
        return decay * (omega * omega / damped) * math.sin(damped * time)

    @cached_property
    def settle_duration(self) -> int:
        """Milliseconds until the spring rests within 0.001 of its target
        with a velocity below 0.01, sampled at 240 Hz like Flutter.
        """
        time = 0.0
        while time < 5 and not (abs(self.position(time) - 1) < 0.001 and abs(self.velocity(time)) < 0.01):
            time += 1 / 240
        return round(time * 1000)


FAST_SPATIAL = Spring(0.6, 800)
"""Small components such as buttons and switches."""
DEFAULT_SPATIAL = Spring(0.8, 380)
"""Partial-screen changes such as dialogs and the navigation rail."""
SLOW_SPATIAL = Spring(0.8, 200)
"""Full-screen transitions."""
FAST_EFFECTS = Spring(1, 3800)
"""Colour and opacity of small components."""
DEFAULT_EFFECTS = Spring(1, 1600)
"""Colour and opacity of partial-screen changes."""
SLOW_EFFECTS = Spring(1, 800)
"""Colour and opacity of full-screen transitions."""

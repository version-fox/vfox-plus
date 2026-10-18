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

"""Domain models shared by every layer of the application.

The models mirror the concepts of the vfox CLI that are described in
`docs/vfox-cli-contract.md`. They are plain immutable values, so that they can
be compared in tests and rebuilt cheaply.
"""

from dataclasses import dataclass, field
from typing import Literal


@dataclass(frozen=True, slots=True)
class InstalledSdk:
    """An installed vfox plugin together with the SDK versions it manages.

    vfox does not distinguish between a plugin and the SDK it installs:
    `vfox add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24`
    installs a version of the `nodejs` SDK through that plugin.
    """

    name: str
    """Plugin name, for example `nodejs`."""
    versions: tuple[str, ...] = ()
    """Installed versions, in the order reported by vfox (newest first)."""
    current: str | None = None
    """The version that is currently in effect, or `None` if none is."""


@dataclass(frozen=True, slots=True)
class PluginInfo:
    """Metadata of an installed plugin, as reported by `vfox info <name>`."""

    name: str
    version: str
    """Version of the plugin itself (not of the SDK)."""
    homepage: str
    """Homepage of the plugin, usually a GitHub repository."""
    install_path: str
    """Directory in which the SDK versions of this plugin are installed."""
    description: str


@dataclass(frozen=True, slots=True)
class AvailablePlugin:
    """A plugin listed in the vfox plugin registry (`vfox available`)."""

    name: str
    """Plugin name that is passed to `vfox add`."""
    homepage: str
    official: bool
    """Whether the plugin is maintained by the version-fox organisation."""


@dataclass(frozen=True, slots=True)
class VersionAddition:
    """A tool that is bundled with an SDK version, such as `npm` for Node.js."""

    name: str
    version: str
    """Version of the bundled tool. May be empty when the plugin omits it."""

    def __str__(self) -> str:
        """The tool as shown to users: `npm 11.19.1`, or just `npm`."""
        return self.name if self.version == "" else f"{self.name} {self.version}"


@dataclass(frozen=True, slots=True)
class AvailableVersion:
    """A version that can be installed, as listed by `vfox search <name>`."""

    version: str
    """The version string to pass to `vfox install <name>@<version>`."""
    note: str | None = None
    """Optional remark from the plugin, for example `LTS`."""
    additions: tuple[VersionAddition, ...] = field(default=())
    """Tools bundled with this version."""
    installed: bool = False
    """Whether this version is already installed."""


@dataclass(frozen=True, slots=True)
class ConfigEntry:
    """One `key = value` line of `vfox config --list`."""

    key: str
    """Dotted configuration key, for example `proxy.url`."""
    value: str
    """Current value as printed by vfox. Empty when the value is unset."""


type UseScope = Literal["global", "project"]
"""Where a version selected with `vfox use` takes effect.

vfox also knows a `session` scope, which only exists inside an activated
shell. A desktop application has no shell session, so it is not offered.

- `global`: the default for the current user (`~/.vfox/.vfox.toml`);
- `project`: pinned in the `.vfox.toml` of a project directory.
"""

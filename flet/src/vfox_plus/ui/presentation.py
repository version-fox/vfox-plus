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

"""What the pages show, apart from how they show it: constants and functions
with the same values and results as in the other implementations.
"""

from urllib.parse import urlparse

from vfox_plus.core.i18n.localizations import Localizations
from vfox_plus.core.vfox.command import VfoxCommand, VfoxOperation

ACTIVATION_COMMAND = 'eval "$(vfox activate bash)"'
"""The line that activates vfox in Bash."""

TWO_PANE_WIDTH = 840
"""The narrowest SDKs page that shows the list and the detail side by
side."""

LIST_PANE_WIDTH = 360
"""The width of the list of the SDKs page next to the detail."""

WRAP_WIDTH = 560
"""The width below which wide trailing controls of list items, such as button
groups and text fields, move below the text."""

SEED_COLORS = (
    0xFFF28C1B,
    0xFFD9473B,
    0xFFC2407A,
    0xFF7E57C2,
    0xFF3F6FD8,
    0xFF0B8A9A,
    0xFF3E8E41,
    0xFF8D7B1E,
)
"""The seed colours offered when the desktop accent colour is not used, the
first being the orange of the vfox logo."""


class ConfigKeys:
    """The configuration keys of vfox that get a dedicated editor."""

    PROXY_ENABLE = "proxy.enable"
    PROXY_URL = "proxy.url"
    REGISTRY = "registry.address"
    STORAGE = "storage.sdkPath"
    LEGACY_ENABLE = "legacyVersionFile.enable"
    LEGACY_STRATEGY = "legacyVersionFile.strategy"
    CACHE = "cache.availableHookDuration"
    HTTP_TIMEOUT = "plugin.http.timeout"
    DOWNLOAD_TIMEOUT = "plugin.http.downloadTimeout"
    GITIGNORE = "gitignore.enable"


KNOWN_CONFIG_KEYS = frozenset(value for name, value in vars(ConfigKeys).items() if name.isupper())
"""The keys of :class:`ConfigKeys`; the other keys are listed as plain text
fields."""


def repository_path(homepage: str) -> str:
    """The repository path of a homepage, such as `version-fox/vfox-nodejs`."""
    segments = [segment for segment in urlparse(homepage).path.split("/") if segment]
    return "/".join(segments) if segments else homepage


def task_title(l10n: Localizations, command: VfoxCommand) -> str:
    """The localized title of a task that runs `command`."""
    subject = command.subject or ""
    match command.operation:
        case VfoxOperation.INSTALL:
            return l10n.t("taskInstall", subject=subject)
        case VfoxOperation.UNINSTALL:
            return l10n.t("taskUninstall", subject=subject)
        case VfoxOperation.USE:
            return l10n.t("taskUse", subject=subject)
        case VfoxOperation.UNUSE:
            return l10n.t("taskUnuse", subject=subject)
        case VfoxOperation.ADD_PLUGIN:
            return l10n.t("taskAddPlugin", subject=subject) if subject else l10n.t("taskAddPluginFromSource")
        case VfoxOperation.REMOVE_PLUGIN:
            return l10n.t("taskRemovePlugin", subject=subject)
        case VfoxOperation.UPDATE_PLUGIN:
            return l10n.t("taskUpdatePlugin", subject=subject)
        case VfoxOperation.UPDATE_ALL_PLUGINS:
            return l10n.t("taskUpdateAllPlugins")

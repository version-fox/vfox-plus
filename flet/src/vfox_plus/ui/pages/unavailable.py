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

"""Shown instead of the pages while vfox is missing or broken."""

import dataclasses

import flet as ft

from vfox_plus.state.controllers import VfoxAvailability
from vfox_plus.state.notifier import spawn
from vfox_plus.ui import links
from vfox_plus.ui.context import use_app
from vfox_plus.ui.hooks import use_listenable
from vfox_plus.ui.widgets.parts import (
    filled_button,
    loading_message,
    outlined_button,
    pane,
    status_message,
    tonal_button,
)


@ft.component
def UnavailablePage() -> ft.Control:
    """Explains how to install vfox, or that the executable is broken."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.vfox)
    vfox = model.vfox
    if vfox.availability in (VfoxAvailability.UNKNOWN, VfoxAvailability.CHECKING, VfoxAvailability.READY):
        return pane(loading_message(l10n.t("vfoxCheckingTitle")))

    async def locate() -> None:
        path = await app.services.pick_file("/usr/local/bin")
        if path is not None:
            model.settings.update(lambda settings: dataclasses.replace(settings, vfox_executable=path))

    missing = vfox.availability == VfoxAvailability.NOT_FOUND
    return pane(
        status_message(
            "notFound" if missing else "warning",
            l10n.t("vfoxMissingTitle") if missing else l10n.t("vfoxBrokenTitle"),
            body=l10n.t("vfoxMissingBody") if missing else l10n.t("vfoxBrokenBody", path=vfox.executable or "vfox"),
            shape="clover4Leaf" if missing else "softBurst",
            is_error=not missing,
            actions=[
                filled_button(
                    l10n.t("vfoxInstallGuide"),
                    lambda: spawn(app.services.open_url(links.VFOX_INSTALL)),
                    meaning="openInNew",
                ),
                tonal_button(l10n.t("vfoxLocate"), lambda: spawn(locate()), meaning="folder"),
                outlined_button(
                    l10n.t("actionRetry"),
                    lambda: spawn(vfox.apply(model.settings.settings)),
                    meaning="retry",
                ),
            ],
        )
    )

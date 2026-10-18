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

"""Runs the whole user interface in the test process: a Flet session whose
connection records the messages for the client instead of sending them, the
application on a scripted process runner, and the driver of the end-to-end
tests to use it."""

import dataclasses
from collections.abc import Sequence
from typing import Any

import flet as ft
from flet.controls.context import _context_page
from flet.messaging.connection import Connection
from flet.messaging.session import Session
from flet.pubsub.pubsub_hub import PubSubHub

from support import FakeProcessRunner
from vfox_plus.core.settings.app_settings import DEFAULT_SETTINGS, AppSettings, LanguagePreference
from vfox_plus.core.settings.settings_store import MemorySettingsStore
from vfox_plus.e2e.driver import Driver
from vfox_plus.ui.app import Application, create_application, start


class RecordingConnection(Connection):
    """A connection that keeps the messages for the client."""

    def __init__(self) -> None:
        super().__init__()
        self.pubsubhub = PubSubHub()
        self.messages: list[Any] = []

    def send_message(self, message: Any) -> None:
        self.messages.append(message)


class FakeServices:
    """Platform services that record what the application asked for."""

    def __init__(self) -> None:
        self.opened: list[str] = []
        self.copied: list[str] = []
        self.picked_directory: str | None = None
        self.picked_file: str | None = None
        self.accent: int | None = None

    async def accent_color(self) -> int | None:
        return self.accent

    async def open_url(self, url: str) -> None:
        self.opened.append(url)

    async def pick_directory(self, initial_directory: str | None = None) -> str | None:
        return self.picked_directory

    async def pick_file(self, initial_directory: str | None, extensions: Sequence[str] = ()) -> str | None:
        return self.picked_file

    async def copy_text(self, text: str) -> None:
        self.copied.append(text)


@dataclasses.dataclass
class Running:
    """A running application in a test."""

    application: Application
    driver: Driver
    services: FakeServices
    settings: MemorySettingsStore
    connection: RecordingConnection
    session: Session
    """Kept here: the page refers to its session weakly only."""

    @property
    def page(self) -> ft.Page:
        return self.application.page


async def launch(
    runner: FakeProcessRunner,
    *,
    settings: AppSettings | None = None,
    environment: dict[str, str] | None = None,
) -> Running:
    """Starts the application on `runner` in a recording session."""
    connection = RecordingConnection()
    session = Session(connection)
    page = session.page
    # Flet binds the page of the session before it calls the main function.
    _context_page.set(page)
    store = MemorySettingsStore(
        settings or dataclasses.replace(DEFAULT_SETTINGS, language=LanguagePreference.ENGLISH, vfox_executable="vfox")
    )
    services = FakeServices()
    application = create_application(
        page,
        environment=environment or {"HOME": "/home/user", "PATH": "/usr/bin"},
        runner=runner,
        settings_store=store,
        services=services,
    )
    await start(application)
    # The client receives the whole page on connect, which indexes and
    # mounts the rendered controls.
    session.get_page_patch()
    driver = Driver(page)
    await driver.settle()
    return Running(application, driver, services, store, connection, session)

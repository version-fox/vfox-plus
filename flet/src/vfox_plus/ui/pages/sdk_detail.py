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

"""Everything about one installed plugin: its metadata, its installed
versions and the versions that can be installed.
"""

from collections.abc import Callable
from enum import StrEnum

import flet as ft

from vfox_plus.core.models import AvailableVersion, InstalledSdk, PluginInfo
from vfox_plus.core.vfox.command import VfoxOperation
from vfox_plus.state.controllers import SdkDetailController, VersionFilter
from vfox_plus.state.notifier import spawn
from vfox_plus.state.task_queue import TaskStatus, VfoxTask
from vfox_plus.ui.context import AppContext, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.presentation import WRAP_WIDTH, task_title
from vfox_plus.ui.tokens import Gaps
from vfox_plus.ui.widgets.buttons import MenuEntry, Segment, SplitButton, button_group, overflow_menu
from vfox_plus.ui.widgets.dialogs import ask_project_directory, confirm
from vfox_plus.ui.widgets.lists import ListPosition, list_item
from vfox_plus.ui.widgets.parts import (
    error_message,
    filled_button,
    icon_button,
    loading_message,
    status_message,
    tag,
    text,
    text_field,
    tonal_button,
)
from vfox_plus.ui.widgets.virtual_list import Row, VirtualList
from vfox_plus_m3e import LoadingIndicator, ShapeAvatar, WavyProgressIndicator

C = ft.Colors
S = ft.TextThemeStyle

_INSTALLED_HEIGHT = 64
_AVAILABLE_HEIGHT = 60
_NARROW_HEADER = 520


class _Tab(StrEnum):
    INSTALLED = "installed"
    AVAILABLE = "available"


@ft.component
def SdkDetail(name: str, on_back: Callable[[], object] | None = None, key: str | None = None) -> ft.Control:
    """The detail of the installed plugin `name`; `on_back` shows a back
    button.
    """
    del key
    app = use_app()
    model = app.model
    l10n = app.l10n
    controller = ft.use_memo(lambda: model.sdk_detail(name), [name])
    tab, set_tab = ft.use_state(_Tab.INSTALLED)
    width, on_size_change = use_width()

    def mount() -> None:
        controller.info.load()

    def reload_info(task: VfoxTask) -> None:
        if task.command.sdk == name and task.command.operation == VfoxOperation.UPDATE_PLUGIN:
            controller.info.load()

    stop: ft.MutableRef[Callable[[], None]] = ft.use_ref(lambda: model.tasks.on_finished(reload_info))

    def unmount() -> None:
        if stop.current is not None:
            stop.current()
        controller.dispose()

    ft.use_effect(mount, [], cleanup=unmount)
    use_listenable(model.installed, model.tasks, controller, controller.info)
    sdk = model.installed_sdk(name) or InstalledSdk(name)
    pane_width = width or 900
    wrap = pane_width - 2 * Gaps.XL < WRAP_WIDTH

    def show(new_tab: _Tab) -> None:
        set_tab(new_tab)
        if new_tab == _Tab.AVAILABLE:
            spawn(controller.versions.ensure_loaded())

    header: list[ft.Control] = [
        _header(app, sdk, controller.info.value, on_back, narrow=pane_width < _NARROW_HEADER),
        ft.Container(
            padding=ft.Padding.symmetric(horizontal=Gaps.XL),
            content=ft.Row(
                [
                    button_group(
                        [
                            Segment(
                                _Tab.INSTALLED,
                                f"{l10n.t('sdkTabInstalled')} ({len(sdk.versions)})",
                                "versions",
                                key="tab-installed",
                            ),
                            Segment(_Tab.AVAILABLE, l10n.t("sdkTabAvailable"), "install", key="tab-available"),
                        ],
                        tab,
                        show,
                    )
                ]
            ),
        ),
        ft.Container(height=Gaps.L),
    ]
    rows: list[Row] = []
    if tab == _Tab.INSTALLED:
        if not sdk.versions:
            header.append(
                status_message(
                    "install",
                    l10n.t("sdkInstalledEmpty", name=sdk.name),
                    shape="pill",
                    actions=[
                        filled_button(l10n.t("sdkBrowseVersions"), lambda: show(_Tab.AVAILABLE), meaning="install")
                    ],
                )
            )

        def installed_row(index: int, version: str) -> Row:
            position = ListPosition.of(index, len(sdk.versions))
            return Row(
                f"installed-{version}",
                _INSTALLED_HEIGHT + (40 if wrap else 0),
                lambda: _installed_item(app, sdk, version, position, wrap=wrap),
            )

        rows = [installed_row(index, version) for index, version in enumerate(sdk.versions)]
    else:
        installed = set(sdk.versions)
        visible = controller.visible_versions(installed)
        if not controller.versions.has_value:
            error = controller.versions.error
            header.append(
                ft.Container(
                    height=320,
                    content=error_message(
                        l10n.t("availableError", name=sdk.name), error, l10n.t("actionRetry"), controller.versions.load
                    )
                    if error is not None
                    else loading_message(l10n.t("availableLoading")),
                )
            )
        else:
            header.append(_toolbar(app, controller, len(visible)))
            if not visible:
                header.append(status_message("notFound", l10n.t("availableEmpty")))

            def available_row(index: int, version: AvailableVersion) -> Row:
                position = ListPosition.of(index, len(visible))
                return Row(
                    f"available-{version.version}",
                    _AVAILABLE_HEIGHT + (48 if wrap else 0),
                    lambda: _available_item(app, sdk.name, version, version.version in installed, position, wrap=wrap),
                )

            rows = [available_row(index, version) for index, version in enumerate(visible)]
    return ft.Container(
        content=VirtualList(
            header,
            rows,
            spacing=Gaps.HAIRLINE,
            padding=ft.Padding.only(bottom=Gaps.XL),
            key=f"{name}-{tab}",
        ),
        expand=True,
        on_size_change=on_size_change,
        size_change_interval=100,
    )


def _header(
    app: AppContext, sdk: InstalledSdk, info: PluginInfo | None, on_back: Callable[[], object] | None, *, narrow: bool
) -> ft.Container:
    model = app.model
    l10n = app.l10n
    updating = model.tasks.active_task(VfoxOperation.UPDATE_PLUGIN, sdk.name)
    tags: list[ft.Control] = [
        tag(
            l10n.t("sdksCurrent", version=sdk.current) if sdk.current is not None else l10n.t("sdksNoCurrent"),
            meaning="current" if sdk.current is not None else None,
            emphasized=sdk.current is not None,
        )
    ]
    if info is not None and info.version:
        tags.append(tag(l10n.t("sdkPluginVersion", version=info.version)))
    texts: list[ft.Control] = [
        text(sdk.name, S.HEADLINE_MEDIUM, color=C.ON_SURFACE, emphasized=True, max_lines=1),
        ft.Row(tags, spacing=Gaps.S, run_spacing=Gaps.XS, wrap=True),
    ]
    if info is not None and info.description:
        texts.append(text(info.description, S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT, max_lines=3))
    back = icon_button("back", l10n.t("actionClose"), on_back, key="sdk-back") if on_back is not None else None

    async def remove() -> None:
        confirmed = await confirm(
            app,
            title=l10n.t("sdkRemoveTitle", name=sdk.name),
            body=l10n.t("sdkRemoveBody", count=len(sdk.versions)),
            confirm_label=l10n.t("sdkRemoveConfirm"),
            meaning="removePlugin",
            destructive=True,
        )
        if confirmed:
            model.run(model.cli.remove_plugin(sdk.name))
            app.shell.show_sdk(None)

    entries = []
    if info is not None and info.homepage:
        homepage = info.homepage
        entries.append(
            MenuEntry(l10n.t("actionOpenHomepage"), lambda: spawn(app.services.open_url(homepage)), "openInNew")
        )
    entries.append(
        MenuEntry(l10n.t("sdkRemovePlugin"), lambda: spawn(remove()), "removePlugin", destructive=True, key="remove")
    )
    actions: list[ft.Control] = [
        ft.Container(content=LoadingIndicator(size=40), padding=Gaps.XS)
        if updating is not None
        else icon_button(
            "update", l10n.t("sdkUpdatePlugin"), lambda: model.run(model.cli.update_plugin(sdk.name)), key="update"
        ),
        overflow_menu(l10n.t("sdkMoreActions"), entries, key="sdk-menu"),
    ]
    column = ft.Column(texts, spacing=Gaps.XS, expand=True)
    if narrow:
        # Narrow panes put the buttons on their own row above the name.
        content: ft.Control = ft.Column(
            [
                ft.Row([*([back] if back is not None else []), ft.Container(expand=True), *actions]),
                ft.Row(
                    [ft.Container(width=Gaps.S), ShapeAvatar(name=sdk.name, size=56), column],
                    spacing=Gaps.L,
                    vertical_alignment=ft.CrossAxisAlignment.START,
                ),
            ],
            spacing=Gaps.S,
        )
    else:
        content = ft.Row(
            [
                *([back] if back is not None else [ft.Container(width=Gaps.S)]),
                ShapeAvatar(name=sdk.name, size=72),
                column,
                *actions,
            ],
            spacing=Gaps.L,
            vertical_alignment=ft.CrossAxisAlignment.START,
        )
    return ft.Container(
        content=content, padding=ft.Padding.only(left=Gaps.L, top=Gaps.XL, right=Gaps.L, bottom=Gaps.XL)
    )


def _installed_item(
    app: AppContext, sdk: InstalledSdk, version: str, position: ListPosition, *, wrap: bool
) -> ft.Control:
    model = app.model
    l10n = app.l10n
    current = sdk.current == version
    task = model.tasks.active_task(sdk=sdk.name, version=version)

    async def use_in_project() -> None:
        directory = await ask_project_directory(app, name=sdk.name, version=version)
        if directory is not None:
            model.run(model.cli.use(sdk.name, version, "project", directory))

    async def copy_path() -> None:
        path = await model.cli.install_path(sdk.name, version)
        if path is not None:
            await app.services.copy_text(path)
            app.messenger.snack(l10n.t("actionCopied"))

    async def uninstall() -> None:
        confirmed = await confirm(
            app,
            title=l10n.t("versionUninstallTitle", name=sdk.name, version=version),
            body=l10n.t("versionUninstallCurrentBody") if current else l10n.t("versionUninstallBody"),
            confirm_label=l10n.t("versionUninstall"),
            meaning="uninstall",
            destructive=True,
        )
        if confirmed:
            model.run(model.cli.uninstall(sdk.name, version))

    if task is not None:
        actions: ft.Control = LoadingIndicator(size=40)
    else:
        menu = [
            *(
                [
                    MenuEntry(l10n.t("versionUseInProject"), lambda: spawn(use_in_project()), "projectScope"),
                    MenuEntry(l10n.t("versionUnuse"), lambda: model.run(model.cli.unuse(sdk.name)), "cancel"),
                ]
                if current
                else []
            ),
            MenuEntry(l10n.t("versionCopyPath"), lambda: spawn(copy_path()), "copy"),
            MenuEntry(l10n.t("versionUninstall"), lambda: spawn(uninstall()), "uninstall", destructive=True),
        ]
        actions = ft.Row(
            [
                tag(l10n.t("versionInUse"), meaning="current", emphasized=True)
                if current
                else SplitButton(
                    l10n.t("versionUse"),
                    lambda: model.run(model.cli.use(sdk.name, version)),
                    [
                        MenuEntry(
                            l10n.t("versionUseGlobally"),
                            lambda: model.run(model.cli.use(sdk.name, version)),
                            "globalScope",
                        ),
                        MenuEntry(l10n.t("versionUseInProject"), lambda: spawn(use_in_project()), "projectScope"),
                    ],
                    l10n.t("versionUseOptions"),
                    meaning="check",
                    key_prefix=f"use-{version}-",
                ),
                overflow_menu(l10n.t("versionMoreActions", version=version), menu, key=f"version-menu-{version}"),
            ],
            spacing=Gaps.XS,
            tight=True,
        )
    return ft.Container(
        padding=ft.Padding.symmetric(horizontal=Gaps.XL),
        content=list_item(
            ft.Text(
                version,
                theme_style=S.BODY_LARGE,
                color=C.ON_SURFACE,
                weight=ft.FontWeight.W_500 if current else None,
            ),
            position=position,
            leading=icon_control(
                "current" if current else "versions",
                filled=current,
                color=C.PRIMARY if current else C.ON_SURFACE_VARIANT,
            ),
            trailing=actions,
            wrap_trailing=wrap,
            key=f"installed-item-{version}",
        ),
    )


def _available_item(
    app: AppContext, sdk: str, version: AvailableVersion, installed: bool, position: ListPosition, *, wrap: bool
) -> ft.Control:
    model = app.model
    l10n = app.l10n
    task = model.tasks.active_task(VfoxOperation.INSTALL, sdk, version.version)
    additions = " · ".join(str(addition) for addition in version.additions)
    if task is not None:
        trailing: ft.Control = InstallProgress(task)
    elif installed:
        trailing = ft.TextButton(
            content=l10n.t("versionInstalled"), icon=icon("check"), disabled=True, key=f"installed-{version.version}"
        )
    else:
        trailing = tonal_button(
            l10n.t("versionInstall"),
            lambda: model.run(model.cli.install(sdk, version.version)),
            meaning="install",
            key=f"install-{version.version}",
        )
    title: list[ft.Control] = [ft.Text(version.version, theme_style=S.BODY_LARGE, color=C.ON_SURFACE)]
    if version.note is not None:
        title.append(tag(version.note, emphasized=True))
    return ft.Container(
        padding=ft.Padding.symmetric(horizontal=Gaps.XL),
        content=list_item(
            ft.Row(title, spacing=Gaps.S, wrap=True),
            position=position,
            subtitle=additions or None,
            trailing=trailing,
            wrap_trailing=wrap,
            padding=ft.Padding.symmetric(horizontal=Gaps.L, vertical=Gaps.S),
        ),
    )


@ft.component
def InstallProgress(task: VfoxTask) -> ft.Control:
    """The wavy progress of an installation, with its status and a cancel
    button.
    """
    app = use_app()
    l10n = app.l10n
    use_listenable(task)
    progress = task.progress
    if progress is None:
        label = l10n.t("taskStatusQueued") if task.status == TaskStatus.QUEUED else l10n.t("taskStatusRunning")
    else:
        label = l10n.t("taskDownloading", percent=round(progress * 100))
    return ft.Container(
        width=220,
        content=ft.Row(
            [
                ft.Column(
                    [
                        WavyProgressIndicator(value=progress, semantics_label=task_title(l10n, task.command)),
                        text(label, S.LABEL_SMALL, color=C.ON_SURFACE_VARIANT),
                    ],
                    spacing=Gaps.XS,
                    tight=True,
                    expand=True,
                ),
                icon_button("cancel", l10n.t("actionCancel"), lambda: app.model.tasks.cancel(task), compact=True),
            ],
            spacing=0,
        ),
    )


def _toolbar(app: AppContext, controller: SdkDetailController, count: int) -> ft.Container:
    l10n = app.l10n

    def set_query(value: str) -> None:
        controller.query = value

    def set_filter(value: VersionFilter) -> None:
        controller.version_filter = value

    return ft.Container(
        padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.M),
        content=ft.Row(
            [
                ft.Container(
                    width=280,
                    content=text_field(
                        hint=l10n.t("availableSearchHint"),
                        value=controller.query,
                        prefix="search",
                        dense=True,
                        on_change=set_query,
                        key="available-filter",
                    ),
                ),
                button_group(
                    [
                        Segment(VersionFilter.ALL, l10n.t("availableFilterAll"), key="filter-all"),
                        Segment(VersionFilter.WITH_NOTE, l10n.t("availableFilterTagged"), key="filter-tagged"),
                        Segment(VersionFilter.INSTALLED, l10n.t("availableFilterInstalled"), key="filter-installed"),
                    ],
                    controller.version_filter,
                    set_filter,
                ),
                text(l10n.t("availableCount", count=count), S.LABEL_LARGE, color=C.ON_SURFACE_VARIANT),
            ],
            spacing=Gaps.M,
            run_spacing=Gaps.S,
            wrap=True,
            vertical_alignment=ft.CrossAxisAlignment.CENTER,
        ),
    )

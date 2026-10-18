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

"""Running, queued and finished operations with their output."""

import math

import flet as ft

from vfox_plus.core.vfox.command import VfoxOperation
from vfox_plus.state.notifier import spawn
from vfox_plus.state.task_queue import TaskStatus, VfoxTask
from vfox_plus.ui import motion
from vfox_plus.ui.context import AppContext, use_app
from vfox_plus.ui.hooks import use_listenable
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.presentation import task_title
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus.ui.widgets.lists import ListPosition, list_item
from vfox_plus.ui.widgets.parts import icon_button, page_header, pane, status_message, text
from vfox_plus_m3e import ConstrainedBox, LoadingIndicator, WavyProgressIndicator

C = ft.Colors
S = ft.TextThemeStyle


@ft.component
def TasksPage() -> ft.Control:
    """The Tasks page."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.tasks)
    tasks = model.tasks.tasks
    header = page_header(
        l10n.t("navTasks"),
        actions=[
            ft.TextButton(
                content=l10n.t("tasksClearFinished"),
                icon=icon("succeeded"),
                on_click=lambda _: model.tasks.clear_finished(),
                disabled=not model.tasks.has_finished,
                key="clear-finished",
            )
        ],
    )
    if not tasks:
        return pane(
            ft.Column(
                [
                    header,
                    status_message("tasks", l10n.t("tasksEmptyTitle"), body=l10n.t("tasksEmptyBody"), shape="sunny"),
                ],
                spacing=0,
                expand=True,
            )
        )
    return pane(
        ft.Column(
            [
                header,
                ft.Container(
                    padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.XL),
                    content=ft.Column(
                        [
                            TaskItem(task, ListPosition.of(index, len(tasks)), key=f"task-{task.id}")
                            for index, task in enumerate(tasks)
                        ],
                        spacing=Gaps.HAIRLINE,
                        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
                    ),
                ),
            ],
            spacing=0,
            scroll=ft.ScrollMode.AUTO,
            expand=True,
        )
    )


@ft.component
def TaskItem(task: VfoxTask, position: ListPosition, key: str | None = None) -> ft.Control:
    """A task: its status, its title and its actions, and its output when
    expanded.
    """
    del key
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(task)
    expanded, set_expanded = ft.use_state(False)
    status = task.status
    match status:
        case TaskStatus.QUEUED:
            leading: ft.Control = icon_control("queued", color=C.ON_SURFACE_VARIANT)
            status_text = l10n.t("taskStatusQueued")
        case TaskStatus.RUNNING:
            leading = LoadingIndicator(size=32)
            status_text = (
                l10n.t("taskStatusRunning")
                if task.progress is None
                else l10n.t("taskDownloading", percent=round(task.progress * 100))
            )
        case TaskStatus.SUCCEEDED:
            leading = icon_control("succeeded", color=C.PRIMARY, filled=True)
            status_text = l10n.t("taskStatusSucceeded")
        case TaskStatus.FAILED:
            leading = icon_control("error", color=C.ERROR, filled=True)
            status_text = f"{l10n.t('taskStatusFailed')} · {task.error_message or ''}"
        case TaskStatus.CANCELLED:
            leading = icon_control("cancel", color=C.ON_SURFACE_VARIANT)
            status_text = l10n.t("taskStatusCancelled")
    subtitle: list[ft.Control] = [
        ft.Text(
            status_text,
            theme_style=S.BODY_MEDIUM,
            color=C.ERROR if status == TaskStatus.FAILED else C.ON_SURFACE_VARIANT,
            max_lines=2,
            overflow=ft.TextOverflow.ELLIPSIS,
        )
    ]
    if status == TaskStatus.RUNNING and task.command.operation == VfoxOperation.INSTALL:
        subtitle.append(
            ft.Container(
                padding=ft.Padding.only(top=Gaps.S),
                content=ConstrainedBox(max_width=480, content=WavyProgressIndicator(value=task.progress)),
            )
        )
    actions: list[ft.Control] = []
    if status.is_active:
        actions.append(icon_button("cancel", l10n.t("actionCancel"), lambda: model.tasks.cancel(task), compact=True))
    elif status in (TaskStatus.FAILED, TaskStatus.CANCELLED):
        actions.append(icon_button("retry", l10n.t("actionRetry"), lambda: model.tasks.retry(task), compact=True))

    def toggle() -> None:
        set_expanded(lambda value: not value)

    toggle_button = icon_button(
        "expandMore", l10n.t("taskHideOutput") if expanded else l10n.t("taskShowOutput"), toggle, compact=True
    )
    toggle_button.rotate = math.pi if expanded else 0
    toggle_button.animate_rotation = ft.Animation(motion.FAST_SPATIAL.settle_duration, ft.AnimationCurve.EASE_OUT_BACK)
    actions.append(toggle_button)

    shown_position = position
    if expanded and position == ListPosition.LAST:
        shown_position = ListPosition.MIDDLE
    elif expanded and position == ListPosition.SINGLE:
        shown_position = ListPosition.FIRST
    item = list_item(
        ft.Text(task_title(l10n, task.command), theme_style=S.BODY_LARGE, weight=ft.FontWeight.W_500),
        position=shown_position,
        leading=ft.Container(content=leading, width=32, height=32, alignment=ft.Alignment.CENTER),
        subtitle=ft.Column(subtitle, spacing=0, tight=True),
        trailing=ft.Row(actions, spacing=0, tight=True),
        on_click=toggle,
    )
    output = _output(app, task, is_last=position in (ListPosition.LAST, ListPosition.SINGLE)) if expanded else None
    return ft.Column(
        [
            item,
            ft.Container(
                content=output,
                padding=ft.Padding.only(top=Gaps.HAIRLINE) if output is not None else None,
                animate_size=ft.Animation(motion.DEFAULT_SPATIAL.settle_duration, ft.AnimationCurve.EASE_OUT_CUBIC),
            ),
        ],
        spacing=0,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
    )


def _output(app: AppContext, task: VfoxTask, *, is_last: bool) -> ft.Control:
    l10n = app.l10n
    lines = task.log
    command = f"$ {task.command.command_line}"
    full = "\n".join([command, *lines])

    async def copy() -> None:
        await app.services.copy_text(full)
        app.messenger.snack(l10n.t("actionCopied"))

    bottom = Corners.LARGE_INCREASED if is_last else Corners.EXTRA_SMALL
    return ft.Container(
        bgcolor=C.SURFACE_CONTAINER_HIGHEST,
        border_radius=ft.BorderRadius(
            top_left=Corners.EXTRA_SMALL, top_right=Corners.EXTRA_SMALL, bottom_left=bottom, bottom_right=bottom
        ),
        padding=ft.Padding.only(left=Gaps.L, top=Gaps.S, right=Gaps.XS, bottom=Gaps.M),
        content=ft.Row(
            [
                ft.Container(
                    expand=True,
                    content=ConstrainedBox(
                        max_height=320,
                        content=ft.Column(
                            [
                                text(
                                    f"{command}\n{l10n.t('taskNoOutput')}" if not lines else full,
                                    S.BODY_SMALL,
                                    selectable=True,
                                    font_family="monospace",
                                    height=1.5,
                                )
                            ],
                            scroll=ft.ScrollMode.AUTO,
                            auto_scroll=True,
                            tight=True,
                        ),
                    ),
                ),
                icon_button("copy", l10n.t("taskCopyOutput"), lambda: spawn(copy()), compact=True),
            ],
            vertical_alignment=ft.CrossAxisAlignment.START,
        ),
    )

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

"""Settings of the application and of vfox."""

import dataclasses
from collections.abc import Awaitable, Callable

import flet as ft

from vfox_plus import APP_VERSION
from vfox_plus.core.settings.app_settings import (
    AppSettings,
    LanguagePreference,
    ThemePreference,
    format_environment_lines,
    parse_environment_lines,
)
from vfox_plus.state.controllers import VfoxAvailability
from vfox_plus.state.notifier import spawn
from vfox_plus.ui import links
from vfox_plus.ui.context import AppContext, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.licenses import LicenseEntry, bundled_licenses
from vfox_plus.ui.presentation import KNOWN_CONFIG_KEYS, SEED_COLORS, WRAP_WIDTH, ConfigKeys
from vfox_plus.ui.theme import color_hex
from vfox_plus.ui.tokens import Gaps
from vfox_plus.ui.widgets.buttons import Segment, button_group
from vfox_plus.ui.widgets.lists import ListPosition, list_group, list_item
from vfox_plus.ui.widgets.parts import (
    content_width,
    describe_error,
    error_message,
    filled_button,
    group_header,
    icon_button,
    loading_message,
    page_header,
    pane,
    text,
    text_button,
    text_field,
    tonal_button,
)
from vfox_plus_m3e import ColorSwatch, ConstrainedBox, ExpressiveDialog, LoadingIndicator, ShapeAvatar

C = ft.Colors
S = ft.TextThemeStyle

_SPACER = ft.Container(width=24)


@ft.component
def SettingsPage() -> ft.Control:
    """The Settings page."""
    app = use_app()
    l10n = app.l10n
    width, on_size_change = use_width()
    wrap = (min(width or 900, 840 + 2 * Gaps.XL) - 2 * Gaps.XL - 2 * Gaps.L) < WRAP_WIDTH
    return pane(
        ft.Container(
            on_size_change=on_size_change,
            size_change_interval=100,
            expand=True,
            content=ft.Column(
                [
                    page_header(l10n.t("navSettings")),
                    ft.Container(
                        padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.XXL),
                        content=content_width(
                            ft.Column(
                                [
                                    AppearanceSection(wrap),
                                    VfoxSection(wrap),
                                    VfoxConfigSection(wrap),
                                    _about(app),
                                ],
                                spacing=0,
                                horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
                            ),
                            max_width=840,
                        ),
                    ),
                ],
                spacing=0,
                scroll=ft.ScrollMode.AUTO,
                expand=True,
            ),
        )
    )


@ft.component
def AppearanceSection(wrap: bool) -> ft.Control:
    """Theme, colours and language."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.settings)
    settings = model.settings.settings

    def update(change: Callable[[AppSettings], AppSettings]) -> None:
        model.settings.update(change)

    def theme(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("settingsTheme"),
            position=position,
            leading="darkMode",
            wrap_trailing=wrap,
            trailing=button_group(
                [
                    Segment(ThemePreference.SYSTEM, l10n.t("themeSystem"), "systemMode", key="theme-system"),
                    Segment(ThemePreference.LIGHT, l10n.t("themeLight"), "lightMode", key="theme-light"),
                    Segment(ThemePreference.DARK, l10n.t("themeDark"), "darkMode", key="theme-dark"),
                ],
                settings.theme,
                lambda value: update(lambda current: dataclasses.replace(current, theme=value)),
            ),
        )

    def system_color(position: ListPosition) -> ft.Control:
        def toggle() -> None:
            update(lambda current: dataclasses.replace(current, use_system_color=not current.use_system_color))

        return list_item(
            l10n.t("settingsSystemColor"),
            position=position,
            leading="palette",
            trailing=ft.Switch(value=settings.use_system_color, on_change=lambda _: toggle(), key="system-color"),
            on_click=toggle,
        )

    def colors(position: ListPosition) -> ft.Control:
        def choose(seed: int) -> None:
            update(lambda current: dataclasses.replace(current, seed_color=seed, use_system_color=False))

        return list_item(
            l10n.t("settingsColor"),
            position=position,
            leading=_SPACER,
            subtitle=ft.Container(
                padding=ft.Padding.only(top=Gaps.S),
                content=ft.Row(
                    [
                        ColorSwatch(
                            seed=color_hex(seed),
                            selected=not settings.use_system_color and settings.seed_color == seed,
                            label=l10n.t("settingsColorOption", index=index + 1),
                            check_icon=icon("check"),
                            on_click=lambda _, seed=seed: choose(seed),
                            key=f"swatch-{index + 1}",
                        )
                        for index, seed in enumerate(SEED_COLORS)
                    ],
                    spacing=Gaps.S,
                    run_spacing=Gaps.S,
                    wrap=True,
                ),
            ),
        )

    def language(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("settingsLanguage"),
            position=position,
            leading="language",
            wrap_trailing=wrap,
            trailing=button_group(
                [
                    Segment(LanguagePreference.SYSTEM, l10n.t("languageSystem"), key="language-system"),
                    # Language names are always shown in their own language.
                    Segment(LanguagePreference.ENGLISH, "English", key="language-english"),
                    Segment(LanguagePreference.SIMPLIFIED_CHINESE, "简体中文", key="language-chinese"),
                ],
                settings.language,
                lambda value: update(lambda current: dataclasses.replace(current, language=value)),
            ),
        )

    return ft.Column(
        [group_header(l10n.t("settingsAppearance")), list_group([theme, system_color, colors, language])],
        spacing=0,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
    )


@ft.component
def VfoxSection(wrap: bool) -> ft.Control:
    """The vfox executable and the environment of vfox."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.vfox, model.settings)
    vfox = model.vfox
    stored_environment: str = format_environment_lines(model.settings.settings.extra_environment)
    environment, set_environment = ft.use_state(stored_environment)
    initial_error: str | None = None
    error, set_error = ft.use_state(initial_error)

    async def choose_executable() -> None:
        path = await app.services.pick_file("/usr/local/bin")
        if path is not None:
            model.settings.update(lambda current: dataclasses.replace(current, vfox_executable=path))

    def detect() -> None:
        model.settings.update(lambda current: dataclasses.replace(current, vfox_executable=None))

    def save_environment() -> None:
        parsed = parse_environment_lines(environment)
        if parsed.invalid_lines:
            set_error(
                l10n.t(
                    "settingsEnvironmentInvalid",
                    count=len(parsed.invalid_lines),
                    lines=", ".join(str(line) for line in parsed.invalid_lines),
                )
            )
            return
        set_error(None)
        model.settings.update(lambda current: dataclasses.replace(current, extra_environment=parsed.variables))
        app.messenger.snack(l10n.t("settingsSaved"))

    if vfox.executable is None:
        source = l10n.t("settingsExecutableMissing")
    elif vfox.is_custom_executable:
        source = l10n.t("settingsExecutableCustom")
    else:
        source = l10n.t("settingsExecutableDetected")
    details = [vfox.executable or "vfox", source]
    if vfox.version is not None:
        details.append(l10n.t("settingsVfoxVersion", version=vfox.version))
    actions: list[ft.Control] = []
    if vfox.availability == VfoxAvailability.CHECKING:
        actions.append(LoadingIndicator(size=36))
    elif vfox.is_custom_executable:
        actions.append(icon_button("refresh", l10n.t("settingsExecutableDetect"), detect, compact=True))
    actions.append(tonal_button(l10n.t("actionBrowse"), lambda: spawn(choose_executable()), key="executable-browse"))

    def executable(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("settingsExecutable"),
            position=position,
            leading="terminal",
            subtitle=" · ".join(details),
            wrap_trailing=wrap,
            trailing=ft.Row(actions, spacing=Gaps.XS, tight=True),
        )

    def variables(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("settingsEnvironment"),
            position=position,
            leading="environment",
            subtitle=ft.Container(
                padding=ft.Padding.only(top=Gaps.S),
                content=ft.Column(
                    [
                        text(l10n.t("settingsEnvironmentHelp"), S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT),
                        ft.Container(height=Gaps.XS),
                        text_field(
                            value=environment,
                            hint="VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node",
                            error=error,
                            multiline=True,
                            monospace=True,
                            on_change=set_environment,
                            key="environment",
                        ),
                        ft.Row(
                            [filled_button(l10n.t("actionSave"), save_environment, key="environment-save")],
                            alignment=ft.MainAxisAlignment.END,
                        ),
                    ],
                    spacing=Gaps.S,
                    horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
                ),
            ),
        )

    return ft.Column(
        [group_header(l10n.t("settingsVfox")), list_group([executable, variables])],
        spacing=0,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
    )


async def _save_config(app: AppContext, save: Callable[[], Awaitable[None]]) -> None:
    """Saves a vfox setting and reports the result in a snack bar."""
    try:
        await save()
        app.messenger.snack(app.l10n.t("settingsSaved"))
    except Exception as error:  # noqa: BLE001 - vfox rejected the value; the reload shows the stored one.
        app.messenger.snack(describe_error(error))


@ft.component
def VfoxConfigSection(wrap: bool) -> ft.Control:
    """An editor for each key of the vfox configuration."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    config = model.vfox_config
    use_listenable(config, model.vfox)
    ready = model.vfox.availability == VfoxAvailability.READY

    def load() -> None:
        if ready and not config.has_value and not config.is_loading:
            config.load()

    ft.use_effect(load, [ready])
    children: list[ft.Control] = []
    if not ready:
        pass
    elif not config.has_value:
        children.append(
            ft.Container(
                height=240,
                content=error_message(
                    l10n.t("settingsVfoxConfigError"), config.error, l10n.t("actionRetry"), config.load
                )
                if config.error is not None
                else loading_message(semantics_label=l10n.t("loading")),
            )
        )
    else:
        other = [entry.key for entry in config.value or () if entry.key not in KNOWN_CONFIG_KEYS]

        def switch(
            key: str, meaning: str | None, title: str, subtitle: str | None = None
        ) -> Callable[[ListPosition], ft.Control]:
            return lambda position: ConfigSwitch(key, meaning, title, subtitle, position, key=f"config-{key}")

        def field(
            key: str,
            meaning: str | None,
            title: str,
            *,
            hint: str | None = None,
            help_text: str | None = None,
            directory: bool = False,
        ) -> Callable[[ListPosition], ft.Control]:
            return lambda position: ConfigText(
                key, meaning, title, hint, help_text, directory, position, wrap, key=f"config-{key}"
            )

        def choice(position: ListPosition) -> ft.Control:
            return ConfigChoice(
                ConfigKeys.LEGACY_STRATEGY,
                l10n.t("configLegacyStrategy"),
                {
                    "specified": l10n.t("strategySpecified"),
                    "latest_installed": l10n.t("strategyLatestInstalled"),
                    "latest_available": l10n.t("strategyLatestAvailable"),
                },
                position,
                wrap,
                key="config-strategy",
            )

        children += [
            list_group(
                [
                    switch(ConfigKeys.PROXY_ENABLE, "proxy", l10n.t("configProxyEnable")),
                    field(ConfigKeys.PROXY_URL, None, l10n.t("configProxyUrl"), hint="http://127.0.0.1:7890"),
                    field(ConfigKeys.REGISTRY, "registry", l10n.t("configRegistry"), hint=l10n.t("configRegistryHint")),
                    field(
                        ConfigKeys.STORAGE,
                        "storage",
                        l10n.t("configStorage"),
                        hint=l10n.t("configStorageHint"),
                        directory=True,
                    ),
                ]
            ),
            ft.Container(height=Gaps.L),
            list_group(
                [
                    switch(
                        ConfigKeys.LEGACY_ENABLE,
                        "legacyFiles",
                        l10n.t("configLegacyEnable"),
                        l10n.t("configLegacyEnableHelp"),
                    ),
                    choice,
                    switch(ConfigKeys.GITIGNORE, "gitignore", l10n.t("configGitignore"), l10n.t("configGitignoreHelp")),
                ]
            ),
            ft.Container(height=Gaps.L),
            list_group(
                [
                    field(
                        ConfigKeys.CACHE,
                        "cache",
                        l10n.t("configCache"),
                        hint="12h",
                        help_text=l10n.t("configCacheHelp"),
                    ),
                    field(
                        ConfigKeys.HTTP_TIMEOUT,
                        "timeout",
                        l10n.t("configHttpTimeout"),
                        hint="30s",
                        help_text=l10n.t("configTimeoutHelp"),
                    ),
                    field(
                        ConfigKeys.DOWNLOAD_TIMEOUT,
                        None,
                        l10n.t("configDownloadTimeout"),
                        hint="30m",
                        help_text=l10n.t("configTimeoutHelp"),
                    ),
                ]
            ),
        ]
        if other:
            children += [group_header(l10n.t("configOther")), list_group([field(key, None, key) for key in other])]
    return ft.Column(
        [
            group_header(l10n.t("settingsVfoxConfig")),
            ft.Container(
                padding=ft.Padding.only(left=Gaps.L, bottom=Gaps.M),
                content=text(l10n.t("settingsVfoxConfigHelp"), S.BODY_SMALL, color=C.ON_SURFACE_VARIANT),
            ),
            *children,
        ],
        spacing=0,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
    )


@ft.component
def ConfigSwitch(
    config_key: str,
    meaning: str | None,
    title: str,
    subtitle: str | None,
    position: ListPosition,
    key: str | None = None,
) -> ft.Control:
    """A boolean key of the vfox configuration."""
    del key
    app = use_app()
    config = app.model.vfox_config
    value = config.value_for(config_key) == "true"
    saving = config.is_saving(config_key)

    def toggle() -> None:
        spawn(_save_config(app, lambda: config.set(config_key, str(not value).lower())))

    return list_item(
        title,
        position=position,
        leading=meaning if meaning is not None else _SPACER,
        subtitle=subtitle,
        trailing=ft.Switch(value=value, disabled=saving, on_change=lambda _: toggle(), key=f"switch-{config_key}"),
        on_click=None if saving else toggle,
    )


@ft.component
def ConfigChoice(
    config_key: str,
    title: str,
    choices: dict[str, str],
    position: ListPosition,
    wrap: bool,
    key: str | None = None,
) -> ft.Control:
    """A key of the vfox configuration with a few possible values."""
    del key
    app = use_app()
    config = app.model.vfox_config
    value = config.value_for(config_key) or ""

    def select(event: ft.Event[ft.Dropdown]) -> None:
        selection = event.control.value
        if selection is not None and selection != value:
            spawn(_save_config(app, lambda: config.set(config_key, selection)))

    return list_item(
        title,
        position=position,
        leading=_SPACER,
        wrap_trailing=wrap,
        trailing=ft.Dropdown(
            value=value if value in choices else None,
            options=[ft.DropdownOption(key=choice, text=label) for choice, label in choices.items()],
            disabled=config.is_saving(config_key),
            on_select=select,
            filled=True,
            border=ft.NoInputBorder(),
            key=f"choice-{config_key}",
        ),
    )


@ft.component
def ConfigText(
    config_key: str,
    meaning: str | None,
    title: str,
    hint: str | None,
    help_text: str | None,
    directory: bool,
    position: ListPosition,
    wrap: bool,
    key: str | None = None,
) -> ft.Control:
    """A key of the vfox configuration edited as text: saved on Enter or when
    the field loses the focus; clearing the field resets the key.
    """
    del key
    app = use_app()
    l10n = app.l10n
    config = app.model.vfox_config
    stored = config.value_for(config_key) or ""
    draft, set_draft = ft.use_state(stored)
    saved: ft.MutableRef[str] = ft.use_ref(stored)
    if saved.current != stored and draft == saved.current:
        # vfox stored another value meanwhile, and the field is not being edited.
        saved.current = stored
        set_draft(stored)

    def save(value: str) -> None:
        value = value.strip()
        if value == saved.current:
            return
        # Submitting and then leaving the field must not save twice. If vfox
        # rejects the value, the reload shows the stored value again.
        saved.current = value
        spawn(_save_config(app, lambda: config.reset(config_key) if not value else config.set(config_key, value)))

    async def browse() -> None:
        chosen = await app.services.pick_directory(draft or None)
        if chosen is not None:
            set_draft(chosen)
            save(chosen)

    def clear() -> None:
        set_draft("")
        save("")

    field = text_field(
        value=draft,
        hint=hint,
        dense=True,
        on_change=set_draft,
        on_submit=save,
        on_blur=save,
        key=f"text-{config_key}",
    )
    field.label = None
    field.disabled = config.is_saving(config_key)
    if directory:
        field.suffix = icon_button("folder", l10n.t("actionBrowse"), lambda: spawn(browse()), compact=True)
    elif draft:
        field.suffix = icon_button("close", l10n.t("actionReset"), clear, compact=True)
    return list_item(
        title,
        position=position,
        leading=meaning if meaning is not None else _SPACER,
        subtitle=help_text,
        wrap_trailing=wrap,
        trailing=ft.Container(width=300, content=field),
    )


def _about(app: AppContext) -> ft.Control:
    l10n = app.l10n
    services = app.services

    def identity(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("appTitle"),
            position=position,
            leading=ShapeAvatar(name="vfox+", icon=icon("sdks"), size=40),
            subtitle=f"{l10n.t('aboutVersion', version=APP_VERSION)} · {l10n.t('aboutLicense')}",
        )

    def link(meaning: str, title: str, url: str) -> Callable[[ListPosition], ft.Control]:
        return lambda position: list_item(
            title,
            position=position,
            leading=meaning,
            subtitle=url,
            trailing=icon_control("openInNew", color=C.ON_SURFACE_VARIANT),
            on_click=lambda: spawn(services.open_url(url)),
        )

    def licenses(position: ListPosition) -> ft.Control:
        return list_item(
            l10n.t("aboutLicenses"),
            position=position,
            leading="license",
            trailing=icon_control("chevronRight", color=C.ON_SURFACE_VARIANT),
            on_click=lambda: show_licenses(app),
            key="about-licenses",
        )

    return ft.Column(
        [
            group_header(l10n.t("settingsAbout")),
            list_group(
                [
                    identity,
                    link("sourceCode", l10n.t("aboutSource"), links.REPOSITORY),
                    link("docs", l10n.t("aboutDocs"), links.VFOX_DOCS),
                    licenses,
                ]
            ),
        ],
        spacing=0,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
    )


def show_licenses(app: AppContext) -> None:
    """Lists the open-source licenses of the bundled code; choosing a package
    shows its license.
    """
    l10n = app.l10n
    entries = bundled_licenses()

    def show(entry: LicenseEntry) -> None:
        app.messenger.show(
            ExpressiveDialog(
                title=", ".join(entry.packages),
                content=ConstrainedBox(
                    max_width=640,
                    max_height=480,
                    content=ft.Column(
                        [text(entry.text, S.BODY_SMALL, selectable=True, font_family="monospace")],
                        scroll=ft.ScrollMode.AUTO,
                        tight=True,
                    ),
                ),
                actions=[text_button(l10n.t("actionClose"), app.messenger.close, key="license-close")],
            )
        )

    items: list[ft.Control] = [
        ft.ListTile(
            title=ft.Text(entry.packages[0]),
            subtitle=ft.Text(", ".join(entry.packages[1:]), max_lines=1, overflow=ft.TextOverflow.ELLIPSIS)
            if len(entry.packages) > 1
            else None,
            on_click=lambda _, entry=entry: show(entry),
        )
        for entry in entries
    ]
    app.messenger.show(
        ExpressiveDialog(
            icon=icon("license"),
            icon_color=C.SECONDARY,
            title=l10n.t("aboutLicenses"),
            content=ConstrainedBox(
                max_width=560,
                max_height=480,
                content=ft.Column(
                    [
                        ft.Text(f"{l10n.t('appTitle')} {APP_VERSION} · {l10n.t('aboutLicense')}"),
                        ft.ListView(items, height=400) if items else ft.Container(),
                    ],
                    tight=True,
                    spacing=Gaps.S,
                ),
            ),
            actions=[text_button(l10n.t("actionClose"), app.messenger.close, key="licenses-close")],
        )
    )

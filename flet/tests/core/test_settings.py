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

"""The settings, their file and the plugin sources."""

import dataclasses
import json
import os
from pathlib import Path

import pytest

from vfox_plus.core.settings.app_settings import (
    DEFAULT_SEED_COLOR,
    DEFAULT_SETTINGS,
    AppSettings,
    LanguagePreference,
    ThemePreference,
    format_environment_lines,
    parse_environment_lines,
    settings_from_json,
    settings_to_json,
)
from vfox_plus.core.settings.settings_store import JsonFileSettingsStore, MemorySettingsStore, settings_path
from vfox_plus.core.vfox.plugin_source import (
    is_remote_plugin_source,
    prepare_plugin_source,
    vfox_moves_plugin_source,
    vfox_temporary_directory,
)


def test_settings_round_trip_through_json() -> None:
    settings = AppSettings(
        theme=ThemePreference.DARK,
        use_system_color=False,
        seed_color=0xFF123456,
        language=LanguagePreference.SIMPLIFIED_CHINESE,
        vfox_executable="/opt/vfox",
        extra_environment={"A": "1"},
        rail_expanded=True,
    )
    assert settings_from_json(json.loads(json.dumps(settings_to_json(settings)))) == settings


def test_settings_write_the_keys_of_the_specification_in_order() -> None:
    assert list(settings_to_json(DEFAULT_SETTINGS)) == [
        "theme",
        "useSystemColor",
        "seedColor",
        "language",
        "extraEnvironment",
        "railExpanded",
    ]
    assert settings_to_json(dataclasses.replace(DEFAULT_SETTINGS, vfox_executable="/v"))["vfoxExecutable"] == "/v"


def test_settings_ignore_malformed_values_and_keep_unknown_keys() -> None:
    settings = settings_from_json(
        {
            "theme": "purple",
            "useSystemColor": "yes",
            "seedColor": "#fff",
            "language": 42,
            "vfoxExecutable": "",
            "extraEnvironment": {"A": "1", "B": 2},
            "railExpanded": 1,
            "future": True,
        }
    )
    assert settings.theme == ThemePreference.SYSTEM
    assert settings.use_system_color is True
    assert settings.seed_color == DEFAULT_SEED_COLOR
    assert settings.language == LanguagePreference.SYSTEM
    assert settings.vfox_executable is None
    assert dict(settings.extra_environment) == {"A": "1"}
    assert settings.rail_expanded is False
    assert settings_to_json(settings)["future"] is True
    assert settings_from_json([1, 2]) is DEFAULT_SETTINGS
    assert dict(settings_from_json({"extraEnvironment": ["x"]}).extra_environment) == {}
    assert settings_from_json({"seedColor": True}).seed_color == DEFAULT_SEED_COLOR


def test_settings_compare_by_value() -> None:
    assert dataclasses.replace(DEFAULT_SETTINGS, extra_environment={}) == DEFAULT_SETTINGS
    assert dataclasses.replace(DEFAULT_SETTINGS, rail_expanded=True) != DEFAULT_SETTINGS
    assert hash(dataclasses.replace(DEFAULT_SETTINGS, extra_environment={})) == hash(DEFAULT_SETTINGS)
    assert DEFAULT_SETTINGS != "settings"


def test_language_codes() -> None:
    assert LanguagePreference.SYSTEM.language_code is None
    assert LanguagePreference.ENGLISH.language_code == "en"
    assert LanguagePreference.SIMPLIFIED_CHINESE.language_code == "zh"


def test_environment_lines_parse_key_value_skipping_comments_and_blank_lines() -> None:
    result = parse_environment_lines("# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n")
    assert result.variables == {"A": "1", "B": "x=y"}
    assert result.invalid_lines == [5, 6]


def test_environment_lines_are_formatted_sorted_by_key() -> None:
    assert format_environment_lines({"B": "2", "A": "1"}) == "A=1\nB=2"


def test_settings_path_uses_xdg_config_home_and_falls_back_to_the_home_directory() -> None:
    assert settings_path({"XDG_CONFIG_HOME": "/x", "HOME": "/h"}) == "/x/vfox-plus/settings.json"
    assert settings_path({"XDG_CONFIG_HOME": "relative", "HOME": "/h"}) == "/h/.config/vfox-plus/settings.json"
    assert settings_path({}, "/fallback") == "/fallback/.config/vfox-plus/settings.json"


def test_json_store_returns_defaults_without_a_file_and_what_was_saved_afterwards(tmp_path: Path) -> None:
    path = tmp_path / "nested" / "settings.json"
    store = JsonFileSettingsStore(str(path))
    assert store.load() == DEFAULT_SETTINGS
    settings = dataclasses.replace(DEFAULT_SETTINGS, theme=ThemePreference.LIGHT, extra_environment={"Ä": "é"})
    store.save(settings)
    assert store.load() == settings
    text = path.read_text(encoding="utf-8")
    assert text.endswith("}\n")
    assert '"Ä": "é"' in text
    assert os.listdir(path.parent) == ["settings.json"]


def test_json_store_returns_defaults_for_a_corrupted_file(tmp_path: Path) -> None:
    path = tmp_path / "settings.json"
    path.write_text("{not json", encoding="utf-8")
    assert JsonFileSettingsStore(str(path)).load() == DEFAULT_SETTINGS


def test_json_store_leaves_no_temporary_file_when_writing_fails(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    def fail(source: str, target: str) -> None:
        raise OSError("disk full")

    monkeypatch.setattr(os, "replace", fail)
    store = JsonFileSettingsStore(str(tmp_path / "settings.json"))
    with pytest.raises(OSError, match="disk full"):
        store.save(DEFAULT_SETTINGS)
    assert os.listdir(tmp_path) == []


def test_memory_store_counts_saves() -> None:
    store = MemorySettingsStore()
    store.save(dataclasses.replace(DEFAULT_SETTINGS, rail_expanded=True))
    assert store.save_count == 1
    assert store.load().rail_expanded is True


def test_only_local_lua_files_are_moved_by_vfox() -> None:
    assert vfox_moves_plugin_source("/home/u/plugin.lua")
    assert vfox_moves_plugin_source("/home/u/PLUGIN.LUA")
    assert not vfox_moves_plugin_source("/home/u/plugin.zip")
    assert not vfox_moves_plugin_source("https://example.com/plugin.lua")
    assert is_remote_plugin_source("http://127.0.0.1:8080/p.zip")


def test_the_temporary_directory_of_vfox_is_in_its_home_directory(tmp_path: Path) -> None:
    home = str(tmp_path)
    assert vfox_temporary_directory({"HOME": home}) == f"{home}/.vfox/tmp"
    (tmp_path / ".version-fox").mkdir()
    assert vfox_temporary_directory({"HOME": home}) == f"{home}/.version-fox/tmp"
    assert vfox_temporary_directory({}).startswith("/")


def test_urls_and_archives_are_passed_through(tmp_path: Path) -> None:
    for source in ("https://example.com/p.lua", str(tmp_path / "p.zip")):
        prepared = prepare_plugin_source(source, str(tmp_path / "tmp"))
        assert prepared.path == source
        assert not prepared.is_copy
        prepared.dispose()


def test_a_local_lua_file_is_copied_and_the_original_is_kept(tmp_path: Path) -> None:
    original = tmp_path / "mine.lua"
    original.write_text("PLUGIN = {}", encoding="utf-8")
    root = tmp_path / ".vfox" / "tmp"
    prepared = prepare_plugin_source(str(original), str(root))
    assert prepared.is_copy
    assert prepared.path != str(original)
    assert prepared.path.endswith("/mine.lua")
    assert Path(prepared.path).read_text(encoding="utf-8") == "PLUGIN = {}"
    prepared.dispose()
    assert not Path(prepared.path).exists()
    assert original.exists()
    assert os.listdir(root) == []


def test_a_file_that_cannot_be_copied_is_left_for_vfox_to_report(tmp_path: Path) -> None:
    root = tmp_path / "tmp"
    prepared = prepare_plugin_source(str(tmp_path / "missing.lua"), str(root))
    assert prepared.path == str(tmp_path / "missing.lua")
    assert not prepared.is_copy
    assert os.listdir(root) == []

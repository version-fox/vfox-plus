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

"""Runs the parsers against the shared vfox output fixtures.

The expected results live in shared/fixtures/vfox-*/cases.json and are shared
by every implementation of the GUI; see shared/fixtures/README.md.
"""

import dataclasses
import json
from collections.abc import Callable
from typing import Any

import pytest

from support import SHARED, read_capture
from vfox_plus.core.terminal_text import split_terminal_lines, strip_ansi
from vfox_plus.core.vfox.output import (
    parse_available_plugins,
    parse_config_list,
    parse_current,
    parse_download_progress,
    parse_install_path,
    parse_installed_tree,
    parse_plugin_info,
    parse_search,
    parse_vfox_version,
    summarize_error,
)

FIXTURES = SHARED / "fixtures"


def _camel_case(name: str) -> str:
    head, *rest = name.split("_")
    return head + "".join(part.title() for part in rest)


def _to_json(value: Any) -> Any:
    """The shape of a parser result in `cases.json`: models become objects
    with camel-case keys, tuples become lists."""
    if dataclasses.is_dataclass(value) and not isinstance(value, type):
        return {_camel_case(item.name): _to_json(getattr(value, item.name)) for item in dataclasses.fields(value)}
    if isinstance(value, (list, tuple)):
        return [_to_json(item) for item in value]
    if isinstance(value, dict):
        return {key: _to_json(item) for key, item in value.items()}
    return value


def _indexed(items: list[Any]) -> dict[str, Any]:
    return {"count": len(items), "items": {str(index): item for index, item in enumerate(items)}}


def _final_progress(_: str, stderr: str) -> float | None:
    last = None
    for line in split_terminal_lines(strip_ansi(stderr)):
        progress = parse_download_progress(line)
        last = progress if progress is not None else last
    return last


CHECKS: dict[str, Callable[[str, str], Any]] = {
    "version": lambda stdout, _: parse_vfox_version(stdout),
    "current": lambda stdout, _: parse_current(stdout),
    "list": lambda stdout, _: parse_installed_tree(stdout),
    "search": lambda stdout, _: _indexed(parse_search(stdout)),
    "available": lambda stdout, _: _indexed(parse_available_plugins(stdout)),
    "pluginInfo": lambda stdout, _: parse_plugin_info(stdout),
    "installPath": lambda stdout, _: parse_install_path(stdout),
    "config": lambda stdout, _: parse_config_list(stdout),
    "finalProgress": _final_progress,
    "error": summarize_error,
}


def _select(check: str, actual: Any, expected: Any) -> Any:
    """For `search` and `available`, only the listed items are compared."""
    if check not in ("search", "available"):
        return actual
    return {"count": actual["count"], "items": {index: actual["items"].get(index) for index in expected["items"]}}


def _cases() -> list[Any]:
    result = []
    for manifest_path in sorted(FIXTURES.glob("vfox-*/cases.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        for case in manifest["cases"]:
            for check, expected in case["expect"].items():
                result.append(
                    pytest.param(
                        manifest_path.parent,
                        case["name"],
                        check,
                        expected,
                        id=f"{manifest['vfox']}-{case['name']}-{check}",
                    )
                )
    return result


CASES = _cases()


def test_finds_the_fixtures() -> None:
    assert len({case.values[0] for case in CASES}) >= 2
    assert len(CASES) > 50


@pytest.mark.parametrize(("directory", "name", "check", "expected"), CASES)
def test_parser_matches_the_fixture(directory: Any, name: str, check: str, expected: Any) -> None:
    stdout = read_capture(directory / f"{name}.stdout")
    stderr = read_capture(directory / f"{name}.stderr")
    assert check in CHECKS, f'Unknown parser "{check}" in cases.json'
    actual = _to_json(CHECKS[check](stdout, stderr))
    assert _select(check, actual, expected) == expected

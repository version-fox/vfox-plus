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

"""Edge cases of the output parsers and the terminal text helpers that the
fixtures do not cover."""

from vfox_plus.core.models import AvailableVersion, ConfigEntry, InstalledSdk, VersionAddition
from vfox_plus.core.terminal_text import TerminalLineSplitter, split_terminal_lines, strip_ansi
from vfox_plus.core.vfox.output import (
    merge_installed_sdks,
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


def test_version_reads_release_and_pre_release_versions() -> None:
    assert parse_vfox_version("vfox version 1.0.12\n") == "1.0.12"
    assert parse_vfox_version("vfox version 2.0.0-beta.1") == "2.0.0-beta.1"
    assert parse_vfox_version("command not found") is None


def test_current_strips_the_v_prefix_only_once() -> None:
    assert parse_current("go -> vv1.2\n") == {"go": "v1.2"}


def test_current_ignores_colour_codes_and_unrelated_lines() -> None:
    output = "Warning: something\njava -> \x1b[92mv21\x1b[0m\nmaven -> N/A \nbad name -> v1\n"
    assert parse_current(output) == {"java": "21", "maven": None}


def test_installed_tree_handles_a_single_plugin_without_versions() -> None:
    assert parse_installed_tree("All installed sdk versions\n└──maven\n") == {"maven": []}


def test_installed_tree_ignores_the_header_and_blank_lines() -> None:
    output = "All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n"
    assert parse_installed_tree(output) == {"nodejs": ["24.11.1"]}


def test_installed_tree_ignores_versions_before_any_plugin() -> None:
    assert parse_installed_tree("  └──24.11.1\n") == {}


def test_merge_keeps_plugins_that_only_one_command_reported() -> None:
    sdks = merge_installed_sdks({"java": ["21"]}, {"java": "21", "go": None})
    assert sdks == [InstalledSdk("java", ("21",), "21"), InstalledSdk("go", (), None)]


def test_search_reads_notes_additions_and_the_installed_marker() -> None:
    versions = parse_search(
        "Available versions:\n"
        " - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n"
        " - 1.0 (release candidate)\n"
        " - 0.1.14 [npm ]\n"
        " - 2.0\n"
        " - 3.0 ()\n"
    )
    assert versions == [
        AvailableVersion(
            "24.22.0", "LTS", (VersionAddition("npm", "11.19.1"), VersionAddition("corepack", "0.34.0")), True
        ),
        AvailableVersion("1.0", "release candidate"),
        AvailableVersion("0.1.14", None, (VersionAddition("npm", ""),)),
        AvailableVersion("2.0"),
        AvailableVersion("3.0"),
    ]
    assert [str(addition) for addition in versions[0].additions] == ["npm 11.19.1", "corepack 0.34.0"]
    assert [str(addition) for addition in versions[2].additions] == ["npm"]


def test_search_keeps_a_note_that_is_not_in_parentheses() -> None:
    assert parse_search(" - 1.0 preview\n")[0].note == "preview"


def test_search_ignores_lines_that_are_not_list_entries() -> None:
    assert parse_search("Please select a version\n -\n - \n") == []


def test_plugin_info_keeps_separators_in_the_description() -> None:
    info = parse_plugin_info("a\x1f1.0\x1fhttps://x\x1f/p\x1fone\x1ftwo")
    assert info is not None
    assert info.description == "one\x1ftwo"
    assert info.homepage == "https://x"
    assert parse_plugin_info("nodejs not supported") is None


def test_install_path_maps_notfound_to_none() -> None:
    assert parse_install_path("notfound\n") is None
    assert parse_install_path("/opt/sdk/v-1\n") == "/opt/sdk/v-1"


def test_config_keeps_empty_values_and_values_containing_spaces() -> None:
    assert parse_config_list("proxy.url = \nfoo.bar = a = b\nnot a setting\n = x\n") == [
        ConfigEntry("proxy.url", ""),
        ConfigEntry("foo.bar", "a = b"),
    ]
    assert parse_config_list("registry.address =\n") == [ConfigEntry("registry.address", "")]


def test_download_progress() -> None:
    assert parse_download_progress("Downloading...  42% [===>   ] (1 MB/s) [1s:2s]") == 0.42
    assert parse_download_progress("Downloading... 100% [======]") == 1
    assert parse_download_progress("Downloading... 250% [======]") == 1
    assert parse_download_progress("Preinstalling maven@3.9.16...") is None


def test_summary_prefers_standard_error() -> None:
    assert summarize_error("Removing...\n", "Use the -y flag\n") == "Use the -y flag"


def test_summary_skips_progress_bars_and_the_generic_install_trailer() -> None:
    assert (
        summarize_error(
            "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n",
            "\rDownloading...  3% [>   ]\r      \r",
        )
        == "failed to download x@1 file, err:EOF"
    )


def test_summary_finds_the_reason_when_the_trailer_is_on_standard_error() -> None:
    # vfox 1.0.13 prints the error a command returns to standard error.
    assert (
        summarize_error(
            "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n",
            "\rDownloading...  3% [>   ]\r      \rfailed to install x\n",
        )
        == "failed to download x@1 file, err:EOF"
    )


def test_summary_is_empty_without_output() -> None:
    assert summarize_error("", "  \n") == ""
    assert summarize_error("", "failed to install x\n") == "failed to install x"


def test_strip_ansi_removes_csi_osc_and_two_byte_escapes() -> None:
    assert strip_ansi("\x1b[1;32mok\x1b[0m \x1b]8;;https://x\x07link\x1b]8;;\x07 \x1bMx") == "ok link x"


def test_split_terminal_lines_on_new_lines_and_lone_carriage_returns() -> None:
    assert split_terminal_lines("a\r\nb\rc\nd") == ["a", "b", "c", "d"]


def test_line_splitter_splits_streamed_output_into_clean_lines() -> None:
    splitter = TerminalLineSplitter()
    assert splitter.push(b"one\r\x1b[32mtw") == ["one"]
    # A multi-byte character split between two chunks stays intact.
    euro = "€".encode()
    assert splitter.push(b"o " + euro[:1]) == []
    assert splitter.push(euro[1:] + b"\n \n") == ["two €"]
    assert splitter.push(b"last") == []
    assert splitter.close() == ["last"]
    assert splitter.close() == []


def test_line_splitter_replaces_malformed_bytes() -> None:
    assert TerminalLineSplitter().push(b"a\xffb\n") == ["a�b"]

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

"""The vfox client and the locator, against a scripted process runner."""

from pathlib import Path

import pytest

from support import FakeProcessRunner, fixture_output, fixture_runner
from vfox_plus.core.models import InstalledSdk
from vfox_plus.core.process import ProcessOutput
from vfox_plus.core.vfox.cli import VfoxCli, VfoxLaunchConfig, is_valid_plugin_name, vfox_environment
from vfox_plus.core.vfox.command import VfoxError, VfoxNotFoundError, VfoxOperation
from vfox_plus.core.vfox.locator import VfoxLocator, is_executable_file
from vfox_plus.core.vfox.output import PLUGIN_INFO_TEMPLATE


class _Listener:
    def stdout(self, chunk: bytes) -> None:
        pass

    def stderr(self, chunk: bytes) -> None:
        pass


@pytest.fixture
def runner() -> FakeProcessRunner:
    return fixture_runner()


@pytest.fixture
def cli(runner: FakeProcessRunner) -> VfoxCli:
    return VfoxCli(
        runner,
        VfoxLaunchConfig(
            executable="/usr/local/bin/vfox",
            base_environment={
                "HOME": "/home/user",
                "PATH": "/usr/bin",
                "__VFOX_SHELL": "bash",
                "__VFOX_PID": "42",
                "__VFOX_CURTMPPATH": "/home/user/.vfox/tmp/x",
            },
            extra_environment={"VFOX_NODEJS_MIRROR": "https://mirror", "PATH": "/opt/bin"},
        ),
    )


def test_environment_removes_the_variables_of_an_activated_shell_and_disables_colours(cli: VfoxCli) -> None:
    environment = vfox_environment(cli.launch_config)
    assert "__VFOX_SHELL" not in environment
    assert "__VFOX_PID" not in environment
    assert "__VFOX_CURTMPPATH" not in environment
    assert environment["NO_COLOR"] == "1"
    assert environment["HOME"] == "/home/user"


def test_environment_lets_user_variables_override_inherited_ones(cli: VfoxCli) -> None:
    environment = vfox_environment(cli.launch_config)
    assert environment["PATH"] == "/opt/bin"
    assert environment["VFOX_NODEJS_MIRROR"] == "https://mirror"


def test_environment_marks_the_hook_environment_only_when_asked_to(cli: VfoxCli) -> None:
    assert vfox_environment(cli.launch_config, hook_shell=True)["__VFOX_SHELL"] == "bash"


async def test_queries_run_in_the_neutral_directory_with_the_configured_executable(
    cli: VfoxCli, runner: FakeProcessRunner
) -> None:
    assert await cli.version() == "1.0.13"
    [call] = runner.invocations
    assert call.executable == "/usr/local/bin/vfox"
    assert call.working_directory == "/"
    assert call.environment["NO_COLOR"] == "1"
    assert not call.streamed


async def test_rejects_an_executable_that_does_not_print_a_version(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("--version", ProcessOutput(0, "hello\n", ""))
    with pytest.raises(VfoxError, match="Unrecognised version output"):
        await cli.version()


async def test_installed_sdks_merges_list_and_current(cli: VfoxCli) -> None:
    assert await cli.installed_sdks() == [
        InstalledSdk("maven", ("3.9.16", "3.8.9"), "3.9.16"),
        InstalledSdk("mvn2", (), None),
        InstalledSdk("nodejs", ("24.11.1", "22.21.0"), "24.11.1"),
    ]


async def test_installed_sdks_is_empty_when_no_plugin_is_installed(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("list", fixture_output("list-empty", 1)).on("current", fixture_output("current-empty"))
    assert await cli.installed_sdks() == []


async def test_installed_sdks_reports_a_failing_list(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("list", ProcessOutput(1, "boom\n", ""))
    with pytest.raises(VfoxError) as raised:
        await cli.installed_sdks()
    assert raised.value.message == "boom"
    assert raised.value.arguments == ("list",)
    assert raised.value.exit_code == 1


async def test_plugin_info_uses_the_format_template(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    info = await cli.plugin_info("nodejs")
    assert info.version == "0.5.1"
    assert info.install_path == "/tmp/vfox-sb/home/.vfox/cache/nodejs"
    assert runner.invocations[0].arguments[:3] == ("info", "nodejs", "--format")


async def test_plugin_info_rejects_output_without_fields(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    with pytest.raises(VfoxError):
        await cli.plugin_info("maven")
    runner.on(f"info maven --format {PLUGIN_INFO_TEMPLATE}", ProcessOutput(0, "maven", ""))
    with pytest.raises(VfoxError, match="Unrecognised plugin information"):
        await cli.plugin_info("maven")


async def test_search_versions(cli: VfoxCli) -> None:
    versions = await cli.search_versions("maven")
    assert "3.9.16" in [version.version for version in versions]


async def test_available_plugins(cli: VfoxCli) -> None:
    assert len(await cli.available_plugins()) == 43


async def test_configuration_is_listed_set_and_reset(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("config proxy.url http://proxy:8080", ProcessOutput(0, "", "")).on(
        "config --unset proxy.url", ProcessOutput(0, "", "")
    )
    assert len(await cli.config_entries()) == 10
    await cli.set_config("proxy.url", "http://proxy:8080")
    await cli.unset_config("proxy.url")
    assert [call.command for call in runner.calls("config")] == [
        "config --list",
        "config proxy.url http://proxy:8080",
        "config --unset proxy.url",
    ]


async def test_a_rejected_value_becomes_an_error_with_the_message_of_vfox(
    cli: VfoxCli, runner: FakeProcessRunner
) -> None:
    runner.on("config cache.availableHookDuration notaduration", fixture_output("config-set-invalid-duration", 1))
    with pytest.raises(VfoxError, match='time: invalid duration "notaduration"'):
        await cli.set_config("cache.availableHookDuration", "notaduration")


async def test_reports_the_exit_code_when_vfox_prints_nothing(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("config --unset proxy.url", ProcessOutput(3, "", ""))
    with pytest.raises(VfoxError, match="vfox exited with code 3"):
        await cli.unset_config("proxy.url")


async def test_install_path(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.on("info nodejs@24.11.1", fixture_output("info-version-path")).on(
        "info nodejs@1.0.0", fixture_output("info-version-not-installed")
    )
    assert await cli.install_path("nodejs", "24.11.1") == "/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1"
    assert await cli.install_path("nodejs", "1.0.0") is None


async def test_a_missing_executable_becomes_a_not_found_error(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.missing_executable = True
    with pytest.raises(VfoxNotFoundError) as raised:
        await cli.version()
    assert raised.value.executable == "/usr/local/bin/vfox"


def test_install_answers_prompts_with_yes(cli: VfoxCli) -> None:
    command = cli.install("nodejs", "24.11.1")
    assert command.arguments == ("install", "--yes", "nodejs@24.11.1")
    assert command.operation == VfoxOperation.INSTALL
    assert command.subject == "nodejs@24.11.1"
    assert command.command_line == "vfox install --yes nodejs@24.11.1"
    assert command.working_directory is None


def test_use_in_the_global_scope_needs_the_hook_environment(cli: VfoxCli) -> None:
    command = cli.use("nodejs", "24.11.1")
    assert command.arguments == ("use", "--global", "nodejs@24.11.1")
    assert command.requires_hook_environment


def test_use_in_a_project_runs_in_the_project_directory(cli: VfoxCli) -> None:
    command = cli.use("java", "21", "project", "/work/app")
    assert command.arguments == ("use", "--project", "java@21")
    assert command.working_directory == "/work/app"


def test_the_project_scope_requires_a_directory(cli: VfoxCli) -> None:
    with pytest.raises(ValueError, match="project directory is required"):
        cli.use("java", "21", "project")
    with pytest.raises(ValueError, match="project directory is required"):
        cli.unuse("java", "project", " ")


def test_unuse_uninstall_remove_and_update(cli: VfoxCli) -> None:
    assert cli.unuse("java").arguments == ("unuse", "--global", "java")
    assert cli.unuse("java", "project", "/p").working_directory == "/p"
    assert cli.uninstall("java", "21").arguments == ("uninstall", "java@21")
    assert cli.remove_plugin("java").arguments == ("remove", "--yes", "java")
    assert cli.update_plugin("java").arguments == ("update", "java")
    assert cli.update_all_plugins().arguments == ("update", "--all")
    assert cli.update_all_plugins().subject is None
    assert cli.update_plugin("java").subject == "java"


def test_add_plugin_from_the_registry_or_from_a_source(cli: VfoxCli) -> None:
    assert cli.add_plugin("nodejs").arguments == ("add", "nodejs")
    custom = cli.add_plugin("", source=" https://x/p.zip ", alias="mine")
    assert custom.arguments == ("add", "--source", "https://x/p.zip", "--alias", "mine")
    assert custom.sdk == "mine"
    assert cli.add_plugin("", source="/p.lua").sdk is None
    with pytest.raises(ValueError, match="name or a source is required"):
        cli.add_plugin("  ")


@pytest.mark.parametrize(
    ("name", "version"), [("--all", "1"), ("java", "-1"), ("java", "1 2"), ("java", "a@b"), ("java", "")]
)
def test_rejects_names_and_versions_that_look_like_options(cli: VfoxCli, name: str, version: str) -> None:
    with pytest.raises(ValueError, match="Not a valid"):
        cli.install(name, version)


def test_plugin_names() -> None:
    assert is_valid_plugin_name("my-sdk.2")
    assert not is_valid_plugin_name("-x")
    assert not is_valid_plugin_name("nodejs\n")


async def test_rejects_malformed_configuration_keys(cli: VfoxCli) -> None:
    with pytest.raises(ValueError, match="Not a valid configuration key"):
        await cli.set_config("proxy..url", "x")


async def test_start_launches_the_command_with_the_right_environment(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    process = await cli.start(cli.use("java", "21", "project", "/work/app"), _Listener())
    [call] = runner.invocations
    assert call.streamed
    assert call.working_directory == "/work/app"
    assert call.environment["__VFOX_SHELL"] == "bash"
    assert await process.wait() == 0


async def test_start_reports_a_missing_executable(cli: VfoxCli, runner: FakeProcessRunner) -> None:
    runner.missing_executable = True
    with pytest.raises(VfoxNotFoundError):
        await cli.start(cli.update_all_plugins(), _Listener())


def test_locator_searches_path_first_then_the_usual_install_locations_without_duplicates() -> None:
    locator = VfoxLocator({"PATH": "/opt/bin::/usr/bin", "HOME": "/home/me"}, lambda path: False)
    assert locator.search_directories == [
        "/opt/bin",
        "/usr/bin",
        "/usr/local/bin",
        "/home/me/.local/bin",
        "/home/me/bin",
        "/home/me/.linuxbrew/bin",
        "/home/linuxbrew/.linuxbrew/bin",
        "/snap/bin",
    ]


def test_locator_finds_the_first_executable() -> None:
    found = {"/home/me/.local/bin/vfox", "/snap/bin/vfox"}
    assert VfoxLocator({"HOME": "/home/me"}, found.__contains__).locate() == "/home/me/.local/bin/vfox"


def test_locator_returns_none_without_vfox() -> None:
    assert VfoxLocator({}, lambda path: False).locate() is None
    assert "None/bin" not in VfoxLocator({}, lambda path: False).fallback_directories


def test_is_executable_file(tmp_path: Path) -> None:
    script = tmp_path / "script"
    script.write_text("#!/bin/sh\n", encoding="utf-8")
    assert not is_executable_file(str(script))
    script.chmod(0o755)
    assert is_executable_file(str(script))
    assert not is_executable_file(str(tmp_path))
    assert not is_executable_file(str(tmp_path / "missing"))

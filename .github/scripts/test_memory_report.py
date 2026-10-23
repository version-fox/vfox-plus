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

"""Tests of memory_report.py: python3 -m unittest discover -s .github/scripts"""

import contextlib
import io
import os
import re
import subprocess
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest import mock

import memory_report as report
from memory_report import Memory, Point, Row

REPOSITORY = "version-fox/vfox-plus"
HEAD = "a" * 40
MAIN = "b" * 40


def setUpModule():
    # CI sets these. The tests must not write to the summary of the job that runs them, or
    # take the repository and the implementations from it.
    patcher = mock.patch.dict(os.environ)
    patcher.start()
    unittest.addModuleCleanup(patcher.stop)
    for name in ("GITHUB_STEP_SUMMARY", "GITHUB_REPOSITORY", "MEMORY_IMPLEMENTATIONS", "GH_TOKEN"):
        os.environ.pop(name, None)


def memory_text(home=(55, 42, 1), versions=(64, 49, 1)) -> str:
    return (
        f"memory (home): RSS {home[0]} MB, PSS {home[1]} MB in {home[2]} processes\n"
        f"memory (nodejs): RSS {versions[0]} MB, PSS {versions[1]} MB in {versions[2]} processes\n"
    )


def make_zip(files: dict[str, str | bytes]) -> bytes:
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as archive:
        for name, content in files.items():
            archive.writestr(name, content)
    return buffer.getvalue()


def memory(home_pss=100, versions_pss=120, processes=1) -> Memory:
    return Memory(Point(home_pss + 10, home_pss, processes), Point(versions_pss + 10, versions_pss, processes))


class FakeGitHub:
    """The parts of the API of GitHub that the report reads and writes."""

    repository = REPOSITORY

    def __init__(self) -> None:
        self.default_branch = "main"
        self.workflows: list[dict] = []
        self.runs: list[dict] = []
        self.artifacts: list[dict] = []
        self.zips: dict[int, bytes] = {}
        self.pulls: dict[int, dict] = {}
        self.comments: dict[int, list[dict]] = {}
        self.writes: list[tuple[str, str]] = []
        self._ids = 1000

    def next_id(self) -> int:
        self._ids += 1
        return self._ids

    # What the tests set up.
    def add_workflow(self, implementation: str, name: str | None = None, state: str = "active") -> None:
        self.workflows.append(
            {
                "path": f".github/workflows/{implementation}.yml",
                "name": name or implementation.capitalize(),
                "state": state,
            }
        )

    def add_pull_request(self, number: int = 7, sha: str = HEAD, state: str = "open") -> None:
        head = {"sha": sha, "ref": "feature", "owner": "contributor"}
        self.pulls[number] = {"number": number, "state": state, "head": head}

    def add_run(
        self,
        implementation: str,
        text: str | bytes | None = None,
        *,
        status: str = "completed",
        conclusion: str | None = "success",
        sha: str = HEAD,
        event: str = "pull_request",
        run_id: int | None = None,
        artifact_files: dict | None = None,
    ) -> int:
        run_id = run_id or self.next_id()
        self.runs.append(
            {
                "id": run_id,
                "path": f".github/workflows/{implementation}.yml",
                "name": implementation.capitalize(),
                "event": event,
                "status": status,
                "conclusion": conclusion,
                "head_sha": sha,
                "head_branch": "feature",
                "html_url": f"https://github.com/{REPOSITORY}/actions/runs/{run_id}",
                "head_repository": {"owner": {"login": "contributor"}},
            }
        )
        if text is not None or artifact_files is not None:
            files = artifact_files if artifact_files is not None else {"memory.txt": text}
            self.add_artifact(implementation, files, run_id=run_id, branch="feature", sha=sha)
        return run_id

    def add_artifact(
        self,
        implementation: str,
        files: dict | str,
        *,
        run_id: int | None = None,
        branch: str = "main",
        sha: str = MAIN,
        created: str = "2026-10-20T10:00:00Z",
        expired: bool = False,
        head_repository_id: int = 1,
        size: int | None = None,
    ) -> int:
        files = {"memory.txt": files} if isinstance(files, str) else files
        archive = make_zip(files)
        artifact_id = self.next_id()
        self.zips[artifact_id] = archive
        self.artifacts.append(
            {
                "id": artifact_id,
                "name": f"memory-{implementation}",
                "size_in_bytes": len(archive) if size is None else size,
                "expired": expired,
                "created_at": created,
                "workflow_run": {
                    "id": run_id or self.next_id(),
                    "head_branch": branch,
                    "head_sha": sha,
                    "head_repository_id": head_repository_id,
                    "repository_id": 1,
                },
            }
        )
        return artifact_id

    # The API.
    def get(self, path: str, **params):
        if path == "":
            return {"default_branch": self.default_branch}
        if path == "actions/workflows":
            return {"workflows": self.workflows}
        if path == "actions/runs":
            return {
                "workflow_runs": [
                    run
                    for run in self.runs
                    if run["head_sha"] == params["head_sha"] and run["event"] == params["event"]
                ]
            }
        if match := re.fullmatch(r"actions/runs/(\d+)", path):
            return next(run for run in self.runs if run["id"] == int(match[1]))
        if match := re.fullmatch(r"actions/runs/(\d+)/artifacts", path):
            return {
                "artifacts": [
                    artifact
                    for artifact in self.artifacts
                    if artifact["workflow_run"]["id"] == int(match[1]) and artifact["name"] == params["name"]
                ]
            }
        if path == "actions/artifacts":
            return {"artifacts": [a for a in self.artifacts if a["name"] == params["name"]]}
        if path == "pulls":
            owner, _, branch = params["head"].partition(":")
            return [
                pull
                for pull in self.pulls.values()
                if pull["state"] == params["state"] and pull["head"]["owner"] == owner and pull["head"]["ref"] == branch
            ]
        if match := re.fullmatch(r"pulls/(\d+)", path):
            return self.pulls[int(match[1])]
        if match := re.fullmatch(r"commits/(\w+)/pulls", path):
            return [pull for pull in self.pulls.values() if pull["head"]["sha"] == match[1]]
        raise AssertionError(f"Unexpected GET {path} {params}")

    def pages(self, path: str, key: str | None = None, **params):
        match = re.fullmatch(r"issues/(\d+)/comments", path)
        assert match, path
        yield from list(self.comments.get(int(match[1]), []))

    def raw(self, path: str) -> bytes:
        match = re.fullmatch(r"actions/artifacts/(\d+)/zip", path)
        assert match, path
        return self.zips[int(match[1])]

    def send(self, method: str, path: str, payload=None):
        self.writes.append((method, path))
        if match := re.fullmatch(r"issues/(\d+)/comments", path):
            comment = {"id": self.next_id(), "body": payload["body"], "user": {"login": report.BOT}}
            self.comments.setdefault(int(match[1]), []).append(comment)
            return comment
        if match := re.fullmatch(r"issues/comments/(\d+)", path):
            for comments in self.comments.values():
                for comment in list(comments):
                    if comment["id"] == int(match[1]):
                        if method == "PATCH":
                            comment["body"] = payload["body"]
                        else:
                            comments.remove(comment)
                        return None
        raise AssertionError(f"Unexpected {method} {path}")


class ParseMemoryTest(unittest.TestCase):
    def test_reads_both_points(self):
        self.assertEqual(
            report.parse_memory(memory_text((55, 42, 1), (64, 49, 3))),
            Memory(Point(55, 42, 1), Point(64, 49, 3)),
        )

    def test_the_order_and_blank_lines_do_not_matter(self):
        text = "\n" + memory_text().splitlines()[1] + "\n\n" + memory_text().splitlines()[0] + "\n"
        self.assertIsNotNone(report.parse_memory(text))

    def test_needs_both_points(self):
        self.assertIsNone(report.parse_memory(memory_text().splitlines()[0]))
        self.assertIsNone(report.parse_memory(""))

    def test_rejects_a_point_twice(self):
        line = memory_text().splitlines()[0]
        self.assertIsNone(report.parse_memory(f"{line}\n{line}\n{memory_text().splitlines()[1]}"))

    def test_rejects_anything_else(self):
        self.assertIsNone(report.parse_memory(memory_text() + "e2e: MEASURE home\n"))
        self.assertIsNone(report.parse_memory(memory_text().replace("MB, PSS", "MB, PSS -")))
        self.assertIsNone(report.parse_memory(memory_text().replace("home", "other")))
        self.assertIsNone(report.parse_memory(memory_text((10**6, 1, 1), (1, 1, 1))))

    def test_rejects_digits_that_are_not_ascii(self):
        self.assertIsNone(report.parse_memory(memory_text().replace("55", "٥٥")))


class ToleranceTest(unittest.TestCase):
    def test_a_small_implementation_is_judged_by_the_absolute_floor(self):
        self.assertFalse(report.exceeds_noise("mygo", 42, 47))
        self.assertTrue(report.exceeds_noise("mygo", 42, 48))
        self.assertTrue(report.exceeds_noise("mygo", 42, 36))
        self.assertFalse(report.exceeds_noise("mygo", 42, 37))

    def test_a_large_implementation_is_judged_by_the_percentage(self):
        self.assertFalse(report.exceeds_noise("tauri", 300, 315))
        self.assertTrue(report.exceeds_noise("tauri", 300, 316))

    def test_the_noisy_implementations_get_more_room(self):
        self.assertFalse(report.exceeds_noise("flutter", 350, 350 + 52))
        self.assertTrue(report.exceeds_noise("flutter", 350, 350 + 53))
        self.assertFalse(report.exceeds_noise("gio", 200, 224))
        self.assertTrue(report.exceeds_noise("gio", 200, 225))

    def test_no_difference_is_no_change(self):
        for implementation in ("mygo", "gio", "unknown"):
            self.assertFalse(report.exceeds_noise(implementation, 100, 100))


class TechnologiesTest(unittest.TestCase):
    PAGE = """# Memory use

Text with a | pipe.

| Implementation                   | Technology         | Processes |
| -------------------------------- | ------------------ | --------: |
| [mygo](../mygo/README.md#memory) | mygo 0.5, Go, GTK  |         1 |
| [Uno](../uno/README.md#memory)   | Uno Platform 6     |         1 |

| Implementation | Other |
| --- | --- |
| [later](../later/README.md) | not read |
"""

    def test_reads_the_first_table_by_directory(self):
        self.assertEqual(
            report.read_technologies(self.PAGE), {"mygo": "mygo 0.5, Go, GTK", "uno": "Uno Platform 6"}
        )

    def test_a_page_without_the_table_gives_nothing(self):
        self.assertEqual(report.read_technologies("# Nothing\n\n| a | b |\n| - | - |\n"), {})

    def test_reads_the_page_of_this_repository(self):
        page = Path(__file__).resolve().parents[2] / "docs" / "memory.md"
        technologies = report.read_technologies(page.read_text(encoding="utf-8"))
        self.assertTrue(technologies)
        self.assertTrue(all(technologies.values()))


class RenderTest(unittest.TestCase):
    def rows(self) -> list[Row]:
        return [
            Row("tauri", "Tauri", "Tauri 2, Rust", "measured", memory(283, 305, 3), memory(283, 305, 3)),
            Row("mygo", "mygo", "mygo 0.5, Go", "inherited", None, memory(42, 49, 1)),
            Row("uno", "Uno", "", "measured", memory(101, 128), memory(101, 127)),
        ]

    def render(self, rows: list[Row]) -> str:
        return report.render(rows, head_sha=HEAD)

    def table(self, text: str) -> list[list[str]]:
        lines = [line for line in text.splitlines() if line.startswith("|")]
        return [[cell.strip() for cell in line.strip("|").split("|")] for line in lines]

    def test_starts_with_the_marker_that_finds_it_again(self):
        text = self.render(self.rows())
        self.assertEqual(text.splitlines()[0], report.MARKER)
        self.assertIn("`aaaaaaa`", text.splitlines()[1])

    def test_is_sorted_by_the_pss_of_the_home_page(self):
        table = self.table(self.render(self.rows()))
        self.assertEqual([line[0] for line in table[2:]], ["mygo (main)", "Uno", "Tauri"])

    def test_shows_the_difference_from_main_for_what_was_measured_only(self):
        table = self.table(self.render(self.rows()))
        by_name = {line[0]: line for line in table[2:]}
        self.assertEqual(by_name["Tauri"][3], "283 (0)")
        self.assertEqual(by_name["Uno"][5], "128 (+1)")
        self.assertEqual(by_name["mygo (main)"][3], "42")

    def test_the_list_adds_what_the_versions_use_beyond_the_home_page(self):
        table = self.table(self.render(self.rows()))
        self.assertEqual({line[0]: line[-1] for line in table[2:]}, {"mygo (main)": "+7", "Uno": "+27", "Tauri": "+22"})

    def test_a_change_beyond_the_noise_is_bold_and_in_the_summary(self):
        rows = [Row("tauri", "Tauri", "", "measured", memory(330, 305), memory(283, 305))]
        text = self.render(rows)
        self.assertIn("330 (**+47**)", text)
        self.assertIn("- Changed from main beyond the noise of CI: Tauri: Home PSS +47 MB (+17%).", text)
        self.assertNotIn("Within the noise", text)

    def test_a_decrease_counts_too(self):
        rows = [Row("tauri", "Tauri", "", "measured", memory(200, 305), memory(283, 305))]
        self.assertIn("200 (**-83**)", self.render(rows))

    def test_a_change_in_the_number_of_processes_is_a_change(self):
        rows = [Row("tauri", "Tauri", "", "measured", memory(283, 305, 4), memory(283, 305, 3))]
        text = self.render(rows)
        self.assertIn("4 (**+1**)", text)
        self.assertIn("Tauri: processes 3 -> 4", text)

    def test_unchanged_implementations_are_listed_apart_from_the_changed_one(self):
        rows = [
            Row("tauri", "Tauri", "", "measured", memory(330, 305), memory(283, 305)),
            Row("uno", "Uno", "", "measured", memory(101, 127), memory(101, 127)),
        ]
        text = self.render(rows)
        self.assertIn("- Within the noise of CI: Uno.", text)

    def test_says_so_when_nothing_changed(self):
        rows = [Row("uno", "Uno", "", "measured", memory(101, 127), memory(101, 127))]
        self.assertIn("- No change from main beyond the noise of CI in Uno.", self.render(rows))

    def test_an_implementation_without_numbers_on_main_has_nothing_to_compare_with(self):
        rows = [Row("uno", "Uno", "", "measured", memory(101, 127), None)]
        text = self.render(rows)
        self.assertIn("Uno (no baseline)", text)
        self.assertIn("- No numbers on main to compare with yet: Uno.", text)
        self.assertIn("| 101 |", text)

    def test_failed_and_running_implementations_come_last_with_a_link(self):
        rows = self.rows() + [
            Row("flet", "Flet", "", "failed", run_url="https://example.test/failed"),
            Row("gio", "Gio", "", "running", run_url="https://example.test/running"),
            Row("wails", "Wails", "", "missing"),
        ]
        text = self.render(rows)
        names = [line[0] for line in self.table(text)[2:]]
        self.assertEqual(names, ["mygo (main)", "Uno", "Tauri", "Gio", "Flet", "Wails"])
        self.assertIn("[no result](https://example.test/failed)", text)
        self.assertIn("[running](https://example.test/running)", text)
        self.assertIn("- Still running: Gio.", text)
        self.assertIn("- No result, the run failed or was cancelled: Flet.", text)
        self.assertIn("- No numbers anywhere: Wails.", text)

    def test_says_when_the_pull_request_rebuilt_nothing(self):
        rows = [Row("mygo", "mygo", "", "inherited", None, memory(42, 49))]
        self.assertIn("did not rebuild any implementation", self.render(rows))
        self.assertNotIn("did not rebuild any implementation", self.render(self.rows()))

    def test_drops_the_technology_column_without_technologies(self):
        rows = [Row("uno", "Uno", "", "measured", memory(101, 127), memory(101, 127))]
        table = self.table(self.render(rows))
        self.assertEqual(table[0][:3], ["Implementation", "Processes", "Home PSS"])
        self.assertEqual({len(line) for line in table}, {7})

    def test_keeps_the_technology_column_with_technologies(self):
        table = self.table(self.render(self.rows()))
        self.assertEqual(table[0][:2], ["Implementation", "Technology"])
        self.assertEqual({len(line) for line in table}, {8})

    def test_a_cell_cannot_break_the_table(self):
        rows = [Row("uno", "Un|o\n", "a | b\nc", "measured", memory(101, 127), memory(101, 127))]
        table = self.table(self.render(rows).replace("\\|", "/"))
        self.assertEqual({len(line) for line in table}, {8})

    def test_links_the_runs(self):
        rows = [
            Row(
                "uno",
                "Uno",
                "",
                "measured",
                memory(101, 127),
                memory(101, 127),
                run_url="https://example.test/pr",
                baseline_url="https://example.test/main",
                baseline_sha=MAIN,
            )
        ]
        text = self.render(rows)
        self.assertIn("[Uno](https://example.test/pr)", text)
        self.assertIn("[Uno](https://example.test/main) (`bbbbbbb`)", text)

    def test_explains_the_noise_that_it_allows(self):
        rows = [
            Row("flutter", "Flutter", "", "measured", memory(300, 310), memory(300, 310)),
            Row("gio", "Gio", "", "measured", memory(200, 210), memory(200, 210)),
        ]
        text = self.render(rows)
        self.assertIn("more than 5 MB and more than 5% of the number on main", text)
        self.assertIn("12% for Gio, 15% for Flutter, which vary more", text)

    def test_is_ascii(self):
        self.render(self.rows()).encode("ascii")


class ReadArtifactTest(unittest.TestCase):
    def read(self, files, **overrides):
        github = FakeGitHub()
        artifact_id = github.add_artifact("uno", files, **overrides)
        artifact = github.artifacts[-1]
        self.assertEqual(artifact["id"], artifact_id)
        return report.read_artifact(github, artifact)

    def test_reads_memory_txt(self):
        self.assertEqual(self.read(memory_text((55, 42, 1), (64, 49, 1))), Memory(Point(55, 42, 1), Point(64, 49, 1)))

    def test_rejects_other_files(self):
        self.assertIsNone(self.read({"other.txt": memory_text()}))
        self.assertIsNone(self.read({"memory.txt": memory_text(), "extra.txt": "x"}))
        self.assertIsNone(self.read({"../memory.txt": memory_text()}))

    def test_rejects_an_expired_or_a_large_artifact_without_downloading_it(self):
        github = FakeGitHub()
        github.add_artifact("uno", memory_text(), expired=True)
        github.add_artifact("uno", memory_text(), size=report.MAX_ARTIFACT_BYTES + 1)
        github.raw = mock.Mock(side_effect=AssertionError("downloaded"))
        for artifact in github.artifacts:
            self.assertIsNone(report.read_artifact(github, artifact))

    def test_rejects_a_large_file_inside_the_zip(self):
        self.assertIsNone(self.read(memory_text() + "x" * (report.MAX_MEMORY_FILE_BYTES + 1)))

    def test_rejects_what_is_not_a_zip(self):
        github = FakeGitHub()
        artifact_id = github.add_artifact("uno", memory_text())
        github.zips[artifact_id] = b"not a zip"
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertIsNone(report.read_artifact(github, github.artifacts[-1]))

    def test_rejects_wrong_numbers(self):
        self.assertIsNone(self.read("memory (home): RSS many MB, PSS 1 MB in 1 processes\n"))


class BaselineTest(unittest.TestCase):
    def test_takes_the_newest_valid_artifact_of_the_default_branch(self):
        github = FakeGitHub()
        github.add_artifact("uno", memory_text((10, 9, 1), (10, 9, 1)), created="2026-10-18T00:00:00Z", sha="1" * 40)
        github.add_artifact("uno", memory_text((20, 19, 1), (20, 19, 1)), created="2026-10-20T00:00:00Z", sha="2" * 40)
        baseline, artifact = report.baseline_of(github, "uno", "main")
        self.assertEqual(baseline.home.pss, 19)
        self.assertEqual(artifact["workflow_run"]["head_sha"], "2" * 40)

    def test_ignores_pull_requests_forks_and_expired_artifacts(self):
        github = FakeGitHub()
        github.add_artifact("uno", memory_text(), branch="feature", created="2026-10-22T00:00:00Z")
        github.add_artifact("uno", memory_text(), head_repository_id=2, created="2026-10-22T00:00:00Z")
        github.add_artifact("uno", memory_text(), expired=True, created="2026-10-22T00:00:00Z")
        self.assertIsNone(report.baseline_of(github, "uno", "main"))

    def test_falls_back_to_an_older_artifact_when_the_newest_is_invalid(self):
        github = FakeGitHub()
        github.add_artifact("uno", memory_text((10, 9, 1), (10, 9, 1)), created="2026-10-18T00:00:00Z")
        github.add_artifact("uno", "garbage", created="2026-10-20T00:00:00Z")
        self.assertEqual(report.baseline_of(github, "uno", "main")[0].home.pss, 9)

    def test_only_reads_the_artifacts_of_the_implementation(self):
        github = FakeGitHub()
        github.add_artifact("tauri", memory_text())
        self.assertIsNone(report.baseline_of(github, "uno", "main"))

    def test_ignores_an_artifact_that_no_workflow_run_made(self):
        github = FakeGitHub()
        github.add_artifact("uno", memory_text())
        github.artifacts[-1]["workflow_run"] = None
        self.assertIsNone(report.baseline_of(github, "uno", "main"))


class PullRequestTest(unittest.TestCase):
    def test_finds_it_by_the_branch_of_the_run(self):
        github = FakeGitHub()
        github.add_pull_request(7)
        pulls = report.find_pull_requests(github, "contributor", "feature", HEAD)
        self.assertEqual([pull["number"] for pull in pulls], [7])

    def test_finds_it_by_the_commit_without_the_owner(self):
        github = FakeGitHub()
        github.add_pull_request(7)
        self.assertEqual([pull["number"] for pull in report.find_pull_requests(github, None, "feature", HEAD)], [7])

    def test_a_pull_request_that_has_a_newer_commit_is_not_the_one_of_the_run(self):
        github = FakeGitHub()
        github.add_pull_request(7, sha="c" * 40)
        self.assertEqual(report.find_pull_requests(github, "contributor", "feature", HEAD), [])

    def test_a_closed_pull_request_is_not_found(self):
        github = FakeGitHub()
        github.add_pull_request(7, state="closed")
        self.assertEqual(report.find_pull_requests(github, "contributor", "feature", HEAD), [])


class UpsertCommentTest(unittest.TestCase):
    def comment(self, github, body, login=report.BOT, number=7):
        comment = {"id": github.next_id(), "body": body, "user": {"login": login}}
        github.comments.setdefault(number, []).append(comment)
        return comment

    def test_creates_the_first_comment(self):
        github = FakeGitHub()
        self.assertEqual(report.upsert_comment(github, 7, f"{report.MARKER}\nnew"), "created")
        self.assertEqual(github.writes, [("POST", "issues/7/comments")])

    def test_replaces_the_earlier_comment_instead_of_adding_another(self):
        github = FakeGitHub()
        old = self.comment(github, f"{report.MARKER}\nold")
        self.assertEqual(report.upsert_comment(github, 7, f"{report.MARKER}\nnew"), "updated")
        self.assertEqual(github.writes, [("PATCH", f"issues/comments/{old['id']}")])
        self.assertEqual([c["body"] for c in github.comments[7]], [f"{report.MARKER}\nnew"])

    def test_leaves_an_identical_comment_alone(self):
        github = FakeGitHub()
        self.comment(github, f"{report.MARKER}\r\nsame\r\n")
        self.assertEqual(report.upsert_comment(github, 7, f"{report.MARKER}\nsame\n"), "unchanged")
        self.assertEqual(github.writes, [])

    def test_keeps_the_oldest_comment_and_deletes_the_rest(self):
        github = FakeGitHub()
        first = self.comment(github, f"{report.MARKER}\nfirst")
        second = self.comment(github, f"{report.MARKER}\nsecond")
        report.upsert_comment(github, 7, f"{report.MARKER}\nnew")
        self.assertEqual(
            github.writes, [("PATCH", f"issues/comments/{first['id']}"), ("DELETE", f"issues/comments/{second['id']}")]
        )
        self.assertEqual([c["body"] for c in github.comments[7]], [f"{report.MARKER}\nnew"])

    def test_never_touches_the_comments_of_other_people(self):
        github = FakeGitHub()
        theirs = self.comment(github, f"{report.MARKER}\nquoted by a person", login="contributor")
        other = self.comment(github, "a remark by the bot")
        self.assertEqual(report.upsert_comment(github, 7, f"{report.MARKER}\nnew"), "created")
        self.assertEqual(github.writes, [("POST", "issues/7/comments")])
        self.assertIn(theirs, github.comments[7])
        self.assertIn(other, github.comments[7])

    def test_tolerates_a_comment_of_a_deleted_user(self):
        github = FakeGitHub()
        github.comments[7] = [{"id": 1, "body": "text", "user": None}]
        self.assertEqual(report.upsert_comment(github, 7, f"{report.MARKER}\nnew"), "created")


class ReportTest(unittest.TestCase):
    """The whole report, against a fake GitHub."""

    # The numbers on main: (name of the workflow, Home as RSS, PSS, processes, Versions as RSS, PSS, processes).
    MAIN_NUMBERS = {
        "mygo": ("mygo", (55, 42, 1), (64, 49, 1)),
        "tauri": ("Tauri", (470, 283, 3), (492, 305, 3)),
        "uno": ("Uno", (110, 101, 1), (136, 127, 1)),
    }

    def setUp(self):
        self.github = FakeGitHub()
        for implementation, (name, home, versions) in self.MAIN_NUMBERS.items():
            self.github.add_workflow(implementation, name)
            self.github.add_artifact(implementation, memory_text(home, versions))
        self.github.add_pull_request(7)

    def run_report(self, **overrides):
        arguments = dict(
            implementations=["mygo", "tauri", "uno"],
            technologies={"mygo": "mygo 0.5, Go"},
            run_id=None,
            pull_request=7,
            dry_run=False,
        )
        arguments.update(overrides)
        output, errors = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
            status = report.report(self.github, **arguments)
        return status, output.getvalue(), errors.getvalue()

    def body(self) -> str:
        (comment,) = self.github.comments[7]
        return comment["body"]

    def test_comments_with_the_measured_implementation_and_main_for_the_rest(self):
        run = self.github.add_run("mygo", memory_text((56, 43, 1), (65, 50, 1)))
        status, _, _ = self.run_report(run_id=run, pull_request=None)
        self.assertEqual(status, 0)
        table = [line for line in self.body().splitlines() if line.startswith("|")]
        self.assertEqual([line.split("|")[1].strip() for line in table[2:]], ["mygo", "Uno (main)", "Tauri (main)"])
        self.assertIn("43 (+1)", self.body())
        self.assertIn("No change from main beyond the noise of CI in mygo.", self.body())

    def test_waits_for_the_other_implementations_that_are_still_running(self):
        run = self.github.add_run("mygo", memory_text())
        self.github.add_run("tauri", status="in_progress", conclusion=None)
        status, _, errors = self.run_report(run_id=run, pull_request=None)
        self.assertEqual(status, 0)
        self.assertIn("Waiting for Tauri", errors)
        self.assertEqual(self.github.comments, {})

    def test_waiting_takes_few_requests(self):
        # Every workflow that ends starts a report, and all but the last one only wait.
        run = self.github.add_run("mygo", memory_text())
        self.github.add_run("tauri", status="in_progress", conclusion=None)
        paths = []
        original = self.github.get

        def get(path, **params):
            paths.append(path)
            return original(path, **params)

        self.github.get = get
        self.github.raw = mock.Mock(side_effect=AssertionError("downloaded an artifact"))
        self.run_report(run_id=run, pull_request=None)
        self.assertLessEqual(len(paths), 6, paths)
        self.assertFalse([path for path in paths if "artifacts" in path], paths)

    def test_the_last_run_to_end_writes_the_report_with_all_of_them(self):
        self.github.add_run("tauri", memory_text((470, 283, 3), (492, 305, 3)))
        run = self.github.add_run("mygo", memory_text())
        self.run_report(run_id=run, pull_request=None)
        table = [line for line in self.body().splitlines() if line.startswith("|")]
        self.assertEqual([line.split("|")[1].strip() for line in table[2:]], ["mygo", "Uno (main)", "Tauri"])

    def test_the_run_that_ended_counts_as_ended_even_if_the_list_is_late(self):
        run = self.github.add_run("mygo", memory_text(), status="in_progress", conclusion=None)
        status, _, errors = self.run_report(run_id=run, pull_request=None)
        self.assertNotIn("Waiting", errors)
        self.assertIn("mygo", self.body())

    def test_a_failed_run_is_reported_as_such(self):
        self.github.add_run("tauri", conclusion="failure")
        run = self.github.add_run("mygo", memory_text())
        self.run_report(run_id=run, pull_request=None)
        self.assertIn("No result, the run failed or was cancelled: Tauri.", self.body())

    def test_an_invalid_artifact_is_reported_as_no_result(self):
        run = self.github.add_run("mygo", "not the numbers")
        self.run_report(run_id=run, pull_request=None)
        self.assertIn("No result, the run failed or was cancelled: mygo.", self.body())

    def test_the_second_report_replaces_the_comment_of_the_first(self):
        run = self.github.add_run("mygo", memory_text((56, 43, 1), (65, 50, 1)))
        self.run_report(run_id=run, pull_request=None)
        self.github.pulls[7]["head"]["sha"] = "c" * 40
        later = self.github.add_run("mygo", memory_text((60, 47, 1), (70, 54, 1)), sha="c" * 40)
        self.run_report(run_id=later, pull_request=None)
        self.assertEqual(len(self.github.comments[7]), 1)
        self.assertIn("47 (+5)", self.body())
        self.assertIn("`ccccccc`", self.body())
        self.assertEqual([method for method, _ in self.github.writes], ["POST", "PATCH"])

    def test_ignores_the_run_of_a_commit_that_is_no_longer_the_latest(self):
        run = self.github.add_run("mygo", memory_text())
        self.github.pulls[7]["head"]["sha"] = "c" * 40
        status, _, errors = self.run_report(run_id=run, pull_request=None)
        self.assertEqual(status, 0)
        self.assertIn("No open pull request has aaaaaaa", errors)
        self.assertEqual(self.github.writes, [])

    def test_does_not_comment_when_the_commit_changes_during_the_report(self):
        run = self.github.add_run("mygo", memory_text())
        original = self.github.get

        def get(path, **params):
            # The first read of the pull request is the one that finds it; later reads see a push.
            if path == "pulls/7":
                self.github.pulls[7]["head"]["sha"] = "c" * 40
            return original(path, **params)

        self.github.get = get
        status, _, errors = self.run_report(run_id=run, pull_request=None)
        self.assertEqual(status, 0)
        self.assertIn("has moved on", errors)
        self.assertEqual(self.github.writes, [])

    def test_ignores_runs_that_are_not_of_a_pull_request(self):
        run = self.github.add_run("mygo", memory_text(), event="push", sha=MAIN)
        status, _, errors = self.run_report(run_id=run, pull_request=None)
        self.assertEqual(status, 0)
        self.assertIn("not one of a pull request", errors)
        self.assertEqual(self.github.writes, [])

    def test_skips_a_workflow_that_no_longer_exists(self):
        self.github.workflows[1]["state"] = "deleted"
        self.run_report()
        self.assertNotIn("Tauri", self.body())

    def test_an_implementation_without_a_workflow_is_skipped(self):
        status, _, errors = self.run_report(implementations=["nothing"])
        self.assertEqual(status, 0)
        self.assertIn("None of the implementations has a workflow", errors)

    def test_manual_mode_does_not_wait(self):
        self.github.add_run("mygo", memory_text())
        self.github.add_run("tauri", status="in_progress", conclusion=None)
        self.run_report()
        self.assertIn("Still running: Tauri.", self.body())

    def test_manual_mode_on_a_closed_pull_request_does_nothing(self):
        self.github.pulls[7]["state"] = "closed"
        status, _, errors = self.run_report()
        self.assertEqual(status, 0)
        self.assertIn("is not open", errors)

    def test_dry_run_prints_and_writes_nothing(self):
        self.github.add_run("mygo", memory_text())
        _, output, _ = self.run_report(dry_run=True)
        self.assertTrue(output.startswith(report.MARKER))
        self.assertEqual(self.github.writes, [])

    def test_a_new_implementation_has_no_baseline_and_is_still_reported(self):
        self.github.artifacts = [a for a in self.github.artifacts if a["name"] != "memory-uno"]
        run = self.github.add_run("uno", memory_text())
        self.run_report(run_id=run, pull_request=None)
        self.assertIn("Uno (no baseline)", self.body())

    def test_writes_the_step_summary(self):
        self.github.add_run("mygo", memory_text())
        with tempfile.TemporaryDirectory() as directory:
            summary = Path(directory) / "summary.md"
            with mock.patch.dict("os.environ", {"GITHUB_STEP_SUMMARY": str(summary)}):
                self.run_report()
            text = summary.read_text(encoding="utf-8")
        self.assertTrue(text.startswith("### Memory use"))
        self.assertNotIn(report.MARKER, text)


class GitHubTest(unittest.TestCase):
    def completed(self, returncode=0, stdout=b"{}", stderr=b""):
        return subprocess.CompletedProcess([], returncode, stdout, stderr)

    def test_builds_the_endpoint_of_the_repository(self):
        github = report.GitHub(REPOSITORY)
        with mock.patch("subprocess.run", return_value=self.completed()) as run:
            github.get("actions/runs", head_sha="abc", per_page=100)
        endpoint = f"repos/{REPOSITORY}/actions/runs?head_sha=abc&per_page=100"
        self.assertEqual(run.call_args.args[0], ["gh", "api", "--method", "GET", endpoint])

    def test_retries_a_failure_of_the_server(self):
        github = report.GitHub(REPOSITORY)
        answers = [self.completed(1, b"", b"gh: HTTP 502"), self.completed(0, b'{"a": 1}')]
        with mock.patch("subprocess.run", side_effect=answers), mock.patch("time.sleep") as sleep:
            self.assertEqual(github.get("x"), {"a": 1})
        sleep.assert_called_once()

    def test_does_not_retry_a_refusal(self):
        github = report.GitHub(REPOSITORY)
        with mock.patch("subprocess.run", return_value=self.completed(1, b"", b"gh: Not Found (HTTP 404)")) as run:
            with self.assertRaises(report.ApiError):
                github.get("x")
        self.assertEqual(run.call_count, 1)

    def test_gives_up_after_three_attempts(self):
        github = report.GitHub(REPOSITORY)
        with mock.patch("subprocess.run", return_value=self.completed(1, b"", b"gh: HTTP 503")) as run:
            with mock.patch("time.sleep"), self.assertRaises(report.ApiError):
                github.get("x")
        self.assertEqual(run.call_count, 3)

    def test_creating_a_comment_is_tried_once(self):
        github = report.GitHub(REPOSITORY)
        with mock.patch("subprocess.run", return_value=self.completed(1, b"", b"gh: HTTP 502")) as run:
            with mock.patch("time.sleep"), self.assertRaises(report.ApiError):
                github.send("POST", "issues/7/comments", {"body": "x"})
        self.assertEqual(run.call_count, 1)

    def test_sends_the_body_on_standard_input_and_accepts_an_empty_answer(self):
        github = report.GitHub(REPOSITORY)
        with mock.patch("subprocess.run", return_value=self.completed(0, b"")) as run:
            self.assertIsNone(github.send("DELETE", "issues/comments/1"))
            github.send("PATCH", "issues/comments/1", {"body": "text"})
        self.assertEqual(run.call_args.kwargs["input"], b'{"body": "text"}')
        self.assertEqual(run.call_args.args[0][-2:], ["--input", "-"])

    def test_reads_every_page(self):
        github = report.GitHub(REPOSITORY)
        pages = [list(range(100)), list(range(5))]
        with mock.patch.object(report.GitHub, "get", side_effect=pages):
            self.assertEqual(len(list(github.pages("issues/7/comments"))), 105)


class MainTest(unittest.TestCase):
    def run_main(self, *arguments):
        with contextlib.redirect_stderr(io.StringIO()):
            return report.main(list(arguments))

    def test_rejects_a_repository_that_is_not_owner_and_name(self):
        with self.assertRaises(SystemExit):
            self.run_main("--repository", "x;y", "--implementations", "mygo", "--pull-request", "1")

    def test_rejects_an_implementation_that_is_not_a_plain_name(self):
        with self.assertRaises(SystemExit):
            self.run_main("--repository", REPOSITORY, "--implementations", "../mygo", "--pull-request", "1")
        with self.assertRaises(SystemExit):
            self.run_main("--repository", REPOSITORY, "--implementations", "", "--pull-request", "1")

    def test_needs_a_run_or_a_pull_request(self):
        with self.assertRaises(SystemExit):
            self.run_main("--repository", REPOSITORY, "--implementations", "mygo")

    def test_reports_a_failure_of_the_api_as_an_exit_code(self):
        with mock.patch.object(report, "report", side_effect=report.ApiError("boom")):
            status = self.run_main(
                "--repository", REPOSITORY, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"
            )
        self.assertEqual(status, 1)

    def test_the_warning_is_one_line_of_a_workflow_command(self):
        error = report.ApiError("POST x: gh: Resource not accessible by integration (HTTP 403)\nsecond line 100%")
        output = io.StringIO()
        with mock.patch.object(report, "report", side_effect=error), contextlib.redirect_stdout(output):
            self.run_main(
                "--repository", REPOSITORY, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"
            )
        self.assertEqual(len(output.getvalue().splitlines()), 1)
        self.assertIn("%0Asecond line 100%25", output.getvalue())

    def test_a_token_that_cannot_write_is_a_warning_and_not_a_failure(self):
        error = report.ApiError("POST issues/7/comments: gh: Resource not accessible by integration (HTTP 403)")
        output = io.StringIO()
        with mock.patch.object(report, "report", side_effect=error), contextlib.redirect_stdout(output):
            status = self.run_main(
                "--repository", REPOSITORY, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"
            )
        self.assertEqual(status, 0)
        self.assertTrue(output.getvalue().startswith("::warning::"))


if __name__ == "__main__":
    unittest.main()

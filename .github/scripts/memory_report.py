#!/usr/bin/env python3
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

"""Comments on a pull request with the memory that every implementation uses.

The workflow of each implementation (.github/workflows/<implementation>.yml)
measures its application with shared/tools/measure.sh (Flutter:
flutter/tool/measure.py) and uploads the result as the artifact
memory-<implementation>, which holds one file, memory.txt, with the lines
"memory (<point>): RSS <n> MB, PSS <n> MB in <n> processes" of the points home
and nodejs. .github/workflows/memory.yml runs this script when one of those
workflows ends. Once all of them have ended for the latest commit of the pull
request, the script writes one comment: a table like the one of docs/memory.md,
sorted by the PSS of the Home page, with the difference from the latest
measurement on the default branch next to the numbers. A later run replaces
that comment instead of adding another one.

An implementation that the pull request did not rebuild, because its workflow
only runs for changes in its own paths, shows the numbers of the default branch.

Usage:

  memory_report.py --run RUN_ID [--dry-run]
      Reports on the pull request of the workflow run RUN_ID, if that was the
      last of the implementation workflows to end for the commit.

  memory_report.py --pull-request NUMBER [--dry-run]
      Reports on that pull request now, with the implementations that are still
      running marked as such.

--dry-run prints the comment instead of writing it. The implementations come
from --implementations or MEMORY_IMPLEMENTATIONS (names separated by spaces).
Needs the gh CLI, which GH_TOKEN authenticates. The artifacts come from the
pull request, so they are not trusted: only the numbers in the lines above
are read from them.
"""

import argparse
import io
import json
import os
import re
import subprocess
import sys
import time
import urllib.parse
import zipfile
from collections import defaultdict
from collections.abc import Iterator
from dataclasses import dataclass
from pathlib import Path
from typing import Any

# The first line of the comment, which finds it again.
MARKER = "<!-- vfox-plus-memory-report -->"
# The author of the comments that GITHUB_TOKEN writes.
BOT = "github-actions[bot]"
WORKFLOWS = ".github/workflows"
MEMORY_FILE = "memory.txt"
# memory.txt has two short lines; an artifact much larger than that is not one.
MAX_ARTIFACT_BYTES = 64 * 1024
MAX_MEMORY_FILE_BYTES = 4 * 1024

# The same code does not use the same memory in every CI run. Comparing every
# two of the 107 runs on 2026-10-18 and 19 that shared their source with another
# one (30 groups of 2 to 8 runs of the ten implementations; see docs/memory.md),
# the Home and the Versions PSS never differed by more than 5 MB and 5 % of the
# first number for most of the implementations, and never by more than the
# percentages below for the four others. Only a larger difference counts as a
# change.
MIN_CHANGE_MB = 5
DEFAULT_TOLERANCE = 0.05
TOLERANCES = {"compose": 0.08, "flet": 0.08, "gio": 0.12, "flutter": 0.15}

LINE = re.compile(r"memory \((home|nodejs)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes")
IMPLEMENTATION = re.compile(r"[a-z0-9][a-z0-9-]*")
REPOSITORY = re.compile(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+")


@dataclass(frozen=True)
class Point:
    """The memory at one measuring point, in MB."""

    rss: int
    pss: int
    processes: int


@dataclass(frozen=True)
class Memory:
    """What an implementation uses on the Home page and with the versions of Node.js listed."""

    home: Point
    versions: Point


@dataclass
class Row:
    """One implementation of the report.

    The state is "measured" (the pull request rebuilt it), "inherited" (it did
    not, so the numbers are those of the default branch), "running" or "failed"
    (the workflow of the pull request has not ended, or ended without numbers)
    or "missing" (no workflow ran and the default branch has no numbers).
    """

    implementation: str
    name: str
    technology: str = ""
    state: str = "missing"
    memory: Memory | None = None
    baseline: Memory | None = None
    run_url: str = ""
    baseline_url: str = ""
    baseline_sha: str = ""

    @property
    def shown(self) -> Memory | None:
        """The numbers of the table."""
        if self.state == "measured":
            return self.memory
        if self.state == "inherited":
            return self.baseline
        return None


class ApiError(RuntimeError):
    """A request to the API of GitHub failed."""


def parse_memory(text: str) -> Memory | None:
    """Parses memory.txt: one line for the point home and one for nodejs, nothing else."""
    points: dict[str, Point] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        match = LINE.fullmatch(line)
        if match is None or match[1] in points:
            return None
        points[match[1]] = Point(rss=int(match[2]), pss=int(match[3]), processes=int(match[4]))
    if points.keys() != {"home", "nodejs"}:
        return None
    return Memory(home=points["home"], versions=points["nodejs"])


def tolerance(implementation: str, baseline: int) -> float:
    """The difference in MB from the baseline that is still the noise of CI."""
    return max(MIN_CHANGE_MB, TOLERANCES.get(implementation, DEFAULT_TOLERANCE) * baseline)


def exceeds_noise(implementation: str, baseline: int, current: int) -> bool:
    """Whether the difference is a change and not the variation between CI runs."""
    return abs(current - baseline) > tolerance(implementation, baseline)


def signed(number: int) -> str:
    return f"+{number}" if number > 0 else str(number)


def cell(text: str) -> str:
    """Text that cannot break out of a table cell."""
    return re.sub(r"\s+", " ", text).replace("|", "\\|").strip()


def read_technologies(text: str) -> dict[str, str]:
    """The Technology column of the first table of docs/memory.md, by implementation directory."""
    technologies: dict[str, str] = {}
    columns: tuple[int, int] | None = None
    for line in text.splitlines():
        if not line.startswith("|"):
            if columns is not None:
                break
            continue
        cells = [part.strip() for part in line.strip().strip("|").split("|")]
        if columns is None:
            if "Implementation" in cells and "Technology" in cells:
                columns = (cells.index("Implementation"), cells.index("Technology"))
            continue
        if len(cells) <= max(columns):
            continue
        link = re.search(r"\]\(\.\./([A-Za-z0-9_-]+)/README\.md", cells[columns[0]])
        if link:
            technologies[link[1]] = cells[columns[1]]
    return technologies


def pss_cell(row: Row, point: str) -> str:
    """The PSS of a point, with the difference from the default branch if it was measured."""
    shown = getattr(row.shown, point).pss
    if row.state != "measured" or row.baseline is None:
        return str(shown)
    before = getattr(row.baseline, point).pss
    difference = signed(shown - before)
    if exceeds_noise(row.implementation, before, shown):
        difference = f"**{difference}**"
    return f"{shown} ({difference})"


def table_cells(row: Row) -> list[str]:
    name = row.name
    if row.state == "inherited":
        name += " (main)"
    elif row.state == "measured" and row.baseline is None:
        name += " (no baseline)"
    shown = row.shown
    if shown is None:
        label = {"running": "running", "failed": "no result", "missing": "no data"}[row.state]
        if row.run_url:
            label = f"[{label}]({row.run_url})"
        return [cell(name), cell(row.technology), "-", label, "-", "-", "-", "-"]
    processes = str(shown.home.processes)
    if row.state == "measured" and row.baseline is not None:
        # The number of processes of an implementation does not vary between CI runs.
        difference = shown.home.processes - row.baseline.home.processes
        if difference:
            processes += f" (**{signed(difference)}**)"
    return [
        cell(name),
        cell(row.technology),
        processes,
        pss_cell(row, "home"),
        str(shown.home.rss),
        pss_cell(row, "versions"),
        str(shown.versions.rss),
        signed(shown.versions.pss - shown.home.pss),
    ]


def sort_key(row: Row) -> tuple[int, int, str]:
    group = {"measured": 0, "inherited": 0, "running": 1, "failed": 2, "missing": 3}[row.state]
    shown = row.shown
    return (group, shown.home.pss if shown else 0, row.name.lower())


def changes_of(row: Row) -> list[str]:
    """The differences from the default branch that are more than the noise."""
    if row.state != "measured" or row.memory is None or row.baseline is None:
        return []
    changes = []
    for label, point in (("Home PSS", "home"), ("Versions PSS", "versions")):
        before = getattr(row.baseline, point).pss
        after = getattr(row.memory, point).pss
        if exceeds_noise(row.implementation, before, after):
            percent = round(100 * (after - before) / before) if before else 0
            changes.append(f"{label} {signed(after - before)} MB ({signed(percent)}%)")
    if row.memory.home.processes != row.baseline.home.processes:
        changes.append(f"processes {row.baseline.home.processes} -> {row.memory.home.processes}")
    return changes


def summary(rows: list[Row]) -> list[str]:
    """The lines above the table: what changed, and what has no numbers."""
    lines = []
    compared = [row for row in rows if row.state == "measured" and row.baseline is not None]
    changed = {row.name: changes_of(row) for row in compared}
    changed = {name: parts for name, parts in changed.items() if parts}
    if changed:
        details = "; ".join(f"{name}: {', '.join(parts)}" for name, parts in changed.items())
        lines.append(f"Changed from main beyond the noise of CI: {details}.")
    same = ", ".join(row.name for row in compared if row.name not in changed)
    if same and changed:
        lines.append(f"Within the noise of CI: {same}.")
    elif same:
        lines.append(f"No change from main beyond the noise of CI in {same}.")
    without_baseline = [row.name for row in rows if row.state == "measured" and row.baseline is None]
    for text, names in (
        ("No numbers on main to compare with yet", without_baseline),
        ("Still running", [row.name for row in rows if row.state == "running"]),
        ("No result, the run failed or was cancelled", [row.name for row in rows if row.state == "failed"]),
        ("No numbers anywhere", [row.name for row in rows if row.state == "missing"]),
    ):
        if names:
            lines.append(f"{text}: {', '.join(names)}.")
    if all(row.state in ("inherited", "missing") for row in rows):
        lines.append("This pull request did not rebuild any implementation, so the numbers are those of main.")
    return lines


def explanation(rows: list[Row]) -> str:
    wider = defaultdict(list)
    for row in rows:
        if row.implementation in TOLERANCES:
            wider[TOLERANCES[row.implementation]].append(row.name)
    exceptions = ", ".join(
        f"{percent:.0%} for {' and '.join(names)}" for percent, names in sorted(wider.items())
    )
    return (
        "The same code does not use the same memory in every CI run, so a difference counts as a change only "
        f"when it is more than {MIN_CHANGE_MB} MB and more than {DEFAULT_TOLERANCE:.0%} of the number on main"
        + (f" ({exceptions}, which vary more)" if exceptions else "")
        + "."
    )


def render(rows: list[Row], *, head_sha: str) -> str:
    """The comment."""
    rows = sorted(rows, key=sort_key)
    header = [
        "Implementation",
        "Technology",
        "Processes",
        "Home PSS",
        "Home RSS",
        "Versions PSS",
        "Versions RSS",
        "The list adds (PSS)",
    ]
    alignment = ["---", "---", *["---:"] * 6]
    body = [table_cells(row) for row in rows]
    if not any(row.technology for row in rows):
        # Without the table of docs/memory.md there is nothing to put there.
        for line in (header, alignment, *body):
            del line[1]
    table = ["| " + " | ".join(line) + " |" for line in (header, alignment, *body)]

    runs = [f"[{row.name}]({row.run_url})" for row in rows if row.run_url]
    baselines = [
        f"[{row.name}]({row.baseline_url}) (`{row.baseline_sha[:7]}`)" for row in rows if row.baseline_url
    ]
    details = [
        "- Measured by `shared/tools/measure.sh` (Flutter: `flutter/tool/measure.py`) in the workflow of each "
        "implementation, on a GitHub-hosted runner with Xvfb and no GPU. These are not the numbers of "
        "`docs/memory.md`, which come from another machine: compare the rows of one comment.",
        "- " + explanation(rows),
    ]
    if runs:
        details.append("- Runs of this pull request: " + ", ".join(runs) + ".")
    if baselines:
        details.append("- Latest runs on main: " + ", ".join(baselines) + ".")

    lines = [
        MARKER,
        f"### Memory use at `{head_sha[:7]}`",
        "",
        *[f"- {line}" for line in summary(rows)],
        "",
        *table,
        "",
        "In MB, for the application and all of its child processes, sorted by the PSS of the Home page; "
        '"Versions" is the SDKs page with the available versions of Node.js listed. A number in parentheses is '
        "the difference from main, in bold when it is more than the noise of CI. "
        '"(main)" marks an implementation that this pull request did not rebuild: its numbers are those of main.',
        "",
        "<details>",
        "<summary>Details</summary>",
        "",
        *details,
        "</details>",
        "",
    ]
    return "\n".join(lines)


class GitHub:
    """The REST API of one repository, through the gh CLI (GH_TOKEN authenticates it)."""

    def __init__(self, repository: str) -> None:
        self.repository = repository

    def _request(self, endpoint: str, *, method: str = "GET", payload: Any = None, attempts: int = 3) -> bytes:
        command = ["gh", "api", "--method", method, endpoint]
        data = None
        if payload is not None:
            command += ["--input", "-"]
            data = json.dumps(payload).encode()
        for attempt in range(1, attempts + 1):
            result = subprocess.run(command, input=data, capture_output=True, check=False)
            if result.returncode == 0:
                return result.stdout
            error = result.stderr.decode(errors="replace").strip()
            transient = re.search(r"HTTP 5[0-9][0-9]|timeout|timed out|connection reset|EOF", error, re.IGNORECASE)
            if attempt == attempts or not transient:
                raise ApiError(f"{method} {endpoint}: {error}")
            time.sleep(2**attempt)
        raise AssertionError("unreachable")

    def _endpoint(self, path: str, params: dict[str, Any]) -> str:
        endpoint = f"repos/{self.repository}" + (f"/{path}" if path else "")
        return endpoint + ("?" + urllib.parse.urlencode(params) if params else "")

    def get(self, path: str, **params: Any) -> Any:
        return json.loads(self._request(self._endpoint(path, params)))

    def raw(self, path: str) -> bytes:
        """The body of a response that is not JSON, such as the zip of an artifact."""
        return self._request(self._endpoint(path, {}))

    def pages(self, path: str, key: str | None = None, **params: Any) -> Iterator[Any]:
        """Every item of a list, 100 at a time."""
        for page in range(1, 11):
            data = self.get(path, per_page=100, page=page, **params)
            items = data[key] if key else data
            yield from items
            if len(items) < 100:
                return

    def send(self, method: str, path: str, payload: Any = None) -> Any:
        # Creating a comment again after a lost response would write it twice.
        attempts = 1 if method == "POST" else 3
        body = self._request(self._endpoint(path, {}), method=method, payload=payload, attempts=attempts)
        return json.loads(body) if body.strip() else None


def log(message: str) -> None:
    print(message, file=sys.stderr, flush=True)


def read_artifact(gh: GitHub, artifact: dict[str, Any]) -> Memory | None:
    """The numbers in an artifact, or None if it is not a valid memory.txt."""
    if artifact.get("expired") or artifact.get("size_in_bytes", 0) > MAX_ARTIFACT_BYTES:
        return None
    try:
        data = gh.raw(f"actions/artifacts/{artifact['id']}/zip")
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            files = archive.infolist()
            if len(files) != 1 or files[0].filename != MEMORY_FILE:
                return None
            with archive.open(files[0]) as file:
                content = file.read(MAX_MEMORY_FILE_BYTES + 1)
    except (zipfile.BadZipFile, ApiError) as error:
        log(f"Cannot read the artifact {artifact.get('name')} ({artifact.get('id')}): {error}")
        return None
    if len(content) > MAX_MEMORY_FILE_BYTES:
        return None
    return parse_memory(content.decode("utf-8", errors="replace"))


def newest(artifacts: list[dict[str, Any]]) -> dict[str, Any] | None:
    live = [artifact for artifact in artifacts if not artifact.get("expired")]
    return max(live, key=lambda artifact: artifact["created_at"], default=None)


def baseline_of(gh: GitHub, implementation: str, default_branch: str) -> tuple[Memory, dict[str, Any]] | None:
    """The latest numbers of an implementation that were measured on the default branch."""
    artifacts = gh.get("actions/artifacts", name=f"memory-{implementation}", per_page=100)["artifacts"]
    # A pull request from a fork can have a branch called like the default branch.
    trusted = [
        artifact
        for artifact in artifacts
        if (artifact.get("workflow_run") or {}).get("head_branch") == default_branch
        and artifact["workflow_run"].get("head_repository_id") == artifact["workflow_run"].get("repository_id")
    ]
    for artifact in sorted(trusted, key=lambda candidate: candidate["created_at"], reverse=True):
        memory = read_artifact(gh, artifact)
        if memory is not None:
            return memory, artifact
    return None


def find_pull_requests(gh: GitHub, owner: str | None, branch: str, sha: str) -> list[dict[str, Any]]:
    """The open pull requests whose latest commit is sha."""
    pulls: list[dict[str, Any]] = []
    if owner:
        pulls = gh.get("pulls", state="open", head=f"{owner}:{branch}", per_page=100)
    if not pulls:
        pulls = [pull for pull in gh.get(f"commits/{sha}/pulls", per_page=100) if pull["state"] == "open"]
    return [pull for pull in pulls if pull["head"]["sha"] == sha]


def workflow_path(implementation: str) -> str:
    return f"{WORKFLOWS}/{implementation}.yml"


def active_workflows(gh: GitHub, implementations: list[str]) -> dict[str, dict[str, Any]]:
    """The workflows of the implementations, by implementation; one that is gone or disabled is left out."""
    workflows = {workflow["path"]: workflow for workflow in gh.get("actions/workflows", per_page=100)["workflows"]}
    return {
        implementation: workflows[workflow_path(implementation)]
        for implementation in implementations
        if workflows.get(workflow_path(implementation), {}).get("state") == "active"
    }


def latest_runs(gh: GitHub, head_sha: str) -> dict[str, dict[str, Any]]:
    """The latest run of every workflow of the pull request for a commit, by the path of the workflow."""
    runs: dict[str, dict[str, Any]] = {}
    for run in gh.get("actions/runs", head_sha=head_sha, event="pull_request", per_page=100)["workflow_runs"]:
        if run["path"] not in runs or run["id"] > runs[run["path"]]["id"]:
            runs[run["path"]] = run
    return runs


def unfinished(
    workflows: dict[str, dict[str, Any]], runs: dict[str, dict[str, Any]], finished_run: int | None
) -> list[str]:
    """The names of the workflows with a run that has not ended, other than the run that has just ended."""
    return [
        workflow["name"]
        for implementation, workflow in workflows.items()
        if (run := runs.get(workflow_path(implementation)))
        and run["status"] != "completed"
        and run["id"] != finished_run
    ]


def collect(
    gh: GitHub,
    *,
    workflows: dict[str, dict[str, Any]],
    runs: dict[str, dict[str, Any]],
    technologies: dict[str, str],
    default_branch: str,
    finished_run: int | None,
) -> list[Row]:
    """The rows of the report: the numbers of the pull request and of the default branch."""
    rows = []
    for implementation, workflow in workflows.items():
        row = Row(implementation, workflow["name"], technologies.get(implementation, ""))
        run = runs.get(workflow_path(implementation))
        if run is None:
            row.state = "inherited"
        else:
            row.run_url = run["html_url"]
            if run["status"] != "completed" and run["id"] != finished_run:
                row.state = "running"
            else:
                artifacts = gh.get(
                    f"actions/runs/{run['id']}/artifacts", name=f"memory-{implementation}", per_page=100
                )["artifacts"]
                artifact = newest(artifacts)
                row.memory = read_artifact(gh, artifact) if artifact else None
                row.state = "measured" if row.memory else "failed"
        if row.state in ("measured", "inherited"):
            baseline = baseline_of(gh, implementation, default_branch)
            if baseline is not None:
                memory, artifact = baseline
                row.baseline = memory
                row.baseline_sha = artifact["workflow_run"]["head_sha"]
                row.baseline_url = f"https://github.com/{gh.repository}/actions/runs/{artifact['workflow_run']['id']}"
            elif row.state == "inherited":
                row.state = "missing"
        rows.append(row)
    return rows


def same_text(left: str, right: str) -> bool:
    return left.replace("\r\n", "\n").strip() == right.replace("\r\n", "\n").strip()


def upsert_comment(gh: GitHub, number: int, body: str) -> str:
    """Writes the comment of the pull request, or replaces the one that an earlier run wrote."""
    mine = sorted(
        (
            comment
            for comment in gh.pages(f"issues/{number}/comments")
            if (comment.get("user") or {}).get("login") == BOT and comment["body"].startswith(MARKER)
        ),
        key=lambda comment: comment["id"],
    )
    if not mine:
        gh.send("POST", f"issues/{number}/comments", {"body": body})
        return "created"
    result = "unchanged"
    if not same_text(mine[0]["body"], body):
        gh.send("PATCH", f"issues/comments/{mine[0]['id']}", {"body": body})
        result = "updated"
    # Two runs at the same time can both have created one.
    for extra in mine[1:]:
        gh.send("DELETE", f"issues/comments/{extra['id']}")
    return result


def report(
    gh: GitHub,
    *,
    implementations: list[str],
    technologies: dict[str, str],
    run_id: int | None,
    pull_request: int | None,
    dry_run: bool,
) -> int:
    # Most runs end here, on the way to the one that finds that all of them have: keep it to a few requests.
    if run_id is not None:
        run = gh.get(f"actions/runs/{run_id}")
        if run["event"] != "pull_request":
            log(f"The run {run_id} is a {run['event']} run, not one of a pull request.")
            return 0
        head_sha = run["head_sha"]
        owner = ((run.get("head_repository") or {}).get("owner") or {}).get("login")
        pulls = find_pull_requests(gh, owner, run["head_branch"], head_sha)
    else:
        pull = gh.get(f"pulls/{pull_request}")
        if pull["state"] != "open":
            log(f"The pull request {pull_request} is not open.")
            return 0
        head_sha = pull["head"]["sha"]
        pulls = [pull]
    if not pulls:
        log(f"No open pull request has {head_sha[:7]} as its latest commit; there is nothing to report on.")
        return 0
    workflows = active_workflows(gh, implementations)
    if not workflows:
        log("None of the implementations has a workflow.")
        return 0
    runs = latest_runs(gh, head_sha)
    waiting = unfinished(workflows, runs, run_id) if run_id is not None else []
    if waiting:
        log(f"Waiting for {', '.join(waiting)}; the report is written when they have ended.")
        return 0

    rows = collect(
        gh,
        workflows=workflows,
        runs=runs,
        technologies=technologies,
        default_branch=gh.get("")["default_branch"],
        finished_run=run_id,
    )
    body = render(rows, head_sha=head_sha)
    summary_file = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_file:
        with open(summary_file, "a", encoding="utf-8") as file:
            file.write(body.removeprefix(MARKER + "\n"))
    if dry_run:
        print(body)
        return 0
    for pull in pulls:
        # A commit that came while the numbers were collected has its own report.
        current = gh.get(f"pulls/{pull['number']}")
        if current["state"] != "open" or current["head"]["sha"] != head_sha:
            log(f"The pull request {pull['number']} has moved on from {head_sha[:7]}.")
            continue
        log(f"Pull request {pull['number']}: comment {upsert_comment(gh, pull['number'], body)}.")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--repository", default=os.environ.get("GITHUB_REPOSITORY", ""), help="OWNER/NAME")
    parser.add_argument(
        "--implementations",
        default=os.environ.get("MEMORY_IMPLEMENTATIONS", ""),
        help="the implementations, such as 'mygo tauri', separated by spaces",
    )
    parser.add_argument("--docs", type=Path, default=Path("docs/memory.md"), help="the page with the Technology column")
    parser.add_argument("--dry-run", action="store_true", help="print the comment instead of writing it")
    target = parser.add_mutually_exclusive_group(required=True)
    target.add_argument("--run", type=int, help="the ID of the workflow run that has just ended")
    target.add_argument("--pull-request", type=int, help="the number of the pull request")
    arguments = parser.parse_args(argv)

    implementations = arguments.implementations.split()
    if not REPOSITORY.fullmatch(arguments.repository):
        parser.error("--repository (or GITHUB_REPOSITORY) must be OWNER/NAME")
    if not implementations or not all(IMPLEMENTATION.fullmatch(name) for name in implementations):
        parser.error("--implementations (or MEMORY_IMPLEMENTATIONS) must list names such as 'mygo tauri'")
    technologies = {}
    if arguments.docs.is_file():
        technologies = read_technologies(arguments.docs.read_text(encoding="utf-8"))
    try:
        return report(
            GitHub(arguments.repository),
            implementations=implementations,
            technologies=technologies,
            run_id=arguments.run,
            pull_request=arguments.pull_request,
            dry_run=arguments.dry_run,
        )
    except ApiError as error:
        if "Resource not accessible by integration" in str(error):
            # A token that cannot write, such as that of a run for a bot's pull request: the report is
            # already in the summary of the run, which is as far as it can go.
            message = f"The token of this run cannot write the comment; the report is in the summary: {error}"
            # The encoding of the data of a workflow command.
            message = message.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
            print(f"::warning::{message}")
            return 0
        log(f"memory_report.py: {error}")
        return 1


if __name__ == "__main__":
    sys.exit(main())

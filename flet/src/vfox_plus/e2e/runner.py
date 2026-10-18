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

"""Runs the end-to-end scenario named by VFOX_PLUS_E2E_SCENARIO in the
running application, as the runner of the web front end does: every step is
reported as `e2e: …` on the output of the application, which then ends with
exit code 0 when the scenario passed and 1 when it failed. `web/e2e/run.sh`,
`shared/tools/measure.sh` and `shared/tools/screenshots.sh` start the
application and read the reports.
"""

import os
import sys
import time
import traceback
from typing import TYPE_CHECKING

from vfox_plus.e2e.driver import Driver
from vfox_plus.e2e.scenarios import SCENARIOS, ScenarioContext

if TYPE_CHECKING:
    from vfox_plus.ui.app import Application


def report(message: str) -> None:
    """Writes `message` on the standard output of the application.

    The built application sends what Python prints to its log file only, so
    the report goes to the file descriptor of the process directly.
    """
    os.write(1, f"e2e: {message}\n".encode())


def finish(code: int) -> None:
    """Ends the application with `code`.

    The Python interpreter runs inside the Flutter application, so leaving the
    process ends the window too; `sys.exit()` would only end the task of the
    scenario.
    """
    sys.stdout.flush()
    sys.stderr.flush()
    os._exit(code)


async def run_scenario(application: Application, name: str) -> None:
    """Runs the scenario `name` and ends the application."""
    scenario = SCENARIOS.get(name)
    if scenario is None:
        report(f'unknown scenario "{name}"; use one of {", ".join(SCENARIOS)}')
        finish(2)
        return
    context = ScenarioContext(application, Driver(application.page), dict(os.environ), report)
    started = time.monotonic()
    try:
        report(f"scenario {name} started")
        await scenario(context)
        report(f"scenario {name} passed in {round(time.monotonic() - started)} s")
        code = 0
    except Exception:  # noqa: BLE001 - every failure fails the scenario.
        report(f"scenario {name} FAILED: {traceback.format_exc()}")
        code = 1
    finish(code)

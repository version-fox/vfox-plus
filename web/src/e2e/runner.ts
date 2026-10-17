// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * The end-to-end test runner, bundled only into builds made with
 * `VFOX_PLUS_E2E=1`. It runs the scenario named by `VFOX_PLUS_E2E_SCENARIO`
 * inside the page, reports every step on the standard output of the
 * application (`e2e: …`) and ends the application with exit code 0 when the
 * scenario passed, 1 when it failed.
 */

import type { Host } from '../core/host';
import { fakeScenario, measureScenario, realScenario, screenshotsScenario, type ScenarioContext } from './scenarios';

const SCENARIOS: Readonly<Record<string, (context: ScenarioContext) => Promise<void>>> = {
  fake: fakeScenario,
  real: realScenario,
  measure: measureScenario,
  screenshots: screenshotsScenario,
};

/** Runs the scenario named by the environment and exits. */
export async function runScenario(host: Host): Promise<void> {
  const environment = await host.environment();
  const name = environment['VFOX_PLUS_E2E_SCENARIO'] ?? '';
  const scenario = SCENARIOS[name];
  const context: ScenarioContext = {
    host,
    environment,
    step: (message) => host.log(`e2e: ${message}`),
  };
  if (scenario === undefined) {
    await context.step(`unknown scenario "${name}"; use one of ${Object.keys(SCENARIOS).join(', ')}`);
    await host.exit(2);
    return;
  }
  const started = Date.now();
  try {
    await context.step(`scenario ${name} started`);
    await scenario(context);
    await context.step(`scenario ${name} passed in ${Math.round((Date.now() - started) / 1000)} s`);
    await host.exit(0);
  } catch (error) {
    // WebKit puts only the frames into the stack, without the message.
    const details = error instanceof Error ? `${error.message}\n${error.stack ?? ''}` : String(error);
    await context.step(`scenario ${name} FAILED: ${details}`);
    await host.exit(1);
  }
}

/*
 * Copyright 2026 Han Li and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// How a terminal activates vfox: the choice among the shells of the computer that
// runs it, and what to do for the chosen one.

import type { Activation } from "./api";
import { t } from "./i18n";

/** The activation of the shell that was chosen, or of the first one, which is the shell of the person. */
export function chosenActivation(activations: Activation[], shell: string): Activation | null {
  return activations.find((a) => a.shell === shell) || activations[0] || null;
}

/** What to do with the command of an activation. */
export function activationText(activation: Activation): string {
  switch (activation.kind) {
    case "once":
      return t("shellOnce").replace("{shell}", () => activation.name);
    case "script":
      return t("shellScript");
    default:
      // A function, so that no `$` of a file such as $PROFILE is read as a pattern.
      return t("shellStartup").replace("{file}", () => activation.file || "");
  }
}

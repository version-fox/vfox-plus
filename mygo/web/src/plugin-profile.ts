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

import type { Profile, Variable } from "./api";

// needsConfirmation reports whether the value comes from a profile that the
// application did not ship, so that the page shows it and asks before it sets
// it. An empty value takes the variable away, which asks nothing.
export function needsConfirmation(variable: Variable, value: string): boolean {
  const chosen = value.trim();
  return chosen !== "" && (variable.unvetted || []).includes(chosen);
}

// canDetect reports whether the plugin may find installs that are on the machine
// already, which the page offers to link: unless it says that it cannot. An
// application that does not say is older.
export function canDetect(profile: Profile | null): boolean {
  return profile?.capabilities?.detect !== false;
}

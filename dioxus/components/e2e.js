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

// The driver of the end-to-end scenarios (src/ui/e2e.rs), which finds and
// clicks elements as a user would. It is the driver of the web front end;
// build.mjs bundles it into assets/e2e-driver.js, which only builds with the
// e2e feature carry.

import * as driver from '../../web/src/e2e/driver.ts';

/** Clicks the visible element matching `selector` whose text is `text`, or starts with it. */
function clickWith(selector, text, { prefix = false } = {}) {
  const matches = (element) => (prefix ? driver.ownText(element).startsWith(text) : driver.ownText(element) === text);
  return driver.clickElement(`${selector} "${text}"`, selector, matches);
}

window.vfoxPlusDriver = {
  ...driver,
  clickWith,
  /** How many visible elements show exactly `text`. */
  countText: (text) => driver.findText(text).length,
};

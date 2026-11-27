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

import { readdirSync, readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// An icon that the page refers to and its sprite lacks draws nothing, which went unseen for the filled icon of the
// settings: it is the one that is shown while the settings are open.
describe("icons", () => {
  const page = readFileSync(new URL("../index.html", import.meta.url), "utf8");
  const sources = readdirSync(new URL(".", import.meta.url))
    .filter((name) => name.endsWith(".ts") && !name.endsWith(".test.ts"))
    .map((name) => readFileSync(new URL(name, import.meta.url), "utf8"));
  const symbols = new Set([...page.matchAll(/<symbol id="(i-[a-z0-9-]+)"/g)].map((m) => m[1]));
  const referred = new Set<string>();
  for (const text of [page, ...sources]) {
    for (const m of text.matchAll(/#(i-[a-z0-9-]+)/g)) referred.add(m[1]);
    for (const m of text.matchAll(/\bicon\(\s*["']([a-z0-9-]+)["']/g)) referred.add("i-" + m[1]);
    for (const m of text.matchAll(/\bicon:\s*["']([a-z0-9-]+)["']/g)) referred.add("i-" + m[1]);
  }

  it("has a symbol for every icon that it names", () => {
    expect([...referred].filter((name) => !symbols.has(name))).toEqual([]);
  });

  it("draws the open destination filled", () => {
    for (const tab of ["home", "deployed-code", "extension", "pending-actions", "settings"]) {
      expect(referred, tab).toContain("i-" + tab + "-fill");
    }
  });
});

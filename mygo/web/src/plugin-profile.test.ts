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

import { describe, expect, it } from "vitest";
import type { Profile, Variable } from "./api";
import { canDetect, needsConfirmation } from "./plugin-profile";

const variable: Variable = {
  key: "VFOX_X_MIRROR",
  label: { en: "Mirror", zh: "镜像" },
  help: { en: "h", zh: "h" },
  value: "",
  kind: "url",
  presets: [],
  unvetted: ["https://new.example.org"],
};

describe("needsConfirmation", () => {
  it("asks for the values that the application did not ship, and no other", () => {
    expect(needsConfirmation(variable, " https://new.example.org ")).toBe(true);
    expect(needsConfirmation(variable, "https://typed.example.org")).toBe(false);
    expect(needsConfirmation(variable, "")).toBe(false);
    expect(needsConfirmation({ ...variable, unvetted: undefined }, "https://new.example.org")).toBe(false);
    expect(needsConfirmation({ ...variable, kind: "toggle", on: "1", unvetted: ["1"] }, "1")).toBe(true);
  });
});

describe("canDetect", () => {
  it("offers detection unless the plugin says it cannot", () => {
    const profile = { capabilities: { detect: false } } as Profile;
    expect(canDetect(profile)).toBe(false);
    expect(canDetect({ capabilities: { detect: true } } as Profile)).toBe(true);
    expect(canDetect({} as Profile)).toBe(true);
    expect(canDetect(null)).toBe(true);
  });
});

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

import { afterEach, describe, expect, it } from "vitest";
import type { Activation } from "./api";
import { setLang } from "./i18n";
import { activationText, chosenActivation } from "./shell";

const pwsh: Activation = { shell: "pwsh", name: "PowerShell", kind: "startup", file: "$PROFILE", command: 'Invoke-Expression "$(vfox activate pwsh)"' };
const nushell: Activation = { shell: "nushell", name: "Nushell", kind: "once", command: "vfox activate nushell" };
const clink: Activation = { shell: "clink", name: "Clink", kind: "script", command: "clink info", url: "https://example.org/clink_vfox.lua" };

describe("chosenActivation", () => {
  it("is the shell of the person until they choose another", () => {
    expect(chosenActivation([pwsh, nushell], "")).toBe(pwsh);
    expect(chosenActivation([pwsh, nushell], "nushell")).toBe(nushell);
  });
  it("falls back to the first when the machine does not offer the chosen shell", () => {
    expect(chosenActivation([pwsh], "bash")).toBe(pwsh);
  });
  it("is nothing without any", () => {
    expect(chosenActivation([], "bash")).toBeNull();
  });
});

describe("activationText", () => {
  afterEach(() => setLang("en"));
  it("names the startup file, whose $ a replacement leaves alone", () => {
    expect(activationText(pwsh)).toBe("Add this line to $PROFILE so that new shells use the versions chosen here.");
  });
  it("names the shell of a command that is run once", () => {
    expect(activationText(nushell)).toBe("Run this once in Nushell so that new shells use the versions chosen here.");
  });
  it("says where a script goes", () => {
    expect(activationText(clink)).toContain("clink_vfox.lua");
  });
  it("speaks the language of the page", () => {
    setLang("zh");
    expect(activationText(pwsh)).toBe("将下面这行添加到 $PROFILE，新的 Shell 就会使用这里选择的版本。");
  });
});

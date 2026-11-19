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
import { webAddress } from "./links";

describe("webAddress", () => {
  it("accepts absolute http and https addresses", () => {
    expect(webAddress("https://github.com/version-fox/vfox-dart")).toBe("https://github.com/version-fox/vfox-dart");
    expect(webAddress(" http://127.0.0.1:8080/app ")).toBe("http://127.0.0.1:8080/app");
    expect(webAddress("HTTPS://example.com/docs")).toBe("https://example.com/docs");
  });

  it("refuses every other scheme, and anything that is not an absolute URL", () => {
    const refused = [
      "javascript:alert(1)",
      "JavaScript:alert(1)",
      " javascript:alert(1)",
      "data:text/html,<script>alert(1)</script>",
      "file:///C:/Windows/System32/calc.exe",
      "ms-msdt:/id PCWDiagnostic",
      "ftp://example.com/installer.exe",
      "mailto:someone",
      "https://",
      "github.com/version-fox/vfox-dart",
      "",
    ];
    for (const raw of refused) {
      expect(webAddress(raw), raw).toBeNull();
    }
  });
});

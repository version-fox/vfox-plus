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
import { aliasesOf, avatar, esc, icon, txt } from "./dom";

// The page writes both double- and single-quoted attributes, so esc must
// neutralize both quotes along with & < and > (replaces the Go regex test
// that pinned the exact source text of esc).
describe("esc", () => {
  it("escapes the HTML metacharacters", () => {
    expect(esc('<a href="x">y & z</a>')).toBe("&lt;a href=&quot;x&quot;&gt;y &amp; z&lt;/a&gt;");
  });
  it("escapes the single quote for single-quoted attributes and JS strings", () => {
    expect(esc("it's")).toBe("it&#39;s");
  });
  it("treats null and undefined as empty", () => {
    expect(esc(null)).toBe("");
    expect(esc(undefined)).toBe("");
  });
  it("passes plain text through", () => {
    expect(esc("flutter 3.47.7-x64")).toBe("flutter 3.47.7-x64");
  });
});

describe("icon", () => {
  it("references the symbol", () => {
    expect(icon("home")).toContain('href="#i-home"');
  });
  it("adds the size class", () => {
    expect(icon("home", "sm")).toContain('class="ic sm"');
  });
});

describe("avatar", () => {
  it("is deterministic per name", () => {
    expect(avatar("flutter")).toBe(avatar("flutter"));
  });
  it("shows the uppercased first letter", () => {
    expect(avatar("flutter")).toContain(">F<");
  });
});

describe("txt", () => {
  it("picks the language", () => {
    expect(txt({ en: "Hello", zh: "你好" }, "zh")).toBe("你好");
    expect(txt({ en: "Hello", zh: "你好" }, "en")).toBe("Hello");
  });
  it("falls back to English", () => {
    expect(txt({ en: "Hello", zh: "" }, "zh")).toBe("Hello");
  });
  it("passes strings through", () => {
    expect(txt("plain", "zh")).toBe("plain");
  });
});

describe("aliasesOf", () => {
  const aliases = [
    { sdk: "java", alias: "j17", target: "java@17.0.9-tem" },
    { sdk: "java", alias: "lts", target: "java@17.0.9-tem" },
    { sdk: "java", alias: "j21", target: "21.0.1-tem" },
    { sdk: "nodejs", alias: "j17", target: "nodejs@17.0.9-tem" },
  ];
  it("names the aliases of a version, of that plugin only", () => {
    expect(aliasesOf(aliases, "java", "17.0.9-tem")).toBe("@j17, @lts");
  });
  it("takes a target that is only the version", () => {
    expect(aliasesOf(aliases, "java", "21.0.1-tem")).toBe("@j21");
  });
  it("is empty for a version without one", () => {
    expect(aliasesOf(aliases, "java", "11.0.2")).toBe("");
  });
});

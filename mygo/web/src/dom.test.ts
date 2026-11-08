import { describe, expect, it } from "vitest";
import { avatar, esc, icon, txt } from "./dom";

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

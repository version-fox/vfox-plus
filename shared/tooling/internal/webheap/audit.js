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

// The census of one screen of the web interface: what a person reads, and what could not be read. `webheap -audit`
// evaluates it in the settled page, and it changes nothing. docs/en/dev/design-system.md has the budgets that the
// numbers are held to, and docs/en/dev/contributing.md the way to take the census of every screen.
(() => {
  const page = document.documentElement;
  const width = page.clientWidth;
  const dark = matchMedia("(prefers-color-scheme: dark)").matches;

  // ---- colour: the contrast of a text against what it stands on, with the opacity of everything above it
  const parse = (value) => {
    let found = /rgba?\(([^)]+)\)/.exec(value);
    if (found) {
      const parts = found[1].split(/[\s,/]+/).filter(Boolean).map(Number);
      return [parts[0], parts[1], parts[2], parts.length > 3 ? parts[3] : 1];
    }
    found = /color\(srgb\s+([^)]+)\)/.exec(value);
    if (found) {
      const parts = found[1].split(/[\s/]+/).filter(Boolean).map(Number);
      return [parts[0] * 255, parts[1] * 255, parts[2] * 255, parts.length > 3 ? parts[3] : 1];
    }
    return null;
  };
  const over = (top, bottom) => {
    const alpha = top[3] + bottom[3] * (1 - top[3]);
    if (alpha === 0) return [0, 0, 0, 0];
    return [0, 1, 2].map((i) => (top[i] * top[3] + bottom[i] * bottom[3] * (1 - top[3])) / alpha).concat([alpha]);
  };
  const luminance = (colour) => {
    const channel = (v) => {
      v /= 255;
      return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
    };
    return 0.2126 * channel(colour[0]) + 0.7152 * channel(colour[1]) + 0.0722 * channel(colour[2]);
  };
  const ratio = (a, b) => {
    const [light, dim] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (light + 0.05) / (dim + 0.05);
  };
  const opacity = (element) => {
    let all = 1;
    for (let e = element; e; e = e.parentElement) all *= parseFloat(getComputedStyle(e).opacity);
    return all;
  };
  const background = (element) => {
    const chain = [];
    for (let e = element; e; e = e.parentElement) chain.push(e);
    let colour = [255, 255, 255, 1];
    for (let i = chain.length - 1; i >= 0; i--) {
      const own = parse(getComputedStyle(chain[i]).backgroundColor);
      if (own && own[3] > 0) colour = over(own, colour);
    }
    return colour;
  };

  // ---- what is on the screen
  const visible = (element) => {
    const box = element.getBoundingClientRect();
    if (box.width < 1 || box.height < 1) return false;
    const style = getComputedStyle(element);
    return style.visibility !== "hidden" && style.display !== "none" && opacity(element) > 0.05;
  };
  const words = (text) =>
    (text.match(/[A-Za-z0-9][A-Za-z0-9'.\-/]*/g) || []).length + (text.match(/[一-鿿]/g) || []).length;
  const describe = (element) =>
    element.tagName.toLowerCase() +
    (element.id ? "#" + element.id : "") +
    (typeof element.className === "string" && element.className.trim()
      ? "." + element.className.trim().split(/\s+/).slice(0, 2).join(".")
      : "");

  const texts = [];
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let node; (node = walker.nextNode()); ) {
    const text = node.nodeValue.replace(/\s+/g, " ").trim();
    const element = node.parentElement;
    if (!text || !element || ["SCRIPT", "STYLE", "NOSCRIPT", "TITLE"].includes(element.tagName) || !visible(element)) continue;
    const style = getComputedStyle(element);
    texts.push({
      element,
      text,
      words: words(text),
      size: parseFloat(style.fontSize),
      mono: /mono|courier|consolas/i.test(style.fontFamily),
    });
  }

  // ---- contrast: WCAG AA, 4.5 for a text and 3 for a large one
  const failures = [];
  let measured = 0;
  for (const { element, text, size } of texts) {
    if (element.closest("[disabled],[aria-disabled=true]")) continue;
    const style = getComputedStyle(element);
    const colour = parse(style.color);
    if (!colour) continue;
    const behind = background(element);
    const shown = over([colour[0], colour[1], colour[2], colour[3] * opacity(element)], behind);
    const found = ratio(shown, behind);
    const large = size >= 24 || (size >= 18.66 && parseInt(style.fontWeight, 10) >= 700);
    measured++;
    if (found < (large ? 3 : 4.5)) failures.push({ ratio: +found.toFixed(2), text: text.slice(0, 34), at: describe(element) });
  }
  failures.sort((a, b) => a.ratio - b.ratio);

  // ---- copy: the sentences, and the small lines under a title that say what the title says
  const prose = texts.filter((t) => t.words >= 8).sort((a, b) => b.words - a.words);
  const helpers = texts.filter(
    (t) => t.size <= 13 && t.words >= 4 && !t.mono && !t.element.closest("button,nav,.navbar,.rail,.tag,.chip,code"),
  );

  // ---- actions: a row has one action, and the others are in its menu; a split button is one action, and so are the
  // segments of a segmented control
  const rows = [...document.querySelectorAll(".list > *")].filter(visible);
  const twoActions = rows.filter(
    (row) =>
      [...row.querySelectorAll("button,[role=button],a[href]")].filter((b) => visible(b) && words(b.textContent) > 0 && !b.closest(".seg")).length >= 2,
  ).length;

  // ---- icons: a list in which three rows or more start with the same icon has an icon that says nothing
  const wallpaper = [];
  for (const list of document.querySelectorAll(".list")) {
    if (!visible(list)) continue;
    const names = {};
    for (const row of list.children) {
      const lead = row.firstElementChild;
      const icon = lead && (lead.matches("svg.ic") ? lead : lead.matches(".icon-tile,.lead") ? lead.querySelector("svg.ic") : null);
      const use = icon && icon.querySelector("use");
      const name = use && use.getAttribute("href");
      if (name) names[name] = (names[name] || 0) + 1;
    }
    for (const [name, n] of Object.entries(names)) if (n >= 3) wallpaper.push(name.replace("#i-", "") + "×" + n);
  }

  // ---- type: the sizes that are not on the type scale of Material 3 (the monospace of code is read as a size of its own)
  const scale = new Set([11, 12, 14, 16, 22, 24, 28, 32, 36, 45, 57]);
  const offScale = {};
  for (const t of texts) {
    const size = Math.round(t.size);
    if (!t.mono && !scale.has(size)) offScale[size] = (offScale[size] || 0) + 1;
  }

  // ---- targets: what a finger or a pointer has to hit
  const targets = [
    ...document.querySelectorAll("button,a[href],input:not([type=hidden]),select,textarea,[role=button],[role=tab],[role=switch],[role=checkbox]"),
  ].filter(visible);
  const smallest = targets.map((e) => {
    const box = e.getBoundingClientRect();
    return Math.min(box.width, box.height);
  });

  return JSON.stringify({
    dark,
    width,
    words: texts.reduce((all, t) => all + t.words, 0),
    prose: { blocks: prose.length, words: prose.reduce((all, t) => all + t.words, 0), longest: prose.slice(0, 3).map((t) => t.words + ": " + t.text.slice(0, 60)) },
    helpers: helpers.length,
    contrast: { measured, failing: failures.length, worst: failures.slice(0, 4) },
    rowsWithTwoActions: twoActions,
    wallpaper,
    offScale,
    targets: { total: targets.length, under24: smallest.filter((s) => s < 24).length, under40: smallest.filter((s) => s < 40).length },
    overflow: page.scrollWidth > page.clientWidth,
  });
})()

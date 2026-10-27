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

// Measures the JavaScript heap of the paired web interface (/app) with the
// Chrome DevTools Protocol and writes memory.txt lines for the memory
// report: `webheap (app)` after the SDK list settles, and
// `webheap (app-settled)` after idling on the open page, which catches leaks
// in the live event stream. No dependencies beyond Node 22 and a Chrome that
// speaks CDP; the browser binary is auto-detected (CHROME_BIN wins).
//
// Usage: node check.cjs --server http://127.0.0.1:PORT --pairing <pairing-url> --output memory.txt [--settle SECONDS] [--screenshot shot.png] [--chrome PATH] [--timeout SECONDS]
//
// The caller starts the server and passes its pairing link; this script only
// drives one throwaway browser profile. Desktop and server memory are not
// measured here, only the web client in front of a human.
'use strict';

const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

function arg(name, fallback) {
  const i = process.argv.indexOf('--' + name);
  if (i < 0 || i + 1 >= process.argv.length) return fallback;
  return process.argv[i + 1];
}

function fail(message) {
  console.error('webheap: ' + message);
  process.exit(1);
}

const server = arg('server', '');
const pairing = arg('pairing', '');
const output = arg('output', '');
const screenshot = arg('screenshot', '');
const shotLang = arg('lang', '');
const settleSeconds = Number(arg('settle', '5'));
const timeoutSeconds = Number(arg('timeout', '180'));
if (!server || !pairing || !output) fail('usage: node check.cjs --server URL --pairing URL --output FILE [--settle SECONDS] [--screenshot FILE] [--chrome PATH] [--timeout SECONDS]');
if (!Number.isFinite(settleSeconds) || settleSeconds < 0) fail('bad --settle');
if (!Number.isFinite(timeoutSeconds) || timeoutSeconds <= 0) fail('bad --timeout');

const deadline = Date.now() + timeoutSeconds * 1000;
function checkTimeout(step) {
  if (Date.now() > deadline) fail('timed out ' + step);
}

function findChrome() {
  const override = arg('chrome', '') || process.env.CHROME_BIN || '';
  if (override) return override;
  for (const candidate of ['google-chrome', 'chromium', 'chromium-browser']) {
    try {
      const found = spawnSync('which', [candidate], { encoding: 'utf8' });
      if (found.status === 0 && found.stdout.trim()) return found.stdout.trim().split('\n')[0];
    } catch (_) { /* try the next candidate */ }
  }
  const macOS = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
  try {
    fs.accessSync(macOS, fs.constants.X_OK);
    return macOS;
  } catch (_) { /* not found */ }
  return '';
}

// Minimal CDP client over the global WebSocket: one message id series, one
// pending map; unsolicited events are ignored. Enough for Page/Runtime.
function connect(url) {
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(url);
    let nextId = 1;
    const pending = new Map();
    socket.addEventListener('open', () => resolve({
      close() { socket.close(); },
      send(method, params) {
        return new Promise((resolveOne, rejectOne) => {
          const id = nextId++;
          pending.set(id, { resolveOne, rejectOne });
          socket.send(JSON.stringify({ id, method, params: params || {} }));
        });
      },
    }));
    socket.addEventListener('error', () => reject(new Error('websocket error')));
    socket.addEventListener('close', () => {
      for (const { rejectOne } of pending.values()) rejectOne(new Error('websocket closed'));
      pending.clear();
    });
    socket.addEventListener('message', (message) => {
      let data;
      try { data = JSON.parse(String(message.data)); } catch (_) { return; }
      if (data.id != null && pending.has(data.id)) {
        const { resolveOne, rejectOne } = pending.get(data.id);
        pending.delete(data.id);
        if (data.error) rejectOne(new Error(data.error.message || JSON.stringify(data.error)));
        else resolveOne(data.result || {});
      }
    });
  });
}

async function httpJson(url) {
  const response = await fetch(url);
  if (!response.ok) throw new Error('GET ' + url + ': ' + response.status);
  return response.json();
}

function median(values) {
  const sorted = values.slice().sort((a, b) => a - b);
  return sorted[Math.floor(sorted.length / 2)];
}

(async () => {
  const chrome = findChrome();
  if (!chrome) fail('no Chrome found (set --chrome or CHROME_BIN)');
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'vfox-webheap-'));
  const child = spawn(chrome, [
    '--headless=new',
    '--remote-debugging-port=0',
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-background-networking',
    '--disable-sync',
    '--disable-default-apps',
    '--mute-audio',
    '--hide-scrollbars',
    '--no-proxy-server',
    '--window-size=1280,800',
    '--user-data-dir=' + profile,
    '--no-sandbox',
    'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] });
  let endpoint = '';
  try {
    endpoint = await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('no DevTools endpoint')), 60000);
      let text = '';
      child.stderr.on('data', (chunk) => {
        text += String(chunk);
        const match = text.match(/DevTools listening on (ws:\/\/\S+)/);
        if (match) {
          clearTimeout(timer);
          resolve(match[1]);
        }
      });
      child.on('error', reject);
      child.on('exit', (code) => reject(new Error('chrome exited with ' + code + ': ' + text.slice(-500))));
    });
  } catch (error) {
    try { child.kill('SIGKILL'); } catch (_) {}
    fail(error.message);
  }
  // The debugger URL names the browser; list the targets from its origin.
  const origin = endpoint.replace(/^ws:/, 'http:').replace(/\/devtools\/.*$/, '');
  let tab;
  try {
    const targets = await httpJson(origin + '/json/list');
    const page = targets.find((t) => t.type === 'page' && t.url !== 'about:blank') || targets.find((t) => t.type === 'page');
    if (!page || !page.webSocketDebuggerUrl) throw new Error('no page target');
    // Talk to the tab directly: events then belong to this page.
    tab = await connect(page.webSocketDebuggerUrl);
  } catch (error) {
    try { child.kill('SIGKILL'); } catch (_) {}
    fail(error.message);
  }
  const finish = (code) => {
    try { tab.close(); } catch (_) {}
    try { child.kill('SIGKILL'); } catch (_) {}
    try { fs.rmSync(profile, { recursive: true, force: true }); } catch (_) {}
    process.exit(code);
  };
  try {
    const use = (method, params) => tab.send(method, params);
    const tabEvaluate = async (expression) => {
      const result = await use('Runtime.evaluate', { expression, returnByValue: true });
      if (result.exceptionDetails) throw new Error('evaluate failed');
      return result.result ? result.result.value : undefined;
    };
    const tabWaitFor = async (expression, label) => {
      for (;;) {
        checkTimeout('waiting for ' + label);
        try {
          if (await tabEvaluate(expression)) return;
        } catch (_) {
          checkTimeout('waiting for ' + label);
        }
        await new Promise((resolve) => setTimeout(resolve, 500));
      }
    };
    await use('Page.enable', {});
    await use('Runtime.enable', {});
    const navigate = async (url, label) => {
      await use('Page.navigate', { url });
      await tabWaitFor('document.readyState === "complete"', label);
    };
    // Pair first, exactly like a browser: the pairing page sets the session
    // cookie, and /app opens paired.
    await navigate(pairing, 'pairing');
    await new Promise((resolve) => setTimeout(resolve, 1000));
    await navigate(server.replace(/\/$/, '') + '/app', 'the app');
    if (shotLang === 'en' || shotLang === 'zh') {
      // Pin the language before settling, so screenshots are deterministic.
      await tabEvaluate(`localStorage.setItem("vfox-plus-lang", ${JSON.stringify(shotLang)}); location.reload();`);
      await tabWaitFor('document.readyState === "complete"', 'the app reload');
    }
    await tabWaitFor('(() => { const list = document.querySelector("#sdkList"); return list && list.children.length > 0 && !list.querySelector(".loader"); })()', 'the SDK list');
    await new Promise((resolve) => setTimeout(resolve, settleSeconds * 1000));
    if (screenshot) {
      // A screenshot of the settled page, for the documentation; the memory
      // measurement does not need it.
      const shot = await use('Page.captureScreenshot', { format: 'png' });
      fs.writeFileSync(screenshot, Buffer.from(shot.data, 'base64'));
    }
    const heap = [];
    for (let i = 0; i < 3; i++) heap.push((await use('Runtime.getHeapUsage', {})).usedSize);
    const domNodes = await tabEvaluate('document.querySelectorAll("*").length');
    // Idle on the open page with live server-sent events: catches leaks.
    await new Promise((resolve) => setTimeout(resolve, 20000));
    const settled = [];
    for (let i = 0; i < 3; i++) settled.push((await use('Runtime.getHeapUsage', {})).usedSize);
    let version = 'unknown';
    try {
      const info = await use('Browser.getVersion', {});
      version = info.product || version;
    } catch (_) { /* best effort */ }
    const lines = `webheap (app): ${median(heap)} bytes\nwebheap (app-settled): ${median(settled)} bytes\n`;
    fs.writeFileSync(output, lines);
    console.log(JSON.stringify({ chrome: version, domNodes, heapUsedBytes: median(heap), heapSettledBytes: median(settled) }));
    finish(0);
  } catch (error) {
    console.error('webheap: ' + (error && error.message ? error.message : error));
    finish(1);
  }
})().catch((error) => {
  console.error('webheap: ' + (error && error.message ? error.message : error));
  process.exit(1);
});

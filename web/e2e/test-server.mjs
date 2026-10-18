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

// A local HTTP server for the offline test plugin of shared/test-plugin,
// following the protocol of shared/test-plugin/README.md: a plugin registry,
// the plugin and two SDK versions. SDK archives are sent slowly, in chunks, so
// that vfox reports download progress. GET /control/hold makes downloads stop
// after their first chunk until GET /control/release.
//
//   node test-server.mjs <work directory>
//
// Prints the base URL on the first line of its standard output and serves
// until it is killed.

import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, utimesSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { fileURLToPath } from 'node:url';

const work = process.argv[2];
if (work === undefined) {
  console.error('Usage: node test-server.mjs <work directory>');
  process.exit(2);
}
const plugin = readFileSync(fileURLToPath(new URL('../../shared/test-plugin/testsdk.lua', import.meta.url)), 'utf8');

// Deterministic, incompressible padding makes the downloads long enough to
// report progress.
function padding(seed) {
  const bytes = Buffer.alloc(512 * 1024);
  let state = seed;
  for (let i = 0; i < bytes.length; i++) {
    state = (Math.imul(state, 1103515245) + 12345) >>> 0;
    bytes[i] = state >>> 24;
  }
  return bytes;
}

const archives = new Map();
for (const [index, version] of ['1.0.0', '2.0.0'].entries()) {
  const root = `${work}/build/testsdk-${version}`;
  mkdirSync(`${root}/bin`, { recursive: true });
  writeFileSync(`${root}/bin/testsdk`, `#!/bin/sh\necho "testsdk ${version}"\n`, { mode: 0o755 });
  writeFileSync(`${root}/payload.bin`, padding(42 + index));
  utimesSync(`${root}/bin/testsdk`, new Date('2026-01-01'), new Date('2026-01-01'));
  const archive = `${work}/testsdk-${version}.tar.gz`;
  execFileSync('tar', ['-czf', archive, '-C', `${work}/build`, `testsdk-${version}`]);
  archives.set(version, readFileSync(archive));
}

let held = null;
let url = '';

const server = createServer((request, response) => {
  const path = new URL(request.url ?? '/', 'http://localhost').pathname;
  const json = (body) => {
    response.writeHead(200, { 'content-type': 'application/json' });
    response.end(JSON.stringify(body));
  };
  const sdk = /^\/sdk\/testsdk-([0-9.]+)\.tar\.gz$/.exec(path);
  if (path === '/registry/index.json') {
    json([
      {
        name: 'testsdk',
        desc: 'An offline SDK for the vfox-plus end-to-end tests.',
        homepage: 'https://github.com/version-fox/vfox-plus',
      },
    ]);
  } else if (path === '/registry/testsdk.json') {
    json({
      name: 'testsdk',
      version: '0.1.0',
      license: 'Apache-2.0',
      author: 'vfox-plus',
      downloadUrl: `${url}/plugin/testsdk.lua`,
      minRuntimeVersion: '1.0.0',
    });
  } else if (path === '/plugin/testsdk.lua') {
    response.writeHead(200, { 'content-type': 'text/plain' });
    response.end(plugin);
  } else if (sdk !== null && archives.has(sdk[1])) {
    const bytes = archives.get(sdk[1]);
    response.writeHead(200, { 'content-type': 'application/gzip', 'content-length': bytes.length });
    const chunk = 64 * 1024;
    let offset = 0;
    const next = async () => {
      while (offset < bytes.length && !response.destroyed) {
        response.write(bytes.subarray(offset, offset + chunk));
        offset += chunk;
        await (held?.promise ?? new Promise((resolve) => setTimeout(resolve, 60)));
      }
      response.end();
    };
    void next();
  } else if (path === '/control/hold') {
    let release;
    const promise = new Promise((resolve) => {
      release = resolve;
    });
    held = { promise, release };
    response.end('held\n');
  } else if (path === '/control/release') {
    held?.release();
    held = null;
    response.end('released\n');
  } else {
    response.writeHead(404);
    response.end();
  }
});

server.listen(0, '127.0.0.1', () => {
  const address = server.address();
  url = `http://127.0.0.1:${address.port}`;
  console.log(url);
});

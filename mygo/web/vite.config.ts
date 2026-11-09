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

import { defineConfig } from "vite";
import { viteSingleFile } from "vite-plugin-singlefile";

// The whole interface builds into one self-contained dist/index.html, which
// npm run build copies over mygo/pair/webapp.html: the Go server embeds that
// file and serves it as /app, with no other requests. Keep it that way: no
// external scripts, stylesheets, fonts or images.
// CSS stays unminified on purpose: the Go design tests read the served
// file and pin the exact custom properties, spring curves and one-
// declaration-per-line shape. (Only the JavaScript is minified.)
export default defineConfig({
  plugins: [viteSingleFile()],
  build: {
    cssMinify: false,
    outDir: "dist",
    assetsInlineLimit: 100000000,
    chunkSizeWarningLimit: 100000000,
  },
  server: {
    // npm run dev proxies the API to a running application, whose port the
    // end-to-end build prints in VFOX_PLUS_E2E_LINK, e.g. VFOX_WEB_PROXY=
    // http://127.0.0.1:39275 npm run dev.
    proxy: {
      "/api": {
        target: process.env.VFOX_WEB_PROXY ?? "http://127.0.0.1:8369",
        changeOrigin: false,
      },
    },
  },
});

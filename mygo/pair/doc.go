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

// Package pair connects a browser to a vfox-plus instance.
//
// `vfox-plus pair` starts a loopback HTTP server and prints a one-time link
// like `http://127.0.0.1:PORT/auth/connect/TOKEN`, in the style of
// `opencode pair`. Opening the link pairs the browser: the token works once
// and expires after a few minutes, and the server issues a session cookie
// for the paired browser.
//
// A paired browser gets the full web interface at `/app` (a single embedded
// page, no external requests): installed SDKs, the plugin registry, tasks
// with live progress and logs over server-sent events, and the settings.
// `vfox-plus serve` keeps the same server up for containers and remote
// instances, with expiring sessions and mintable links.
package pair

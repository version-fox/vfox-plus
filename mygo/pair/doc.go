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

// Package pair is the web interface of vfox-plus: the servers that connect a
// browser, or the Android app, to the application, and the host that runs them as
// the settings say.
//
// The application is a desktop application and has no command line: the user
// turns the web interface on in its settings, and the Host starts a server on the
// loopback interface. Its link to the browsers of the computer is a one-time URL
// like `http://127.0.0.1:PORT/auth/connect/TOKEN`, which the settings open in the
// browser: the token works once and expires after a few minutes, and the server
// issues a session cookie for the paired browser and takes it to the interface.
// Other devices on the network, a phone for one, connect to a second server, that
// the user allows in the settings, over HTTPS with a certificate that the application
// makes for itself or the user brings; the pairing code of the settings is the
// link to it. Both servers serve the same application state.
//
// A paired browser gets the full web interface at `/app` (a single embedded
// page, no external requests): installed SDKs, the plugin registry, tasks with
// live progress and logs over server-sent events, and the settings, among them how
// other devices connect and the lightweight mode, in which the application stays
// running without its window.
package pair

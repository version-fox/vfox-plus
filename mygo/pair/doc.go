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

// Package pair is the server that other devices connect to: the Docker gateway,
// the Android app and a browser that the gateway pairs, and the host that runs that
// server as the settings say.
//
// The application is a desktop application and has no command line, and it shows no
// web interface on the computer that it runs on. The user allows other devices in
// its settings, and then the Host starts a server on the network, over HTTPS with a
// certificate that the application makes for itself or the user brings. The pairing
// code of the settings is a one-time URL like `https://ADDRESS:PORT/auth/connect/TOKEN`:
// the token works once and expires after a few minutes, and the server issues a
// session cookie for the paired device and takes it to the interface. The Docker
// gateway (package gateway) and the Android app pair with that server.
//
// A paired device gets the interface at `/app` (a single embedded page, no external
// requests): installed SDKs, the plugin registry, tasks with live progress and logs
// over server-sent events, and the settings, among them how other devices connect
// and the lightweight mode, in which the application stays running without its
// window.
package pair

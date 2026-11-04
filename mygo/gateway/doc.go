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

// Package gateway is the web interface of one vfox+ instance, served from another place: the
// program of the Docker image of the same name. It is a client of an instance, as the Android
// app is, and not an instance: it has no vfox, no window and no state of vfox of its own.
//
// An instance is the application, run by a person with its web interface switched on (see
// package pair, and docs/en/dev/remote-protocol.md). The gateway is paired with it the way
// the Android app is: it is given a pairing link of the instance, checks the certificate of
// the instance against the fingerprint that the person compared (or against the roots of the
// system when the instance has a certificate of an authority), and holds the session that the
// instance gives it. It then serves the interface of the instance to the browsers that are
// paired with the gateway: the page /app and the routes of /api are the instance's, forwarded
// with the session of the gateway, and the page is the instance's own, so that it always
// matches the routes it calls.
//
// A browser is paired with the gateway as it is with an instance, with a one-time link that
// the gateway prints when it starts, and has a session of its own. What the gateway serves
// itself is the pairing of the browsers (/auth/connect/), the page that says to which
// instance it is paired and pairs it with another (/ and /gateway/), and /api/token for the
// second browser; every other route is the instance's, and nothing the instance sets (its
// cookie, its headers about the origin) crosses to the browser or the other way.
//
// The session at the instance lasts a day, and ends with the instance. The gateway renews it
// while it runs, with the route that the instance gives a paired device to mint a link for a
// second one, so that it stays paired for as long as the instance runs; when the instance
// has ended, or was not reachable for a day, the person pairs it again on its page.
//
// Like the application, the program has no flags and no switches: what it remembers is in
// its directory (the certificate that it serves with, the instance, its session), and the
// instance it shows is chosen on its page.
package gateway

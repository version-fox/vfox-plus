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

//go:build e2e

package main

// The end-to-end tests drive the application from outside, through the automation
// plugin of mygo: it listens on the Unix socket that MYGO_AUTOMATION names, and
// answers the requests of vfxtool (shared/tooling/internal/automation) to list the
// elements of the window, act on them and capture it. Only builds with the tag e2e
// have it, and builds with mygo_noinspector, which the releases are, leave it out
// even then.
import _ "github.com/egoist/mygo/plugins/automation"

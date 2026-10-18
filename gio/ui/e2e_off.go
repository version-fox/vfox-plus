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

//go:build !e2e

package ui

// e2eHooks is what the end-to-end tests add to the application; the build
// without the tag `e2e` has none.
type e2eHooks struct{} //nolint:unused // See App.

func (a *App) startE2E() {}

func (a *App) afterFrame() {}

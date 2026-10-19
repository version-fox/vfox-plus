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

//go:build !linux

package ui

// ownFontCache is for Linux, where mygo hands the fonts that an application
// registers to fontconfig as files; there is nothing to do elsewhere.
func ownFontCache() bool { return false }

// prepareProcess does what the application needs before mygo starts: nothing
// outside Linux.
func prepareProcess() {}

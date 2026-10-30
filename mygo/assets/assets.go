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

// Package assets embeds what the application needs at run time and that
// `vfxtool prepare` collects: the icons, the translations and the licenses.
// Run it before building.
package assets

import "embed"

// Files holds the collected assets:
//
//   - icons.json: the symbols of the icons, by meaning (shared/icons.json);
//   - icons/SYMBOL.svg and icons/SYMBOL-fill.svg: the icons, outlined and
//     filled, cut out of the npm package that shared/icons.json pins;
//   - i18n/app_<language>.arb: the translations of shared/i18n;
//   - licenses.txt: the licenses of everything the application is built from.
//
//go:embed icons.json icons/*.svg i18n/*.arb licenses.txt
var Files embed.FS

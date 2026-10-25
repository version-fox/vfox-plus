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
// `vfxtool prepare` generates: the icon fonts, the code points of the icons,
// the translations and the licenses. Run it before building.
package assets

import "embed"

// Files holds the generated assets:
//
//   - fonts/MaterialSymbolsRounded.ttf: the icon font, outlined (fill 0);
//   - fonts/MaterialSymbolsRoundedFilled.ttf: the icon font, filled (fill 1);
//   - icons.json: the code points of the icons, by meaning;
//   - i18n/app_<language>.arb: the translations of shared/i18n;
//   - licenses.txt: the licenses of everything the application is built from.
//
//go:embed fonts/MaterialSymbolsRounded.ttf fonts/MaterialSymbolsRoundedFilled.ttf icons.json i18n/*.arb licenses.txt
var Files embed.FS

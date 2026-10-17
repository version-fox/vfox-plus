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

/// Licenses of bundled assets that do not come from a Dart package.
library;

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// The license of the icon font in `assets/fonts`.
const String iconFontLicenseAsset = 'assets/fonts/MaterialSymbolsRounded-LICENSE.txt';

/// Adds the licenses of bundled assets to the [LicenseRegistry], which the
/// license page shows.
///
/// Flutter registers the licenses of the Dart packages by itself. The subset
/// of the Material Symbols font is generated from a development dependency, so
/// its license needs an entry of its own.
void registerAssetLicenses({AssetBundle? bundle}) {
  LicenseRegistry.addLicense(() async* {
    yield LicenseEntryWithLineBreaks(const <String>[
      'Material Symbols',
    ], await (bundle ?? rootBundle).loadString(iconFontLicenseAsset));
  });
}

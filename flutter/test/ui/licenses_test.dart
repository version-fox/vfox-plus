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

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/ui/licenses.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(LicenseRegistry.reset);
  tearDown(LicenseRegistry.reset);

  test('the license page lists the bundled icon font', () async {
    registerAssetLicenses();
    final LicenseEntry font = (await LicenseRegistry.licenses.toList()).single;
    expect(font.packages, <String>['Material Symbols']);
    final String text = font.paragraphs.map((LicenseParagraph paragraph) => paragraph.text).join('\n');
    expect(text, contains('Copyright Google LLC'));
    expect(text, contains('Apache License'));
  });
}

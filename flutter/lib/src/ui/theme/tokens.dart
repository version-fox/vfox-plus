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

/// Design tokens of the Material 3 Expressive system used by the app.
///
/// The values are documented in `docs/design-system.md`, which every
/// implementation of the GUI follows.
library;

import 'package:flutter/widgets.dart';

/// The Material 3 Expressive corner radius scale, in logical pixels.
///
/// See https://m3.material.io/styles/shape/corner-radius-scale.
abstract final class Corners {
  /// 0 dp.
  static const double none = 0;

  /// 4 dp.
  static const double extraSmall = 4;

  /// 8 dp.
  static const double small = 8;

  /// 12 dp.
  static const double medium = 12;

  /// 16 dp.
  static const double large = 16;

  /// 20 dp.
  static const double largeIncreased = 20;

  /// 28 dp.
  static const double extraLarge = 28;

  /// 32 dp.
  static const double extraLargeIncreased = 32;

  /// 48 dp.
  static const double extraExtraLarge = 48;

  /// Fully rounded: half of the shorter side.
  static const double full = 9999;
}

/// Spacing between elements, on a 4 dp grid.
abstract final class Gaps {
  /// 2 dp: between the segments of connected groups.
  static const double hairline = 2;

  /// 4 dp.
  static const double xs = 4;

  /// 8 dp.
  static const double s = 8;

  /// 12 dp.
  static const double m = 12;

  /// 16 dp.
  static const double l = 16;

  /// 24 dp.
  static const double xl = 24;

  /// 32 dp.
  static const double xxl = 32;
}

/// Material 3 window size classes, by window width.
///
/// See https://m3.material.io/foundations/layout/applying-layout/window-size-classes.
enum WindowSizeClass {
  /// Narrower than 600 dp.
  compact,

  /// 600 dp to 839 dp.
  medium,

  /// 840 dp to 1199 dp.
  expanded,

  /// 1200 dp to 1599 dp.
  large,

  /// 1600 dp and wider.
  extraLarge;

  /// The size class of a window [width] logical pixels wide.
  static WindowSizeClass fromWidth(double width) {
    if (width < 600) {
      return compact;
    }
    if (width < 840) {
      return medium;
    }
    if (width < 1200) {
      return expanded;
    }
    if (width < 1600) {
      return large;
    }
    return extraLarge;
  }

  /// The size class of the window that contains [context].
  static WindowSizeClass of(BuildContext context) => fromWidth(MediaQuery.sizeOf(context).width);

  /// Whether this class is at least as wide as [other].
  bool operator >=(WindowSizeClass other) => index >= other.index;
}

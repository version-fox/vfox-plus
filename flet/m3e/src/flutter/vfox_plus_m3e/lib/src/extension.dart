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

import 'package:flet/flet.dart';
import 'package:flutter/widgets.dart';

import 'constrained_box.dart';
import 'expressive_dialog.dart';
import 'loading_indicator.dart';
import 'navigation_rail.dart';
import 'shapes.dart';
import 'spring_surface.dart';
import 'transitions.dart';
import 'wavy_progress_indicator.dart';

/// Registers the controls and the icon set of this package with Flet.
class Extension extends FletExtension {
  /// The icon set ID of Material Symbols Rounded in Flet's icon codes, which
  /// are `set << 16 | index`. Flet uses 1 for its Material icons and 2 for
  /// the Cupertino icons; the index of a symbol is its code point.
  static const int symbolsSetId = 0x56;

  @override
  Widget? createWidget(Key? key, Control control) {
    return switch (control.type) {
      'ColorSwatch' => ColorSwatchControl(key: key, control: control),
      'ConstrainedBox' => ConstrainedBoxControl(key: key, control: control),
      'ExpressiveDialog' => ExpressiveDialogControl(key: key, control: control),
      'ExpressiveNavigationRail' => ExpressiveNavigationRailControl(key: key, control: control),
      'LoadingIndicator' => LoadingIndicatorControl(key: key, control: control),
      'MaterialShape' => MaterialShapeControl(key: key, control: control),
      'ShapeAvatar' => ShapeAvatarControl(key: key, control: control),
      'SpringSurface' => SpringSurfaceControl(key: key, control: control),
      'TransitionSwitcher' => TransitionSwitcherControl(key: key, control: control),
      'WavyProgressIndicator' => WavyProgressIndicatorControl(key: key, control: control),
      _ => null,
    };
  }

  @override
  IconData? createIconData(int iconCode) {
    if ((iconCode >> 16) & 0xFF != symbolsSetId) {
      return null;
    }
    // The code points come from Python at run time, so the icon font cannot
    // be tree-shaken; it is a subset of the icons of the application already.
    // ignore: non_const_argument_for_const_parameter
    return IconData(iconCode & 0xFFFF, fontFamily: 'MaterialSymbolsRounded', fontPackage: 'vfox_plus_m3e');
  }
}

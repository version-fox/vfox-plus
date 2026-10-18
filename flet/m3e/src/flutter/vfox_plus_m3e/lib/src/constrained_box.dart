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

/// The `ConstrainedBox` control: limits the size of its content, which Flet
/// controls cannot do, and aligns it in the remaining space — the
/// `ContentWidth` of `flutter/lib/src/ui/widgets/page_parts.dart`.
class ConstrainedBoxControl extends StatelessWidget {
  const ConstrainedBoxControl({super.key, required this.control});

  final Control control;

  @override
  Widget build(BuildContext context) {
    Widget child = ConstrainedBox(
      constraints: BoxConstraints(
        minWidth: control.getDouble('min_width', 0)!,
        maxWidth: control.getDouble('max_width', double.infinity)!,
        minHeight: control.getDouble('min_height', 0)!,
        maxHeight: control.getDouble('max_height', double.infinity)!,
      ),
      child: control.buildWidget('content'),
    );
    final Alignment? alignment = control.getAlignment('alignment');
    if (alignment != null) {
      child = Align(alignment: alignment, heightFactor: 1, child: child);
    }
    return LayoutControl(control: control, child: child);
  }
}

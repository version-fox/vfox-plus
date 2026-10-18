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

import 'package:animations/animations.dart';
import 'package:flet/flet.dart';
import 'package:flutter/material.dart';

import 'motion.dart';

/// The `TransitionSwitcher` control: replaces its content with a Material
/// transition whenever the content key changes.
///
/// `fade_through` switches between unrelated content, such as the pages of
/// the shell or two SDKs; `shared_axis_x` moves along a horizontal axis,
/// such as from a list to the detail of an item (`reverse` goes back). The
/// transitions last as long as the slow effects spring needs to settle, as
/// in `flutter/lib/src/ui/shell/app_shell.dart`.
class TransitionSwitcherControl extends StatelessWidget {
  const TransitionSwitcherControl({super.key, required this.control});

  final Control control;

  @override
  Widget build(BuildContext context) {
    final String transition = control.getString('transition', 'fade_through')!;
    final Widget content = control.buildWidget('content') ?? const SizedBox.shrink();
    return LayoutControl(
      control: control,
      child: PageTransitionSwitcher(
        duration: reduceMotion(context) ? Duration.zero : Springs.slowEffects.settleDuration,
        reverse: control.getBool('reverse', false)!,
        transitionBuilder: (Widget child, Animation<double> primary, Animation<double> secondary) =>
            switch (transition) {
              'shared_axis_x' => SharedAxisTransition(
                animation: primary,
                secondaryAnimation: secondary,
                transitionType: SharedAxisTransitionType.horizontal,
                fillColor: Colors.transparent,
                child: child,
              ),
              _ => FadeThroughTransition(
                animation: primary,
                secondaryAnimation: secondary,
                fillColor: Colors.transparent,
                child: child,
              ),
            },
        child: KeyedSubtree(key: ValueKey<String>(control.getString('content_key', '')!), child: content),
      ),
    );
  }
}

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
import 'package:flutter/material.dart';

import 'motion.dart';

/// The `ExpressiveDialog` control: an alert dialog that scales in on the
/// default spatial spring and fades in on the default effects spring, as
/// Material 3 Expressive dialogs do (`showExpressiveDialog` of
/// `flutter/lib/src/ui/widgets/dialogs.dart`).
///
/// It is opened and closed like Flet's `AlertDialog`, whose life cycle it
/// follows: `page.show_dialog()` sets `open`, closing the route sets it back
/// and triggers `dismiss`.
class ExpressiveDialogControl extends StatefulWidget {
  const ExpressiveDialogControl({super.key, required this.control});

  final Control control;

  @override
  State<ExpressiveDialogControl> createState() => _ExpressiveDialogControlState();
}

class _ExpressiveDialogControlState extends State<ExpressiveDialogControl> {
  // The route this widget pushed, so that closing pops this dialog and not
  // whatever route is on top.
  ModalRoute<dynamic>? _route;

  Control get control => widget.control;

  Widget _dialog(BuildContext context) {
    // Rebuilds the open dialog when the control changes, such as an error
    // text below a field.
    return ListenableBuilder(
      listenable: control,
      builder: (BuildContext context, Widget? _) {
        final ColorScheme colors = Theme.of(context).colorScheme;
        final IconData? icon = control.getIconData('icon');
        return SafeArea(
          child: BaseControl(
            control: control,
            child: AlertDialog(
              icon: icon == null ? null : Icon(icon, color: control.getColor('icon_color', context)),
              title: control.buildTextOrWidget('title'),
              content: control.buildWidget('content'),
              actions: control.buildWidgets('actions'),
              scrollable: control.getBool('scrollable', false)!,
              semanticLabel: control.getString('semantics_label'),
              backgroundColor: colors.surfaceContainerHigh,
            ),
          ),
        );
      },
    );
  }

  void _show() {
    final bool reduced = reduceMotion(context);
    final ColorScheme colors = Theme.of(context).colorScheme;
    showGeneralDialog<void>(
      context: context,
      useRootNavigator: false,
      barrierDismissible: !control.getBool('modal', false)!,
      barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
      barrierColor: colors.scrim.withValues(alpha: 0.32),
      transitionDuration: reduced ? Duration.zero : Springs.defaultSpatial.settleDuration,
      pageBuilder: (BuildContext context, Animation<double> animation, Animation<double> secondary) {
        _route ??= ModalRoute.of(context);
        return _dialog(context);
      },
      transitionBuilder:
          (BuildContext context, Animation<double> animation, Animation<double> secondary, Widget child) {
            final scale = CurvedAnimation(
              parent: animation,
              curve: Springs.defaultSpatial.curve,
              reverseCurve: Curves.easeIn,
            );
            final opacity = CurvedAnimation(parent: animation, curve: Springs.defaultEffects.curve);
            return FadeTransition(
              opacity: opacity,
              child: ScaleTransition(scale: Tween<double>(begin: 0.8, end: 1).animate(scale), child: child),
            );
          },
    ).then((_) {
      final ModalRoute<dynamic>? route = _route;
      _route = null;
      // The future completes when the route is popped, before its exit
      // transition ends; dismiss once the transition has ended.
      (route?.completed ?? Future<void>.value()).then((_) {
        control.updateProperties(<String, dynamic>{'_open': false}, python: false);
        control.updateProperties(<String, dynamic>{'open': false});
        control.triggerEvent('dismiss');
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    final bool open = control.getBool('open', false)!;
    final bool wasOpen = control.getBool('_open', false)!;
    if (open && !wasOpen) {
      control.updateProperties(<String, dynamic>{'_open': true}, python: false);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _show();
        }
      });
    } else if (!open && wasOpen) {
      control.updateProperties(<String, dynamic>{'_open': false}, python: false);
      final ModalRoute<dynamic>? route = _route;
      if (route != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) => closeModalRoute(route));
      }
    }
    return const SizedBox.shrink();
  }
}

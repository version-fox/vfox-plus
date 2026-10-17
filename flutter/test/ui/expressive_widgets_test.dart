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

import 'package:flutter_test/flutter_test.dart';
import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/ui/theme/app_theme.dart';
import 'package:vfox_plus/src/ui/theme/motion.dart';
import 'package:vfox_plus/src/ui/theme/tokens.dart';
import 'package:vfox_plus/src/ui/widgets/button_group.dart';
import 'package:vfox_plus/src/ui/widgets/list_group.dart';
import 'package:vfox_plus/src/ui/widgets/loading_indicator.dart';
import 'package:vfox_plus/src/ui/widgets/navigation_rail.dart';
import 'package:vfox_plus/src/ui/widgets/polygon_border.dart';
import 'package:vfox_plus/src/ui/widgets/split_button.dart';
import 'package:vfox_plus/src/ui/widgets/wavy_progress_indicator.dart';

Widget _app(Widget child) {
  return MaterialApp(
    theme: AppTheme.build(AppTheme.scheme(const Color(0xFFF28C1B), Brightness.light)),
    home: Scaffold(body: Center(child: child)),
  );
}

void main() {
  group('motion', () {
    test('the motion schemes use the Material 3 Expressive spring tokens', () {
      expect(MotionScheme.expressive.fastSpatial, const Spring(dampingRatio: 0.6, stiffness: 800));
      expect(MotionScheme.expressive.defaultSpatial, const Spring(dampingRatio: 0.8, stiffness: 380));
      expect(MotionScheme.expressive.slowSpatial, const Spring(dampingRatio: 0.8, stiffness: 200));
      expect(MotionScheme.standard.defaultSpatial, const Spring(dampingRatio: 0.9, stiffness: 700));
      for (final MotionScheme scheme in <MotionScheme>[MotionScheme.standard, MotionScheme.expressive]) {
        expect(scheme.fastEffects, const Spring(dampingRatio: 1, stiffness: 3800));
        expect(scheme.defaultEffects, const Spring(dampingRatio: 1, stiffness: 1600));
        expect(scheme.slowEffects, const Spring(dampingRatio: 1, stiffness: 800));
      }
    });

    test('stiffer springs settle sooner', () {
      final Duration fast = MotionScheme.expressive.fastEffects.settleDuration;
      final Duration slow = MotionScheme.expressive.slowEffects.settleDuration;
      expect(fast, lessThan(slow));
      expect(slow, lessThan(const Duration(seconds: 1)));
    });

    test('spring curves start at 0, end near 1 and spatial ones overshoot', () {
      final Curve curve = MotionScheme.expressive.fastSpatial.curve;
      expect(curve.transform(0), 0);
      expect(curve.transform(1), closeTo(1, 0.01));
      final double peak = List<double>.generate(
        100,
        (int i) => curve.transform(i / 100),
      ).reduce((a, b) => a > b ? a : b);
      expect(peak, greaterThan(1));
      final Curve effects = MotionScheme.expressive.defaultEffects.curve;
      for (var i = 0; i <= 100; i++) {
        expect(effects.transform(i / 100), lessThanOrEqualTo(1.0001));
      }
    });

    testWidgets('SpringBuilder animates towards a new value and settles', (WidgetTester tester) async {
      double target = 0;
      late StateSetter setTarget;
      final values = <double>[];
      await tester.pumpWidget(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter setState) {
            setTarget = setState;
            return SpringBuilder(
              value: target,
              spring: MotionScheme.expressive.fastSpatial,
              builder: (BuildContext context, double value, Widget? _) {
                values.add(value);
                return const SizedBox();
              },
            );
          },
        ),
      );
      setTarget(() => target = 1);
      await tester.pumpAndSettle();
      expect(values.last, closeTo(1, 0.01));
      expect(values.any((double v) => v > 0 && v < 1), isTrue);
    });
  });

  group('tokens', () {
    test('window size classes', () {
      expect(WindowSizeClass.fromWidth(599), WindowSizeClass.compact);
      expect(WindowSizeClass.fromWidth(600), WindowSizeClass.medium);
      expect(WindowSizeClass.fromWidth(840), WindowSizeClass.expanded);
      expect(WindowSizeClass.fromWidth(1200), WindowSizeClass.large);
      expect(WindowSizeClass.fromWidth(1600), WindowSizeClass.extraLarge);
      expect(WindowSizeClass.large >= WindowSizeClass.expanded, isTrue);
    });

    test('the theme opts icon buttons into Material 3 Expressive', () {
      final ThemeData theme = AppTheme.build(AppTheme.scheme(Colors.blue, Brightness.dark));
      expect(theme.iconButtonTheme.variant, StyleVariant.material3Expressive);
      expect(theme.extension<MotionScheme>(), MotionScheme.expressive);
      expect(theme.colorScheme.brightness, Brightness.dark);
    });
  });

  group('shapes', () {
    test('stableHash is deterministic', () {
      expect(stableHash('nodejs'), stableHash('nodejs'));
      expect(stableHash('nodejs'), isNot(stableHash('java')));
      expect(decorativeShapeFor('nodejs'), same(decorativeShapeFor('nodejs')));
    });

    test('PolygonBorder fills the rectangle it is given', () {
      final border = PolygonBorder(polygon: MaterialShapes.square);
      final Rect bounds = border.getOuterPath(const Rect.fromLTWH(10, 20, 100, 50)).getBounds();
      expect(bounds.left, closeTo(10, 1));
      expect(bounds.top, closeTo(20, 1));
      expect(bounds.width, closeTo(100, 1));
      expect(bounds.height, closeTo(50, 1));
    });

    testWidgets('ShapeAvatar shows the initial', (WidgetTester tester) async {
      await tester.pumpWidget(_app(const ShapeAvatar(name: 'nodejs')));
      expect(find.text('N'), findsOneWidget);
    });
  });

  group('ConnectedButtonGroup', () {
    testWidgets('selects a value and exposes the selection to accessibility', (WidgetTester tester) async {
      String selected = 'a';
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (BuildContext context, StateSetter setState) => ConnectedButtonGroup<String>(
              segments: const <ButtonGroupSegment<String>>[
                ButtonGroupSegment<String>(value: 'a', label: 'Alpha'),
                ButtonGroupSegment<String>(value: 'b', label: 'Beta'),
              ],
              selected: selected,
              onSelected: (String value) => setState(() => selected = value),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Beta'));
      await tester.pumpAndSettle();
      expect(selected, 'b');
      expect(
        tester.getSemantics(find.text('Beta')),
        isSemantics(label: 'Beta', isButton: true, isSelected: true, isInMutuallyExclusiveGroup: true),
      );
      expect(tester.getSemantics(find.text('Alpha')), isSemantics(label: 'Alpha', isSelected: false));
    });
  });

  group('SplitButton', () {
    testWidgets('runs the main action and opens the menu', (WidgetTester tester) async {
      var pressed = 0;
      var menu = 0;
      await tester.pumpWidget(
        _app(
          SplitButton(
            label: 'Use',
            menuTooltip: 'More',
            onPressed: () => pressed++,
            menuChildren: <Widget>[MenuItemButton(onPressed: () => menu++, child: const Text('In a project'))],
          ),
        ),
      );
      await tester.tap(find.text('Use'));
      expect(pressed, 1);
      await tester.tap(find.byTooltip('More'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('In a project'));
      await tester.pumpAndSettle();
      expect(menu, 1);
    });
  });

  group('ListGroup', () {
    testWidgets('gives each item its position', (WidgetTester tester) async {
      await tester.pumpWidget(
        _app(
          SizedBox(
            width: 300,
            child: ListGroup(
              children: <Widget>[
                for (final String title in <String>['one', 'two', 'three'])
                  ListGroupItem(title: Text(title), onTap: () {}),
              ],
            ),
          ),
        ),
      );
      RoundedRectangleBorder shapeOf(String title) {
        final Material material = tester.widget<Material>(
          find.ancestor(of: find.text(title), matching: find.byType(Material)).first,
        );
        return material.shape! as RoundedRectangleBorder;
      }

      final BorderRadius first = shapeOf('one').borderRadius as BorderRadius;
      final BorderRadius middle = shapeOf('two').borderRadius as BorderRadius;
      final BorderRadius last = shapeOf('three').borderRadius as BorderRadius;
      expect(first.topLeft.x, greaterThan(first.bottomLeft.x));
      expect(middle.topLeft.x, middle.bottomLeft.x);
      expect(last.bottomLeft.x, greaterThan(last.topLeft.x));
    });

    test('positions', () {
      expect(ListGroupPosition.of(0, 1), ListGroupPosition.single);
      expect(ListGroupPosition.of(0, 3), ListGroupPosition.first);
      expect(ListGroupPosition.of(1, 3), ListGroupPosition.middle);
      expect(ListGroupPosition.of(2, 3), ListGroupPosition.last);
    });
  });

  group('progress', () {
    testWidgets('LoadingIndicator morphs over time without errors', (WidgetTester tester) async {
      await tester.pumpWidget(_app(const LoadingIndicator(contained: true, semanticsLabel: 'Loading')));
      for (var i = 0; i < 20; i++) {
        await tester.pump(const Duration(milliseconds: 160));
      }
      expect(find.bySemanticsLabel('Loading'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('LoadingIndicator stops when animations are disabled', (WidgetTester tester) async {
      await tester.pumpWidget(
        MediaQuery(data: const MediaQueryData(disableAnimations: true), child: _app(const LoadingIndicator())),
      );
      await tester.pumpAndSettle();
      expect(tester.hasRunningAnimations, isFalse);
    });

    testWidgets('WavyProgressIndicator reports determinate progress', (WidgetTester tester) async {
      await tester.pumpWidget(_app(const SizedBox(width: 200, child: WavyProgressIndicator(value: 0.42))));
      await tester.pump(const Duration(seconds: 1));
      expect(tester.getSemantics(find.byType(WavyProgressIndicator)).value, '42%');
      await tester.pumpWidget(_app(const SizedBox(width: 200, child: WavyProgressIndicator())));
      await tester.pump(const Duration(milliseconds: 700));
      expect(tester.takeException(), isNull);
    });
  });

  group('ExpressiveNavigationRail', () {
    testWidgets('selects destinations and toggles its labels', (WidgetTester tester) async {
      var selected = 0;
      var expanded = false;
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (BuildContext context, StateSetter setState) => SizedBox(
              height: 600,
              child: ExpressiveNavigationRail(
                destinations: const <NavigationDestinationData>[
                  NavigationDestinationData(icon: Icons.home, label: 'Home'),
                  NavigationDestinationData(icon: Icons.settings, label: 'Settings', badgeCount: 2),
                ],
                selectedIndex: selected,
                onDestinationSelected: (int index) => setState(() => selected = index),
                expanded: expanded,
                onExpandedChanged: (bool value) => setState(() => expanded = value),
                expandTooltip: 'Expand',
                collapseTooltip: 'Collapse',
              ),
            ),
          ),
        ),
      );
      expect(tester.getSize(find.byType(ExpressiveNavigationRail)).width, ExpressiveNavigationRail.collapsedWidth);
      await tester.tap(find.bySemanticsLabel('Settings'));
      await tester.pumpAndSettle();
      expect(selected, 1);
      expect(find.text('2'), findsOneWidget);

      await tester.tap(find.byTooltip('Expand'));
      await tester.pumpAndSettle();
      expect(expanded, isTrue);
      expect(tester.getSize(find.byType(ExpressiveNavigationRail)).width, ExpressiveNavigationRail.expandedWidth);
    });
  });
}

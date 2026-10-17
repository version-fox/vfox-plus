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

import 'package:material_ui/material_ui.dart';

import 'motion.dart';
import 'tokens.dart';

/// Builds the Material 3 Expressive theme of the application.
abstract final class AppTheme {
  /// The colour scheme for [seed] and [brightness].
  ///
  /// The tonal spot variant is the Material 3 default; it keeps large
  /// surfaces calm and saves the vivid tones for primary actions.
  static ColorScheme scheme(Color seed, Brightness brightness) =>
      ColorScheme.fromSeed(seedColor: seed, brightness: brightness);

  /// The theme for [colors].
  static ThemeData build(ColorScheme colors) {
    // Text styles are deliberately not set here: the text theme of a fresh
    // ThemeData has no font sizes yet. Theme.of() adds them later, matching
    // the script of the current locale, so components use their defaults.
    final base = ThemeData(colorScheme: colors, useMaterial3: true, visualDensity: VisualDensity.standard);
    final shapeTransitionDuration = MotionScheme.expressive.fastEffects.settleDuration;

    ButtonStyle commonButtonStyle() => ButtonStyle(
      // Material 3 Expressive buttons are round at rest and morph into a
      // squarer shape while pressed.
      shape: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        if (states.contains(WidgetState.pressed)) {
          return RoundedRectangleBorder(borderRadius: BorderRadius.circular(Corners.small));
        }
        return const StadiumBorder();
      }),
      animationDuration: shapeTransitionDuration,
      minimumSize: const WidgetStatePropertyAll<Size>(Size(48, 40)),
      padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(EdgeInsets.symmetric(horizontal: Gaps.l)),
      iconSize: const WidgetStatePropertyAll<double>(20),
    );

    return base.copyWith(
      extensions: const <ThemeExtension<dynamic>>[MotionScheme.expressive],
      scaffoldBackgroundColor: colors.surface,
      iconTheme: IconThemeData(color: colors.onSurfaceVariant, size: 24),
      iconButtonTheme: const IconButtonThemeData(variant: StyleVariant.material3Expressive),
      filledButtonTheme: FilledButtonThemeData(style: commonButtonStyle()),
      outlinedButtonTheme: OutlinedButtonThemeData(style: commonButtonStyle()),
      textButtonTheme: TextButtonThemeData(style: commonButtonStyle()),
      elevatedButtonTheme: ElevatedButtonThemeData(style: commonButtonStyle()),
      floatingActionButtonTheme: FloatingActionButtonThemeData(
        backgroundColor: colors.primaryContainer,
        foregroundColor: colors.onPrimaryContainer,
        elevation: 3,
        focusElevation: 3,
        hoverElevation: 4,
        highlightElevation: 3,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.large))),
        extendedPadding: const EdgeInsetsDirectional.only(start: 16, end: 20),
        extendedIconLabelSpacing: 12,
      ),
      cardTheme: CardThemeData(
        elevation: 0,
        margin: EdgeInsets.zero,
        color: colors.surfaceContainerLow,
        clipBehavior: Clip.antiAlias,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.large))),
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: colors.surfaceContainerHigh,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.extraLarge))),
      ),
      menuTheme: MenuThemeData(
        style: MenuStyle(
          backgroundColor: WidgetStatePropertyAll<Color>(colors.surfaceContainer),
          shape: const WidgetStatePropertyAll<OutlinedBorder>(
            RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.large))),
          ),
          padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(EdgeInsets.symmetric(vertical: Gaps.xs)),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        height: 64,
        backgroundColor: colors.surfaceContainer,
        indicatorColor: colors.secondaryContainer,
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: colors.inverseSurface,
        actionTextColor: colors.inversePrimary,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.medium))),
        width: 480,
      ),
      inputDecorationTheme: InputDecorationThemeData(
        filled: true,
        fillColor: colors.surfaceContainerHighest,
        border: const OutlineInputBorder(
          borderSide: BorderSide.none,
          borderRadius: BorderRadius.all(Radius.circular(Corners.medium)),
        ),
        enabledBorder: const OutlineInputBorder(
          borderSide: BorderSide.none,
          borderRadius: BorderRadius.all(Radius.circular(Corners.medium)),
        ),
        focusedBorder: OutlineInputBorder(
          borderSide: BorderSide(color: colors.primary, width: 2),
          borderRadius: const BorderRadius.all(Radius.circular(Corners.medium)),
        ),
        errorBorder: OutlineInputBorder(
          borderSide: BorderSide(color: colors.error),
          borderRadius: const BorderRadius.all(Radius.circular(Corners.medium)),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderSide: BorderSide(color: colors.error, width: 2),
          borderRadius: const BorderRadius.all(Radius.circular(Corners.medium)),
        ),
      ),
      searchBarTheme: SearchBarThemeData(
        elevation: const WidgetStatePropertyAll<double>(0),
        backgroundColor: WidgetStatePropertyAll<Color>(colors.surfaceContainerHigh),
        constraints: const BoxConstraints(minHeight: 56, maxWidth: 720),
        padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(EdgeInsets.symmetric(horizontal: Gaps.l)),
      ),
      chipTheme: const ChipThemeData(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.small))),
      ),
      badgeTheme: BadgeThemeData(backgroundColor: colors.error, textColor: colors.onError),
      tooltipTheme: const TooltipThemeData(waitDuration: Duration(milliseconds: 500)),
      dividerTheme: DividerThemeData(color: colors.outlineVariant, space: 1, thickness: 1),
      listTileTheme: ListTileThemeData(
        iconColor: colors.onSurfaceVariant,
        contentPadding: const EdgeInsetsDirectional.only(start: Gaps.l, end: Gaps.l),
        minVerticalPadding: Gaps.m,
      ),
      pageTransitionsTheme: const PageTransitionsTheme(
        builders: <TargetPlatform, PageTransitionsBuilder>{
          TargetPlatform.linux: FadeForwardsPageTransitionsBuilder(),
          TargetPlatform.windows: FadeForwardsPageTransitionsBuilder(),
          TargetPlatform.macOS: FadeForwardsPageTransitionsBuilder(),
          TargetPlatform.android: FadeForwardsPageTransitionsBuilder(),
        },
      ),
    );
  }
}

/// The emphasized type scale of Material 3 Expressive.
///
/// Emphasized styles use a heavier weight than their baseline counterparts
/// and highlight the most important text on a screen, such as headlines and
/// selected items. See https://m3.material.io/styles/typography/type-scale-tokens.
///
/// The scale is derived from the text theme returned by [Theme.of], which
/// carries the font sizes for the script of the current locale.
@immutable
class EmphasizedTextTheme {
  /// Creates an emphasized type scale.
  const EmphasizedTextTheme({
    required this.displayLarge,
    required this.displayMedium,
    required this.displaySmall,
    required this.headlineLarge,
    required this.headlineMedium,
    required this.headlineSmall,
    required this.titleLarge,
    required this.titleMedium,
    required this.titleSmall,
    required this.bodyLarge,
    required this.bodyMedium,
    required this.labelLarge,
    required this.labelMedium,
  });

  /// Derives the emphasized scale from the baseline [text] theme.
  factory EmphasizedTextTheme.from(TextTheme text) {
    TextStyle emphasize(TextStyle? style, FontWeight weight) => (style ?? const TextStyle()).copyWith(
      fontWeight: weight,
      fontVariations: <FontVariation>[FontVariation.weight(weight.value.toDouble())],
    );
    return EmphasizedTextTheme(
      displayLarge: emphasize(text.displayLarge, FontWeight.w500),
      displayMedium: emphasize(text.displayMedium, FontWeight.w500),
      displaySmall: emphasize(text.displaySmall, FontWeight.w500),
      headlineLarge: emphasize(text.headlineLarge, FontWeight.w500),
      headlineMedium: emphasize(text.headlineMedium, FontWeight.w500),
      headlineSmall: emphasize(text.headlineSmall, FontWeight.w500),
      titleLarge: emphasize(text.titleLarge, FontWeight.w500),
      titleMedium: emphasize(text.titleMedium, FontWeight.w700),
      titleSmall: emphasize(text.titleSmall, FontWeight.w700),
      bodyLarge: emphasize(text.bodyLarge, FontWeight.w500),
      bodyMedium: emphasize(text.bodyMedium, FontWeight.w500),
      labelLarge: emphasize(text.labelLarge, FontWeight.w700),
      labelMedium: emphasize(text.labelMedium, FontWeight.w700),
    );
  }

  /// Emphasized display large.
  final TextStyle displayLarge;

  /// Emphasized display medium.
  final TextStyle displayMedium;

  /// Emphasized display small.
  final TextStyle displaySmall;

  /// Emphasized headline large.
  final TextStyle headlineLarge;

  /// Emphasized headline medium.
  final TextStyle headlineMedium;

  /// Emphasized headline small.
  final TextStyle headlineSmall;

  /// Emphasized title large.
  final TextStyle titleLarge;

  /// Emphasized title medium.
  final TextStyle titleMedium;

  /// Emphasized title small.
  final TextStyle titleSmall;

  /// Emphasized body large.
  final TextStyle bodyLarge;

  /// Emphasized body medium.
  final TextStyle bodyMedium;

  /// Emphasized label large.
  final TextStyle labelLarge;

  /// Emphasized label medium.
  final TextStyle labelMedium;

  static final Expando<EmphasizedTextTheme> _cache = Expando<EmphasizedTextTheme>('EmphasizedTextTheme');

  /// The emphasized scale of the closest [Theme].
  static EmphasizedTextTheme of(BuildContext context) {
    final TextTheme text = Theme.of(context).textTheme;
    return _cache[text] ??= EmphasizedTextTheme.from(text);
  }
}

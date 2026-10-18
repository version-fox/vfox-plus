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

package dev.vfox.plus.ui

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialExpressiveTheme
import androidx.compose.material3.MotionScheme
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.materialkolor.dynamiccolor.ColorSpec.SpecVersion
import com.materialkolor.dynamiccolor.DynamicColor
import com.materialkolor.dynamiccolor.MaterialDynamicColors
import com.materialkolor.hct.Hct
import com.materialkolor.scheme.SchemeTonalSpot
import dev.vfox.plus.core.DEFAULT_SEED_COLOR

/** Seed colours offered when the desktop accent colour is not used; the first is the vfox orange. */
val SEED_COLORS: List<Long> =
    listOf(DEFAULT_SEED_COLOR, 0xFFD9473B, 0xFFC2407A, 0xFF7E57C2, 0xFF3F6FD8, 0xFF0B8A9A, 0xFF3E8E41, 0xFF8D7B1E)

/**
 * The tonal spot scheme of the Material colour utilities for [seed], in the
 * 2021 specification that every implementation uses (docs/design-system.md).
 */
fun colorSchemeFor(
    seed: Long,
    dark: Boolean,
): ColorScheme {
    val scheme = SchemeTonalSpot(listOf(Hct.fromInt(seed.toInt())), dark, 0.0, SpecVersion.SPEC_2021)
    val colors = MaterialDynamicColors()

    fun color(dynamic: DynamicColor) = Color(dynamic.getArgb(scheme))
    return ColorScheme(
        primary = color(colors.primary),
        onPrimary = color(colors.onPrimary),
        primaryContainer = color(colors.primaryContainer),
        onPrimaryContainer = color(colors.onPrimaryContainer),
        inversePrimary = color(colors.inversePrimary),
        secondary = color(colors.secondary),
        onSecondary = color(colors.onSecondary),
        secondaryContainer = color(colors.secondaryContainer),
        onSecondaryContainer = color(colors.onSecondaryContainer),
        tertiary = color(colors.tertiary),
        onTertiary = color(colors.onTertiary),
        tertiaryContainer = color(colors.tertiaryContainer),
        onTertiaryContainer = color(colors.onTertiaryContainer),
        background = color(colors.background),
        onBackground = color(colors.onBackground),
        surface = color(colors.surface),
        onSurface = color(colors.onSurface),
        surfaceVariant = color(colors.surfaceVariant),
        onSurfaceVariant = color(colors.onSurfaceVariant),
        surfaceTint = color(colors.surfaceTint),
        inverseSurface = color(colors.inverseSurface),
        inverseOnSurface = color(colors.inverseOnSurface),
        error = color(colors.error),
        onError = color(colors.onError),
        errorContainer = color(colors.errorContainer),
        onErrorContainer = color(colors.onErrorContainer),
        outline = color(colors.outline),
        outlineVariant = color(colors.outlineVariant),
        scrim = color(colors.scrim),
        surfaceBright = color(colors.surfaceBright),
        surfaceDim = color(colors.surfaceDim),
        surfaceContainer = color(colors.surfaceContainer),
        surfaceContainerHigh = color(colors.surfaceContainerHigh),
        surfaceContainerHighest = color(colors.surfaceContainerHighest),
        surfaceContainerLow = color(colors.surfaceContainerLow),
        surfaceContainerLowest = color(colors.surfaceContainerLowest),
        primaryFixed = color(colors.primaryFixed),
        primaryFixedDim = color(colors.primaryFixedDim),
        onPrimaryFixed = color(colors.onPrimaryFixed),
        onPrimaryFixedVariant = color(colors.onPrimaryFixedVariant),
        secondaryFixed = color(colors.secondaryFixed),
        secondaryFixedDim = color(colors.secondaryFixedDim),
        onSecondaryFixed = color(colors.onSecondaryFixed),
        onSecondaryFixedVariant = color(colors.onSecondaryFixedVariant),
        tertiaryFixed = color(colors.tertiaryFixed),
        tertiaryFixedDim = color(colors.tertiaryFixedDim),
        onTertiaryFixed = color(colors.onTertiaryFixed),
        onTertiaryFixedVariant = color(colors.onTertiaryFixedVariant),
    )
}

/** The shape tokens of docs/design-system.md#shape. */
object Corners {
    val extraSmall = 4.dp
    val small = 8.dp
    val medium = 12.dp
    val large = 16.dp
    val largeIncreased = 20.dp
    val extraLarge = 28.dp
}

/** The spacing tokens of docs/design-system.md#spacing. */
object Gaps {
    val hairline = 2.dp
    val xs = 4.dp
    val s = 8.dp
    val m = 12.dp
    val l = 16.dp
    val xl = 24.dp
    val xxl = 32.dp
}

/**
 * The Material 3 type scale with the emphasized styles of
 * docs/design-system.md#typography. `Typography()` resolves to the constructor
 * without emphasized styles, which makes them equal to the baseline ones.
 */
fun expressiveTypography(): Typography {
    val base = Typography()

    fun TextStyle.weight(weight: FontWeight) = copy(fontWeight = weight)
    return Typography(
        displayLarge = base.displayLarge,
        displayMedium = base.displayMedium,
        displaySmall = base.displaySmall,
        headlineLarge = base.headlineLarge,
        headlineMedium = base.headlineMedium,
        headlineSmall = base.headlineSmall,
        titleLarge = base.titleLarge,
        titleMedium = base.titleMedium,
        titleSmall = base.titleSmall,
        bodyLarge = base.bodyLarge,
        bodyMedium = base.bodyMedium,
        bodySmall = base.bodySmall,
        labelLarge = base.labelLarge,
        labelMedium = base.labelMedium,
        labelSmall = base.labelSmall,
        displayLargeEmphasized = base.displayLarge.weight(FontWeight.Medium),
        displayMediumEmphasized = base.displayMedium.weight(FontWeight.Medium),
        displaySmallEmphasized = base.displaySmall.weight(FontWeight.Medium),
        headlineLargeEmphasized = base.headlineLarge.weight(FontWeight.Medium),
        headlineMediumEmphasized = base.headlineMedium.weight(FontWeight.Medium),
        headlineSmallEmphasized = base.headlineSmall.weight(FontWeight.Medium),
        titleLargeEmphasized = base.titleLarge.weight(FontWeight.Medium),
        titleMediumEmphasized = base.titleMedium.weight(FontWeight.Bold),
        titleSmallEmphasized = base.titleSmall.weight(FontWeight.Bold),
        bodyLargeEmphasized = base.bodyLarge.weight(FontWeight.Medium),
        bodyMediumEmphasized = base.bodyMedium.weight(FontWeight.Medium),
        bodySmallEmphasized = base.bodySmall.weight(FontWeight.Medium),
        labelLargeEmphasized = base.labelLarge.weight(FontWeight.Bold),
        labelMediumEmphasized = base.labelMedium.weight(FontWeight.Bold),
        labelSmallEmphasized = base.labelSmall.weight(FontWeight.Bold),
    )
}

/**
 * The Material 3 Expressive theme: the colours of [seed] in light or [dark],
 * the expressive motion scheme, whose springs are those of
 * docs/design-system.md#motion, and the type scale with its emphasized styles.
 */
@Composable
fun VfoxPlusTheme(
    seed: Long,
    dark: Boolean,
    content: @Composable () -> Unit,
) {
    val colors = remember(seed, dark) { colorSchemeFor(seed, dark) }
    val typography = remember { expressiveTypography() }
    MaterialExpressiveTheme(colorScheme = colors, motionScheme = MotionScheme.expressive(), typography = typography, content = content)
}

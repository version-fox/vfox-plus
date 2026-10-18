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

package dev.vfox.plus.ui.components

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.hoverable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ButtonGroupDefaults
import androidx.compose.material3.ContainedLoadingIndicator
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PlainTooltip
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.ToggleButton
import androidx.compose.material3.ToggleButtonDefaults
import androidx.compose.material3.TooltipAnchorPosition
import androidx.compose.material3.TooltipBox
import androidx.compose.material3.TooltipDefaults
import androidx.compose.material3.rememberTooltipState
import androidx.compose.material3.toShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Outline
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.graphics.shapes.Morph
import androidx.graphics.shapes.RoundedPolygon
import dev.vfox.plus.ui.Corners
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.t

/**
 * The shapes of the Material 3 Expressive shape library that look good as
 * decorative containers of small icons and initials, in the order every
 * implementation uses.
 */
val DECORATIVE_SHAPES: List<RoundedPolygon>
    get() =
        listOf(
            MaterialShapes.Cookie9Sided,
            MaterialShapes.Clover4Leaf,
            MaterialShapes.Sunny,
            MaterialShapes.SoftBurst,
            MaterialShapes.Cookie6Sided,
            MaterialShapes.Gem,
            MaterialShapes.Pentagon,
            MaterialShapes.Flower,
            MaterialShapes.PuffyDiamond,
            MaterialShapes.Cookie4Sided,
            MaterialShapes.Arch,
            MaterialShapes.Clover8Leaf,
        )

/**
 * A deterministic string hash (FNV-1a, 32 bit) of the UTF-16 code units of
 * [text], the same as in the other implementations, so that a plugin keeps its
 * shape and colour everywhere.
 */
fun stableHash(text: String): Long {
    var hash = 0x811c9dc5L
    for (unit in text) {
        hash = hash xor unit.code.toLong()
        hash = (hash * 0x01000193L) and 0xFFFFFFFFL
    }
    return hash
}

/** A shape that draws a [Morph] of two shapes of the shape library at [progress]. */
class MorphShape(
    private val morph: Morph,
    private val progress: Float,
) : Shape {
    override fun createOutline(
        size: androidx.compose.ui.geometry.Size,
        layoutDirection: LayoutDirection,
        density: Density,
    ): Outline {
        val path = Path()
        val cubics = morph.asCubics(progress)
        cubics.firstOrNull()?.let { path.moveTo(it.anchor0X * size.width, it.anchor0Y * size.height) }
        for (cubic in cubics) {
            path.cubicTo(
                cubic.control0X * size.width,
                cubic.control0Y * size.height,
                cubic.control1X * size.width,
                cubic.control1Y * size.height,
                cubic.anchor1X * size.width,
                cubic.anchor1Y * size.height,
            )
        }
        path.close()
        return Outline.Generic(path)
    }
}

/**
 * A plugin avatar: an initial on a decorative Material 3 Expressive shape. The
 * shape and the colour follow from the name; hovering spins the shape by 30°
 * on the fast spatial spring.
 */
@Composable
fun Avatar(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 40.dp,
    icon: String? = null,
) {
    val hash = stableHash(name)
    val colors = MaterialTheme.colorScheme
    // Selected list items use the secondary container colour. Avatars avoid it,
    // and the primary container too, which some seeds (the vfox orange among
    // them) make just as light, so that they stay visible on selected items.
    val (background, foreground) =
        when ((hash % 3).toInt()) {
            0 -> colors.primaryFixedDim to colors.onPrimaryFixed
            1 -> colors.secondary to colors.onSecondary
            else -> colors.tertiaryContainer to colors.onTertiaryContainer
        }
    val shape = DECORATIVE_SHAPES[(hash % DECORATIVE_SHAPES.size).toInt()].toShape()
    val interactions = remember { MutableInteractionSource() }
    val hovered by interactions.collectIsHoveredAsState()
    val rotation by animateFloatAsState(if (hovered) 30f else 0f, MaterialTheme.motionScheme.fastSpatialSpec())
    Box(modifier.size(size).hoverable(interactions), contentAlignment = Alignment.Center) {
        Box(Modifier.fillMaxSize().graphicsLayer { rotationZ = rotation }.background(background, shape))
        if (icon != null) {
            Icon(icon, size = size * 0.5f, tint = foreground)
        } else {
            val initial = name.firstOrNull()?.uppercase() ?: "?"
            Text(initial, color = foreground, fontWeight = FontWeight.Bold, fontSize = (size.value * 0.42f).sp)
        }
    }
}

/** A small label, such as "In use: 24.11.1" or "LTS"; [emphasized] uses the tertiary container. */
@Composable
fun Tag(
    text: String,
    modifier: Modifier = Modifier,
    emphasized: Boolean = false,
    icon: String? = null,
) {
    val colors = MaterialTheme.colorScheme
    Surface(
        modifier = modifier,
        shape = CircleShape,
        color = if (emphasized) colors.tertiaryContainer else colors.surfaceContainerHighest,
        contentColor = if (emphasized) colors.onTertiaryContainer else colors.onSurfaceVariant,
    ) {
        Row(Modifier.padding(horizontal = 10.dp, vertical = 3.dp), verticalAlignment = Alignment.CenterVertically) {
            if (icon != null) {
                Icon(icon, filled = true, size = 16.dp)
                Box(Modifier.size(4.dp))
            }
            Text(text, style = MaterialTheme.typography.labelMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/**
 * A friendly placeholder for empty, unavailable or failed content, with an
 * expressive shape behind its icon and buttons in [actions].
 */
@Composable
fun StatusMessage(
    icon: String,
    heading: String,
    modifier: Modifier = Modifier,
    body: String? = null,
    shape: RoundedPolygon = MaterialShapes.Cookie9Sided,
    error: Boolean = false,
    actions: @Composable RowScope.() -> Unit = {},
) {
    val colors = MaterialTheme.colorScheme
    Box(modifier.fillMaxSize().padding(Gaps.xxl), contentAlignment = Alignment.Center) {
        Column(Modifier.widthIn(max = 440.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(Modifier.size(112.dp), contentAlignment = Alignment.Center) {
                Box(
                    Modifier
                        .fillMaxSize()
                        .background(if (error) colors.errorContainer else colors.secondaryContainer, shape.toShape()),
                )
                Icon(icon, filled = true, size = 48.dp, tint = if (error) colors.onErrorContainer else colors.onSecondaryContainer)
            }
            Box(Modifier.size(Gaps.xl))
            Text(
                heading,
                style = MaterialTheme.typography.titleLargeEmphasized,
                color = colors.onSurface,
                textAlign = TextAlign.Center,
                modifier = Modifier.semantics { heading() },
            )
            if (body != null) {
                Text(
                    body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(top = Gaps.s),
                )
            }
            FlowRow(
                Modifier.padding(top = Gaps.xl),
                horizontalArrangement = Arrangement.spacedBy(Gaps.s, Alignment.CenterHorizontally),
                content = { actions() },
            )
        }
    }
}

/** A centred loading indicator with an optional label. */
@Composable
fun LoadingMessage(
    modifier: Modifier = Modifier,
    label: String? = null,
) {
    val description = label ?: t("loading")
    Box(modifier.fillMaxSize().padding(Gaps.xxl), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Gaps.l)) {
            ContainedLoadingIndicator(Modifier.size(64.dp).semantics { contentDescription = description })
            if (label != null) {
                Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

/** A status message for an error, with its text and a retry button. */
@Composable
fun ErrorMessage(
    heading: String,
    error: Throwable?,
    retry: (() -> Unit)? = null,
) {
    StatusMessage(
        icon = "error",
        heading = heading,
        body = error?.message ?: error?.toString(),
        shape = MaterialShapes.SoftBurst,
        error = true,
    ) {
        if (retry != null) {
            FilledTonalButton(onClick = retry) {
                Icon("retry", size = ButtonDefaults.IconSize)
                Box(Modifier.size(ButtonDefaults.IconSpacing))
                Text(t("actionRetry"))
            }
        }
    }
}

/** One choice of a [ChoiceGroup]: its value, label and optional icon. */
data class Choice<T>(
    val value: T,
    val label: String,
    val icon: String? = null,
)

/**
 * A Material 3 Expressive connected button group with single selection. The
 * selected button cannot be unselected by clicking it again.
 */
@Composable
fun <T> ChoiceGroup(
    choices: List<Choice<T>>,
    selected: T,
    modifier: Modifier = Modifier,
    onSelect: (T) -> Unit,
) {
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(ButtonGroupDefaults.ConnectedSpaceBetween)) {
        choices.forEachIndexed { index, choice ->
            ToggleButton(
                checked = choice.value == selected,
                onCheckedChange = { if (choice.value != selected) onSelect(choice.value) },
                shapes =
                    when (index) {
                        0 -> ButtonGroupDefaults.connectedLeadingButtonShapes()
                        choices.lastIndex -> ButtonGroupDefaults.connectedTrailingButtonShapes()
                        else -> ButtonGroupDefaults.connectedMiddleButtonShapes()
                    },
                colors = ToggleButtonDefaults.tonalToggleButtonColors(),
                contentPadding = PaddingValues(horizontal = 14.dp),
            ) {
                if (choice.icon != null) {
                    Icon(choice.icon, size = ButtonDefaults.IconSize, filled = choice.value == selected)
                    Box(Modifier.size(ButtonDefaults.IconSpacing))
                }
                Text(choice.label, maxLines = 1)
            }
        }
    }
}

/**
 * An icon button with a tooltip, which also serves as its accessible name.
 * Every icon-only button of the application is one.
 */
@Composable
fun TooltipIconButton(
    icon: String,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    filled: Boolean = false,
    tonal: Boolean = false,
    onClick: () -> Unit,
) {
    TooltipBox(
        positionProvider = TooltipDefaults.rememberTooltipPositionProvider(TooltipAnchorPosition.Above),
        tooltip = { PlainTooltip { Text(label) } },
        state = rememberTooltipState(),
    ) {
        IconButton(
            onClick = onClick,
            enabled = enabled,
            modifier = modifier.semantics { contentDescription = label },
            colors = if (tonal) IconButtonDefaults.filledTonalIconButtonColors() else IconButtonDefaults.iconButtonColors(),
            shapes = IconButtonDefaults.shapes(),
        ) {
            Icon(icon, filled = filled)
        }
    }
}

/** The heading of a group of list items, in the primary colour. */
@Composable
fun GroupHeader(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(start = Gaps.l, top = Gaps.xl, bottom = Gaps.s, end = Gaps.xs).semantics { heading() },
    )
}

/** The title row of a page, with a subtitle and actions. */
@Composable
fun PageHeader(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    /** Whether the header keeps its distance from the edges of the page itself. */
    inset: Boolean = true,
    actions: @Composable RowScope.() -> Unit = {},
) {
    Row(
        modifier.padding(start = if (inset) Gaps.xl else 0.dp, top = Gaps.xl, end = if (inset) Gaps.l else 0.dp, bottom = Gaps.l),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Gaps.s),
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                title,
                style = MaterialTheme.typography.headlineMediumEmphasized,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.semantics { heading() },
            )
            if (subtitle != null) {
                Text(
                    subtitle,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(top = Gaps.xs),
                )
            }
        }
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.s), content = actions)
    }
}

/** The rounded surface that holds the content of a page. */
@Composable
fun Pane(
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    Box(
        modifier
            .fillMaxSize()
            .clip(RoundedCornerShape(Corners.extraLarge))
            .background(MaterialTheme.colorScheme.surfaceContainerLow),
    ) {
        content()
    }
}

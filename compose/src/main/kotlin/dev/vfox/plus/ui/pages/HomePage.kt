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

package dev.vfox.plus.ui.pages

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.requiredSize
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedListItem
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.toShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.graphics.shapes.RoundedPolygon
import dev.vfox.plus.ui.Corners
import dev.vfox.plus.ui.Destination
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.Links
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.Avatar
import dev.vfox.plus.ui.components.ErrorMessage
import dev.vfox.plus.ui.components.GroupHeader
import dev.vfox.plus.ui.components.LoadingMessage
import dev.vfox.plus.ui.components.PageHeader
import dev.vfox.plus.ui.components.Pane
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.components.segmentedItemColors
import dev.vfox.plus.ui.t
import kotlinx.coroutines.launch

/** The line that activates vfox in Bash. */
const val ACTIVATION_COMMAND = "eval \"\$(vfox activate bash)\""

/** A tile of the statistics, which springs to smaller corners while pressed. */
@Composable
private fun StatTile(
    value: Int,
    label: String,
    icon: String,
    shape: RoundedPolygon,
    container: Color,
    content: Color,
    compact: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val interactions = remember { MutableInteractionSource() }
    val pressed by interactions.collectIsPressedAsState()
    val corner by animateDpAsState(if (pressed) Corners.large else Corners.extraLarge, MaterialTheme.motionScheme.fastSpatialSpec())
    val decoration = shape.toShape()
    Surface(
        onClick = onClick,
        modifier = modifier.heightIn(min = 136.dp),
        shape = RoundedCornerShape(corner),
        color = container,
        contentColor = content,
        interactionSource = interactions,
    ) {
        Box {
            // The shape does not make the tile taller: it is partly hidden behind the edge.
            Box(Modifier.matchParentSize()) {
                Box(
                    Modifier
                        .align(Alignment.BottomEnd)
                        .offset(x = 28.dp, y = 36.dp)
                        .requiredSize(140.dp)
                        .background(content.copy(alpha = 0.08f), decoration),
                )
            }
            Column(Modifier.padding(if (compact) Gaps.l else Gaps.xl).fillMaxHeight()) {
                Icon(icon, filled = true)
                Box(Modifier.weight(1f))
                val typography = MaterialTheme.typography
                Text(value.toString(), style = if (compact) typography.headlineLargeEmphasized else typography.displaySmallEmphasized)
                Text(
                    label,
                    style = MaterialTheme.typography.labelLargeEmphasized,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(top = Gaps.xs),
                )
            }
        }
    }
}

/** A card that leads somewhere else. */
@Composable
private fun Shortcut(
    icon: String,
    title: String,
    body: String,
    trailing: String,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val interactions = remember { MutableInteractionSource() }
    val pressed by interactions.collectIsPressedAsState()
    val corner by animateDpAsState(if (pressed) Corners.small else Corners.large, MaterialTheme.motionScheme.fastSpatialSpec())
    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(corner),
        color = MaterialTheme.colorScheme.surfaceContainer,
        interactionSource = interactions,
    ) {
        Row(
            Modifier.padding(Gaps.l),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(Gaps.l),
        ) {
            Box(Modifier.size(48.dp), contentAlignment = Alignment.Center) {
                Box(
                    Modifier.size(48.dp).background(
                        MaterialTheme.colorScheme.secondaryContainer,
                        MaterialShapes.Cookie6Sided.toShape(),
                    ),
                )
                Icon(icon, tint = MaterialTheme.colorScheme.onSecondaryContainer)
            }
            Column(Modifier.weight(1f)) {
                Text(title, style = MaterialTheme.typography.titleSmallEmphasized)
                Text(
                    body,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = Gaps.xs),
                )
            }
            Icon(trailing, tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

/** The overview: what is installed, what is in use and where to go next. */
@Composable
fun HomePage() {
    val context = app
    val model = context.model
    val navigation = context.navigation
    val sdks = model.installed.value
    Pane {
        if (sdks == null) {
            val error = model.installed.error
            if (error == null) LoadingMessage() else ErrorMessage(t("sdksLoadError"), error) { model.installed.load() }
            return@Pane
        }
        val active = sdks.filter { it.current != null }
        val versions = sdks.sumOf { it.versions.size }
        val subtitle = listOfNotNull(t("homeSubtitle", "version" to (model.vfox.version ?: "?")), model.vfox.executable).joinToString(" · ")
        BoxWithConstraints(Modifier.fillMaxWidth()) {
            val compact = maxWidth < 608.dp
            val narrow = maxWidth < 808.dp
            Column(Modifier.verticalScroll(rememberScrollState())) {
                PageHeader(t("appTitle"), subtitle = subtitle) {
                    TooltipIconButton("refresh", t("actionRefresh"), enabled = !model.installed.isLoading) { model.installed.load() }
                }
                Column(
                    Modifier
                        .align(Alignment.CenterHorizontally)
                        .widthIn(max = 1080.dp)
                        .padding(start = Gaps.xl, end = Gaps.xl, bottom = Gaps.xl),
                ) {
                    val colors = MaterialTheme.colorScheme
                    // Tiles share the height of the tallest, whose label may take two lines.
                    Row(Modifier.height(IntrinsicSize.Min), horizontalArrangement = Arrangement.spacedBy(if (compact) Gaps.s else Gaps.m)) {
                        val tileModifier = Modifier.weight(1f).fillMaxHeight()
                        StatTile(
                            sdks.size,
                            t("homeStatPlugins", "count" to sdks.size),
                            "plugins",
                            MaterialShapes.Cookie9Sided,
                            colors.primaryContainer,
                            colors.onPrimaryContainer,
                            compact,
                            tileModifier,
                        ) { navigation.showSdk(null) }
                        StatTile(
                            versions,
                            t("homeStatVersions", "count" to versions),
                            "versions",
                            MaterialShapes.Clover4Leaf,
                            colors.secondaryContainer,
                            colors.onSecondaryContainer,
                            compact,
                            tileModifier,
                        ) { navigation.showSdk(null) }
                        StatTile(
                            active.size,
                            t("homeStatActive"),
                            "current",
                            MaterialShapes.Sunny,
                            colors.tertiaryContainer,
                            colors.onTertiaryContainer,
                            compact,
                            tileModifier,
                        ) { navigation.showSdk(null) }
                    }
                    GroupHeader(t("homeActiveTitle"))
                    Column(verticalArrangement = Arrangement.spacedBy(ListItemDefaults.SegmentedGap)) {
                        if (active.isEmpty()) {
                            SegmentedListItem(
                                onClick = { navigation.showSdk(null) },
                                shapes = ListItemDefaults.segmentedShapes(0, 1),
                                colors = segmentedItemColors(),
                                leadingContent = { Icon("info") },
                            ) { Text(t("homeActiveEmpty")) }
                        }
                        active.forEachIndexed { index, sdk ->
                            SegmentedListItem(
                                onClick = { navigation.showSdk(sdk.name) },
                                shapes = ListItemDefaults.segmentedShapes(index, active.size),
                                colors = segmentedItemColors(),
                                leadingContent = { Avatar(sdk.name) },
                                supportingContent = { Text(sdk.current.orEmpty()) },
                                trailingContent = { Icon("chevronRight") },
                            ) { Text(sdk.name) }
                        }
                    }
                    val shell: @Composable (Modifier) -> Unit = { modifier -> ShellCard(modifier) }
                    val explore: @Composable (Modifier) -> Unit = { modifier ->
                        Shortcut("explore", t("homeExploreTitle"), t("homeExploreBody"), "chevronRight", modifier) {
                            navigation.go(Destination.PLUGINS)
                        }
                    }
                    val docs: @Composable (Modifier) -> Unit = { modifier ->
                        Shortcut(
                            "docs",
                            t("homeDocsTitle"),
                            t("homeDocsBody"),
                            "openInNew",
                            modifier,
                        ) { context.services.openUrl(Links.VFOX_DOCS) }
                    }
                    Box(Modifier.height(Gaps.xl))
                    if (narrow) {
                        Column(verticalArrangement = Arrangement.spacedBy(Gaps.m)) {
                            shell(Modifier.fillMaxWidth())
                            explore(Modifier.fillMaxWidth())
                            docs(Modifier.fillMaxWidth())
                        }
                    } else {
                        Row(Modifier.height(IntrinsicSize.Min), horizontalArrangement = Arrangement.spacedBy(Gaps.m)) {
                            shell(Modifier.weight(3f).fillMaxHeight())
                            Column(Modifier.weight(2f), verticalArrangement = Arrangement.spacedBy(Gaps.m)) {
                                explore(Modifier.fillMaxWidth().weight(1f))
                                docs(Modifier.fillMaxWidth().weight(1f))
                            }
                        }
                    }
                }
            }
        }
    }
}

/** How to use the chosen versions in a terminal, with the line to copy. */
@Composable
private fun ShellCard(modifier: Modifier) {
    val context = app
    val copied = t("actionCopied")
    Surface(modifier, shape = RoundedCornerShape(Corners.large), color = MaterialTheme.colorScheme.surfaceContainerHighest) {
        Column(Modifier.padding(Gaps.xl)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.m)) {
                Icon("terminal", tint = MaterialTheme.colorScheme.primary)
                Text(t("homeShellTitle"), style = MaterialTheme.typography.titleMediumEmphasized)
            }
            Text(
                t("homeShellBody"),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = Gaps.s, bottom = Gaps.m),
            )
            Row(
                Modifier
                    .fillMaxWidth()
                    .heightIn(min = 48.dp)
                    .clip(RoundedCornerShape(Corners.medium))
                    .background(MaterialTheme.colorScheme.surfaceContainerLowest)
                    .padding(start = Gaps.l, end = Gaps.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                SelectionContainer(Modifier.weight(1f)) {
                    Text(ACTIVATION_COMMAND, fontFamily = FontFamily.Monospace, style = MaterialTheme.typography.bodyMedium, maxLines = 1)
                }
                TooltipIconButton("copy", t("actionCopy")) {
                    context.services.copyText(ACTIVATION_COMMAND)
                    context.scope.launch { context.snackbars.showSnackbar(copied) }
                }
            }
        }
    }
}

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

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.hoverable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.ListItemColors
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import dev.vfox.plus.ui.Corners
import dev.vfox.plus.ui.Gaps

/**
 * A row of a segmented list that is not a list item of its own, such as a
 * version with its actions or a setting with its control: every row is a
 * surface of its own, 2 dp apart; the first and the last have large outer
 * corners, and the inner corners grow on the fast spatial spring while the
 * pointer is on the row. With [onClick] the whole row is a button, whose
 * state layers follow its shape.
 */
@Composable
fun SegmentedRow(
    index: Int,
    count: Int,
    modifier: Modifier = Modifier,
    color: Color = MaterialTheme.colorScheme.surfaceContainer,
    roundBottom: Boolean = true,
    onClick: (() -> Unit)? = null,
    content: @Composable RowScope.() -> Unit,
) {
    val interactions = remember { MutableInteractionSource() }
    val hovered by interactions.collectIsHoveredAsState()
    val pressed by interactions.collectIsPressedAsState()
    val inner by animateDpAsState(
        if (hovered || pressed) Corners.medium else Corners.extraSmall,
        MaterialTheme.motionScheme.fastSpatialSpec(),
    )
    val top = if (index == 0) Corners.largeIncreased else inner
    val bottom = if (index == count - 1 && roundBottom) Corners.largeIncreased else inner
    val shape = RoundedCornerShape(topStart = top, topEnd = top, bottomStart = bottom, bottomEnd = bottom)
    val row: @Composable () -> Unit = {
        Row(
            Modifier.heightIn(min = 56.dp).padding(horizontal = Gaps.l, vertical = Gaps.s),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(Gaps.l),
            content = content,
        )
    }
    if (onClick == null) {
        Surface(modifier = modifier.fillMaxWidth().hoverable(interactions), shape = shape, color = color, content = row)
    } else {
        Surface(
            onClick = onClick,
            modifier = modifier.fillMaxWidth(),
            shape = shape,
            color = color,
            interactionSource = interactions,
            content = row,
        )
    }
}

/** The colours of the items of segmented lists: surface containers on the panes of the pages. */
@Composable
fun segmentedItemColors(): ListItemColors = ListItemDefaults.segmentedColors(containerColor = MaterialTheme.colorScheme.surfaceContainer)

/** A menu in the style of the application: a surface container with large corners. */
@Composable
fun AppMenu(
    expanded: Boolean,
    onDismissRequest: () -> Unit,
    content: @Composable ColumnScope.() -> Unit,
) = DropdownMenu(
    expanded = expanded,
    onDismissRequest = onDismissRequest,
    shape = RoundedCornerShape(Corners.large),
    containerColor = MaterialTheme.colorScheme.surfaceContainer,
    content = content,
)

/** The 2 dp between the rows of a segmented list. */
@Composable
fun SegmentedGap() = Box(Modifier.padding(top = Gaps.hairline))

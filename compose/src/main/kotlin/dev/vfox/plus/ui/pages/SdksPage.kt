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

import androidx.compose.animation.AnimatedContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedListItem
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import dev.vfox.plus.core.InstalledSdk
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.Avatar
import dev.vfox.plus.ui.components.ErrorMessage
import dev.vfox.plus.ui.components.LoadingMessage
import dev.vfox.plus.ui.components.PageHeader
import dev.vfox.plus.ui.components.Pane
import dev.vfox.plus.ui.components.StatusMessage
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.components.segmentedItemColors
import dev.vfox.plus.ui.fadeThrough
import dev.vfox.plus.ui.sharedAxisX
import dev.vfox.plus.ui.t

/** The narrowest page that shows the list and the detail side by side. */
val TWO_PANE_WIDTH = 840.dp

/**
 * Installed SDKs, in the list-detail canonical layout: wide pages show the
 * list and the selected SDK side by side, and the detail fades through when
 * the selection changes; narrow pages show one at a time and move between them
 * along the horizontal shared axis.
 */
@Composable
fun SdksPage() {
    val context = app
    val model = context.model
    val sdks = model.installed.value
    if (sdks == null) {
        Pane {
            val error = model.installed.error
            if (error == null) LoadingMessage() else ErrorMessage(t("sdksLoadError"), error) { model.installed.load() }
        }
        return
    }
    val selected = context.navigation.selectedSdk?.takeIf { name -> sdks.any { it.name == name } }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        if (maxWidth >= TWO_PANE_WIDTH) {
            Row(horizontalArrangement = Arrangement.spacedBy(Gaps.xl)) {
                Pane(Modifier.width(360.dp)) { SdkList(sdks, selected, chevrons = false) }
                Pane(Modifier.weight(1f)) {
                    AnimatedContent(selected, transitionSpec = fadeThrough(), label = "sdk detail") { name ->
                        if (name == null) {
                            StatusMessage("sdks", t("sdksSelectPrompt"), shape = MaterialShapes.Gem)
                        } else {
                            SdkDetail(name, back = false)
                        }
                    }
                }
            }
        } else {
            AnimatedContent(
                selected,
                transitionSpec = sharedAxisX { _, target -> target != null },
                label = "sdk list or detail",
            ) { name ->
                Pane { if (name == null) SdkList(sdks, null, chevrons = true) else SdkDetail(name, back = true) }
            }
        }
    }
}

/** A search field in the style of a Material 3 search bar. */
@Composable
internal fun SearchField(
    query: String,
    hint: String,
    modifier: Modifier = Modifier,
    onQueryChange: (String) -> Unit,
) {
    TextField(
        value = query,
        onValueChange = onQueryChange,
        modifier = modifier,
        placeholder = { Text(hint) },
        leadingIcon = { Icon("search") },
        trailingIcon =
            if (query.isEmpty()) {
                null
            } else {
                { TooltipIconButton("cancel", t("actionClose")) { onQueryChange("") } }
            },
        singleLine = true,
        shape = CircleShape,
        colors =
            TextFieldDefaults.colors(
                focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent,
            ),
    )
}

/** A text field that filters a list, with a search icon. */
@Composable
internal fun FilterField(
    query: String,
    hint: String,
    modifier: Modifier = Modifier,
    onQueryChange: (String) -> Unit,
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        modifier = modifier,
        placeholder = { Text(hint) },
        leadingIcon = { Icon("search") },
        singleLine = true,
    )
}

@Composable
private fun SdkList(
    sdks: List<InstalledSdk>,
    selected: String?,
    chevrons: Boolean,
) {
    val context = app
    val model = context.model
    if (sdks.isEmpty()) {
        StatusMessage("empty", t("sdksEmptyTitle"), body = t("sdksEmptyBody"), shape = MaterialShapes.Flower) {
            androidx.compose.material3.Button(onClick = { context.navigation.go(dev.vfox.plus.ui.Destination.PLUGINS) }) {
                IconLabel("add", t("navAddPlugin"))
            }
        }
        return
    }
    var query by remember { mutableStateOf("") }
    val needle = query.trim().lowercase()
    val visible = sdks.filter { needle.isEmpty() || it.name.lowercase().contains(needle) }
    Column {
        PageHeader(t("navSdks")) {
            TooltipIconButton("refresh", t("actionRefresh"), enabled = !model.installed.isLoading) { model.installed.load() }
        }
        SearchField(
            query,
            t("sdksSearchHint"),
            Modifier.fillMaxWidth().padding(start = Gaps.l, end = Gaps.l, bottom = Gaps.m),
        ) { query = it }
        if (visible.isEmpty()) {
            StatusMessage("notFound", t("sdksNoMatch", "query" to query.trim()))
            return@Column
        }
        LazyColumn(
            contentPadding = PaddingValues(start = Gaps.l, end = Gaps.l, bottom = Gaps.l),
            verticalArrangement = Arrangement.spacedBy(ListItemDefaults.SegmentedGap),
        ) {
            itemsIndexed(visible, key = { _, sdk -> sdk.name }) { index, sdk ->
                SegmentedListItem(
                    modifier = Modifier.testTag("sdk:${sdk.name}"),
                    selected = sdk.name == selected,
                    onClick = { context.navigation.showSdk(sdk.name) },
                    shapes = ListItemDefaults.segmentedShapes(index, visible.size),
                    colors = segmentedItemColors(),
                    leadingContent = { Avatar(sdk.name) },
                    supportingContent = {
                        Text(
                            (if (sdk.current == null) t("sdksNoCurrent") else t("sdksCurrent", "version" to sdk.current)) +
                                "\n" + t("sdksVersionCount", "count" to sdk.versions.size),
                        )
                    },
                    trailingContent = if (chevrons) ({ Icon("chevronRight") }) else null,
                ) { Text(sdk.name) }
            }
        }
    }
}

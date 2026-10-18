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

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PlainTooltip
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TooltipAnchorPosition
import androidx.compose.material3.TooltipBox
import androidx.compose.material3.TooltipDefaults
import androidx.compose.material3.rememberTooltipState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.vfox.plus.core.AvailablePlugin
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.ui.Corners
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.Avatar
import dev.vfox.plus.ui.components.Choice
import dev.vfox.plus.ui.components.ChoiceGroup
import dev.vfox.plus.ui.components.CustomPluginDialog
import dev.vfox.plus.ui.components.ErrorMessage
import dev.vfox.plus.ui.components.LoadingMessage
import dev.vfox.plus.ui.components.PageHeader
import dev.vfox.plus.ui.components.Pane
import dev.vfox.plus.ui.components.StatusMessage
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.t
import kotlinx.coroutines.launch
import java.net.URI

private enum class PluginFilter { ALL, OFFICIAL, COMMUNITY }

/** The repository path of a homepage, such as `version-fox/vfox-nodejs`. */
fun repositoryPath(homepage: String): String =
    runCatching {
        URI(homepage)
            .path
            .split('/')
            .filter { it.isNotEmpty() }
            .joinToString("/")
    }.getOrNull()
        ?.ifEmpty { null } ?: homepage

/** The plugin registry: every plugin that `vfox add` can install. */
@Composable
fun PluginsPage() {
    val context = app
    val model = context.model
    LaunchedEffect(Unit) { model.registry.ensureLoaded() }
    var query by remember { mutableStateOf("") }
    var filter by remember { mutableStateOf(PluginFilter.ALL) }
    var custom by remember { mutableStateOf(false) }
    val plugins = model.registry.value
    val installed =
        model.installed.value
            .orEmpty()
            .map { it.name }
            .toSet()
    val needle = query.trim().lowercase()
    val visible =
        plugins.orEmpty().filter { plugin ->
            (filter == PluginFilter.ALL || (filter == PluginFilter.OFFICIAL) == plugin.official) &&
                (needle.isEmpty() || plugin.name.lowercase().contains(needle) || plugin.homepage.contains(needle))
        }
    val updatingAll = model.tasks.activeTask(VfoxOperation.UPDATE_ALL_PLUGINS)

    Pane {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(260.dp),
            contentPadding = PaddingValues(start = Gaps.xl, end = Gaps.xl, bottom = Gaps.xl),
            horizontalArrangement = Arrangement.spacedBy(Gaps.m),
            verticalArrangement = Arrangement.spacedBy(Gaps.m),
        ) {
            item(span = { GridItemSpan(maxLineSpan) }) {
                Column {
                    PageHeader(t("pluginsTitle"), subtitle = t("pluginsSubtitle"), inset = false) {
                        if (updatingAll == null) {
                            TooltipIconButton("update", t("pluginsUpdateAll"), enabled = installed.isNotEmpty()) {
                                model.run(model.cli.updateAllPlugins())
                            }
                        } else {
                            val label = t("pluginsUpdateAll")
                            LoadingIndicator(Modifier.size(40.dp).semantics { contentDescription = label })
                        }
                        TooltipIconButton("refresh", t("actionRefresh"), enabled = !model.registry.isLoading) { model.registry.load() }
                        FilledTonalButton(onClick = { custom = true }) { IconLabel("add", t("pluginsAddCustom")) }
                    }
                    BoxWithConstraints {
                        val searchWidth = minOf(420.dp, maxWidth)
                        FlowRow(
                            Modifier.padding(bottom = Gaps.l),
                            horizontalArrangement = Arrangement.spacedBy(Gaps.m),
                            verticalArrangement = Arrangement.spacedBy(Gaps.s),
                            itemVerticalAlignment = Alignment.CenterVertically,
                        ) {
                            SearchField(query, t("pluginsSearchHint"), Modifier.width(searchWidth)) { query = it }
                            ChoiceGroup(
                                listOf(
                                    Choice(PluginFilter.ALL, t("pluginsFilterAll")),
                                    Choice(PluginFilter.OFFICIAL, t("pluginsFilterOfficial"), "verified"),
                                    Choice(PluginFilter.COMMUNITY, t("pluginsFilterCommunity")),
                                ),
                                filter,
                            ) { filter = it }
                        }
                    }
                }
            }
            when {
                plugins == null -> {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        val error = model.registry.error
                        Box(Modifier.height(360.dp)) {
                            if (error == null) LoadingMessage() else ErrorMessage(t("pluginsLoadError"), error) { model.registry.load() }
                        }
                    }
                }

                visible.isEmpty() -> {
                    item(
                        span = { GridItemSpan(maxLineSpan) },
                    ) { Box(Modifier.height(360.dp)) { StatusMessage("notFound", t("pluginsNoMatch")) } }
                }

                else -> {
                    items(visible, key = { it.name }) { plugin -> PluginCard(plugin, plugin.name in installed) }
                }
            }
        }
    }
    if (custom) {
        CustomPluginDialog(onDismiss = { custom = false }) { source, alias ->
            context.scope.launch { model.addPluginFromSource(source, alias) }
        }
    }
}

@Composable
private fun PluginCard(
    plugin: AvailablePlugin,
    installed: Boolean,
) {
    val context = app
    val model = context.model
    val task = model.tasks.activeTask(VfoxOperation.ADD_PLUGIN, plugin.name)
    val colors = MaterialTheme.colorScheme
    Surface(
        shape = RoundedCornerShape(Corners.largeIncreased),
        color = if (installed) colors.surfaceContainerHighest else colors.surfaceContainer,
        modifier = Modifier.testTag("plugin:${plugin.name}").fillMaxWidth().height(152.dp),
    ) {
        Column(Modifier.padding(Gaps.l)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.m)) {
                Avatar(plugin.name, size = 44.dp)
                Column(Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.xs)) {
                        Text(
                            plugin.name,
                            style = MaterialTheme.typography.titleMediumEmphasized,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f, fill = false),
                        )
                        if (plugin.official) {
                            val official = t("pluginsOfficial")
                            TooltipBox(
                                positionProvider = TooltipDefaults.rememberTooltipPositionProvider(TooltipAnchorPosition.Above),
                                tooltip = { PlainTooltip { Text(official) } },
                                state = rememberTooltipState(),
                            ) {
                                Box(Modifier.semantics { contentDescription = official }) {
                                    Icon("verified", filled = true, size = 18.dp, tint = colors.primary)
                                }
                            }
                        }
                    }
                    Text(
                        repositoryPath(plugin.homepage),
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Box(Modifier.weight(1f))
            Row(verticalAlignment = Alignment.CenterVertically) {
                TooltipIconButton("openInNew", t("actionOpenHomepage")) { context.services.openUrl(plugin.homepage) }
                Box(Modifier.weight(1f))
                when {
                    task != null -> {
                        val label = t("loading")
                        LoadingIndicator(Modifier.size(40.dp).semantics { contentDescription = label })
                    }

                    installed -> {
                        TextButton(onClick = { context.navigation.showSdk(plugin.name) }) { IconLabel("check", t("pluginsAdded")) }
                    }

                    else -> {
                        FilledTonalButton(onClick = { model.run(model.cli.addPlugin(plugin.name)) }) { IconLabel("add", t("pluginsAdd")) }
                    }
                }
            }
        }
    }
}

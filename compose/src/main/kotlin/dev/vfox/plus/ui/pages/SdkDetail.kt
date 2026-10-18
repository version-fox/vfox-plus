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

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.LinearWavyProgressIndicator
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SplitButtonDefaults
import androidx.compose.material3.SplitButtonLayout
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.vfox.plus.core.AvailableVersion
import dev.vfox.plus.core.InstalledSdk
import dev.vfox.plus.core.UseScope
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.state.SdkDetailController
import dev.vfox.plus.state.SdkTab
import dev.vfox.plus.state.TaskStatus
import dev.vfox.plus.state.VersionFilter
import dev.vfox.plus.state.VfoxTask
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.AppMenu
import dev.vfox.plus.ui.components.Avatar
import dev.vfox.plus.ui.components.Choice
import dev.vfox.plus.ui.components.ChoiceGroup
import dev.vfox.plus.ui.components.ConfirmDialog
import dev.vfox.plus.ui.components.ErrorMessage
import dev.vfox.plus.ui.components.LoadingMessage
import dev.vfox.plus.ui.components.ProjectDirectoryDialog
import dev.vfox.plus.ui.components.SegmentedRow
import dev.vfox.plus.ui.components.StatusMessage
import dev.vfox.plus.ui.components.Tag
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.t
import dev.vfox.plus.ui.taskTitle
import kotlinx.coroutines.launch

/** A dialog that the detail shows, if any. */
private sealed interface DetailDialog {
    data object RemovePlugin : DetailDialog

    data class Uninstall(
        val version: String,
        val current: Boolean,
    ) : DetailDialog

    data class UseInProject(
        val version: String,
    ) : DetailDialog
}

/** Everything about one installed plugin: metadata, installed and installable versions. */
@Composable
fun SdkDetail(
    name: String,
    back: Boolean,
) {
    val context = app
    val model = context.model
    val navigation = context.navigation
    val controller = remember(name) { model.sdkDetail(name) }
    DisposableEffect(controller) {
        navigation.shownDetail = controller
        val stop =
            model.tasks.onFinished { task ->
                if (task.command.sdk == name && task.command.operation == VfoxOperation.UPDATE_PLUGIN) controller.info.load()
            }
        onDispose {
            stop()
            if (navigation.shownDetail === controller) navigation.shownDetail = null
        }
    }
    LaunchedEffect(controller) {
        controller.info.load()
        navigation.requestedTab?.let {
            controller.showTab(it)
            navigation.requestedTab = null
        }
    }
    var dialog by remember { mutableStateOf<DetailDialog?>(null) }
    val sdk = model.installedSdk(name) ?: InstalledSdk(name, emptyList(), null)

    BoxWithConstraints(Modifier.fillMaxSize()) {
        val narrow = maxWidth < 520.dp
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = Gaps.xl)) {
            item(key = "header") { Header(sdk, controller, back, narrow) { dialog = DetailDialog.RemovePlugin } }
            item(key = "tabs") {
                ChoiceGroup(
                    listOf(
                        Choice(SdkTab.INSTALLED, "${t("sdkTabInstalled")} (${sdk.versions.size})", "versions"),
                        Choice(SdkTab.AVAILABLE, t("sdkTabAvailable"), "install"),
                    ),
                    controller.tab,
                    Modifier.padding(start = Gaps.xl, end = Gaps.xl, bottom = Gaps.l),
                ) { controller.showTab(it) }
            }
            if (controller.tab == SdkTab.INSTALLED) {
                installedVersions(sdk, onUninstall = { version, current -> dialog = DetailDialog.Uninstall(version, current) }) {
                    dialog = DetailDialog.UseInProject(it)
                }
            } else {
                availableVersions(sdk, controller)
            }
        }
    }

    when (val shown = dialog) {
        DetailDialog.RemovePlugin -> {
            ConfirmDialog(
                title = t("sdkRemoveTitle", "name" to name),
                body = t("sdkRemoveBody", "count" to sdk.versions.size),
                confirmLabel = t("sdkRemoveConfirm"),
                icon = "removePlugin",
                destructive = true,
                onDismiss = { dialog = null },
            ) {
                model.run(model.cli.removePlugin(name))
                navigation.showSdk(null)
            }
        }

        is DetailDialog.Uninstall -> {
            ConfirmDialog(
                title = t("versionUninstallTitle", "name" to name, "version" to shown.version),
                body = if (shown.current) t("versionUninstallCurrentBody") else t("versionUninstallBody"),
                confirmLabel = t("versionUninstall"),
                icon = "uninstall",
                destructive = true,
                onDismiss = { dialog = null },
            ) { model.run(model.cli.uninstall(name, shown.version)) }
        }

        is DetailDialog.UseInProject -> {
            ProjectDirectoryDialog(name, shown.version, onDismiss = { dialog = null }) { directory ->
                model.run(model.cli.use(name, shown.version, UseScope.PROJECT, directory))
            }
        }

        null -> {}
    }
}

@Composable
private fun Header(
    sdk: InstalledSdk,
    controller: SdkDetailController,
    back: Boolean,
    narrow: Boolean,
    onRemove: () -> Unit,
) {
    val context = app
    val model = context.model
    val info = controller.info.value
    val updating = model.tasks.activeTask(VfoxOperation.UPDATE_PLUGIN, sdk.name)
    var menu by remember { mutableStateOf(false) }
    val backButton: @Composable () -> Unit = {
        if (back) TooltipIconButton("back", t("actionClose")) { context.navigation.showSdk(null) }
    }
    val actions: @Composable () -> Unit = {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (updating == null) {
                TooltipIconButton("update", t("sdkUpdatePlugin")) { model.run(model.cli.updatePlugin(sdk.name)) }
            } else {
                val label = t("loading")
                LoadingIndicator(Modifier.size(40.dp).semantics { contentDescription = label })
            }
            Box {
                TooltipIconButton("more", t("sdkMoreActions")) { menu = true }
                AppMenu(expanded = menu, onDismissRequest = { menu = false }) {
                    if (info != null && info.homepage.isNotEmpty()) {
                        DropdownMenuItem(
                            text = { Text(t("actionOpenHomepage")) },
                            leadingIcon = { Icon("openInNew") },
                            onClick = {
                                menu = false
                                context.services.openUrl(info.homepage)
                            },
                        )
                    }
                    DropdownMenuItem(
                        text = { Text(t("sdkRemovePlugin"), color = MaterialTheme.colorScheme.error) },
                        leadingIcon = { Icon("removePlugin", tint = MaterialTheme.colorScheme.error) },
                        onClick = {
                            menu = false
                            onRemove()
                        },
                    )
                }
            }
        }
    }
    val text: @Composable () -> Unit = {
        Column {
            Text(sdk.name, style = MaterialTheme.typography.headlineMediumEmphasized, maxLines = 1, overflow = TextOverflow.Ellipsis)
            FlowRow(
                Modifier.padding(top = Gaps.xs),
                horizontalArrangement = Arrangement.spacedBy(Gaps.s),
                verticalArrangement = Arrangement.spacedBy(Gaps.xs),
            ) {
                Tag(
                    if (sdk.current == null) t("sdksNoCurrent") else t("sdksCurrent", "version" to sdk.current),
                    emphasized = sdk.current != null,
                    icon = if (sdk.current == null) null else "current",
                )
                if (info != null && info.version.isNotEmpty()) Tag(t("sdkPluginVersion", "version" to info.version))
            }
            if (info != null && info.description.isNotEmpty()) {
                Text(
                    info.description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(top = Gaps.s),
                )
            }
        }
    }
    if (narrow) {
        Column(
            Modifier.padding(start = Gaps.s, top = Gaps.xl, end = Gaps.l, bottom = Gaps.xl),
            verticalArrangement = Arrangement.spacedBy(Gaps.s),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                backButton()
                Box(Modifier.weight(1f))
                actions()
            }
            Row(Modifier.padding(start = Gaps.s), horizontalArrangement = Arrangement.spacedBy(Gaps.l)) {
                Avatar(sdk.name, size = 56.dp)
                text()
            }
        }
    } else {
        Row(
            Modifier.padding(start = Gaps.s, top = Gaps.xl, end = Gaps.l, bottom = Gaps.xl),
            horizontalArrangement = Arrangement.spacedBy(Gaps.l),
        ) {
            if (back) backButton() else Box(Modifier.width(Gaps.s))
            Avatar(sdk.name, size = 72.dp)
            Box(Modifier.weight(1f)) { text() }
            actions()
        }
    }
}

private fun LazyListScope.installedVersions(
    sdk: InstalledSdk,
    onUninstall: (version: String, current: Boolean) -> Unit,
    onUseInProject: (String) -> Unit,
) {
    if (sdk.versions.isEmpty()) {
        item(key = "empty") {
            val controller = app.navigation.shownDetail
            StatusMessage("install", t("sdkInstalledEmpty", "name" to sdk.name), shape = MaterialShapes.Pill) {
                androidx.compose.material3.Button(onClick = { controller?.showTab(SdkTab.AVAILABLE) }) {
                    IconLabel("install", t("sdkBrowseVersions"))
                }
            }
        }
        return
    }
    itemsIndexed(sdk.versions, key = { _, version -> "installed:$version" }) { index, version ->
        InstalledVersion(sdk, version, index, onUninstall, onUseInProject)
    }
}

@Composable
private fun InstalledVersion(
    sdk: InstalledSdk,
    version: String,
    index: Int,
    onUninstall: (version: String, current: Boolean) -> Unit,
    onUseInProject: (String) -> Unit,
) {
    val context = app
    val model = context.model
    val current = sdk.current == version
    val task = model.tasks.activeTask(sdk = sdk.name, version = version)
    val copied = t("actionCopied")
    var useMenu by remember { mutableStateOf(false) }
    var moreMenu by remember { mutableStateOf(false) }
    Box(Modifier.testTag("installed:$version").padding(horizontal = Gaps.xl).padding(bottom = Gaps.hairline)) {
        SegmentedRow(index, sdk.versions.size) {
            Icon(
                if (current) "current" else "versions",
                filled = current,
                tint = if (current) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                version,
                style = if (current) MaterialTheme.typography.bodyLargeEmphasized else MaterialTheme.typography.bodyLarge,
                modifier = Modifier.weight(1f),
            )
            if (task != null) {
                val label = context.localizations.taskTitle(task.command)
                LoadingIndicator(Modifier.size(40.dp).semantics { contentDescription = label })
                return@SegmentedRow
            }
            if (current) {
                Tag(t("versionInUse"), emphasized = true, icon = "current")
            } else {
                Box {
                    val rotation by animateFloatAsState(if (useMenu) 180f else 0f, MaterialTheme.motionScheme.fastSpatialSpec())
                    SplitButtonLayout(
                        leadingButton = {
                            SplitButtonDefaults.TonalLeadingButton(onClick = { model.run(model.cli.use(sdk.name, version)) }) {
                                IconLabel("check", t("versionUse"))
                            }
                        },
                        trailingButton = {
                            val label = t("versionUseOptions")
                            SplitButtonDefaults.TonalTrailingButton(
                                checked = useMenu,
                                onCheckedChange = { useMenu = it },
                                modifier = Modifier.semantics { contentDescription = label },
                            ) { Icon("expandMore", Modifier.rotate(rotation), size = SplitButtonDefaults.TrailingIconSize) }
                        },
                    )
                    AppMenu(expanded = useMenu, onDismissRequest = { useMenu = false }) {
                        DropdownMenuItem(
                            text = { Text(t("versionUseGlobally")) },
                            leadingIcon = { Icon("globalScope") },
                            onClick = {
                                useMenu = false
                                model.run(model.cli.use(sdk.name, version))
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(t("versionUseInProject")) },
                            leadingIcon = { Icon("projectScope") },
                            onClick = {
                                useMenu = false
                                onUseInProject(version)
                            },
                        )
                    }
                }
            }
            Box {
                TooltipIconButton("more", t("versionMoreActions", "version" to version)) { moreMenu = true }
                AppMenu(expanded = moreMenu, onDismissRequest = { moreMenu = false }) {
                    if (current) {
                        DropdownMenuItem(
                            text = { Text(t("versionUseInProject")) },
                            leadingIcon = { Icon("projectScope") },
                            onClick = {
                                moreMenu = false
                                onUseInProject(version)
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(t("versionUnuse")) },
                            leadingIcon = { Icon("cancel") },
                            onClick = {
                                moreMenu = false
                                model.run(model.cli.unuse(sdk.name))
                            },
                        )
                    }
                    DropdownMenuItem(
                        text = { Text(t("versionCopyPath")) },
                        leadingIcon = { Icon("copy") },
                        onClick = {
                            moreMenu = false
                            context.scope.launch {
                                runCatching { model.cli.installPath(sdk.name, version) }.getOrNull()?.let {
                                    context.services.copyText(it)
                                    context.snackbars.showSnackbar(copied)
                                }
                            }
                        },
                    )
                    DropdownMenuItem(
                        text = { Text(t("versionUninstall"), color = MaterialTheme.colorScheme.error) },
                        leadingIcon = { Icon("uninstall", tint = MaterialTheme.colorScheme.error) },
                        onClick = {
                            moreMenu = false
                            onUninstall(version, current)
                        },
                    )
                }
            }
        }
    }
}

private fun LazyListScope.availableVersions(
    sdk: InstalledSdk,
    controller: SdkDetailController,
) {
    val versions = controller.versions
    if (!versions.hasValue) {
        item(key = "loading") {
            val error = versions.error
            Box(Modifier.fillMaxWidth().padding(top = Gaps.xxl)) {
                if (error == null) {
                    LoadingMessage(label = t("availableLoading"))
                } else {
                    ErrorMessage(t("availableError", "name" to sdk.name), error) { versions.load() }
                }
            }
        }
        return
    }
    val installed = sdk.versions.toSet()
    val visible = controller.visibleVersions(installed)
    item(key = "toolbar") {
        FlowRow(
            Modifier.padding(start = Gaps.xl, end = Gaps.xl, bottom = Gaps.m),
            horizontalArrangement = Arrangement.spacedBy(Gaps.m),
            verticalArrangement = Arrangement.spacedBy(Gaps.s),
            itemVerticalAlignment = Alignment.CenterVertically,
        ) {
            FilterField(controller.query, t("availableSearchHint"), Modifier.width(280.dp)) { controller.query = it }
            ChoiceGroup(
                listOf(
                    Choice(VersionFilter.ALL, t("availableFilterAll")),
                    Choice(VersionFilter.WITH_NOTE, t("availableFilterTagged")),
                    Choice(VersionFilter.INSTALLED, t("availableFilterInstalled")),
                ),
                controller.filter,
            ) { controller.filter = it }
            Text(
                t("availableCount", "count" to visible.size),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
    if (visible.isEmpty()) {
        item(key = "none") { StatusMessage("notFound", t("availableEmpty")) }
        return
    }
    itemsIndexed(visible, key = { _, version -> "available:${version.version}" }) { index, version ->
        Box(Modifier.testTag("available:${version.version}").padding(horizontal = Gaps.xl).padding(bottom = Gaps.hairline)) {
            AvailableVersionRow(sdk.name, version, version.version in installed, index, visible.size)
        }
    }
}

@Composable
private fun AvailableVersionRow(
    name: String,
    version: AvailableVersion,
    installed: Boolean,
    index: Int,
    count: Int,
) {
    val model = app.model
    val task = model.tasks.activeTask(VfoxOperation.INSTALL, name, version.version)
    SegmentedRow(index, count) {
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.s)) {
                Text(version.version, style = MaterialTheme.typography.bodyLarge)
                version.note?.let { Tag(it, emphasized = true) }
            }
            val additions = version.additions.joinToString(" · ") { it.label }
            if (additions.isNotEmpty()) {
                Text(additions, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        when {
            task != null -> {
                InstallProgress(task)
            }

            installed -> {
                TextButton(onClick = {}, enabled = false) { IconLabel("check", t("versionInstalled")) }
            }

            else -> {
                FilledTonalButton(onClick = { model.run(model.cli.install(name, version.version)) }) {
                    IconLabel("install", t("versionInstall"))
                }
            }
        }
    }
}

/** The download progress of an installation, on the wavy progress indicator, with a cancel button. */
@Composable
fun InstallProgress(task: VfoxTask) {
    val context = app
    val progress = task.progress
    val label =
        when {
            progress != null -> t("taskDownloading", "percent" to Math.round(progress * 100).toInt())
            task.status == TaskStatus.QUEUED -> t("taskStatusQueued")
            else -> t("taskStatusRunning")
        }
    val title = context.localizations.taskTitle(task.command)
    Row(Modifier.width(220.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.xs)) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Gaps.xs)) {
            val indicator = Modifier.fillMaxWidth().semantics { contentDescription = title }
            if (progress == null) {
                LinearWavyProgressIndicator(indicator)
            } else {
                val animated by animateFloatAsState(progress.toFloat(), MaterialTheme.motionScheme.defaultSpatialSpec())
                LinearWavyProgressIndicator(progress = { animated }, modifier = indicator)
            }
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        TooltipIconButton("cancel", t("actionCancel")) { context.model.tasks.cancel(task) }
    }
}

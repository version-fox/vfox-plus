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

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.LinearWavyProgressIndicator
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.state.TaskStatus
import dev.vfox.plus.state.VfoxTask
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.PageHeader
import dev.vfox.plus.ui.components.Pane
import dev.vfox.plus.ui.components.SegmentedRow
import dev.vfox.plus.ui.components.StatusMessage
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.t
import dev.vfox.plus.ui.taskTitle
import kotlinx.coroutines.launch

/** Running, queued and finished operations with their output. */
@Composable
fun TasksPage() {
    val model = app.model
    val tasks = model.tasks.tasks
    Pane {
        Column {
            PageHeader(t("navTasks")) {
                TextButton(onClick = { model.tasks.clearFinished() }, enabled = model.tasks.hasFinished) {
                    IconLabel("succeeded", t("tasksClearFinished"))
                }
            }
            if (tasks.isEmpty()) {
                StatusMessage("tasks", t("tasksEmptyTitle"), body = t("tasksEmptyBody"), shape = MaterialShapes.Sunny)
                return@Column
            }
            LazyColumn(contentPadding = PaddingValues(start = Gaps.xl, end = Gaps.xl, bottom = Gaps.xl)) {
                itemsIndexed(tasks, key = { _, task -> task.id }) { index, task -> TaskItem(task, index, tasks.size) }
            }
        }
    }
}

@Composable
private fun TaskItem(
    task: VfoxTask,
    index: Int,
    count: Int,
) {
    val context = app
    val model = context.model
    var expanded by rememberSaveable(task.id) { mutableStateOf(false) }
    val status = task.status
    val progress = task.progress
    val colors = MaterialTheme.colorScheme
    val detail =
        when (status) {
            TaskStatus.QUEUED -> {
                t("taskStatusQueued")
            }

            TaskStatus.RUNNING -> {
                if (progress ==
                    null
                ) {
                    t("taskStatusRunning")
                } else {
                    t("taskDownloading", "percent" to Math.round(progress * 100).toInt())
                }
            }

            TaskStatus.SUCCEEDED -> {
                t("taskStatusSucceeded")
            }

            TaskStatus.FAILED -> {
                "${t("taskStatusFailed")} · ${task.errorMessage.orEmpty()}"
            }

            TaskStatus.CANCELLED -> {
                t("taskStatusCancelled")
            }
        }
    val command = "$ ${task.command.commandLine}"
    val text = (listOf(command) + task.log).joinToString("\n")
    val copied = t("actionCopied")
    Column(Modifier.testTag("task:${task.id}").padding(bottom = Gaps.hairline)) {
        SegmentedRow(index, count, roundBottom = !expanded, onClick = { expanded = !expanded }) {
            Box(Modifier.size(32.dp), contentAlignment = Alignment.Center) {
                when (status) {
                    TaskStatus.QUEUED -> {
                        Icon("queued", tint = colors.onSurfaceVariant)
                    }

                    TaskStatus.RUNNING -> {
                        val label = t("taskStatusRunning")
                        LoadingIndicator(Modifier.size(32.dp).semantics { contentDescription = label })
                    }

                    TaskStatus.SUCCEEDED -> {
                        Icon("succeeded", filled = true, tint = colors.primary)
                    }

                    TaskStatus.FAILED -> {
                        Icon("error", filled = true, tint = colors.error)
                    }

                    TaskStatus.CANCELLED -> {
                        Icon("cancel", tint = colors.onSurfaceVariant)
                    }
                }
            }
            Column(Modifier.weight(1f).padding(vertical = Gaps.xs)) {
                Text(context.localizations.taskTitle(task.command), style = MaterialTheme.typography.bodyLargeEmphasized)
                Text(
                    detail,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (status == TaskStatus.FAILED) colors.error else colors.onSurfaceVariant,
                    maxLines = 2,
                )
                if (status == TaskStatus.RUNNING && task.command.operation == VfoxOperation.INSTALL) {
                    val indicator = Modifier.padding(top = Gaps.s).widthIn(max = 480.dp).fillMaxWidth()
                    if (progress == null) {
                        LinearWavyProgressIndicator(indicator)
                    } else {
                        val animated by animateFloatAsState(progress.toFloat(), MaterialTheme.motionScheme.defaultSpatialSpec())
                        LinearWavyProgressIndicator(progress = { animated }, modifier = indicator)
                    }
                }
            }
            if (status.isActive) {
                TooltipIconButton("cancel", t("actionCancel")) { model.tasks.cancel(task) }
            } else if (status == TaskStatus.FAILED || status == TaskStatus.CANCELLED) {
                TooltipIconButton("retry", t("actionRetry")) { model.tasks.retry(task) }
            }
            val rotation by animateFloatAsState(if (expanded) 180f else 0f, MaterialTheme.motionScheme.fastSpatialSpec())
            TooltipIconButton(
                "expandMore",
                if (expanded) t("taskHideOutput") else t("taskShowOutput"),
                Modifier.rotate(rotation),
            ) { expanded = !expanded }
        }
        AnimatedVisibility(
            expanded,
            enter =
                expandVertically(MaterialTheme.motionScheme.defaultSpatialSpec()) + fadeIn(MaterialTheme.motionScheme.defaultEffectsSpec()),
            exit =
                shrinkVertically(
                    MaterialTheme.motionScheme.defaultSpatialSpec(),
                ) + fadeOut(MaterialTheme.motionScheme.fastEffectsSpec()),
        ) {
            // Joined to the task above it; rounded at the bottom when it ends the list.
            SegmentedRow(
                1,
                if (index ==
                    count - 1
                ) {
                    2
                } else {
                    3
                },
                Modifier.padding(top = Gaps.hairline),
                color = colors.surfaceContainerHighest,
            ) {
                val scroll = rememberScrollState()
                // Show the end of the output, where vfox writes.
                LaunchedEffect(task.log.size) { scroll.scrollTo(scroll.maxValue) }
                SelectionContainer(Modifier.weight(1f)) {
                    Text(
                        if (task.log.isEmpty()) "$command\n${t("taskNoOutput")}" else text,
                        fontFamily = FontFamily.Monospace,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.heightIn(max = 320.dp).verticalScroll(scroll).padding(vertical = Gaps.s),
                    )
                }
                TooltipIconButton("copy", t("taskCopyOutput")) {
                    context.services.copyText(text)
                    context.scope.launch { context.snackbars.showSnackbar(copied) }
                }
            }
        }
    }
}

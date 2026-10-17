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

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import dev.vfox.plus.core.isValidPluginName
import dev.vfox.plus.ui.Gaps
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.t
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.nio.file.Files
import java.nio.file.Path

/** The width of the fields of dialogs, which makes the dialogs 440 dp wide. */
private val DIALOG_CONTENT_WIDTH = 392.dp

/** Asks the user to confirm an action; [destructive] actions are confirmed in the error colour. */
@Composable
fun ConfirmDialog(
    title: String,
    body: String,
    confirmLabel: String,
    icon: String,
    destructive: Boolean = false,
    onDismiss: () -> Unit,
    onConfirm: () -> Unit,
) {
    val colors = MaterialTheme.colorScheme
    AlertDialog(
        onDismissRequest = onDismiss,
        icon = { Icon(icon, tint = if (destructive) colors.error else colors.secondary) },
        title = { Text(title) },
        text = { Text(body) },
        dismissButton = { TextButton(onClick = onDismiss) { Text(t("actionCancel")) } },
        confirmButton = {
            Button(
                onClick = {
                    onDismiss()
                    onConfirm()
                },
                colors =
                    if (destructive) {
                        ButtonDefaults.buttonColors(containerColor = colors.error, contentColor = colors.onError)
                    } else {
                        ButtonDefaults.buttonColors()
                    },
            ) { Text(confirmLabel) }
        },
    )
}

/** Asks for the project directory in which to pin [name] [version]. */
@Composable
fun ProjectDirectoryDialog(
    name: String,
    version: String,
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
) {
    val context = app
    val scope = rememberCoroutineScope()
    var directory by remember { mutableStateOf("") }
    var invalid by remember { mutableStateOf(false) }
    val invalidText = t("useProjectInvalid")
    val focus = remember { FocusRequester() }
    LaunchedEffect(focus) { focus.requestFocus() }
    val submit: () -> Unit = {
        scope.launch {
            val path = directory.trim()
            val valid = path.isNotEmpty() && withContext(Dispatchers.IO) { Files.isDirectory(Path.of(path)) }
            if (valid) {
                onDismiss()
                onConfirm(path)
            } else {
                invalid = true
            }
        }
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        icon = { Icon("projectScope", tint = MaterialTheme.colorScheme.secondary) },
        title = { Text(t("useProjectTitle", "name" to name, "version" to version)) },
        text = {
            Column(Modifier.width(DIALOG_CONTENT_WIDTH), verticalArrangement = Arrangement.spacedBy(Gaps.l)) {
                Text(t("useProjectBody"))
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.s)) {
                    OutlinedTextField(
                        value = directory,
                        onValueChange = {
                            directory = it
                            invalid = false
                        },
                        label = { Text(t("useProjectDirectory")) },
                        isError = invalid,
                        supportingText = if (invalid) ({ Text(invalidText) }) else null,
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = { submit() }),
                        modifier = Modifier.weight(1f).focusRequester(focus),
                    )
                    TooltipIconButton("folder", t("actionBrowse"), tonal = true) {
                        scope.launch {
                            val home = context.model.environment["HOME"]
                            context.services.pickDirectory(directory.trim().ifEmpty { home })?.let {
                                directory = it
                                invalid = false
                            }
                        }
                    }
                }
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text(t("actionCancel")) } },
        confirmButton = { Button(onClick = submit) { Text(t("useProjectConfirm")) } },
    )
}

/** Asks for the source and the optional alias of a plugin that is not in the registry. */
@Composable
fun CustomPluginDialog(
    onDismiss: () -> Unit,
    onConfirm: (source: String, alias: String?) -> Unit,
) {
    val context = app
    val scope = rememberCoroutineScope()
    var source by remember { mutableStateOf("") }
    var alias by remember { mutableStateOf("") }
    var sourceError by remember { mutableStateOf<String?>(null) }
    var aliasError by remember { mutableStateOf<String?>(null) }
    val focus = remember { FocusRequester() }
    LaunchedEffect(focus) { focus.requestFocus() }
    val required = t("pluginsCustomRequired")
    val invalidName = t("pluginsCustomInvalidName")
    val fileType = t("pluginsCustomFileType")
    val submit: () -> Unit = {
        sourceError = if (source.isBlank()) required else null
        aliasError = if (alias.isNotBlank() && !isValidPluginName(alias.trim())) invalidName else null
        if (sourceError == null && aliasError == null) {
            onDismiss()
            onConfirm(source.trim(), alias.trim().ifEmpty { null })
        }
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        icon = { Icon("plugins", tint = MaterialTheme.colorScheme.secondary) },
        title = { Text(t("pluginsCustomTitle")) },
        text = {
            Column(Modifier.width(DIALOG_CONTENT_WIDTH), verticalArrangement = Arrangement.spacedBy(Gaps.m)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Gaps.s)) {
                    OutlinedTextField(
                        value = source,
                        onValueChange = {
                            source = it
                            sourceError = null
                        },
                        label = { Text(t("pluginsCustomSource")) },
                        supportingText = { Text(sourceError ?: t("pluginsCustomSourceHelp")) },
                        isError = sourceError != null,
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = { submit() }),
                        modifier = Modifier.weight(1f).focusRequester(focus),
                    )
                    TooltipIconButton("folder", t("actionBrowse"), tonal = true) {
                        scope.launch {
                            context.services.pickFile(context.model.environment["HOME"], fileType, listOf("zip", "lua"))?.let {
                                source = it
                                sourceError = null
                            }
                        }
                    }
                }
                OutlinedTextField(
                    value = alias,
                    onValueChange = {
                        alias = it
                        aliasError = null
                    },
                    label = { Text(t("pluginsCustomAlias")) },
                    supportingText = { Text(aliasError ?: t("pluginsCustomAliasHelp")) },
                    isError = aliasError != null,
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { submit() }),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text(t("actionCancel")) } },
        confirmButton = { Button(onClick = submit) { Text(t("pluginsCustomConfirm")) } },
    )
}

/** Shows a long, monospaced text, such as the open-source licenses. */
@Composable
fun TextDialog(
    title: String,
    text: String,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            SelectionContainer {
                Text(
                    text,
                    fontFamily = FontFamily.Monospace,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.widthIn(max = 720.dp).heightIn(max = 480.dp).verticalScroll(rememberScrollState()),
                )
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text(t("actionClose")) } },
    )
}

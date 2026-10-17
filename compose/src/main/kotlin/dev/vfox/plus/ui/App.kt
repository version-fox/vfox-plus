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

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import dev.vfox.plus.core.ThemePreference
import dev.vfox.plus.core.i18n.Localizations
import dev.vfox.plus.core.i18n.systemLocales
import dev.vfox.plus.state.AppModel
import dev.vfox.plus.state.TaskStatus
import kotlinx.coroutines.launch

/**
 * The root of the user interface: the theme of the settings and the desktop,
 * the language, the shell and the snack bars that report finished tasks.
 */
@Composable
fun VfoxPlusApp(
    model: AppModel,
    navigation: Navigation,
    services: PlatformServices,
) {
    val settings = model.settings.settings
    var accent by remember { mutableStateOf<Long?>(null) }
    var desktopDark by remember { mutableStateOf<Boolean?>(null) }
    LaunchedEffect(services) {
        accent = services.accentColor()
        desktopDark = services.prefersDark()
    }
    val dark =
        when (settings.theme) {
            ThemePreference.DARK -> true
            ThemePreference.LIGHT -> false
            ThemePreference.SYSTEM -> desktopDark ?: isSystemInDarkTheme()
        }
    val seed = accent?.takeIf { settings.useSystemColor } ?: settings.seedColor
    val localizations = remember(settings.language) { Localizations.forPreference(settings.language, systemLocales(model.environment)) }
    val snackbars = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    val context = remember(localizations) { AppContext(model, navigation, services, localizations, snackbars, scope) }

    VfoxPlusTheme(seed, dark) {
        CompositionLocalProvider(LocalApp provides context) {
            DisposableEffect(context) {
                val stop =
                    model.tasks.onFinished { task ->
                        if (task.status == TaskStatus.CANCELLED) return@onFinished
                        val title = localizations.taskTitle(task.command)
                        scope.launch {
                            if (task.status == TaskStatus.FAILED) {
                                // Failures stay until dismissed, so that they can be read in full.
                                val result =
                                    snackbars.showSnackbar(
                                        "${localizations.t("taskFailed", "title" to title)}\n${task.errorMessage.orEmpty()}".trim(),
                                        actionLabel = localizations.t("taskView"),
                                        withDismissAction = true,
                                        duration = SnackbarDuration.Indefinite,
                                    )
                                if (result == SnackbarResult.ActionPerformed) navigation.go(Destination.TASKS)
                            } else if (navigation.destination != Destination.TASKS) {
                                snackbars.showSnackbar(localizations.t("taskSucceeded", "title" to title))
                            }
                        }
                    }
                onDispose { stop() }
            }
            Scaffold(
                containerColor = MaterialTheme.colorScheme.surface,
                snackbarHost = { SnackbarHost(snackbars) },
            ) { Shell() }
        }
    }
}

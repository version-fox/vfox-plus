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

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialShapes
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import dev.vfox.plus.state.VfoxAvailability
import dev.vfox.plus.ui.AppContext
import dev.vfox.plus.ui.Icon
import dev.vfox.plus.ui.Links
import dev.vfox.plus.ui.app
import dev.vfox.plus.ui.components.LoadingMessage
import dev.vfox.plus.ui.components.Pane
import dev.vfox.plus.ui.components.StatusMessage
import dev.vfox.plus.ui.t
import kotlinx.coroutines.launch

/** Lets the user choose the vfox executable and stores the choice. */
fun AppContext.locateVfox() {
    scope.launch {
        services.pickFile("/usr/local/bin")?.let { path -> model.updateSettings { it.copy(vfoxExecutable = path) } }
    }
}

/** A button with an icon before its label. */
@Composable
internal fun IconLabel(
    icon: String,
    label: String,
) {
    Icon(icon, size = ButtonDefaults.IconSize)
    Box(Modifier.size(ButtonDefaults.IconSpacing))
    Text(label)
}

/** Shown instead of the pages while vfox is missing or broken. */
@Composable
fun UnavailablePage() {
    val context = app
    val vfox = context.model.vfox
    Pane {
        when (vfox.availability) {
            VfoxAvailability.UNKNOWN, VfoxAvailability.CHECKING, VfoxAvailability.READY -> {
                LoadingMessage(label = t("vfoxCheckingTitle"))
            }

            else -> {
                val missing = vfox.availability == VfoxAvailability.NOT_FOUND
                StatusMessage(
                    icon = if (missing) "notFound" else "warning",
                    heading = if (missing) t("vfoxMissingTitle") else t("vfoxBrokenTitle"),
                    body = if (missing) t("vfoxMissingBody") else t("vfoxBrokenBody", "path" to (vfox.executable ?: "vfox")),
                    shape = if (missing) MaterialShapes.Clover4Leaf else MaterialShapes.SoftBurst,
                    error = !missing,
                ) {
                    Button(onClick = { context.services.openUrl(Links.VFOX_INSTALL) }) { IconLabel("openInNew", t("vfoxInstallGuide")) }
                    FilledTonalButton(onClick = { context.locateVfox() }) { IconLabel("folder", t("vfoxLocate")) }
                    OutlinedButton(
                        onClick = { context.scope.launch { context.model.applySettings() } },
                    ) { IconLabel("retry", t("actionRetry")) }
                }
            }
        }
    }
}

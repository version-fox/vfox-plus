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

import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import dev.vfox.plus.core.VfoxCommand
import dev.vfox.plus.core.VfoxOperation
import dev.vfox.plus.core.i18n.Localizations
import dev.vfox.plus.state.AppModel
import dev.vfox.plus.state.SdkDetailController
import dev.vfox.plus.state.SdkTab
import kotlinx.coroutines.CoroutineScope

/** The top-level destinations, in the order of the navigation. */
enum class Destination(
    val icon: String,
    val label: String,
) {
    HOME("home", "navHome"),
    SDKS("sdks", "navSdks"),
    PLUGINS("plugins", "navPlugins"),
    TASKS("tasks", "navTasks"),
    SETTINGS("settings", "navSettings"),
}

/** Navigation state of the application shell. */
class Navigation {
    /** The visible destination. */
    var destination: Destination by mutableStateOf(Destination.HOME)
        private set

    /** The SDK shown on the SDKs page, if any. */
    var selectedSdk: String? by mutableStateOf(null)
        private set

    /** A tab the detail of [selectedSdk] should show when it opens, if any. */
    var requestedTab: SdkTab? by mutableStateOf(null)

    /** The state of the SDK detail that is shown, for the measure scenario. */
    var shownDetail: SdkDetailController? by mutableStateOf(null)

    fun go(target: Destination) {
        destination = target
    }

    /** Shows the SDKs page with [name] selected, or with no selection. */
    fun showSdk(
        name: String?,
        tab: SdkTab? = null,
    ) {
        destination = Destination.SDKS
        selectedSdk = name
        requestedTab = tab
    }
}

/** The pieces of the running application. */
class AppContext(
    val model: AppModel,
    val navigation: Navigation,
    val services: PlatformServices,
    val localizations: Localizations,
    val snackbars: SnackbarHostState,
    /** The scope of the user interface, for work that outlives a click. */
    val scope: CoroutineScope,
)

/** The running application, provided by the root. */
val LocalApp = staticCompositionLocalOf<AppContext> { error("No application") }

/** The running application. */
val app: AppContext
    @Composable @ReadOnlyComposable
    get() = LocalApp.current

/** The message [key] of the current language, formatted with [arguments]. */
@Composable
@ReadOnlyComposable
fun t(
    key: String,
    vararg arguments: Pair<String, Any>,
): String = LocalApp.current.localizations.t(key, *arguments)

/** The localized title of a task that runs [command]. */
fun Localizations.taskTitle(command: VfoxCommand): String {
    val subject = command.subject.orEmpty()
    return when (command.operation) {
        VfoxOperation.INSTALL -> t("taskInstall", "subject" to subject)
        VfoxOperation.UNINSTALL -> t("taskUninstall", "subject" to subject)
        VfoxOperation.USE -> t("taskUse", "subject" to subject)
        VfoxOperation.UNUSE -> t("taskUnuse", "subject" to subject)
        VfoxOperation.ADD_PLUGIN -> if (subject.isEmpty()) t("taskAddPluginFromSource") else t("taskAddPlugin", "subject" to subject)
        VfoxOperation.REMOVE_PLUGIN -> t("taskRemovePlugin", "subject" to subject)
        VfoxOperation.UPDATE_PLUGIN -> t("taskUpdatePlugin", "subject" to subject)
        VfoxOperation.UPDATE_ALL_PLUGINS -> t("taskUpdateAllPlugins")
    }
}

/** Links that the user interface opens. */
object Links {
    const val VFOX_INSTALL = "https://vfox.dev/guides/quick-start.html"
    const val VFOX_DOCS = "https://vfox.dev/"
    const val REPOSITORY = "https://github.com/version-fox/vfox-plus"
}

/** The version of the application; the build checks it against build.gradle.kts. */
const val APP_VERSION = "0.1.0"

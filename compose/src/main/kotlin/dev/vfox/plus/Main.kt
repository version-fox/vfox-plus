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

package dev.vfox.plus

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.graphics.asComposeImageBitmap
import androidx.compose.ui.graphics.painter.BitmapPainter
import androidx.compose.ui.graphics.painter.Painter
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import dev.vfox.plus.core.JsonFileSettingsStore
import dev.vfox.plus.core.JvmProcessRunner
import dev.vfox.plus.core.settingsPath
import dev.vfox.plus.state.AppModel
import dev.vfox.plus.state.SdkTab
import dev.vfox.plus.ui.DesktopPlatformServices
import dev.vfox.plus.ui.Destination
import dev.vfox.plus.ui.Navigation
import dev.vfox.plus.ui.VfoxPlusApp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.first
import org.jetbrains.skia.Bitmap
import org.jetbrains.skia.Canvas
import org.jetbrains.skia.Data
import org.jetbrains.skia.svg.SVGDOM
import java.awt.Dimension
import java.awt.Toolkit
import kotlin.system.exitProcess

/** Renders the SVG application icon for the window, with the SVG renderer of Skia. */
private fun windowIcon(svg: ByteArray): Painter {
    val size = 256
    val document = SVGDOM(Data.makeFromBytes(svg))
    document.setContainerSize(size.toFloat(), size.toFloat())
    val bitmap = Bitmap().apply { allocN32Pixels(size, size) }
    Canvas(bitmap).use { document.render(it) }
    return BitmapPainter(bitmap.asComposeImageBitmap())
}

/** The application ID, also the name of the desktop entry. */
const val APP_ID = "dev.vfox.VfoxPlus"

/**
 * Names the window class after the application ID, so that the window matches
 * the desktop entry (StartupWMClass). AWT takes it from the main class and has
 * no API for it; build.gradle.kts opens the field to the application.
 */
private fun nameWindowClass() {
    runCatching {
        val toolkit = Toolkit.getDefaultToolkit()
        toolkit.javaClass.getDeclaredField("awtAppClassName").apply {
            isAccessible = true
            set(toolkit, APP_ID)
        }
    }
}

/**
 * The end-to-end scenario "measure" of shared/tools/measure.sh: shows the
 * Home page and the available versions of Node.js, reports each point and
 * exits.
 */
@Composable
private fun MeasureScenario(
    model: AppModel,
    navigation: Navigation,
) {
    LaunchedEffect(Unit) {
        fun report(message: String) {
            println("e2e: $message")
            System.out.flush()
        }
        report("scenario measure started")
        snapshotFlow { model.installed.hasValue }.first { it }
        delay(3000)
        report("MEASURE home")
        delay(5000)
        navigation.showSdk("nodejs", SdkTab.AVAILABLE)
        snapshotFlow { navigation.shownDetail?.versions?.hasValue == true }.first { it }
        delay(3000)
        report("MEASURE nodejs")
        delay(5000)
        navigation.go(Destination.HOME)
        report("scenario measure passed")
        exitProcess(0)
    }
}

fun main() {
    nameWindowClass()
    val environment = System.getenv()
    // The state changes on the event dispatch thread of Swing, where Compose runs.
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    val model = AppModel(JvmProcessRunner(), JsonFileSettingsStore(settingsPath(environment)), environment, scope)
    val navigation = Navigation()
    val services = DesktopPlatformServices(environment)
    model.start()
    val icon = Navigation::class.java.getResourceAsStream("/icon.svg")?.use { windowIcon(it.readBytes()) }
    application {
        Window(
            onCloseRequest = {
                model.dispose()
                exitApplication()
            },
            state = rememberWindowState(width = 1280.dp, height = 820.dp),
            title = "vfox+",
            icon = icon,
        ) {
            LaunchedEffect(Unit) { window.minimumSize = Dimension(360, 480) }
            VfoxPlusApp(model, navigation, services)
            if (environment["VFOX_PLUS_E2E_SCENARIO"] == "measure") MeasureScenario(model, navigation)
        }
    }
}

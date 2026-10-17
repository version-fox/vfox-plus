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

import androidx.compose.animation.AnimatedContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ShortNavigationBar
import androidx.compose.material3.ShortNavigationBarItem
import androidx.compose.material3.Text
import androidx.compose.material3.WideNavigationRail
import androidx.compose.material3.WideNavigationRailItem
import androidx.compose.material3.WideNavigationRailValue
import androidx.compose.material3.rememberWideNavigationRailState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import dev.vfox.plus.state.VfoxAvailability
import dev.vfox.plus.ui.components.TooltipIconButton
import dev.vfox.plus.ui.pages.HomePage
import dev.vfox.plus.ui.pages.PluginsPage
import dev.vfox.plus.ui.pages.SdksPage
import dev.vfox.plus.ui.pages.SettingsPage
import dev.vfox.plus.ui.pages.TasksPage
import dev.vfox.plus.ui.pages.UnavailablePage

/** Material 3 window size classes: compact windows are narrower than 600 dp. */
private val COMPACT_WIDTH = 600.dp

/** The icon of a destination, with the number of active tasks on Tasks. */
@Composable
private fun DestinationIcon(
    destination: Destination,
    selected: Boolean,
) {
    val active = app.model.tasks.activeCount
    BadgedBox(badge = { if (destination == Destination.TASKS && active > 0) Badge { Text(active.toString()) } }) {
        Icon(destination.icon, filled = selected)
    }
}

/**
 * The adaptive frame around the pages: a short navigation bar at the bottom of
 * compact windows, the expressive wide navigation rail at the start of wider
 * ones. Switching destinations fades through. Without vfox, every page but
 * Settings explains how to get it.
 */
@Composable
fun Shell() {
    val context = app
    val model = context.model
    val navigation = context.navigation
    val ready = model.vfox.availability == VfoxAvailability.READY
    val page = if (!ready && navigation.destination != Destination.SETTINGS) null else navigation.destination
    val transition = fadeThrough<Destination?>()
    val stage: @Composable (Modifier) -> Unit = { modifier ->
        AnimatedContent(page, modifier, transitionSpec = transition, label = "destination") { shown ->
            when (shown) {
                null -> UnavailablePage()
                Destination.HOME -> HomePage()
                Destination.SDKS -> SdksPage()
                Destination.PLUGINS -> PluginsPage()
                Destination.TASKS -> TasksPage()
                Destination.SETTINGS -> SettingsPage()
            }
        }
    }
    androidx.compose.foundation.layout.BoxWithConstraints(Modifier.fillMaxSize()) {
        if (maxWidth < COMPACT_WIDTH) {
            Column(Modifier.fillMaxSize()) {
                stage(Modifier.weight(1f))
                ShortNavigationBar(containerColor = MaterialTheme.colorScheme.surfaceContainer) {
                    Destination.entries.forEach { destination ->
                        val label = t(destination.label)
                        ShortNavigationBarItem(
                            selected = navigation.destination == destination,
                            onClick = { navigation.go(destination) },
                            icon = { DestinationIcon(destination, navigation.destination == destination) },
                            label = { Text(label) },
                            modifier = Modifier.semantics { contentDescription = label },
                        )
                    }
                }
            }
        } else {
            val expanded = model.settings.settings.railExpanded
            val railState =
                rememberWideNavigationRailState(if (expanded) WideNavigationRailValue.Expanded else WideNavigationRailValue.Collapsed)
            LaunchedEffect(expanded) { if (expanded) railState.expand() else railState.collapse() }
            Row(Modifier.fillMaxSize()) {
                WideNavigationRail(
                    state = railState,
                    header = {
                        Column {
                            TooltipIconButton(
                                if (expanded) "menuOpen" else "menu",
                                if (expanded) t("navCollapse") else t("navExpand"),
                                Modifier.padding(start = 24.dp),
                            ) { model.updateSettings { it.copy(railExpanded = !expanded) } }
                            if (ready) {
                                val label = t("navAddPlugin")
                                ExtendedFloatingActionButton(
                                    onClick = { navigation.go(Destination.PLUGINS) },
                                    expanded = expanded,
                                    icon = { Icon("add") },
                                    text = { Text(label) },
                                    modifier = Modifier.padding(start = 20.dp, top = 8.dp).semantics { contentDescription = label },
                                )
                            }
                        }
                    },
                ) {
                    Destination.entries.forEach { destination ->
                        val label = t(destination.label)
                        WideNavigationRailItem(
                            selected = navigation.destination == destination,
                            onClick = { navigation.go(destination) },
                            icon = { DestinationIcon(destination, navigation.destination == destination) },
                            label = { Text(label) },
                            railExpanded = expanded,
                            modifier = Modifier.semantics { contentDescription = label },
                        )
                    }
                }
                Box(Modifier.weight(1f).padding(top = 16.dp, end = 16.dp, bottom = 16.dp)) { stage(Modifier.fillMaxSize()) }
            }
        }
    }
}

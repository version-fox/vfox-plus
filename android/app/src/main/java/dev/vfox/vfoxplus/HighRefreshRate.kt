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

package dev.vfox.vfoxplus

import android.app.Activity

// An app that does not ask runs at the rate the system picks, which many
// phones keep at 60 Hz for ordinary windows. This asks for the display mode
// with the highest refresh rate at the current resolution, so that the
// transitions of the web interface and of the app run at 90 or 120 Hz where
// the display has it. Modes of another resolution are left out: switching to
// one blanks the screen. Without such a mode nothing changes.
object HighRefreshRate {
    @Suppress("DEPRECATION")
    fun request(activity: Activity) {
        val display = activity.windowManager.defaultDisplay ?: return
        val current = display.mode
        val best = display.supportedModes
            .filter {
                it.physicalWidth == current.physicalWidth &&
                    it.physicalHeight == current.physicalHeight
            }
            .maxByOrNull { it.refreshRate } ?: return
        if (best.modeId == current.modeId) return
        val attributes = activity.window.attributes
        attributes.preferredDisplayModeId = best.modeId
        activity.window.attributes = attributes
    }
}

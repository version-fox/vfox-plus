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

import android.content.Context
import java.io.File
import java.io.IOException

// The pairing link that `vfxtool measure-apk --pair-url` opens in a debuggable build. The shell
// writes it into the private files directory of the app with `run-as`, which only a debuggable
// application allows, and then starts the launcher, which takes the link and removes the file.
// No other application can write into that directory, and no exported component takes a link,
// so no other application can open a page inside the app this way.
internal object MeasurementLink {
    private const val LINK_FILE = "measurement/pairing-link"
    private const val MAX_LENGTH = 4096L

    // The one-time link waiting for the app, or null when there is none or it is not a pairing
    // link. The file is removed in any case.
    fun take(context: Context): String? {
        val file = File(context.filesDir, LINK_FILE)
        if (!file.isFile) return null
        return try {
            file.takeIf { it.length() <= MAX_LENGTH }?.readText()?.trim()
                ?.takeIf { PairingFragment.pairingBase(it) != null }
        } catch (e: IOException) {
            null
        } finally {
            file.delete()
        }
    }
}

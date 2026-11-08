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

package dev.vfox.vfoxplus.session

import android.content.Context
import androidx.core.content.edit

// The session cookies, kept in SharedPreferences so a restart reconnects
// without pairing again. The server
// sets a single session cookie; entries are stored per host as name=value
// pairs and sent back on every request to that host.
internal class PersistentCookies(context: Context) {
    private val prefs = context.getSharedPreferences("session-cookies", Context.MODE_PRIVATE)

    @Synchronized
    fun header(host: String): String? {
        val jar = prefs.getStringSet(host, null).orEmpty()
        if (jar.isEmpty()) return null
        return jar.sorted().joinToString("; ") { it.substringBefore("; expires=") }
    }

    @Synchronized
    fun store(host: String, setCookies: List<String>) {
        if (setCookies.isEmpty()) return
        val jar = prefs.getStringSet(host, null).orEmpty().associate {
            it.substringBefore("=").trim() to it
        }.toMutableMap()
        for (raw in setCookies) {
            val pair = raw.substringBefore(";").trim()
            val name = pair.substringBefore("=")
            if (name.isNotEmpty()) jar[name] = pair
        }
        prefs.edit {
            putStringSet(host, jar.values.toSet())
        }
    }

    @Synchronized
    fun clear() {
        prefs.edit { clear() }
    }
}

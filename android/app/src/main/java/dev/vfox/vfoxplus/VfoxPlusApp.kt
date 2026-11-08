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

import android.app.Application

// Application setup for the vfox-plus remote client.
class VfoxPlusApp : Application() {
    override fun onCreate() {
        super.onCreate()
        // The pairing session lives in a cookie; keep it across launches so a
        // paired phone reconnects without pasting another link.
        android.webkit.CookieManager.getInstance().apply {
            setAcceptCookie(true)
        }
        // The native chrome takes the system's expressive palette on
        // Android 12 and later; older releases keep the seed scheme built
        // into the theme.
        com.google.android.material.color.DynamicColors.applyToActivitiesIfAvailable(this)
    }
}

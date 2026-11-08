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

import android.webkit.JavascriptInterface

// What the web interface can ask of the app that shows it, as window.vfoxAndroid. The
// page knows that it runs in the app when the object is there: it keeps clear of the
// system bars by the insets (WebView does not give a page its safe area as env(), in
// every version), and its settings offer the two things that only the app can do.
// The WebView shows nothing but the paired server (an address of another site opens
// in the browser), so the page is the one that the user paired.
class WebBridge(private val host: Host) {
    interface Host {
        // The safe area as "top,right,bottom,left,keyboard", in CSS pixels: what the system bars and a display cutout cover, and the
        // height of the keyboard while it is open (then the bottom bar is not in the way).
        fun insets(): String

        // The address of the paired server, scheme and authority.
        fun server(): String

        fun pairNew()

        fun disconnect()
    }

    @JavascriptInterface
    fun insets(): String = host.insets()

    @JavascriptInterface
    fun server(): String = host.server()

    @JavascriptInterface
    fun pairNew() = host.pairNew()

    @JavascriptInterface
    fun disconnect() = host.disconnect()
}

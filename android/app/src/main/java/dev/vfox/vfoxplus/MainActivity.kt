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

import android.content.Intent
import android.os.Bundle
import androidx.activity.enableEdgeToEdge
import androidx.appcompat.app.AppCompatActivity
import dev.vfox.vfoxplus.databinding.ActivityMainBinding

// Single-activity host: pairing first, the WebView after. The app draws edge to edge,
// and every screen keeps clear of the system bars itself (Insets.kt). The system back
// gesture walks the back stack of the screens, and the session screen takes it first
// while its page has a history; the app does not handle it here, so that the system
// can show the predictive back animation.
class MainActivity : AppCompatActivity() {
    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        HighRefreshRate.request(this)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        if (savedInstanceState == null) {
            supportFragmentManager.beginTransaction()
                .setReorderingAllowed(true)
                .add(R.id.container, firstScreen(intent))
                .commit()
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        // A fresh pairing link while the app runs opens its session on top,
        // like pasting the link and tapping connect.
        if (firstScreen(intent) is ViewerFragment) {
            supportFragmentManager.beginTransaction()
                .setReorderingAllowed(true)
                .replace(R.id.container, firstScreen(intent))
                .addToBackStack(null)
                .commit()
        }
    }

    companion object {
        // The measurement entry point of `vfxtool measure-apk --pair-url` (and
        // of anyone holding a pairing link): an explicit VIEW intent whose
        // data is a one-time pairing link or a bare server address opens the
        // session directly, exactly as if the link had been pasted and
        // connected. It grants nothing that the link itself does not grant.
        fun firstScreen(intent: Intent?) =
            if (intent?.action == Intent.ACTION_VIEW &&
                PairingFragment.pairingBase(intent.dataString.orEmpty()) != null
            ) {
                ViewerFragment.newInstance(intent.dataString!!.trim())
            } else {
                PairingFragment()
            }
    }
}

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

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.google.android.material.button.MaterialButton
import dev.vfox.vfoxplus.R
import org.json.JSONObject

// The home tab of the native session: the version, the stat tiles, the
// plugins in use, the terminal card and the registry/docs cards.
class HomeFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var swipe: SwipeRefreshLayout
    private lateinit var body: LinearLayout
    private lateinit var version: TextView

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = (parentFragment as NativeSessionFragment) as SessionHost
        swipe = SwipeRefreshLayout(requireContext())
        val scroll = ScrollView(requireContext())
        body = LinearLayout(requireContext())
        SessionUi.pad(body)
        scroll.addView(body)
        swipe.addView(scroll)
        swipe.setOnRefreshListener { refresh() }
        return swipe
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        refresh()
    }

    override fun refresh() {
        swipe.isRefreshing = true
        host.runApi(
            task = { host.api.get("${host.base}/api/inventory") },
            done = { result ->
                if (!isAdded) return@runApi
                swipe.isRefreshing = false
                result.fold(
                    onSuccess = { response ->
                        if (response.code != 200) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        draw(SessionModels.inventory(JSONObject(response.body)))
                    },
                    onFailure = {
                        host.notify(getString(R.string.request_failed))
                    },
                )
            },
        )
    }

    private fun draw(inventory: SessionModels.Inventory) {
        body.removeAllViews()
        val context = requireContext()
        version = TextView(context).apply { text = "vfox ${inventory.vfox.version.orEmpty()}" }
        body.addView(version)
        // The three stat tiles: plugins, versions, in use.
        val stats = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val installed = inventory.installed
        val inUse = installed.count { it.hasCurrent }
        val versions = installed.sumOf { it.versions.size }
        for ((count, label) in listOf(
            installed.size to getString(R.string.session_plugins),
            versions to getString(R.string.session_versions),
            inUse to getString(R.string.session_in_use),
        )
        ) {
            val tile = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, count.toString(), 28f))
            inner.addView(SessionUi.body(context, label))
            tile.addView(inner)
            tile.layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
            stats.addView(tile)
        }
        body.addView(stats)
        body.addView(SessionUi.section(context, getString(R.string.session_active)))
        for (sdk in installed.filter { it.hasCurrent }) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            SessionUi.pad(inner)
            val avatar = TextView(context).apply {
                text = sdk.name.firstOrNull()?.uppercase().orEmpty()
                textSize = 24f
            }
            val names = LinearLayout(context)
            SessionUi.pad(names, 12, 0)
            names.addView(SessionUi.headline(context, sdk.name, 17f))
            names.addView(SessionUi.body(context, sdk.current.orEmpty()))
            inner.addView(avatar)
            inner.addView(names)
            card.addView(inner)
            card.setOnClickListener { host.openSdkDetail(sdk.name) }
            body.addView(card)
        }
        val terminal = SessionUi.card(context)
        val terminalInner = LinearLayout(context)
        SessionUi.pad(terminalInner)
        terminalInner.addView(SessionUi.headline(context, getString(R.string.session_terminal), 16f))
        terminalInner.addView(SessionUi.body(context, getString(R.string.session_terminal_body)))
        terminalInner.addView(SessionUi.body(context, "eval \"\$(vfox activate bash)\""))
        terminal.addView(terminalInner)
        body.addView(terminal)
        val explore = SessionUi.card(context)
        val exploreInner = LinearLayout(context)
        SessionUi.pad(exploreInner)
        exploreInner.addView(SessionUi.headline(context, getString(R.string.session_explore), 16f))
        explore.addView(exploreInner)
        explore.setOnClickListener { host.openTab(2) }
        body.addView(explore)
        val add = com.google.android.material.floatingactionbutton.FloatingActionButton(context).apply {
            setImageResource(dev.vfox.vfoxplus.R.drawable.ic_add)
            contentDescription = getString(R.string.session_explore)
            setOnClickListener { host.openTab(2) }
        }
        val addRow = LinearLayout(context).apply {
            gravity = android.view.Gravity.END
        }
        addRow.addView(add)
        body.addView(addRow)
        version.isVisible = true
    }
}

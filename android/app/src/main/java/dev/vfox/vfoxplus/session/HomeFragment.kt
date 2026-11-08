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

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ScrollView
import androidx.fragment.app.Fragment
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.google.android.material.button.MaterialButton
import dev.vfox.vfoxplus.Motion
import dev.vfox.vfoxplus.R
import dev.vfox.vfoxplus.springIn
import org.json.JSONObject

// The home tab of the native session: the version, the stat tiles, the
// plugins in use, the terminal card and the registry card.
class HomeFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var swipe: SwipeRefreshLayout
    private lateinit var body: LinearLayout

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enterTransition = Motion.fadeThrough(requireContext(), true)
        exitTransition = Motion.fadeThrough(requireContext(), false)
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = parentFragment as SessionHost
        val root = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        root.addView(SessionUi.bar(requireContext(), "vfox-plus"))
        swipe = SwipeRefreshLayout(requireContext())
        val scroll = ScrollView(requireContext())
        body = LinearLayout(requireContext())
        SessionUi.pad(body)
        scroll.addView(body)
        swipe.addView(scroll)
        swipe.setOnRefreshListener { refresh() }
        root.addView(
            swipe,
            LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f),
        )
        return root
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
        val version = SessionUi.body(context)
        version.text = getString(R.string.session_version, inventory.vfox.version.orEmpty())
        body.addView(version)
        // The three stat tiles: plugins, versions, in use.
        val stats = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val installed = inventory.installed
        val inUse = installed.count { it.hasCurrent }
        val versions = installed.sumOf { it.versions.size }
        val tiles = listOf(
            Triple(installed.size.toString(), getString(R.string.session_plugins), dev.vfox.vfoxplus.R.drawable.ic_nav_plugins),
            Triple(versions.toString(), getString(R.string.session_versions), dev.vfox.vfoxplus.R.drawable.ic_nav_sdks),
            Triple(inUse.toString(), getString(R.string.session_in_use), dev.vfox.vfoxplus.R.drawable.ic_nav_home),
        )
        tiles.forEachIndexed { index, (count, label, icon) ->
            val tile = SessionUi.stat(context, count, label, icon)
            tile.springIn((index * 50).toLong())
            stats.addView(tile)
        }
        body.addView(stats)
        body.addView(SessionUi.section(context, getString(R.string.session_active)))
        installed.filter { it.hasCurrent }.forEachIndexed { index, sdk ->
            body.addView(
                SessionUi.row(
                    context,
                    title = sdk.name,
                    subtitle = sdk.current,
                    avatarName = sdk.name,
                    onClick = { host.openSdkDetail(sdk.name) },
                ).apply { springIn((150 + index * 50).toLong()) },
            )
            body.addView(SessionUi.divider(context))
        }
        val terminal = SessionUi.card(context)
        val terminalInner = LinearLayout(context)
        SessionUi.pad(terminalInner)
        terminalInner.addView(SessionUi.headline(context, getString(R.string.session_terminal), 16f))
        terminalInner.addView(SessionUi.body(context, getString(R.string.session_terminal_body)))
        val shellRow = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = android.view.Gravity.CENTER_VERTICAL
        }
        val command = SessionUi.body(context, "eval \"\$(vfox activate bash)\"").apply {
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        val copy = MaterialButton(context, null, com.google.android.material.R.attr.materialIconButtonStyle).apply {
            setIconResource(dev.vfox.vfoxplus.R.drawable.ic_paste)
            contentDescription = getString(R.string.session_copy)
            setOnClickListener {
                val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("shell", "eval \"\$(vfox activate bash)\""))
                host.notify(getString(R.string.session_copied))
            }
        }
        shellRow.addView(command)
        shellRow.addView(copy)
        terminalInner.addView(shellRow)
        terminal.addView(terminalInner)
        terminal.springIn(250)
        body.addView(terminal)
        val explore = SessionUi.card(context)
        explore.addView(
            SessionUi.row(
                context,
                title = getString(R.string.session_explore),
                subtitle = getString(R.string.session_explore_body),
                chevron = true,
                onClick = { host.openTab(2) },
            ),
        )
        explore.springIn(300)
        body.addView(explore)
        val add = com.google.android.material.floatingactionbutton.FloatingActionButton(context).apply {
            setImageResource(dev.vfox.vfoxplus.R.drawable.ic_add)
            contentDescription = getString(R.string.session_explore)
            setOnClickListener { host.openTab(2) }
        }
        val addRow = LinearLayout(context).apply {
            gravity = android.view.Gravity.END
            val margin = (16 * resources.displayMetrics.density).toInt()
            setPadding(0, margin, margin, margin)
        }
        addRow.addView(add)
        body.addView(addRow)
    }
}

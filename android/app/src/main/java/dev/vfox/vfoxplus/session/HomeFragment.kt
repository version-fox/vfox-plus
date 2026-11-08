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
import android.widget.TextView
import androidx.fragment.app.Fragment
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.google.android.material.button.MaterialButton
import com.google.android.material.color.MaterialColors
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
        // A coordinator: the bar folds away on scroll, the action button
        // anchors to the corner, never cut by the content.
        val root = androidx.coordinatorlayout.widget.CoordinatorLayout(requireContext())
        val appBar = com.google.android.material.appbar.AppBarLayout(requireContext())
        appBar.addView(
            SessionUi.largeBar(requireContext(), "").apply {
                layoutParams = com.google.android.material.appbar.AppBarLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT,
                ).apply {
                    scrollFlags =
                        com.google.android.material.appbar.AppBarLayout.LayoutParams.SCROLL_FLAG_SCROLL or
                        com.google.android.material.appbar.AppBarLayout.LayoutParams.SCROLL_FLAG_ENTER_ALWAYS or
                        com.google.android.material.appbar.AppBarLayout.LayoutParams.SCROLL_FLAG_SNAP
                }
            },
        )
        root.addView(
            appBar,
            androidx.coordinatorlayout.widget.CoordinatorLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT,
            ),
        )
        swipe = SwipeRefreshLayout(requireContext())
        val scroll = androidx.core.widget.NestedScrollView(requireContext())
        body = LinearLayout(requireContext())
        SessionUi.pad(body)
        scroll.addView(body)
        swipe.addView(scroll)
        swipe.setOnRefreshListener { refresh() }
        swipe.layoutParams = androidx.coordinatorlayout.widget.CoordinatorLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT,
        ).apply {
            behavior = com.google.android.material.appbar.AppBarLayout.ScrollingViewBehavior()
        }
        root.addView(swipe)
        val density = resources.displayMetrics.density
        root.addView(
            com.google.android.material.floatingactionbutton.FloatingActionButton(requireContext()).apply {
                setImageResource(dev.vfox.vfoxplus.R.drawable.ic_add)
                contentDescription = getString(R.string.session_explore)
                setOnClickListener { host.openTab(2) }
            }.also { fab ->
                fab.layoutParams = androidx.coordinatorlayout.widget.CoordinatorLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT,
                ).apply {
                    gravity = android.view.Gravity.BOTTOM or android.view.Gravity.END
                    val margin = (16 * density).toInt()
                    setMargins(margin, margin, margin, margin)
                    behavior = com.google.android.material.floatingactionbutton.FloatingActionButton.Behavior()
                }
            },
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
        // The header band: the title, the version, and the three numbers as
        // text columns in one colour field, instead of three squeezed pills.
        val density = context.resources.displayMetrics.density
        val band = SessionUi.band(context, SessionUi.PRIMARY)
        val onBand = MaterialColors.getColor(context, SessionUi.PRIMARY.onContainer, 0)
        band.addView(TextView(context).apply {
            text = "vfox-plus"
            setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_DisplaySmall)
            setTextColor(onBand)
        })
        band.addView(TextView(context).apply {
            text = getString(R.string.session_version, inventory.vfox.version.orEmpty())
            setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyLarge)
            setTextColor(onBand)
        })
        val stats = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(0, (16 * density).toInt(), 0, 0)
        }
        val installed = inventory.installed
        val counts = listOf(
            installed.size.toString() to getString(R.string.session_plugins),
            installed.sumOf { it.versions.size }.toString() to getString(R.string.session_versions),
            installed.count { it.hasCurrent }.toString() to getString(R.string.session_in_use),
        )
        counts.forEach { (count, label) ->
            stats.addView(LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                addView(TextView(context).apply {
                    text = count
                    setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_HeadlineMedium)
                    setTextColor(onBand)
                })
                addView(TextView(context).apply {
                    text = label
                    setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_LabelLarge)
                    setTextColor(onBand)
                })
            })
        }
        band.addView(stats)
        band.springIn(0)
        body.addView(band)
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
        terminalInner.addView(
            SessionUi.settingRow(
                context,
                title = getString(R.string.session_terminal),
                icon = dev.vfox.vfoxplus.R.drawable.ic_terminal,
                hue = SessionUi.TERTIARY,
            ),
        )
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
            SessionUi.settingRow(
                context,
                title = getString(R.string.session_explore),
                subtitle = getString(R.string.session_explore_body),
                icon = dev.vfox.vfoxplus.R.drawable.ic_store,
                hue = SessionUi.PRIMARY,
                trailing = android.widget.ImageView(context).apply {
                    setImageResource(dev.vfox.vfoxplus.R.drawable.ic_forward)
                    imageTintList = android.content.res.ColorStateList.valueOf(
                        com.google.android.material.color.MaterialColors.getColor(
                            context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0,
                        ),
                    )
                    val size = (24 * context.resources.displayMetrics.density).toInt()
                    layoutParams = LinearLayout.LayoutParams(size, size)
                },
                onClick = { host.openTab(2) },
            ),
        )
        explore.springIn(300)
        body.addView(explore)
    }
}

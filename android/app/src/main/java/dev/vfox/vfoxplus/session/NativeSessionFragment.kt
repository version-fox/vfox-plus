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
import android.os.Handler
import android.os.Looper
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import dev.vfox.vfoxplus.PairingFragment
import dev.vfox.vfoxplus.R
import dev.vfox.vfoxplus.ServerStore
import dev.vfox.vfoxplus.databinding.DialogTlsBinding
import dev.vfox.vfoxplus.padToSafeArea
import java.io.Closeable
import java.util.concurrent.Executors

// What the paired session calls back into: the API, background work, and
// leaving the session. The tab fragments share one host.
interface SessionHost {
    val api: SessionApi
    val base: String
    fun <T> runApi(task: () -> T, done: (Result<T>) -> Unit)
    fun notify(message: String)
    fun openSdkDetail(name: String)
    fun openTab(id: Int)
    fun setTaskBadge(count: Int)
    fun pairNew()
    fun disconnect()
    fun refreshAll()
}

// The whole /app page in Views: the five tabs over the paired JSON API, the
// trust dialog for a self-signed certificate, and the unreachable page when
// the PC cannot be reached.
class NativeSessionFragment : Fragment(), SessionHost {
    override lateinit var api: SessionApi
    override lateinit var base: String
    private var startUrl: String = ""
    private lateinit var content: FrameLayout
    private lateinit var loading: View
    private lateinit var errorTitle: TextView
    private lateinit var errorBody: TextView
    private lateinit var errorServer: TextView
    private lateinit var errorBox: LinearLayout
    private lateinit var retryButton: MaterialButton
    private lateinit var pairNewButton: MaterialButton
    private lateinit var tabs: com.google.android.material.bottomnavigation.BottomNavigationView
    private val main = Handler(Looper.getMainLooper())
    private val workers = Executors.newSingleThreadExecutor()
    private var events: Closeable? = null
    private var paired = false
    private var tab: Fragment? = null
    private var tabId = 0

    override fun <T> runApi(task: () -> T, done: (Result<T>) -> Unit) {
        workers.execute {
            val result = try {
                Result.success(task())
            } catch (e: SessionApi.UntrustedChain) {
                Result.failure<T>(SessionError.Untrusted(e.chain))
            } catch (e: RefusedException) {
                Result.failure<T>(SessionError.Refused(e.code, e.body))
            } catch (e: java.io.IOException) {
                Result.failure<T>(SessionError.Unreachable(base, e))
            }
            main.post { if (isAdded) done(result) }
        }
    }

    override fun notify(message: String) {
        view?.let { Snackbar.make(it, message, Snackbar.LENGTH_LONG).show() }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        api = SessionApi(requireContext())
        startUrl = requireArguments().getString(ARG_URL).orEmpty()
        base = PairingFragment.pairingBase(startUrl).orEmpty()
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val root = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(MaterialColors.getColor(context, com.google.android.material.R.attr.colorSurface, 0))
        }
        content = FrameLayout(requireContext()).apply {
            id = View.generateViewId()
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f,
            )
        }
        loading = com.google.android.material.progressindicator.CircularProgressIndicator(requireContext()).apply {
            isIndeterminate = true
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT, Gravity.CENTER,
            )
        }
        errorBox = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER
            visibility = View.GONE
            setPadding(48, 48, 48, 48)
        }
        val density = resources.displayMetrics.density
        val errorArt = FrameLayout(requireContext()).apply {
            layoutParams = LinearLayout.LayoutParams((200 * density).toInt(), (200 * density).toInt()).apply {
                gravity = Gravity.CENTER
            }
        }
        errorArt.addView(
            android.widget.ImageView(requireContext()).apply {
                setImageDrawable(
                    dev.vfox.vfoxplus.PolygonDrawable(
                        dev.vfox.vfoxplus.ExpressiveShapes.softBurst,
                        MaterialColors.getColor(context, com.google.android.material.R.attr.colorErrorContainer, 0),
                    ),
                )
                layoutParams = FrameLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT,
                )
            },
        )
        errorArt.addView(
            android.widget.ImageView(requireContext()).apply {
                setImageResource(dev.vfox.vfoxplus.R.drawable.ic_unreachable)
                imageTintList = android.content.res.ColorStateList.valueOf(
                    MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnErrorContainer, 0),
                )
                layoutParams = FrameLayout.LayoutParams((72 * density).toInt(), (72 * density).toInt(), Gravity.CENTER)
            },
        )
        errorBox.addView(errorArt)
        errorTitle = TextView(requireContext()).apply {
            setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_HeadlineSmall)
            gravity = Gravity.CENTER
        }
        errorBody = TextView(requireContext()).apply {
            gravity = Gravity.CENTER
        }
        errorServer = TextView(requireContext()).apply {
            gravity = Gravity.CENTER
        }
        retryButton = MaterialButton(requireContext()).apply { setText(R.string.try_again) }
        pairNewButton = MaterialButton(requireContext(), null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            setText(R.string.pair_new)
        }
        errorBox.addView(errorTitle)
        errorBox.addView(errorBody)
        errorBox.addView(errorServer)
        errorBox.addView(retryButton)
        errorBox.addView(pairNewButton)
        content.addView(errorBox)
        content.addView(loading)
        tabs = com.google.android.material.bottomnavigation.BottomNavigationView(requireContext()).apply {
            inflateMenu(dev.vfox.vfoxplus.R.menu.session_nav)
            setOnItemSelectedListener { item ->
                showTab(
                    when (item.itemId) {
                        dev.vfox.vfoxplus.R.id.nav_sdks -> 1
                        dev.vfox.vfoxplus.R.id.nav_plugins -> 2
                        dev.vfox.vfoxplus.R.id.nav_tasks -> 3
                        dev.vfox.vfoxplus.R.id.nav_settings -> 4
                        else -> 0
                    },
                )
                true
            }
            visibility = View.GONE
        }
        root.addView(content)
        root.addView(tabs)
        root.padToSafeArea()
        return root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        retryButton.setOnClickListener { pair() }
        pairNewButton.setOnClickListener { pairNew() }
        requireActivity().onBackPressedDispatcher.addCallback(
            viewLifecycleOwner,
            object : androidx.activity.OnBackPressedCallback(true) {
                override fun handleOnBackPressed() {
                    if (!goBack()) {
                        isEnabled = false
                        requireActivity().onBackPressedDispatcher.onBackPressed()
                    }
                }
            },
        )
        buildTabs()
        pair()
    }

    override fun onDestroyView() {
        events?.close()
        super.onDestroyView()
    }

    override fun onDestroy() {
        workers.shutdownNow()
        super.onDestroy()
    }

    private fun buildTabs() {
        // The bar keeps its own checked state; showing a tab only swaps the content.
    }

    private fun showTab(id: Int) {
        tabId = id
        tabs.menu.findItem(
            when (id) {
                1 -> dev.vfox.vfoxplus.R.id.nav_sdks
                2 -> dev.vfox.vfoxplus.R.id.nav_plugins
                3 -> dev.vfox.vfoxplus.R.id.nav_tasks
                4 -> dev.vfox.vfoxplus.R.id.nav_settings
                else -> dev.vfox.vfoxplus.R.id.nav_home
            },
        )?.isChecked = true
        tab = when (id) {
            1 -> SdksFragment()
            2 -> PluginsFragment()
            3 -> TasksFragment()
            4 -> SettingsFragment()
            else -> HomeFragment()
        }
        childFragmentManager.popBackStackImmediate(
            null, androidx.fragment.app.FragmentManager.POP_BACK_STACK_INCLUSIVE,
        )
        childFragmentManager.beginTransaction()
            .setReorderingAllowed(true)
            .replace(content.id, tab!!, "tab")
            .commit()
        errorBox.visibility = View.GONE
    }

    override fun refreshAll() {
        (childFragmentManager.findFragmentByTag("tab") as? Refreshable)?.refresh()
    }

    override fun openTab(id: Int) {
        showTab(id)
    }

    override fun setTaskBadge(count: Int) {
        val badge = tabs.getOrCreateBadge(dev.vfox.vfoxplus.R.id.nav_tasks)
        badge.isVisible = count > 0
        badge.number = count
    }

    override fun openSdkDetail(name: String) {
        childFragmentManager.beginTransaction()
            .setReorderingAllowed(true)
            .replace(content.id, SdkDetailFragment.newInstance(name), "detail")
            .addToBackStack(null)
            .commit()
    }

    override fun pairNew() {
        parentFragmentManager.popBackStack()
    }

    override fun disconnect() {
        api.forgetSession()
        ServerStore(requireContext()).forget()
        parentFragmentManager.popBackStack()
    }

    fun goBack(): Boolean {
        if (childFragmentManager.popBackStackImmediate()) return true
        if (paired) {
            parentFragmentManager.popBackStack()
            return true
        }
        return false
    }

    private fun pair() {
        loading.isVisible = true
        errorBox.visibility = View.GONE
        runApi(
            task = {
                val bare = !("/auth/connect/" in startUrl)
                if (bare) {
                    // A remembered server: only a live session opens it.
                    val probe = api.get("$base/app")
                    if (probe.code != 200) throw RefusedException(probe.code, probe.body)
                } else {
                    val landing = api.get(startUrl)
                    if (!landing.finalUrl.endsWith("/app") || landing.code != 200) {
                        throw RefusedException(landing.code, landing.body)
                    }
                }
                ServerStore(requireContext()).lastPairingUrl = base
            },
            done = { result ->
                loading.isVisible = false
                result.fold(
                    onSuccess = {
                        paired = true
                        tabs.visibility = View.VISIBLE
                        showTab(tabId)
                        watchEvents()
                    },
                    onFailure = { error ->
                        when (error) {
                            is SessionError.Untrusted -> askTrust(error)
                            is SessionError.Unreachable -> showError(
                                getString(R.string.unreachable_title),
                                getString(R.string.unreachable_body),
                                base,
                            )
                            else -> showError(
                                getString(R.string.request_failed),
                                (error as? SessionError.Refused)?.body.orEmpty(),
                                base,
                            )
                        }
                    },
                )
            },
        )
    }

    private fun askTrust(error: SessionError.Untrusted) {
        val fingerprint = error.fingerprint
        val dialog = MaterialAlertDialogBuilder(requireContext())
            .setIcon(R.drawable.ic_lock)
            .setTitle(R.string.tls_title)
            .setPositiveButton(R.string.trust_once) { _, _ ->
                api.trustedFingerprint = fingerprint
                pair()
            }
            .setNegativeButton(android.R.string.cancel, null)
        if (fingerprint != null) {
            val content = DialogTlsBinding.inflate(layoutInflater)
            content.fingerprint.text = fingerprint.split(":").chunked(8).joinToString("\n") { it.joinToString(":") }
            dialog.setMessage(R.string.tls_body).setView(content.root)
        } else {
            dialog.setMessage(R.string.tls_body_unknown)
        }
        dialog.show()
    }

    private fun showError(title: String, body: String, server: String) {
        tabs.visibility = View.GONE
        errorBox.visibility = View.VISIBLE
        errorTitle.text = title
        errorBody.text = body
        errorServer.text = server.substringAfter("://")
    }

    private fun watchEvents() {
        events?.close()
        events = api.openEvents("$base/api/events") { kind ->
            if (!isAdded) return@openEvents
            when (kind) {
                "hello", "task", "inventory", "vfox", "web" -> refreshAll()
            }
        }
    }

    // The sessions own no back press: the container pops the detail first,
    // then leaves the session.
    companion object {
        private const val ARG_URL = "url"

        fun newInstance(startUrl: String) = NativeSessionFragment().apply {
            arguments = android.os.Bundle().apply { putString(ARG_URL, startUrl) }
        }
    }

    // What the tabs implement to redraw on show and on server events.
    interface Refreshable {
        fun refresh()
    }
}

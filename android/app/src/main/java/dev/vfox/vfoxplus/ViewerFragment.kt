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

import android.annotation.SuppressLint
import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.http.SslError
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.webkit.CookieManager
import android.webkit.SslErrorHandler
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.OnBackPressedCallback
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import dev.vfox.vfoxplus.databinding.DialogTlsBinding
import dev.vfox.vfoxplus.databinding.FragmentViewerBinding
import java.util.Locale

// The paired session itself: a WebView of the server's /app interface, which
// owns the Material 3 Expressive look of the session and asks the app for what
// only the app can do through window.vfoxAndroid (WebBridge). The app adds what a
// page cannot show: the state of a server that cannot be reached, and the
// certificate that the user has to compare.
class ViewerFragment : Fragment(), WebBridge.Host {
    private var _binding: FragmentViewerBinding? = null
    private val binding get() = _binding!!
    private var fileCallback: ValueCallback<Array<android.net.Uri>>? = null

    // The safe area that the page was last told, "top,right,bottom,left,keyboard" in CSS pixels.
    @Volatile
    private var insets = "0,0,0,0"

    private val pickFile = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        val uris = if (result.resultCode == Activity.RESULT_OK) {
            result.data?.data?.let { arrayOf(it) } ?: emptyArray()
        } else {
            emptyArray()
        }
        fileCallback?.onReceiveValue(uris)
        fileCallback = null
    }

    // The back gesture walks the history of the page before it leaves the session.
    private val pageBack = object : OnBackPressedCallback(false) {
        override fun handleOnBackPressed() {
            _binding?.web?.goBack()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val context = requireContext()
        enterTransition = Motion.sharedAxisX(context, true)
        exitTransition = Motion.sharedAxisX(context, true)
        reenterTransition = Motion.sharedAxisX(context, false)
        returnTransition = Motion.sharedAxisX(context, false)
    }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?,
    ): View {
        _binding = FragmentViewerBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onSaveInstanceState(outState: Bundle) {
        super.onSaveInstanceState(outState)
        _binding?.web?.saveState(outState)
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        val startUrl = requireArguments().getString(ARG_URL)!!
        val web = binding.web
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        web.settings.mediaPlaybackRequiresUserGesture = false
        // No white flash in the dark theme before the page paints.
        web.setBackgroundColor(
            MaterialColors.getColor(requireContext(), com.google.android.material.R.attr.colorSurface, 0),
        )
        web.addJavascriptInterface(WebBridge(this), "vfoxAndroid")
        CookieManager.getInstance().setAcceptCookie(true)
        web.webViewClient = PairingWebClient(startUrl)
        web.webChromeClient = object : WebChromeClient() {
            override fun onShowFileChooser(
                webView: WebView,
                callback: ValueCallback<Array<android.net.Uri>>,
                params: FileChooserParams,
            ): Boolean {
                fileCallback?.onReceiveValue(null)
                fileCallback = callback
                return try {
                    pickFile.launch(params.createIntent())
                    true
                } catch (_: Exception) {
                    fileCallback = null
                    false
                }
            }

            override fun onProgressChanged(view: WebView, newProgress: Int) {
                val progress = _binding?.progress ?: return
                progress.isVisible = newProgress in 1..99
                progress.setProgressCompat(newProgress, true)
            }
        }
        requireActivity().onBackPressedDispatcher.addCallback(viewLifecycleOwner, pageBack)

        binding.refresh.setColorSchemeColors(
            MaterialColors.getColor(requireContext(), androidx.appcompat.R.attr.colorPrimary, 0),
        )
        binding.refresh.setProgressBackgroundColorSchemeColor(
            MaterialColors.getColor(requireContext(), com.google.android.material.R.attr.colorSurfaceContainerHigh, 0),
        )
        binding.refresh.setOnRefreshListener { web.reload() }

        binding.errorShape.setImageDrawable(
            PolygonDrawable(
                ExpressiveShapes.softBurst,
                MaterialColors.getColor(requireContext(), com.google.android.material.R.attr.colorErrorContainer, 0),
            ),
        )
        binding.errorUrl.text = startUrl.substringAfter("://").substringBefore("/")
        binding.retryButton.setOnClickListener { retry(startUrl) }
        binding.pairNewButton.setOnClickListener { pairNew() }
        binding.errorState.padToSafeArea()

        // The page keeps clear of the bars by the insets that it is told, and the part of the
        // window that the keyboard covers is not the page's.
        ViewCompat.setOnApplyWindowInsetsListener(binding.viewerRoot) { _, windowInsets ->
            val safe = windowInsets.getInsets(WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout())
            val keyboard = if (windowInsets.isVisible(WindowInsetsCompat.Type.ime())) {
                windowInsets.getInsets(WindowInsetsCompat.Type.ime()).bottom
            } else {
                0
            }
            val density = resources.displayMetrics.density
            val bottom = if (keyboard > 0) 0 else safe.bottom
            insets = listOf(safe.top, safe.right, bottom, safe.left, keyboard)
                .joinToString(",") { String.format(Locale.ROOT, "%.1f", it / density) }
            binding.refresh.setPadding(0, 0, 0, keyboard)
            (binding.progress.layoutParams as ViewGroup.MarginLayoutParams).topMargin = safe.top
            binding.progress.requestLayout()
            binding.web.evaluateJavascript("window.vfoxInsets && window.vfoxInsets()", null)
            windowInsets
        }
        ViewCompat.requestApplyInsets(binding.viewerRoot)

        // A new Activity (the dark theme, another language, the phone turned) gets the page and its history back; without them
        // the first address, which is the interface once the session cookie is there.
        if (savedInstanceState == null || web.restoreState(savedInstanceState) == null) web.loadUrl(addressToOpen(startUrl))
    }

    // The address that works: the interface once the session cookie is there, and else the link that was pasted (it
    // works once: it is what makes the cookie).
    private fun addressToOpen(startUrl: String): String {
        val base = PairingFragment.pairingBase(startUrl) ?: return startUrl
        val paired = CookieManager.getInstance().getCookie(base)?.contains("vfox-plus-session") == true
        return if (paired) PairingFragment.appUrl(base) else startUrl
    }

    private fun retry(startUrl: String) {
        binding.errorState.isVisible = false
        binding.web.loadUrl(addressToOpen(startUrl))
    }

    // WebBridge.Host: the page asks, on a thread of its own.
    override fun insets(): String = insets

    override fun server(): String = PairingFragment.pairingBase(requireArguments().getString(ARG_URL).orEmpty()).orEmpty()

    override fun pairNew() {
        view?.post { parentFragmentManager.popBackStack() }
    }

    override fun disconnect() {
        view?.post {
            CookieManager.getInstance().removeAllCookies(null)
            CookieManager.getInstance().flush()
            ServerStore(requireContext()).forget()
            parentFragmentManager.popBackStack()
        }
    }

    // Opens an address of another site in the browser of the phone, which has an
    // address bar, downloads and the certificate checks of its own.
    private fun openInBrowser(uri: android.net.Uri) {
        val scheme = uri.scheme?.lowercase()
        if (scheme != "http" && scheme != "https") return
        try {
            startActivity(Intent(Intent.ACTION_VIEW, uri).addCategory(Intent.CATEGORY_BROWSABLE))
        } catch (_: ActivityNotFoundException) {
            Snackbar.make(binding.root, getString(R.string.open_link_failed), Snackbar.LENGTH_LONG).show()
        }
    }

    override fun onDestroyView() {
        _binding?.web?.destroy()
        _binding = null
        super.onDestroyView()
    }

    // Follows the pairing handshake inside the WebView: after the one-time
    // link sets the session cookie, move to /app. Surfaces TLS problems with
    // the certificate fingerprint for comparison with the PC terminal. Keeps the
    // window on the paired server: what the page links to elsewhere (the
    // homepage of a plugin, its documentation) opens in the browser instead.
    private inner class PairingWebClient(val startUrl: String) : WebViewClient() {
        override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
            if (!request.isForMainFrame) return false
            val server = PairingFragment.pairingBase(startUrl) ?: return false
            if (PairingFragment.pairingBase(request.url.toString()).equals(server, ignoreCase = true)) return false
            openInBrowser(request.url)
            return true
        }

        override fun onPageStarted(view: WebView, url: String, favicon: android.graphics.Bitmap?) {
            _binding?.errorState?.isVisible = false
        }

        // The one-time link is not a page to go back to: once the interface has loaded, the history starts there, so that
        // back leaves the session instead of loading the link again and being sent on to the interface.
        private var dropHistory = false

        override fun onPageFinished(view: WebView, url: String) {
            _binding?.refresh?.isRefreshing = false
            if ("/auth/connect/" in url) {
                val base = PairingFragment.pairingBase(url)
                if (base != null) {
                    dropHistory = true
                    view.loadUrl(PairingFragment.appUrl(base))
                }
            } else if (dropHistory) {
                dropHistory = false
                view.clearHistory()
                pageBack.isEnabled = false
            }
        }

        override fun doUpdateVisitedHistory(view: WebView, url: String, isReload: Boolean) {
            pageBack.isEnabled = view.canGoBack()
        }

        override fun onReceivedError(view: WebView, request: WebResourceRequest, error: WebResourceError) {
            if (request.isForMainFrame) {
                _binding?.refresh?.isRefreshing = false
                _binding?.errorState?.isVisible = true
            }
        }

        override fun onReceivedSslError(view: WebView, handler: SslErrorHandler, error: SslError) {
            val fingerprint = Fingerprint.sha256HexColon(error.certificate)
            val dialog = MaterialAlertDialogBuilder(requireContext())
                .setIcon(R.drawable.ic_lock)
                .setTitle(R.string.tls_title)
                .setPositiveButton(R.string.trust_once) { _, _ -> handler.proceed() }
                .setNegativeButton(android.R.string.cancel) { _, _ -> handler.cancel() }
                .setOnCancelListener { handler.cancel() }
            if (fingerprint != null) {
                // The fingerprint in rows of eight bytes, so that it can be compared with the terminal at a glance.
                val content = DialogTlsBinding.inflate(layoutInflater)
                content.fingerprint.text = fingerprint.split(":").chunked(8).joinToString("\n") { it.joinToString(":") }
                dialog.setMessage(R.string.tls_body).setView(content.root)
            } else {
                dialog.setMessage(R.string.tls_body_unknown)
            }
            dialog.show()
        }
    }

    companion object {
        private const val ARG_URL = "url"

        fun newInstance(startUrl: String): ViewerFragment {
            return ViewerFragment().apply {
                arguments = Bundle().apply { putString(ARG_URL, startUrl) }
            }
        }
    }
}

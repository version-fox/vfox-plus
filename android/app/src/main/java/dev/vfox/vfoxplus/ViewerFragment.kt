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
import androidx.activity.result.contract.ActivityResultContracts
import androidx.fragment.app.Fragment
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import com.google.android.material.transition.MaterialSharedAxis
import dev.vfox.vfoxplus.databinding.FragmentViewerBinding

// The paired session itself: a WebView of the server's /app interface, with a
// native overflow menu (refresh, pair a new link, disconnect). The page owns
// the Material 3 Expressive look; the shell stays out of its way.
class ViewerFragment : Fragment() {
    private var _binding: FragmentViewerBinding? = null
    private val binding get() = _binding!!
    private var fileCallback: ValueCallback<Array<android.net.Uri>>? = null

    private val pickFile = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        val uris = if (result.resultCode == Activity.RESULT_OK) {
            result.data?.data?.let { arrayOf(it) } ?: emptyArray()
        } else {
            emptyArray()
        }
        fileCallback?.onReceiveValue(uris)
        fileCallback = null
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enterTransition = MaterialSharedAxis(MaterialSharedAxis.X, true)
        returnTransition = MaterialSharedAxis(MaterialSharedAxis.X, false)
    }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?,
    ): View {
        _binding = FragmentViewerBinding.inflate(inflater, container, false)
        return binding.root
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        val startUrl = requireArguments().getString(ARG_URL)!!
        val web = binding.web
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        web.settings.mediaPlaybackRequiresUserGesture = false
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
        }
        binding.refresh.setOnRefreshListener { web.reload() }
        binding.menuButton.setOnClickListener { anchor -> showMenu(anchor) }
        if (savedInstanceState == null) web.loadUrl(startUrl)
    }

    private fun showMenu(anchor: View) {
        val popup = androidx.appcompat.widget.PopupMenu(requireContext(), anchor)
        popup.menu.add(getString(R.string.action_refresh))
        popup.menu.add(getString(R.string.action_pair_new))
        popup.menu.add(getString(R.string.action_disconnect))
        popup.setOnMenuItemClickListener { item ->
            when (item.title.toString()) {
                getString(R.string.action_refresh) -> binding.web.reload()
                getString(R.string.action_pair_new) -> parentFragmentManager.popBackStack()
                getString(R.string.action_disconnect) -> disconnect()
            }
            true
        }
        popup.show()
    }

    private fun disconnect() {
        CookieManager.getInstance().removeAllCookies(null)
        CookieManager.getInstance().flush()
        ServerStore(requireContext()).forget()
        parentFragmentManager.popBackStack()
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

    fun goBack(): Boolean {
        val web = _binding?.web ?: return false
        return if (web.canGoBack()) {
            web.goBack()
            true
        } else {
            false
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

        override fun onPageFinished(view: WebView, url: String) {
            binding.refresh.isRefreshing = false
            if ("/auth/connect/" in url) {
                val base = PairingFragment.pairingBase(url)
                if (base != null) view.loadUrl(PairingFragment.appUrl(base))
            }
        }

        override fun onReceivedError(view: WebView, request: WebResourceRequest, error: WebResourceError) {
            if (request.isForMainFrame) {
                binding.refresh.isRefreshing = false
                Snackbar.make(binding.root, getString(R.string.load_failed), Snackbar.LENGTH_LONG).show()
            }
        }

        override fun onReceivedSslError(view: WebView, handler: SslErrorHandler, error: SslError) {
            val fingerprint = Fingerprint.sha256HexColon(error.certificate)
            val body = if (fingerprint != null) {
                getString(R.string.tls_body, fingerprint)
            } else {
                getString(R.string.tls_body_unknown)
            }
            MaterialAlertDialogBuilder(requireContext())
                .setTitle(R.string.tls_title)
                .setMessage(body)
                .setPositiveButton(R.string.trust_once) { _, _ -> handler.proceed() }
                .setNegativeButton(android.R.string.cancel) { _, _ -> handler.cancel() }
                .setOnCancelListener { handler.cancel() }
                .show()
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

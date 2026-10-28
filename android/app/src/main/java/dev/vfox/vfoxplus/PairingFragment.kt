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

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.fragment.app.setFragmentResultListener
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import com.google.android.material.transition.MaterialSharedAxis
import dev.vfox.vfoxplus.databinding.FragmentPairingBinding

// The native Material 3 Expressive pairing flow: scan the terminal QR code or
// paste the one-time link the PC printed, compare the certificate fingerprint,
// connect.
class PairingFragment : Fragment() {
    private var _binding: FragmentPairingBinding? = null
    private val binding get() = _binding!!

    private val cameraPermission = registerForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        if (granted) {
            openScanner()
        } else {
            Snackbar.make(binding.root, getString(R.string.scan_denied), Snackbar.LENGTH_LONG).show()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enterTransition = MaterialSharedAxis(MaterialSharedAxis.X, true)
        returnTransition = MaterialSharedAxis(MaterialSharedAxis.X, false)
        setFragmentResultListener(ScanFragment.REQUEST_KEY) { _, bundle ->
            val link = bundle.getString(ScanFragment.RESULT_LINK).orEmpty()
            if (link.isNotEmpty()) {
                binding.linkField.setText(link)
                tryConnect(link)
            }
        }
    }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?,
    ): View {
        _binding = FragmentPairingBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        val store = ServerStore(requireContext())
        val last = store.lastPairingUrl
        binding.reconnectCard.isVisible = last != null
        if (last != null) {
            binding.reconnectUrl.text = last
            binding.reconnectButton.setOnClickListener { openViewer(appUrl(last)) }
        }
        binding.connectButton.setOnClickListener {
            tryConnect(binding.linkField.text?.toString().orEmpty())
        }
        binding.scanButton.setOnClickListener { scan() }
        binding.howButton.setOnClickListener { showHowTo() }
    }

    private fun tryConnect(link: String) {
        val store = ServerStore(requireContext())
        val base = pairingBase(link)
        if (base == null) {
            binding.linkLayout.error = getString(R.string.invalid_link)
            return
        }
        binding.linkLayout.error = null
        binding.connectProgress.isVisible = true
        binding.connectButton.isEnabled = false
        store.lastPairingUrl = base
        // The link may be a token URL (it sets the session cookie) or a bare
        // server address (it redirects to the intro page when unpaired).
        openViewer(if ("/auth/connect/" in link) link else appUrl(base))
    }

    private fun scan() {
        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.CAMERA) ==
            PackageManager.PERMISSION_GRANTED
        ) {
            openScanner()
        } else {
            cameraPermission.launch(Manifest.permission.CAMERA)
        }
    }

    private fun openScanner() {
        parentFragmentManager.beginTransaction()
            .setReorderingAllowed(true)
            .replace(R.id.container, ScanFragment())
            .addToBackStack(null)
            .commit()
    }

    private fun openViewer(pairingUrl: String) {
        parentFragmentManager.beginTransaction()
            .setReorderingAllowed(true)
            .replace(R.id.container, ViewerFragment.newInstance(pairingUrl))
            .addToBackStack(null)
            .commit()
    }

    private fun showHowTo() {
        MaterialAlertDialogBuilder(requireContext())
            .setTitle(R.string.how_title)
            .setMessage(R.string.how_body)
            .setPositiveButton(android.R.string.ok, null)
            .show()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }

    companion object {
        // Accepts a full one-time pairing link or a bare server address.
        // Returns the server base (scheme + host + port), or null.
        fun pairingBase(link: String): String? {
            val trimmed = link.trim()
            if (trimmed.isEmpty()) return null
            return try {
                val uri = android.net.Uri.parse(if ("://" in trimmed) trimmed else "http://$trimmed")
                val scheme = uri.scheme?.lowercase() ?: return null
                if (scheme != "http" && scheme != "https") return null
                val host = uri.host ?: return null
                val port = if (uri.port != -1) ":${uri.port}" else ""
                "$scheme://$host$port"
            } catch (_: Exception) {
                null
            }
        }

        fun appUrl(base: String): String = base.trimEnd('/') + "/app"
    }
}

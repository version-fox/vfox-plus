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
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.view.inputmethod.EditorInfo
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.content.res.AppCompatResources
import androidx.core.content.ContextCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.isVisible
import androidx.core.widget.doAfterTextChanged
import androidx.fragment.app.Fragment
import androidx.fragment.app.setFragmentResultListener
import com.google.android.material.color.MaterialColors
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import dev.vfox.vfoxplus.databinding.FragmentPairingBinding

// The native Material 3 Expressive pairing flow: scan the terminal QR code or
// paste the one-time link the PC printed, compare the certificate fingerprint,
// connect.
class PairingFragment : Fragment() {
    private var _binding: FragmentPairingBinding? = null
    private val binding get() = _binding!!

    // The session opens once: the Go key of a hardware keyboard reports itself twice to the field, and a second tap is as
    // quick as the first, and each would put another session on the back stack.
    private var connecting = false

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
        val context = requireContext()
        enterTransition = Motion.sharedAxisX(context, true)
        exitTransition = Motion.sharedAxisX(context, true)
        reenterTransition = Motion.sharedAxisX(context, false)
        returnTransition = Motion.sharedAxisX(context, false)
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
        connecting = false
        binding.scroll.padToSafeArea()
        binding.heroShape.setImageDrawable(heroShape())

        val last = ServerStore(requireContext()).lastPairingUrl
        binding.reconnectCard.isVisible = last != null
        if (last != null) {
            binding.reconnectUrl.text = last.substringAfter("://")
            binding.reconnectButton.setOnClickListener { openViewer(appUrl(last)) }
        }

        binding.copyButton.setOnClickListener { copyCommand() }
        binding.scanButton.setOnClickListener { scan() }
        binding.connectButton.setOnClickListener { tryConnect(binding.linkField.text?.toString().orEmpty()) }
        binding.howButton.setOnClickListener { showHowTo() }

        // The end icon of the field pastes while it is empty and clears when it is not.
        binding.linkField.doAfterTextChanged {
            binding.linkLayout.error = null
            updateLinkEndIcon()
        }
        binding.linkLayout.setEndIconOnClickListener {
            if (binding.linkField.text.isNullOrEmpty()) paste() else binding.linkField.text?.clear()
        }
        binding.linkField.setOnEditorActionListener { _, actionId, _ ->
            if (actionId == EditorInfo.IME_ACTION_GO) {
                tryConnect(binding.linkField.text?.toString().orEmpty())
                true
            } else {
                false
            }
        }
        updateLinkEndIcon()

        if (savedInstanceState == null) enter()
    }

    // The way in: the hero first, then each part a little later, on the springs of
    // the theme.
    private fun enter() {
        binding.hero.springIn(0, 40f)
        binding.title.springIn(70)
        binding.body.springIn(110)
        val steps = listOf(binding.reconnectCard, binding.commandCard, binding.scanButton, binding.linkLayout, binding.connectButton)
        steps.filter { it.isVisible }.forEachIndexed { index, step -> step.springIn(170L + index * 50) }
        binding.howButton.springIn(170L + steps.size * 50)
    }

    // The 9-sided cookie of the shape library, in the primary container.
    private fun heroShape() = PolygonDrawable(
        ExpressiveShapes.cookie9,
        MaterialColors.getColor(requireContext(), com.google.android.material.R.attr.colorPrimaryContainer, 0),
    )

    private fun updateLinkEndIcon() {
        val empty = binding.linkField.text.isNullOrEmpty()
        binding.linkLayout.endIconDrawable = AppCompatResources.getDrawable(
            requireContext(),
            if (empty) R.drawable.ic_paste else R.drawable.ic_close,
        )
        binding.linkLayout.endIconContentDescription = getString(if (empty) R.string.paste else R.string.clear)
    }

    private fun clipboard() = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager

    private fun paste() {
        val text = clipboard().primaryClip?.takeIf { it.itemCount > 0 }?.getItemAt(0)?.coerceToText(requireContext())
        if (!text.isNullOrBlank()) binding.linkField.setText(text.toString().trim())
    }

    private fun copyCommand() {
        clipboard().setPrimaryClip(ClipData.newPlainText(getString(R.string.command_label), getString(R.string.pair_command)))
        // From Android 13 the system confirms a copy itself.
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
            Snackbar.make(binding.root, getString(R.string.command_copied), Snackbar.LENGTH_SHORT).show()
        }
    }

    private fun tryConnect(link: String) {
        if (connecting) return
        val store = ServerStore(requireContext())
        val base = pairingBase(link)
        if (base == null) {
            binding.linkLayout.error = getString(R.string.invalid_link)
            return
        }
        binding.linkLayout.error = null
        connecting = true
        // The link works once: when the user comes back to pair another, the field is empty.
        binding.linkField.text?.clear()
        // The keyboard has done its work, and the page that comes has its own fields.
        WindowCompat.getInsetsController(requireActivity().window, binding.root).hide(WindowInsetsCompat.Type.ime())
        // The loading indicator takes the place of the button while the page loads.
        binding.connectButton.isEnabled = false
        binding.connectButton.visibility = View.INVISIBLE
        binding.connectLoading.isVisible = true
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
            .setIcon(R.drawable.ic_help)
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

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

import android.os.Bundle
import android.util.Size
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.camera.core.Camera
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.core.content.ContextCompat
import androidx.core.os.bundleOf
import androidx.core.view.WindowCompat
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import androidx.fragment.app.setFragmentResult
import com.google.common.util.concurrent.ListenableFuture
import com.google.zxing.BinaryBitmap
import com.google.zxing.DecodeHintType
import com.google.zxing.PlanarYUVLuminanceSource
import com.google.zxing.common.HybridBinarizer
import com.google.zxing.qrcode.QRCodeReader
import dev.vfox.vfoxplus.databinding.FragmentScanBinding
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean

// Scans the terminal QR code that `vfox-plus serve` prints on a non-loopback
// bind. The decoded pairing link goes back to the pairing screen, which
// connects immediately; without a camera the user pastes the link instead.
class ScanFragment : Fragment() {
    private var _binding: FragmentScanBinding? = null
    private val binding get() = _binding!!
    private var camera: Camera? = null
    private var cameraProvider: ListenableFuture<ProcessCameraProvider>? = null
    private var executor: ExecutorService? = null
    private val fired = AtomicBoolean(false)
    private var torchOn = false
    private var lightStatusBars = false

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
        _binding = FragmentScanBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        // The bar above and the toolbar below keep clear of the system bars; the square of
        // the viewfinder lies between them.
        binding.topBar.padToSafeArea(bottom = false)
        binding.bottomBar.padToSafeArea(top = false)
        binding.closeButton.setOnClickListener { parentFragmentManager.popBackStack() }
        binding.manualButton.setOnClickListener { parentFragmentManager.popBackStack() }
        binding.torchButton.addOnCheckedChangeListener { button, checked ->
            button.setIconResource(if (checked) R.drawable.ic_torch_on_fill else R.drawable.ic_torch_off)
            setTorch(checked)
        }
        binding.root.post { placeViewfinder() }
        binding.viewfinder.settleIn()
        startCamera()
    }

    // The camera fills the screen under the status bar, so the clock and the icons are light
    // whatever the theme is (the theme gives dark ones in the light theme, which vanish on the
    // picture). The other screens get theirs back as soon as this one pauses, which is when it
    // starts to leave, not when its exit transition has ended.
    override fun onResume() {
        super.onResume()
        val window = requireActivity().window
        val bars = WindowCompat.getInsetsController(window, window.decorView)
        lightStatusBars = bars.isAppearanceLightStatusBars
        bars.isAppearanceLightStatusBars = false
    }

    override fun onPause() {
        val window = requireActivity().window
        WindowCompat.getInsetsController(window, window.decorView).isAppearanceLightStatusBars = lightStatusBars
        super.onPause()
    }

    // The square is placed between the bar at the top and the hint with the toolbar at the bottom.
    private fun placeViewfinder() {
        _binding?.let {
            it.viewfinder.clearTop = it.topBar.bottom.toFloat()
            it.viewfinder.clearBottom = (it.root.height - it.bottomBar.top).toFloat()
        }
    }

    private fun startCamera() {
        val provider = ProcessCameraProvider.getInstance(requireContext())
        cameraProvider = provider
        executor = Executors.newSingleThreadExecutor()
        provider.addListener({
            val analysis = ImageAnalysis.Builder()
                .setTargetResolution(Size(1280, 720))
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()
            analysis.setAnalyzer(executor!!) { image -> analyze(image) }
            val preview = Preview.Builder().build()
            preview.surfaceProvider = binding.preview.surfaceProvider
            try {
                provider.get().unbindAll()
                val bound = provider.get().bindToLifecycle(
                    viewLifecycleOwner,
                    CameraSelector.DEFAULT_BACK_CAMERA,
                    preview,
                    analysis,
                )
                camera = bound
                binding.torchButton.isVisible = bound.cameraInfo.hasFlashUnit()
            } catch (_: Exception) {
                binding.hint.text = getString(R.string.scan_failed)
            }
        }, ContextCompat.getMainExecutor(requireContext()))
    }

    private fun analyze(image: ImageProxy) {
        try {
            val text = decode(image) ?: return
            if (fired.compareAndSet(false, true)) {
                setFragmentResult(REQUEST_KEY, bundleOf(RESULT_LINK to text))
                parentFragmentManager.popBackStack()
            }
        } finally {
            image.close()
        }
    }

    private fun decode(image: ImageProxy): String? {
        val plane = image.planes[0]
        val width = image.width
        val height = image.height
        val rowStride = plane.rowStride
        val buffer = plane.buffer
        val yuv = ByteArray(width * height)
        if (rowStride == width) {
            buffer.get(yuv)
        } else {
            // Tighten strided rows; the decoder needs contiguous luminance.
            val row = ByteArray(rowStride)
            var offset = 0
            for (y in 0 until height) {
                buffer.get(row)
                System.arraycopy(row, 0, yuv, offset, width)
                offset += width
            }
        }
        val source = PlanarYUVLuminanceSource(yuv, width, height, 0, 0, width, height, false)
        return try {
            QRCodeReader().decode(
                BinaryBitmap(HybridBinarizer(source)),
                mapOf(DecodeHintType.TRY_HARDER to true),
            ).text
        } catch (_: Exception) {
            null
        }
    }

    private fun setTorch(on: Boolean) {
        torchOn = on
        camera?.cameraControl?.enableTorch(on)
    }

    override fun onDestroyView() {
        try {
            cameraProvider?.get()?.unbindAll()
        } catch (_: Exception) {
        }
        executor?.shutdown()
        executor = null
        _binding = null
        super.onDestroyView()
    }

    companion object {
        const val REQUEST_KEY = "scan"
        const val RESULT_LINK = "link"
    }
}

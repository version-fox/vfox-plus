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
import android.graphics.Bitmap
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ScrollView
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.switchmaterial.SwitchMaterial
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import dev.vfox.vfoxplus.R
import org.json.JSONObject

// The settings tab of the native session: this phone, the connections of the
// PC (remote switch, port, pairing code, fingerprint, lightweight mode,
// quit), the application, and the vfox config table.
class SettingsFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var body: LinearLayout
    private var web: SessionModels.Web? = null
    private var settings: SessionModels.Settings? = null
    private var config: List<SessionModels.ConfigEntry> = emptyList()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = parentFragment as SessionHost
        val scroll = ScrollView(requireContext())
        body = LinearLayout(requireContext())
        SessionUi.pad(body)
        scroll.addView(body)
        return scroll
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        refresh()
    }

    override fun refresh() {
        host.runApi(
            task = {
                Triple(
                    host.api.get("${host.base}/api/web"),
                    host.api.get("${host.base}/api/settings"),
                    host.api.get("${host.base}/api/inventory"),
                )
            },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { (webResponse, settingsResponse, inventoryResponse) ->
                        if (webResponse.code != 200 || settingsResponse.code != 200) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        web = SessionModels.web(JSONObject(webResponse.body))
                        settings = SessionModels.settings(JSONObject(settingsResponse.body))
                        config = if (inventoryResponse.code == 200) {
                            SessionModels.inventory(JSONObject(inventoryResponse.body)).config
                        } else {
                            emptyList()
                        }
                        draw()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun putWeb(payload: JSONObject, confirmTitle: Int? = null, confirmBody: String? = null) {
        val send = {
            host.runApi(
                task = { host.api.putJson("${host.base}/api/web", payload.toString()) },
                done = { refresh() },
            )
        }
        if (confirmTitle != null) {
            MaterialAlertDialogBuilder(requireContext())
                .setTitle(confirmTitle)
                .setMessage(confirmBody)
                .setPositiveButton(android.R.string.ok) { _, _ -> send() }
                .setNegativeButton(R.string.cancel, null)
                .show()
        } else {
            send()
        }
    }

    private fun draw() {
        body.removeAllViews()
        val context = requireContext()
        val webState = web ?: return
        // This phone: the card that pairs another link or disconnects.
        body.addView(SessionUi.section(context, getString(R.string.session_this_phone)))
        val phone = SessionUi.card(context)
        val phoneInner = LinearLayout(context)
        SessionUi.pad(phoneInner)
        phoneInner.addView(SessionUi.body(context, "${getString(R.string.session_connected_to)} ${host.base.substringAfter("://")}"))
        val phoneRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val pairNew = MaterialButton(context).apply {
            text = getString(R.string.session_pair_new)
            setOnClickListener { host.pairNew() }
        }
        val disconnect = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = getString(R.string.disconnect)
            setOnClickListener { host.disconnect() }
        }
        phoneRow.addView(pairNew)
        phoneRow.addView(disconnect)
        phoneInner.addView(phoneRow)
        phone.addView(phoneInner)
        body.addView(phone)
        // Connections: the web interface of the PC.
        body.addView(SessionUi.section(context, getString(R.string.session_connections)))
        val connections = SessionUi.card(context)
        val connectionsInner = LinearLayout(context)
        SessionUi.pad(connectionsInner)
        connectionsInner.addView(SessionUi.headline(context, getString(R.string.session_other_devices), 16f))
        val remote = SwitchMaterial(context).apply {
            isChecked = webState.remote
            setOnCheckedChangeListener { _, checked ->
                if (checked && !webState.remote) {
                    putWeb(
                        JSONObject().put("remote", true),
                        R.string.session_other_devices, getString(R.string.session_other_devices_body),
                    )
                } else if (!checked) {
                    putWeb(JSONObject().put("remote", false))
                }
            }
        }
        connectionsInner.addView(remote)
        connectionsInner.addView(SessionUi.headline(context, getString(R.string.session_port), 16f))
        connectionsInner.addView(SessionUi.body(context, getString(R.string.session_port_body)))
        val portRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val portBox = SessionUi.textField(context, getString(R.string.session_port), webState.port.toString())
        portBox.layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        val port = portBox.editText!!
        port.inputType = android.text.InputType.TYPE_CLASS_NUMBER
        val portSave = MaterialButton(context).apply {
            text = getString(R.string.save)
            setOnClickListener {
                putWeb(JSONObject().put("port", port.text.toString().toIntOrNull() ?: webState.port))
            }
        }
        portRow.addView(portBox)
        portRow.addView(portSave)
        connectionsInner.addView(portRow)
        if (webState.error.isNotEmpty()) connectionsInner.addView(SessionUi.body(context, webState.error))
        connectionsInner.addView(SessionUi.headline(context, getString(R.string.session_pair_phone), 16f))
        if (webState.listening && webState.addresses.isNotEmpty()) {
            val address = "https://${webState.addresses.first()}:${webState.port}"
            connectionsInner.addView(SessionUi.body(context, "${getString(R.string.session_reachable)} $address"))
            connectionsInner.addView(SessionUi.body(context, "${getString(R.string.session_fingerprint)} ${webState.fingerprint}"))
            val pairButton = MaterialButton(context).apply {
                text = getString(R.string.session_pair_phone)
                setOnClickListener { showPairing() }
            }
            connectionsInner.addView(pairButton)
        }
        connectionsInner.addView(SessionUi.headline(context, getString(R.string.session_lightweight), 16f))
        connectionsInner.addView(SessionUi.body(context, getString(R.string.session_lightweight_body)))
        val lightweight = SwitchMaterial(context).apply {
            isChecked = webState.lightweight
            setOnCheckedChangeListener { _, checked ->
                putWeb(JSONObject().put("lightweight", checked))
            }
        }
        connectionsInner.addView(lightweight)
        connectionsInner.addView(SessionUi.headline(context, getString(R.string.session_quit), 16f))
        val quit = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = getString(R.string.session_quit)
            setOnClickListener {
                MaterialAlertDialogBuilder(context)
                    .setTitle(R.string.session_quit)
                    .setMessage(R.string.session_quit_body)
                    .setPositiveButton(android.R.string.ok) { _, _ -> quitApp() }
                    .setNegativeButton(R.string.cancel, null)
                    .show()
            }
        }
        connectionsInner.addView(quit)
        connections.addView(connectionsInner)
        body.addView(connections)
        // Application: the settings of the desktop app.
        val app = SessionUi.card(context)
        val appInner = LinearLayout(context)
        SessionUi.pad(appInner)
        appInner.addView(SessionUi.section(context, getString(R.string.session_application)))
        val current = settings
        if (current != null) {
            val executableBox = SessionUi.textField(context, "vfox", current.vfoxExecutable)
            appInner.addView(executableBox)
            val envBox = SessionUi.textField(context, "KEY=value", current.extraEnvironment)
            appInner.addView(envBox)
            val executable = executableBox.editText!!
            val env = envBox.editText!!
            val saveApp = MaterialButton(context).apply {
                text = getString(R.string.save)
                setOnClickListener {
                    host.runApi(
                        task = {
                            host.api.putJson(
                                "${host.base}/api/settings",
                                JSONObject()
                                    .put("vfoxExecutable", executable.text.toString())
                                    .put("extraEnvironment", env.text.toString()).toString(),
                            )
                        },
                        done = { refresh() },
                    )
                }
            }
            appInner.addView(saveApp)
        }
        app.addView(appInner)
        body.addView(app)
        // vfox config table.
        val table = SessionUi.card(context)
        val tableInner = LinearLayout(context)
        SessionUi.pad(tableInner)
        tableInner.addView(SessionUi.section(context, getString(R.string.session_vfox_config)))
        for (entry in config) {
            tableInner.addView(SessionUi.headline(context, entry.key, 15f))
            tableInner.addView(SessionUi.body(context, entry.value.ifBlank { "—" }))
            val edit = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = getString(R.string.session_edit)
                setOnClickListener { editConfig(entry) }
            }
            tableInner.addView(edit)
        }
        table.addView(tableInner)
        body.addView(table)
    }

    private fun quitApp() {
        host.runApi(
            task = { host.api.postJson("${host.base}/api/quit", "{}") },
            done = { host.pairNew() },
        )
    }

    private fun editConfig(entry: SessionModels.ConfigEntry) {
        val box = SessionUi.textField(requireContext(), entry.key, entry.value)
        box.setPadding(48, 16, 48, 0)
        val field = box.editText!!
        MaterialAlertDialogBuilder(requireContext())
            .setTitle(entry.key)
            .setView(box)
            .setPositiveButton(R.string.save) { _, _ ->
                host.runApi(
                    task = {
                        host.api.putJson(
                            "${host.base}/api/config",
                            JSONObject().put("key", entry.key).put("value", field.text.toString()).toString(),
                        )
                    },
                    done = { refresh() },
                )
            }
            .setNegativeButton(R.string.cancel, null)
            .show()
    }

    private fun showPairing() {
        host.runApi(
            task = { host.api.postJson("${host.base}/api/web/pairing", "{}") },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { response ->
                        if (response.code != 200) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        val url = JSONObject(response.body).optString("url", "")
                        if (url.isEmpty()) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        val qr = qrBitmap(url, 640)
                        val image = ImageView(requireContext())
                        if (qr != null) image.setImageBitmap(qr)
                        val copy = MaterialButton(requireContext(), null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                            text = getString(R.string.session_copy_link)
                            setOnClickListener {
                                val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                                clipboard.setPrimaryClip(ClipData.newPlainText("pairing", url))
                            }
                        }
                        val layout = LinearLayout(requireContext())
                        SessionUi.pad(layout)
                        layout.addView(SessionUi.body(requireContext(), getString(R.string.session_pair_help)))
                        layout.addView(SessionUi.body(requireContext(), url))
                        if (qr != null) layout.addView(image)
                        layout.addView(copy)
                        MaterialAlertDialogBuilder(requireContext())
                            .setTitle(R.string.session_pair_phone)
                            .setView(layout)
                            .setPositiveButton(R.string.session_close, null)
                            .show()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun qrBitmap(url: String, size: Int): Bitmap? {
        return try {
            val matrix = QRCodeWriter().encode(url, BarcodeFormat.QR_CODE, size, size)
            Bitmap.createBitmap(size, size, Bitmap.Config.RGB_565).apply {
                for (x in 0 until size) {
                    for (y in 0 until size) {
                        setPixel(x, y, if (matrix[x, y]) 0xFF000000.toInt() else 0xFFFFFFFF.toInt())
                    }
                }
            }
        } catch (_: Exception) {
            null
        }
    }
}

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
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.switchmaterial.SwitchMaterial
import dev.vfox.vfoxplus.R
import org.json.JSONArray
import org.json.JSONObject
import java.net.URLEncoder

// The detail of one SDK: its installed versions with use/uninstall, the
// available versions with install, and its options (variables, legacy files,
// shorthands) — the three tabs of the web page over the same API.
class SdkDetailFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var body: LinearLayout
    private var name: String = ""
    private var tab = 0
    private var info: SessionModels.PluginInfo? = null
    private var installed: List<String> = emptyList()
    private var current: String? = null
    private var versions: List<SessionModels.VersionEntry> = emptyList()
    private var profile: SessionModels.Profile? = null
    private var versionQuery = ""

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        name = requireArguments().getString(ARG_NAME).orEmpty()
    }

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
        val encoded = URLEncoder.encode(name, "UTF-8")
        host.runApi(
            task = {
                val infoResponse = host.api.get("${host.base}/api/sdks/$encoded/info")
                val inventory = SessionModels.inventory(JSONObject(host.api.get("${host.base}/api/inventory").body))
                Triple(infoResponse, inventory, host.api.get("${host.base}/api/sdks/$encoded/profile"))
            },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { (infoResponse, inventory, profileResponse) ->
                        if (infoResponse.code != 200) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        info = SessionModels.pluginInfo(JSONObject(infoResponse.body))
                        val sdk = inventory.installed.firstOrNull { it.name == name }
                        installed = sdk?.versions.orEmpty()
                        current = sdk?.current
                        profile = if (profileResponse.code == 200) {
                            SessionModels.profile(JSONObject(profileResponse.body))
                        } else {
                            null
                        }
                        loadVersions()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun loadVersions() {
        val encoded = URLEncoder.encode(name, "UTF-8")
        val query = if (versionQuery.isBlank()) "" else "?search=${URLEncoder.encode(versionQuery, "UTF-8")}"
        host.runApi(
            task = { host.api.get("${host.base}/api/sdks/$encoded/versions$query") },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { response ->
                        versions = if (response.code == 200) {
                            SessionModels.versions(JSONArray(response.body))
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

    private fun task(operation: String, version: String? = null, extra: JSONObject? = null) {
        host.runApi(
            task = {
                val payload = JSONObject()
                    .put("operation", operation)
                    .put("sdk", name)
                if (version != null) payload.put("version", version)
                if (extra != null) {
                    for (key in extra.keys()) payload.put(key, extra.get(key))
                }
                host.api.postJson("${host.base}/api/tasks", payload.toString())
            },
            done = { result ->
                result.fold(
                    onSuccess = { response ->
                        if (response.code !in 200..299) host.notify(getString(R.string.request_failed))
                        else refresh()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun draw() {
        body.removeAllViews()
        val context = requireContext()
        val back = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = "←"
            setOnClickListener { parentFragmentManager.popBackStack() }
        }
        body.addView(back)
        body.addView(SessionUi.headline(context, name, 26f))
        info?.let {
            body.addView(SessionUi.body(context, it.description))
            body.addView(SessionUi.body(context, it.homepage))
        }
        val tabs = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val labels = listOf(R.string.session_installed, R.string.session_available, R.string.session_options)
        labels.forEachIndexed { index, label ->
            val button = MaterialButton(
                context, null,
                com.google.android.material.R.attr.materialButtonOutlinedStyle,
            ).apply {
                setText(label)
                isCheckable = true
                isChecked = index == tab
                layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                setOnClickListener {
                    tab = index
                    draw()
                    if (index == 1 && versions.isEmpty()) loadVersions()
                }
            }
            tabs.addView(button)
        }
        body.addView(tabs)
        when (tab) {
            0 -> drawInstalled()
            1 -> drawAvailable()
            else -> drawOptions()
        }
    }

    private fun drawInstalled() {
        val context = requireContext()
        if (installed.isEmpty()) {
            body.addView(SessionUi.body(context, "—"))
            return
        }
        for (version in installed) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            val row = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            row.addView(SessionUi.headline(context, version, 16f))
            if (version == current) {
                val tag = MaterialButton(context).apply {
                    text = getString(R.string.session_current)
                    isEnabled = false
                }
                row.addView(tag)
            }
            inner.addView(row)
            val actions = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            if (version != current) {
                val use = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                    text = getString(R.string.session_use_globally)
                    setOnClickListener { task("use", version) }
                }
                actions.addView(use)
            }
            val uninstall = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = getString(R.string.session_uninstall)
                setOnClickListener { task("uninstall", version) }
            }
            actions.addView(uninstall)
            inner.addView(actions)
            card.addView(inner)
            body.addView(card)
        }
    }

    private fun drawAvailable() {
        val context = requireContext()
        val searchRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val field = EditText(context).apply {
            hint = versionQuery.ifBlank { "search" }
            setText(versionQuery)
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        val go = MaterialButton(context).apply {
            text = "→"
            setOnClickListener {
                versionQuery = field.text.toString()
                loadVersions()
            }
        }
        searchRow.addView(field)
        searchRow.addView(go)
        body.addView(searchRow)
        profile?.search?.let { spec ->
            if (spec.choices.size > 1) {
                val choose = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                    val chosen = spec.choices.firstOrNull { it.value == spec.current }?.name?.en ?: spec.current
                    text = chosen
                    setOnClickListener { chooseSearch(spec) }
                }
                body.addView(choose)
            }
        }
        for (entry in versions) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, entry.version, 16f))
            if (entry.hasNote) inner.addView(SessionUi.body(context, entry.note))
            for (addition in entry.additions) {
                inner.addView(SessionUi.body(context, "${addition.name} ${addition.version}"))
            }
            if (!entry.installed) {
                val install = MaterialButton(context).apply {
                    text = getString(R.string.session_install)
                    setOnClickListener { task("install", entry.version) }
                }
                inner.addView(install)
            }
            card.addView(inner)
            body.addView(card)
        }
    }

    private fun chooseSearch(spec: SessionModels.SearchSpec) {
        val names = spec.choices.map { it.name.en.ifBlank { it.value } }.toTypedArray()
        MaterialAlertDialogBuilder(requireContext())
            .setItems(names) { _, which ->
                val chosen = spec.choices[which].value
                host.runApi(
                    task = {
                        host.api.putJson(
                            "${host.base}/api/sdks/${URLEncoder.encode(name, "UTF-8")}/search",
                            JSONObject().put("value", chosen).toString(),
                        )
                    },
                    done = { refresh() },
                )
            }
            .show()
    }

    private fun drawOptions() {
        val context = requireContext()
        val spec = profile ?: return
        if (spec.guideEn.isNotEmpty()) body.addView(SessionUi.body(context, spec.guideEn))
        for (variable in spec.variables) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, variable.label.ifBlank { variable.key }, 15f))
            if (variable.help.isNotEmpty()) inner.addView(SessionUi.body(context, variable.help))
            inner.addView(SessionUi.body(context, variable.value.ifBlank { "Not set" }))
            val change = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = getString(R.string.save)
                setOnClickListener { editVariable(variable) }
            }
            inner.addView(change)
            card.addView(inner)
            body.addView(card)
        }
        if (spec.legacyFiles.isNotEmpty()) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, spec.legacyFiles.joinToString(", "), 15f))
            card.addView(inner)
            body.addView(card)
        }
        for (shorthand in spec.shorthands) {
            body.addView(SessionUi.body(context, "${shorthand.name}: ${shorthand.meaningEn}"))
        }
        for ((title, url) in spec.docs) {
            val link = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = title.ifBlank { url }
            }
            body.addView(link)
        }
    }

    private fun editVariable(variable: SessionModels.Variable) {
        val field = EditText(requireContext()).apply { setText(variable.value) }
        MaterialAlertDialogBuilder(requireContext())
            .setTitle(variable.label.ifBlank { variable.key })
            .setView(field)
            .setPositiveButton(R.string.save) { _, _ ->
                host.runApi(
                    task = {
                        host.api.putJson(
                            "${host.base}/api/sdks/${URLEncoder.encode(name, "UTF-8")}/variables",
                            JSONObject().put("key", variable.key).put("value", field.text.toString()).toString(),
                        )
                    },
                    done = { refresh() },
                )
            }
            .setNegativeButton(R.string.cancel, null)
            .show()
    }

    companion object {
        private const val ARG_NAME = "sdk"

        fun newInstance(name: String) = SdkDetailFragment().apply {
            arguments = Bundle().apply { putString(ARG_NAME, name) }
        }
    }
}

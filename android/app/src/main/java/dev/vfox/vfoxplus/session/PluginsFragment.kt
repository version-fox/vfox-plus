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

import android.net.Uri
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import dev.vfox.vfoxplus.R
import org.json.JSONObject

// The plugins tab of the native session: adding by source or file, and the
// registry to install and update from.
class PluginsFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var body: LinearLayout
    private lateinit var registry: LinearLayout
    private var plugins: List<SessionModels.AvailablePlugin> = emptyList()
    private var query = ""
    private var officialOnly = false

    private val pickFile = registerForActivityResult(ActivityResultContracts.GetContent()) { uri: Uri? ->
        if (uri != null) upload(uri)
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
        host.runApi(
            task = { host.api.get("${host.base}/api/inventory") },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { response ->
                        if (response.code != 200) {
                            host.notify(getString(R.string.request_failed))
                            return@fold
                        }
                        plugins = SessionModels.inventory(JSONObject(response.body)).registry
                        draw()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun draw() {
        body.removeAllViews()
        val context = requireContext()
        val add = SessionUi.card(context)
        val addInner = LinearLayout(context)
        SessionUi.pad(addInner)
        addInner.addView(SessionUi.headline(context, getString(R.string.session_add_plugin), 16f))
        val sourceBox = SessionUi.textField(context, getString(R.string.session_source))
        val aliasBox = SessionUi.textField(context, getString(R.string.session_alias))
        val source = sourceBox.editText!!
        val alias = aliasBox.editText!!
        addInner.addView(sourceBox)
        addInner.addView(aliasBox)
        val addRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        val addButton = MaterialButton(context).apply {
            text = getString(R.string.session_add)
            setOnClickListener {
                val payload = JSONObject().put("source", source.text.toString())
                val aliasText = alias.text.toString()
                if (aliasText.isNotBlank()) payload.put("alias", aliasText)
                host.runApi(
                    task = { host.api.postJson("${host.base}/api/plugins", payload.toString()) },
                    done = {
                        source.text.clear()
                        alias.text.clear()
                        refresh()
                    },
                )
            }
        }
        val fileButton = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = "+"
            setOnClickListener { pickFile.launch("*/*") }
        }
        addRow.addView(addButton)
        addRow.addView(fileButton)
        addInner.addView(addRow)
        add.addView(addInner)
        body.addView(add)

        body.addView(SessionUi.section(context, getString(R.string.session_registry)))
        val updateAll = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = getString(R.string.session_update_all)
            setOnClickListener {
                host.runApi(
                    task = {
                        host.api.postJson(
                            "${host.base}/api/tasks",
                            JSONObject().put("operation", "update").toString(),
                        )
                    },
                    done = { host.notify(getString(R.string.session_update_all)) },
                )
            }
        }
        body.addView(updateAll)
        val search = EditText(context).apply {
            hint = getString(R.string.session_search_sdks)
            setText(query)
            setOnEditorActionListener { view, _, _ ->
                query = (view as EditText).text.toString()
                drawRegistry()
                true
            }
        }
        body.addView(search)
        registry = LinearLayout(context)
        body.addView(registry)
        drawRegistry()
    }

    private fun drawRegistry() {
        registry.removeAllViews()
        val context = requireContext()
        val shown = plugins.filter {
            (query.isBlank() || it.name.contains(query, ignoreCase = true)) &&
                (!officialOnly || it.official)
        }
        for (plugin in shown) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, plugin.name, 16f))
            if (plugin.homepage.isNotEmpty()) inner.addView(SessionUi.body(context, plugin.homepage))
            val install = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = getString(R.string.session_add)
                setOnClickListener {
                    host.runApi(
                        task = {
                            host.api.postJson(
                                "${host.base}/api/plugins",
                                JSONObject().put("source", plugin.name).toString(),
                            )
                        },
                        done = { host.notify(plugin.name) },
                    )
                }
            }
            inner.addView(install)
            card.addView(inner)
            registry.addView(card)
        }
    }

    private fun upload(uri: Uri) {
        host.runApi(
            task = {
                val bytes = requireContext().contentResolver.openInputStream(uri)?.use { it.readBytes() }
                    ?: throw java.io.IOException("cannot read the file")
                val name = uri.lastPathSegment?.substringAfterLast('/').orEmpty().ifBlank { "plugin.zip" }
                host.api.postMultipart("${host.base}/api/plugins", name, bytes, "")
            },
            done = { result ->
                result.fold(
                    onSuccess = { refresh() },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }
}

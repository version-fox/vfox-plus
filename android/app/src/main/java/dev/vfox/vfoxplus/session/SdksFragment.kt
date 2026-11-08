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
import androidx.core.widget.doAfterTextChanged
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import dev.vfox.vfoxplus.R
import org.json.JSONObject

// The SDKs tab of the native session: the installed plugins with their
// versions, into whose detail the rows open.
class SdksFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var swipe: SwipeRefreshLayout
    private lateinit var list: RecyclerView
    private lateinit var search: EditText
    private var sdks: List<SessionModels.InstalledSdk> = emptyList()
    private var query: String = ""

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = parentFragment as SessionHost
        val root = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val searchBox = SessionUi.textField(requireContext(), getString(R.string.session_search_sdks))
        search = searchBox.editText!!
        search.doAfterTextChanged {
            query = it?.toString().orEmpty()
            draw()
        }
        swipe = SwipeRefreshLayout(requireContext())
        list = RecyclerView(requireContext()).apply { layoutManager = LinearLayoutManager(requireContext()) }
        swipe.addView(list)
        swipe.setOnRefreshListener { refresh() }
        root.addView(SessionUi.section(requireContext(), getString(R.string.session_installed)))
        root.addView(searchBox)
        root.addView(swipe, LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f))
        root.let { SessionUi.pad(it, 0, 0) }
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
                        sdks = SessionModels.inventory(JSONObject(response.body)).installed
                        draw()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun draw() {
        val shown = if (query.isBlank()) sdks else sdks.filter { it.name.contains(query, ignoreCase = true) }
        list.adapter = SdkAdapter(shown, { host.openSdkDetail(it) })
    }

    private inner class SdkAdapter(
        private val rows: List<SessionModels.InstalledSdk>,
        private val open: (String) -> Unit,
    ) : RecyclerView.Adapter<SdkAdapter.Holder>() {
        inner class Holder(val card: LinearLayout) : RecyclerView.ViewHolder(card)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): Holder {
            val card = LinearLayout(parent.context).apply { orientation = LinearLayout.HORIZONTAL }
            SessionUi.pad(card)
            return Holder(card)
        }

        override fun getItemCount() = rows.size

        override fun onBindViewHolder(holder: Holder, position: Int) {
            val sdk = rows[position]
            holder.card.removeAllViews()
            holder.card.addView(SessionUi.headline(holder.card.context, sdk.name.firstOrNull()?.uppercase().orEmpty(), 24f))
            val names = LinearLayout(holder.card.context)
            SessionUi.pad(names, 12, 0)
            names.addView(SessionUi.headline(holder.card.context, sdk.name, 17f))
            val sub = if (sdk.versions.isEmpty()) "— · 0 Versions"
            else "${sdk.versions.size} Versions · ${sdk.current.orEmpty()}"
            names.addView(SessionUi.body(holder.card.context, sub))
            holder.card.addView(names)
            holder.card.setOnClickListener { open(sdk.name) }
        }
    }
}

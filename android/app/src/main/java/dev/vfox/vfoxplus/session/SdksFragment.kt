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
import dev.vfox.vfoxplus.Motion
import dev.vfox.vfoxplus.R
import dev.vfox.vfoxplus.springIn
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

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enterTransition = Motion.fadeThrough(requireContext(), true)
        exitTransition = Motion.fadeThrough(requireContext(), false)
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = parentFragment as SessionHost
        val root = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        root.addView(SessionUi.bar(requireContext(), getString(R.string.tab_sdks)))
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
        list.adapter = SdkAdapter(shown, { host.openSdkDetail(it) }, query.isNotBlank())
    }

    private inner class SdkAdapter(
        private val rows: List<SessionModels.InstalledSdk>,
        private val open: (String) -> Unit,
        private val animate: Boolean,
    ) : RecyclerView.Adapter<SdkAdapter.Holder>() {
        inner class Holder(val row: LinearLayout) : RecyclerView.ViewHolder(row)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): Holder {
            val context = parent.context
            val row = LinearLayout(context).apply { orientation = LinearLayout.VERTICAL }
            return Holder(row)
        }

        override fun getItemCount() = rows.size

        override fun onBindViewHolder(holder: Holder, position: Int) {
            val sdk = rows[position]
            holder.row.removeAllViews()
            val sub = if (sdk.versions.isEmpty()) {
                "— · 0 ${holder.row.context.getString(R.string.session_versions)}"
            } else {
                "${sdk.versions.size} ${holder.row.context.getString(R.string.session_versions)} · ${sdk.current.orEmpty()}"
            }
            val row = SessionUi.row(
                holder.row.context,
                title = sdk.name,
                subtitle = sub,
                avatarName = sdk.name,
                trailing = android.widget.ImageView(holder.row.context).apply {
                    setImageResource(dev.vfox.vfoxplus.R.drawable.ic_forward)
                    imageTintList = android.content.res.ColorStateList.valueOf(
                        com.google.android.material.color.MaterialColors.getColor(
                            context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0,
                        ),
                    )
                    contentDescription = null
                },
                onClick = { open(sdk.name) },
            )
            if (animate) row.springIn((position * 30).toLong())
            holder.row.addView(row)
            holder.row.addView(SessionUi.divider(holder.row.context))
        }
    }
}

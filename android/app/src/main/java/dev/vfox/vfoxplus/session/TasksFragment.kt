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
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.core.view.isVisible
import androidx.fragment.app.Fragment
import com.google.android.material.button.MaterialButton
import dev.vfox.vfoxplus.R
import org.json.JSONArray
import org.json.JSONObject

// The tasks tab of the native session: what runs, its log, cancel and
// retry, and clearing what finished.
class TasksFragment : Fragment(), NativeSessionFragment.Refreshable {
    private lateinit var host: SessionHost
    private lateinit var body: LinearLayout
    private var expanded: MutableSet<String> = mutableSetOf()
    private var tasks: List<SessionModels.Task> = emptyList()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        host = parentFragment as SessionHost
        val root = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        root.addView(SessionUi.bar(requireContext(), getString(R.string.tab_tasks)))
        val scroll = ScrollView(requireContext())
        body = LinearLayout(requireContext())
        SessionUi.pad(body)
        scroll.addView(body)
        root.addView(
            scroll,
            LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f),
        )
        return root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        refresh()
    }

    override fun refresh() {
        host.runApi(
            task = { host.api.get("${host.base}/api/tasks") },
            done = { result ->
                if (!isAdded) return@runApi
                result.fold(
                    onSuccess = { response ->
                        tasks = if (response.code == 200) {
                            SessionModels.tasks(JSONArray(response.body))
                        } else {
                            emptyList()
                        }
                        host.setTaskBadge(tasks.count { it.status == "running" || it.status == "queued" })
                        draw()
                    },
                    onFailure = { host.notify(getString(R.string.request_failed)) },
                )
            },
        )
    }

    private fun act(id: String, action: String, done: () -> Unit = {}) {
        host.runApi(
            task = { host.api.postJson("${host.base}/api/tasks/$id/$action", "{}") },
            done = {
                done()
                refresh()
            },
        )
    }

    private fun draw() {
        body.removeAllViews()
        val context = requireContext()
        if (tasks.isEmpty()) {
            body.addView(SessionUi.body(context, getString(R.string.session_tasks_empty)))
        }
        for (task in tasks) {
            val card = SessionUi.card(context)
            val inner = LinearLayout(context)
            SessionUi.pad(inner)
            inner.addView(SessionUi.headline(context, task.commandLine.ifBlank { task.operation }, 15f))
            inner.addView(SessionUi.body(context, task.status + (task.progress?.let { " $it" }.orEmpty())))
            task.lastLine?.let { inner.addView(SessionUi.body(context, it)) }
            task.errorMessage?.let { inner.addView(SessionUi.body(context, it)) }
            val row = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            val toggle = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                text = if (expanded.contains(task.id)) "−" else "+"
                setOnClickListener {
                    if (!expanded.add(task.id)) expanded.remove(task.id)
                    draw()
                }
            }
            row.addView(toggle)
            if (task.status == "running" || task.status == "queued") {
                val cancel = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                    text = getString(R.string.cancel)
                    setOnClickListener { act(task.id, "cancel") }
                }
                row.addView(cancel)
            }
            if (task.status == "failed" || task.status == "cancelled") {
                val retry = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
                    text = getString(R.string.retry)
                    setOnClickListener { act(task.id, "retry") }
                }
                row.addView(retry)
            }
            inner.addView(row)
            if (expanded.contains(task.id)) {
                inner.addView(TextView(context).apply { text = task.log.joinToString("\n") })
            }
            card.addView(inner)
            body.addView(card)
        }
        val clear = MaterialButton(context, null, com.google.android.material.R.attr.materialButtonOutlinedStyle).apply {
            text = getString(R.string.session_clear_finished)
            isVisible = tasks.any { it.status == "succeeded" || it.status == "failed" || it.status == "cancelled" }
            setOnClickListener {
                host.runApi(
                    task = { host.api.delete("${host.base}/api/tasks/finished") },
                    done = { refresh() },
                )
            }
        }
        body.addView(clear)
    }
}

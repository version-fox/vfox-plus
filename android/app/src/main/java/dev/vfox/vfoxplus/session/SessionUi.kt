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

import android.content.Context
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.TextView
import com.google.android.material.card.MaterialCardView
import com.google.android.material.color.MaterialColors

// The small Material kit of the native session: cards, section labels and
// body text in the roles of the theme, so the five tabs look like one app.
object SessionUi {
    fun card(context: Context): MaterialCardView = MaterialCardView(context).apply {
        radius = 28f * context.resources.displayMetrics.density
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT,
        ).apply {
            val gap = (12 * context.resources.displayMetrics.density).toInt()
            setMargins(gap, gap / 2, gap, gap / 2)
        }
    }

    fun section(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_LabelLarge)
        setTextColor(MaterialColors.getColor(context, androidx.appcompat.R.attr.colorPrimary, 0))
    }

    fun headline(context: Context, text: String, size: Float = 20f): TextView = TextView(context).apply {
        this.text = text
        setTextAppearance(
            when {
                size >= 24 -> com.google.android.material.R.style.TextAppearance_Material3_TitleLarge
                size >= 17 -> com.google.android.material.R.style.TextAppearance_Material3_TitleMedium
                else -> com.google.android.material.R.style.TextAppearance_Material3_TitleSmall
            },
        )
    }

    fun body(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyMedium)
    }

    // A Material outlined text field with its hint on the outline.
    fun textField(context: Context, hint: String, text: String = ""): com.google.android.material.textfield.TextInputLayout {
        val layout = com.google.android.material.textfield.TextInputLayout(
            context, null,
            com.google.android.material.R.attr.textInputOutlinedStyle,
        )
        layout.hint = hint
        layout.addView(android.widget.EditText(context).apply { setText(text) })
        return layout
    }

    fun pad(view: LinearLayout, horizontalDp: Int = 16, verticalDp: Int = 12) {        val density = view.resources.displayMetrics.density
        view.setPadding(
            (horizontalDp * density).toInt(), (verticalDp * density).toInt(),
            (horizontalDp * density).toInt(), (verticalDp * density).toInt(),
        )
        view.orientation = LinearLayout.VERTICAL
    }
}

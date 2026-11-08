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
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.card.MaterialCardView
import com.google.android.material.color.MaterialColors
import com.google.android.material.divider.MaterialDivider

// The small Material kit of the native session: an app bar, list rows with
// avatars and chevrons, tonal stat tiles, section labels and body text in
// the roles of the theme, so the five tabs look and move like one app.
object SessionUi {
    fun card(context: Context): MaterialCardView = MaterialCardView(context).apply {
        radius = 28f * context.resources.displayMetrics.density
        layoutParams = LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT,
        ).apply {
            val gap = (12 * context.resources.displayMetrics.density).toInt()
            setMargins(gap, gap / 2, gap / 2, gap / 2)
        }
    }

    // A tonal stat tile: the number large, the label under it, an icon after.
    fun stat(context: Context, count: String, label: String, icon: Int): MaterialCardView {
        val card = card(context).apply {
            setCardBackgroundColor(
                MaterialColors.getColor(context, com.google.android.material.R.attr.colorSecondaryContainer, 0),
            )
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        val inner = LinearLayout(context).apply { orientation = LinearLayout.VERTICAL }
        pad(inner)
        val row = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        val number = TextView(context).apply {
            text = count
            setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_DisplaySmall)
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        val dot = ImageView(context).apply {
            setImageResource(icon)
            imageTintList = android.content.res.ColorStateList.valueOf(
                MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnSecondaryContainer, 0),
            )
            val size = (28 * context.resources.displayMetrics.density).toInt()
            layoutParams = LinearLayout.LayoutParams(size, size)
        }
        row.addView(number)
        row.addView(dot)
        inner.addView(row)
        inner.addView(labelView(context, label))
        card.addView(inner)
        return card
    }

    private fun labelView(context: Context, text: String): TextView = TextView(context).apply {
        this.text = text
        setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_LabelLarge)
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

    fun body(context: Context, text: String = ""): TextView = TextView(context).apply {
        this.text = text
        setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyMedium)
    }

    // A small top app bar with a title; the detail screens add a navigation icon.
    fun bar(context: Context, title: String, subtitle: String? = null): MaterialToolbar =
        MaterialToolbar(context).apply {
            this.title = title
            if (subtitle != null) this.subtitle = subtitle
        }

    // A 40 dp circle with the first letter, coloured by the name like the web avatars.
    fun avatar(context: Context, name: String): TextView {
        val density = context.resources.displayMetrics.density
        val (bg, fg) = AVATARS[(name.hashCode() and Int.MAX_VALUE) % AVATARS.size]
        return TextView(context).apply {
            text = name.firstOrNull()?.uppercase().orEmpty()
            gravity = Gravity.CENTER
            textSize = 20f
            setTypeface(typeface, Typeface.BOLD)
            background = GradientDrawable().apply {
                shape = GradientDrawable.OVAL
                setColor(bg)
            }
            setTextColor(fg)
            val size = (40 * density).toInt()
            layoutParams = LinearLayout.LayoutParams(size, size).apply {
                marginEnd = (16 * density).toInt()
            }
        }
    }

    // One settings-style row: an optional leading icon or avatar, a title with
    // an optional subtitle, and a trailing control; tapping the row taps it too.
    fun row(
        context: Context,
        title: String,
        subtitle: String? = null,
        icon: Int? = null,
        avatarName: String? = null,
        chevron: Boolean = false,
        trailing: View? = null,
        onClick: (() -> Unit)? = null,
    ): LinearLayout {
        val density = context.resources.displayMetrics.density
        val row = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            minimumHeight = (64 * density).toInt()
            setPadding((16 * density).toInt(), (8 * density).toInt(), (16 * density).toInt(), (8 * density).toInt())
            if (onClick != null) {
                isClickable = true
                isFocusable = true
                setOnClickListener { onClick() }
            }
        }
        if (icon != null) {
            row.addView(ImageView(context).apply {
                setImageResource(icon)
                imageTintList = android.content.res.ColorStateList.valueOf(
                    MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0),
                )
                val size = (24 * density).toInt()
                layoutParams = LinearLayout.LayoutParams(size, size).apply {
                    marginEnd = (16 * density).toInt()
                }
            })
        } else if (avatarName != null) {
            row.addView(avatar(context, avatarName))
        }
        val texts = LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        texts.addView(TextView(context).apply {
            text = title
            setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyLarge)
        })
        if (!subtitle.isNullOrEmpty()) {
            texts.addView(TextView(context).apply {
                text = subtitle
                setTextAppearance(com.google.android.material.R.style.TextAppearance_Material3_BodyMedium)
                setTextColor(
                    MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0),
                )
            })
        }
        row.addView(texts)
        if (trailing != null) {
            row.addView(trailing)
        } else if (chevron) {
            row.addView(ImageView(context).apply {
                setImageResource(dev.vfox.vfoxplus.R.drawable.ic_forward)
                imageTintList = android.content.res.ColorStateList.valueOf(
                    MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0),
                )
                val size = (24 * context.resources.displayMetrics.density).toInt()
                layoutParams = LinearLayout.LayoutParams(size, size)
            })
        }
        return row
    }

    fun divider(context: Context): MaterialDivider = MaterialDivider(
        context, null, com.google.android.material.R.attr.materialDividerStyle,
    ).apply {
        dividerInsetStart = (16 * context.resources.displayMetrics.density).toInt()
        dividerInsetEnd = (16 * context.resources.displayMetrics.density).toInt()
    }

    // A certificate fingerprint: monospace, selectable, next to its copy button.
    fun fingerprint(context: Context, value: String, onCopy: () -> Unit): LinearLayout {
        val density = context.resources.displayMetrics.density
        val box = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        val code = TextView(context).apply {
            text = value
            typeface = Typeface.MONOSPACE
            textSize = 12f
            setTextIsSelectable(true)
            setTextColor(
                MaterialColors.getColor(context, com.google.android.material.R.attr.colorOnSurfaceVariant, 0),
            )
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
        }
        val copy = com.google.android.material.button.MaterialButton(
            context, null, com.google.android.material.R.attr.materialIconButtonStyle,
        ).apply {
            setIconResource(dev.vfox.vfoxplus.R.drawable.ic_paste)
            contentDescription = context.getString(dev.vfox.vfoxplus.R.string.session_copy)
            setOnClickListener { onCopy() }
        }
        box.addView(code)
        box.addView(copy)
        box.setPadding(0, (4 * density).toInt(), 0, (4 * density).toInt())
        return box
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

    fun pad(view: LinearLayout, horizontalDp: Int = 16, verticalDp: Int = 12) {
        val density = view.resources.displayMetrics.density
        view.setPadding(
            (horizontalDp * density).toInt(), (verticalDp * density).toInt(),
            (horizontalDp * density).toInt(), (verticalDp * density).toInt(),
        )
        view.orientation = LinearLayout.VERTICAL
    }

    // Tonal pairs the avatar letter sits in, hashed by name like the web avatars.
    private val AVATARS = listOf(
        0xFFFFDBCB.toInt() to 0xFF6B3A12.toInt(),
        0xFFC9F2C7.toInt() to 0xFF0E4D1F.toInt(),
        0xFFD4E3FF.toInt() to 0xFF0A305F.toInt(),
        0xFFE8DEF8.toInt() to 0xFF4A4458.toInt(),
        0xFFB8E8E0.toInt() to 0xFF06302B.toInt(),
        0xFFF8E287.toInt() to 0xFF4C4400.toInt(),
    )
}

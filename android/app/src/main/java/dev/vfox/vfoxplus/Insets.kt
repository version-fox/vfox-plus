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

import android.graphics.Rect
import android.view.View
import androidx.core.graphics.Insets
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.widget.NestedScrollView
import com.google.android.material.textfield.TextInputLayout

// The app draws edge to edge, under the status bar and the navigation bar. What a
// screen must keep clear of is the safe area: the system bars, the display cutout
// and, while it is open, the keyboard.
fun WindowInsetsCompat.safeArea(): Insets = getInsets(
    WindowInsetsCompat.Type.systemBars() or
        WindowInsetsCompat.Type.displayCutout() or
        WindowInsetsCompat.Type.ime(),
)

// Adds the safe area to the padding that the view has, now and whenever the insets change,
// so that no layout needs a fixed offset for the bars. The insets stay available to the
// children. When the keyboard comes up, a scroll view brings the field that has the focus
// into the part that is left: the system does not resize an edge-to-edge window for the
// keyboard, so nothing else does it.
fun View.padToSafeArea(top: Boolean = true, bottom: Boolean = true) {
    val laidOut = Insets.of(paddingLeft, paddingTop, paddingRight, paddingBottom)
    ViewCompat.setOnApplyWindowInsetsListener(this) { view, insets ->
        val safe = insets.safeArea()
        view.setPadding(
            laidOut.left + safe.left,
            laidOut.top + if (top) safe.top else 0,
            laidOut.right + safe.right,
            laidOut.bottom + if (bottom) safe.bottom else 0,
        )
        if (insets.isVisible(WindowInsetsCompat.Type.ime()) && view is NestedScrollView) view.post { view.revealFocus() }
        insets
    }
    ViewCompat.requestApplyInsets(this)
}

// Scrolls the field that has the focus into the part of the view that the keyboard leaves. A
// scroll view takes its whole height for what is visible (the padding for the keyboard is not
// subtracted), so it does not do this by itself. The field is the layout around the text, with
// its label and its help.
private fun NestedScrollView.revealFocus() {
    val focused = findFocus() ?: return
    val field = generateSequence(focused) { it.parent as? View }.firstOrNull { it is TextInputLayout } ?: focused
    val box = Rect(0, 0, field.width, field.height)
    offsetDescendantRectToMyCoords(field, box)
    val margin = (16 * resources.displayMetrics.density).toInt()
    val visibleBottom = height - paddingBottom
    val delta = when {
        box.bottom + margin > visibleBottom -> box.bottom + margin - visibleBottom
        box.top - margin < paddingTop -> box.top - margin - paddingTop
        else -> 0
    }
    if (delta != 0) smoothScrollBy(0, delta)
}

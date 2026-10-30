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

import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.Path
import android.graphics.RectF
import android.util.AttributeSet
import android.view.View
import androidx.dynamicanimation.animation.FloatPropertyCompat
import androidx.dynamicanimation.animation.SpringAnimation
import com.google.android.material.color.MaterialColors
import kotlin.math.min

// What the camera is asked to look at: the picture is dimmed except for a square
// with the large corners of the shape scale (32 dp), and the corners of the square
// are drawn as brackets in the colour of the theme. Starting, the square comes in from
// a little larger on the default spatial spring, and it is the one that overshoots.
class ViewfinderView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyle: Int = 0,
) : View(context, attrs, defStyle) {
    private val density = resources.displayMetrics.density
    private val radius = 32 * density
    private val scrim = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = 0x99000000.toInt() }
    private val bracket = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE
        strokeWidth = 4 * density
        strokeCap = Paint.Cap.ROUND
        color = MaterialColors.getColor(context, com.google.android.material.R.attr.colorPrimaryFixedDim, 0xFFFFFFFF.toInt())
    }
    private val hole = RectF()
    private val usable = RectF()
    private val oval = RectF()
    private val dim = Path()
    private val corners = Path()

    // 1 is the square at rest; the springs of the start begin above it.
    private var scale = 1f
        set(value) {
            field = value
            invalidate()
        }

    // The part of the view in which the square lies, below this much (the bar above
    // it) and above that much (the toolbar below it), set by the screen.
    var clearTop = 0f
        set(value) {
            field = value
            invalidate()
        }
    var clearBottom = 0f
        set(value) {
            field = value
            invalidate()
        }

    fun settleIn() {
        if (Motion.reduced(context)) return
        scale = START_SCALE
        SpringAnimation(this, SCALE).apply {
            spring = Motion.defaultSpatial(context).setFinalPosition(1f)
            start()
        }
    }

    override fun onDraw(canvas: Canvas) {
        usable.set(0f, clearTop, width.toFloat(), height - clearBottom)
        val side = min(usable.width() - 2 * 48 * density, usable.height()) * SIDE_OF_AREA
        val half = side * scale / 2
        hole.set(usable.centerX() - half, usable.centerY() - half, usable.centerX() + half, usable.centerY() + half)

        dim.reset()
        dim.addRect(0f, 0f, width.toFloat(), height.toFloat(), Path.Direction.CW)
        dim.addRoundRect(hole, radius, radius, Path.Direction.CCW)
        canvas.drawPath(dim, scrim)

        // The four corners: an arm of this length along each side, around the arc of the corner.
        val arm = min(side * 0.22f, 56 * density) * scale
        corners.reset()
        corner(hole.left, hole.top, 1, 1, arm)
        corner(hole.right, hole.top, -1, 1, arm)
        corner(hole.right, hole.bottom, -1, -1, arm)
        corner(hole.left, hole.bottom, 1, -1, arm)
        canvas.drawPath(corners, bracket)
    }

    // One corner of the square at (x, y), where sx and sy (1 or -1) point to the inside:
    // down the side, around the arc, along the top or the bottom.
    private fun corner(x: Float, y: Float, sx: Int, sy: Int, arm: Float) {
        oval.set(minOf(x, x + sx * 2 * radius), minOf(y, y + sy * 2 * radius), maxOf(x, x + sx * 2 * radius), maxOf(y, y + sy * 2 * radius))
        corners.moveTo(x, y + sy * (radius + arm))
        corners.lineTo(x, y + sy * radius)
        // The arc begins on the side (left of its centre when the square lies to the right) and ends on the top or bottom.
        corners.arcTo(oval, if (sx > 0) 180f else 0f, if ((sx > 0) == (sy > 0)) 90f else -90f)
        corners.lineTo(x + sx * (radius + arm), y)
    }

    private companion object {
        const val START_SCALE = 1.15f
        const val SIDE_OF_AREA = 0.78f
        val SCALE = object : FloatPropertyCompat<ViewfinderView>("scale") {
            override fun getValue(view: ViewfinderView) = view.scale
            override fun setValue(view: ViewfinderView, value: Float) {
                view.scale = value
            }
        }
    }
}

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

import android.graphics.Canvas
import android.graphics.ColorFilter
import android.graphics.Matrix
import android.graphics.Paint
import android.graphics.Path
import android.graphics.PixelFormat
import android.graphics.Rect
import android.graphics.RectF
import android.graphics.drawable.Drawable
import androidx.graphics.shapes.CornerRounding
import androidx.graphics.shapes.RoundedPolygon
import androidx.graphics.shapes.star
import androidx.graphics.shapes.toPath
import androidx.graphics.shapes.transformed
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.sin

// The shapes of the shape library of Material 3 Expressive that the app draws, built from the public
// API of androidx.graphics:graphics-shapes with the numbers of MaterialShapes (COOKIE_9 and
// SOFT_BURST). MaterialShapes itself is restricted to the library group, and its createShapeDrawable
// draws nothing in Material 1.14.0.
object ExpressiveShapes {
    // A star of nine points, rounded by half, with a point at the top.
    val cookie9: RoundedPolygon by lazy {
        RoundedPolygon.star(numVerticesPerRadius = 9, radius = 1f, innerRadius = 0.8f, rounding = CornerRounding(0.5f))
            .transformed(Matrix().apply { setRotate(-90f) })
    }

    // Ten points, each of two vertices that are turned around the centre ten times.
    val softBurst: RoundedPolygon by lazy {
        val point = floatArrayOf(0.193f, 0.277f, 0.176f, 0.055f)
        val repetitions = 10
        val vertices = FloatArray(point.size * repetitions)
        for (turn in 0 until repetitions) {
            val angle = 2 * PI * turn / repetitions
            for (v in 0 until point.size / 2) {
                val dx = point[2 * v] - 0.5
                val dy = point[2 * v + 1] - 0.5
                val at = (turn * point.size + 2 * v)
                vertices[at] = (0.5 + dx * cos(angle) - dy * sin(angle)).toFloat()
                vertices[at + 1] = (0.5 + dx * sin(angle) + dy * cos(angle)).toFloat()
            }
        }
        RoundedPolygon(vertices, rounding = CornerRounding(0.053f), centerX = 0.5f, centerY = 0.5f)
    }
}

// A shape of that library in one colour,
// scaled into whatever size the view gives it. The empty and error states show their
// icon on such a shape, as the desktop interface does.
class PolygonDrawable(private val polygon: RoundedPolygon, color: Int) : Drawable() {
    private val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply { this.color = color }
    private val path = Path()

    override fun onBoundsChange(bounds: Rect) {
        path.rewind()
        polygon.toPath(path)
        val box = polygon.calculateBounds()
        val into = Matrix()
        into.setRectToRect(RectF(box[0], box[1], box[2], box[3]), RectF(bounds), Matrix.ScaleToFit.CENTER)
        path.transform(into)
    }

    override fun draw(canvas: Canvas) {
        canvas.drawPath(path, paint)
    }

    override fun setAlpha(alpha: Int) {
        paint.alpha = alpha
        invalidateSelf()
    }

    override fun setColorFilter(colorFilter: ColorFilter?) {
        paint.colorFilter = colorFilter
        invalidateSelf()
    }

    @Deprecated("Deprecated in Java")
    override fun getOpacity(): Int = PixelFormat.TRANSLUCENT
}

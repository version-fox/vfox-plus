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

import android.animation.TimeInterpolator
import android.annotation.SuppressLint
import android.content.Context
import android.provider.Settings
import android.view.View
import androidx.dynamicanimation.animation.DynamicAnimation
import androidx.dynamicanimation.animation.SpringAnimation
import androidx.dynamicanimation.animation.SpringForce
import com.google.android.material.motion.MotionUtils
import com.google.android.material.R as MaterialR
import com.google.android.material.transition.MaterialSharedAxis
import kotlin.math.cos
import kotlin.math.exp
import kotlin.math.sin
import kotlin.math.sqrt

// The motion of the app is the spring system of Material 3 Expressive
// (docs/en/dev/design-system.md): the tokens come from the theme (motionSpring*),
// as they do for the components of the library. Spatial springs move, resize and
// reshape, and may overshoot; effects springs change opacity and colour, and never
// do.
object Motion {
    // True when the user turned the animations off: everything jumps to its end.
    fun reduced(context: Context): Boolean =
        Settings.Global.getFloat(context.contentResolver, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f

    // The springs come from the attributes that the theme sets. The style that each one is given as its default is the
    // library's own Expressive spring, which it marks private: it is what MotionUtils falls back to when a theme leaves
    // the attribute out, and a Material that drops it breaks this build instead of the app.
    @SuppressLint("PrivateResource")
    fun defaultSpatial(context: Context): SpringForce = MotionUtils.resolveThemeSpringForce(
        context, MaterialR.attr.motionSpringDefaultSpatial, MaterialR.style.Motion_Material3_Spring_Expressive_Default_Spatial,
    )

    @SuppressLint("PrivateResource")
    fun defaultEffects(context: Context): SpringForce = MotionUtils.resolveThemeSpringForce(
        context, MaterialR.attr.motionSpringDefaultEffects, MaterialR.style.Motion_Material3_Spring_Expressive_Default_Effects,
    )

    @SuppressLint("PrivateResource")
    fun slowEffects(context: Context): SpringForce = MotionUtils.resolveThemeSpringForce(
        context, MaterialR.attr.motionSpringSlowEffects, MaterialR.style.Motion_Material3_Spring_Expressive_Slow_Effects,
    )

    // A transition between two screens that go along the horizontal axis. Page
    // transitions use the slow effects spring: the library's transition takes a
    // duration and a curve, so the spring is sampled into a curve over the time it
    // needs to settle.
    fun sharedAxisX(context: Context, forward: Boolean): MaterialSharedAxis {
        val spring = slowEffects(context)
        val curve = SpringInterpolator(spring.stiffness, spring.dampingRatio)
        return MaterialSharedAxis(MaterialSharedAxis.X, forward).apply {
            duration = if (reduced(context)) 0 else curve.settleMillis
            interpolator = curve
        }
    }
}

// The way in of a view: it rises a little on the default spatial spring while it
// fades in on the default effects spring, [delayMillis] after the others of its group.
fun View.springIn(delayMillis: Long = 0, riseDp: Float = 24f) {
    if (Motion.reduced(context)) return
    val rise = riseDp * resources.displayMetrics.density
    alpha = 0f
    translationY = rise
    postDelayed({
        SpringAnimation(this, DynamicAnimation.TRANSLATION_Y).apply {
            spring = Motion.defaultSpatial(context).setFinalPosition(0f)
            start()
        }
        SpringAnimation(this, DynamicAnimation.ALPHA).apply {
            spring = Motion.defaultEffects(context).setFinalPosition(1f)
            start()
        }
    }, delayMillis)
}

// The step response of a spring of unit mass, from 0 to 1, as a curve over the time
// that the spring needs to settle: until it stays within a thousandth of its end.
class SpringInterpolator(stiffness: Float, private val dampingRatio: Float) : TimeInterpolator {
    private val omega = sqrt(stiffness.toDouble())

    // How long the spring needs to settle, in milliseconds.
    val settleMillis: Long = run {
        var last = 0.0
        var t = 0.0
        while (t < MAX_SECONDS) {
            if (kotlin.math.abs(1.0 - displacement(t)) > TOLERANCE) last = t
            t += STEP_SECONDS
        }
        ((last + STEP_SECONDS) * 1000).toLong()
    }

    override fun getInterpolation(input: Float): Float {
        if (input <= 0f) return 0f
        if (input >= 1f) return 1f
        return displacement(input * settleMillis / 1000.0).toFloat()
    }

    // Where the spring is after t seconds, when it is let go at 0 with its end at 1.
    private fun displacement(t: Double): Double {
        val zeta = dampingRatio.toDouble()
        return when {
            zeta < 1.0 -> {
                val damped = omega * sqrt(1.0 - zeta * zeta)
                1.0 - exp(-zeta * omega * t) * (cos(damped * t) + zeta / sqrt(1.0 - zeta * zeta) * sin(damped * t))
            }
            zeta == 1.0 -> 1.0 - exp(-omega * t) * (1.0 + omega * t)
            else -> {
                val root = sqrt(zeta * zeta - 1.0)
                val slow = -omega * (zeta - root)
                val fast = -omega * (zeta + root)
                1.0 - (fast * exp(slow * t) - slow * exp(fast * t)) / (fast - slow)
            }
        }
    }

    private companion object {
        const val TOLERANCE = 0.001
        const val STEP_SECONDS = 0.002
        const val MAX_SECONDS = 3.0
    }
}

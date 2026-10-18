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

package dev.vfox.plus.ui

import androidx.compose.animation.AnimatedContentTransitionScope
import androidx.compose.animation.ContentTransform
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp

// The transitions of docs/design-system.md#motion, on the springs of the
// expressive motion scheme.

/**
 * Fade through, between destinations and between unrelated content: the old
 * content fades out on the fast effects spring while the new one fades in and
 * grows from 92 % on the slow effects spring.
 */
@Composable
fun <S> fadeThrough(): AnimatedContentTransitionScope<S>.() -> ContentTransform {
    val motion = MaterialTheme.motionScheme
    val fadeOut = motion.fastEffectsSpec<Float>()
    val fadeIn = motion.slowEffectsSpec<Float>()
    return { (fadeIn(fadeIn) + scaleIn(fadeIn, initialScale = 0.92f)) togetherWith fadeOut(fadeOut) }
}

/**
 * The horizontal shared axis, between a list and its detail: forwards slides
 * the new content in from the end, backwards from the start, 30 dp, while the
 * contents cross-fade.
 */
@Composable
fun <S> sharedAxisX(forward: (initial: S, target: S) -> Boolean): AnimatedContentTransitionScope<S>.() -> ContentTransform {
    val motion = MaterialTheme.motionScheme
    val distance = with(LocalDensity.current) { 30.dp.roundToPx() }
    val slide = motion.defaultSpatialSpec<IntOffset>()
    val fade = motion.defaultEffectsSpec<Float>()
    return {
        val direction = if (forward(initialState, targetState)) 1 else -1
        (slideInHorizontally(slide) { direction * distance } + fadeIn(fade)) togetherWith
            (slideOutHorizontally(slide) { -direction * distance } + fadeOut(fade))
    }
}

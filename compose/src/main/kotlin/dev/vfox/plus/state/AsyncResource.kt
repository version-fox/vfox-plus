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

package dev.vfox.plus.state

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/**
 * A value that is loaded asynchronously and can be reloaded. While a reload is
 * in flight the previous [value] stays available, so the user interface keeps
 * showing it with a progress indicator instead of flashing an empty state.
 * When loads overlap, only the latest one wins.
 *
 * The properties are snapshot state, which the user interface observes. They
 * change on the dispatcher of [scope], the main thread of the application.
 */
class AsyncResource<T : Any>(
    private val scope: CoroutineScope,
    private val loader: suspend () -> T,
) {
    /** The last successfully loaded value. */
    var value: T? by mutableStateOf(null)
        private set

    /** Whether a value was loaded at least once. */
    var hasValue: Boolean by mutableStateOf(false)
        private set

    /** The error of the last load, or `null` if it succeeded. */
    var error: Throwable? by mutableStateOf(null)
        private set

    /** Whether a load is in progress. */
    var isLoading: Boolean by mutableStateOf(false)
        private set

    private var generation = 0
    private var inFlight: Job? = null

    /** Loads the value, replacing any load that is still in progress. */
    fun load(): Job {
        val current = ++generation
        isLoading = true
        val job =
            scope.launch {
                val result =
                    try {
                        Result.success(loader())
                    } catch (cancellation: CancellationException) {
                        throw cancellation
                    } catch (failure: Exception) {
                        Result.failure(failure)
                    }
                if (current != generation) return@launch
                result.onSuccess {
                    value = it
                    hasValue = true
                    error = null
                }
                result.onFailure { error = it }
                isLoading = false
            }
        inFlight = job
        return job
    }

    /** Loads the value unless it is loaded or loading already. */
    fun ensureLoaded(): Job? =
        when {
            isLoading -> inFlight
            hasValue -> null
            else -> load()
        }

    /** Replaces the value without loading. */
    fun replace(newValue: T) {
        generation++
        value = newValue
        hasValue = true
        error = null
        isLoading = false
    }
}

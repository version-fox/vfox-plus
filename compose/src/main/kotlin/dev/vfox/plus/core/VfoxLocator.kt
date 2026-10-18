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

package dev.vfox.plus.core

import java.nio.file.Files
import java.nio.file.Path

/** Whether [path] is an existing regular file that may be executed. */
fun isExecutableFile(path: String): Boolean {
    val file = Path.of(path)
    return Files.isRegularFile(file) && Files.isExecutable(file)
}

/**
 * Looks for the vfox executable in `PATH` and in common install locations.
 * Applications started from a desktop launcher do not source the user's shell
 * profile, so directories such as `~/.local/bin` are often missing from their
 * `PATH`; the install locations of the official installers and of package
 * managers are therefore searched as well.
 */
class VfoxLocator(
    private val environment: Map<String, String>,
    private val isExecutable: (String) -> Boolean = ::isExecutableFile,
) {
    /** Directories searched in addition to `PATH`, in order. */
    val fallbackDirectories: List<String>
        get() {
            val home = environment["HOME"]
            return buildList {
                add("/usr/local/bin")
                add("/usr/bin")
                if (home != null) addAll(listOf("$home/.local/bin", "$home/bin", "$home/.linuxbrew/bin"))
                add("/home/linuxbrew/.linuxbrew/bin")
                add("/snap/bin")
            }
        }

    /** All directories that are searched, without duplicates. */
    val searchDirectories: List<String>
        get() = (environment["PATH"].orEmpty().split(':').filter { it.isNotEmpty() } + fallbackDirectories).distinct()

    /** The path of the first vfox executable found, or `null`. */
    fun locate(): String? = searchDirectories.map { "$it/$VFOX_EXECUTABLE_NAME" }.firstOrNull(isExecutable)
}

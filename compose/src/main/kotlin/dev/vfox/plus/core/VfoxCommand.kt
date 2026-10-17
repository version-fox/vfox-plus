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

/** The kinds of state-changing operations the application performs. */
enum class VfoxOperation {
    /** `vfox install --yes <name>@<version>` */
    INSTALL,

    /** `vfox uninstall <name>@<version>` */
    UNINSTALL,

    /** `vfox use --global|--project <name>@<version>` */
    USE,

    /** `vfox unuse --global|--project <name>` */
    UNUSE,

    /** `vfox add [--source <url>] [--alias <alias>] <name>` */
    ADD_PLUGIN,

    /** `vfox remove --yes <name>` */
    REMOVE_PLUGIN,

    /** `vfox update <name>` */
    UPDATE_PLUGIN,

    /** `vfox update --all` */
    UPDATE_ALL_PLUGINS,
}

/**
 * A state-changing vfox invocation. Commands are values: `VfoxCli` builds
 * them, the task queue executes them, and running one again retries it.
 */
data class VfoxCommand(
    val operation: VfoxOperation,
    /** Arguments passed to the vfox executable. */
    val arguments: List<String>,
    /** The plugin the command acts on, if any. */
    val sdk: String? = null,
    /** The SDK version the command acts on, if any. */
    val version: String? = null,
    /**
     * Directory to run in; `null` means a neutral directory, so that no project
     * configuration is picked up.
     */
    val workingDirectory: String? = null,
    /**
     * Whether vfox must believe that it runs inside an activated shell. `vfox
     * use` refuses to run on Unix without shell integration, but with
     * `--global` or `--project` it only edits `.vfox.toml` and symlinks. See
     * docs/vfox-cli-contract.md.
     */
    val requiresHookEnvironment: Boolean = false,
) {
    /** `name@version`, or just the name, or `null`. */
    val subject: String? get() = if (sdk != null && version != null) "$sdk@$version" else sdk

    /** The command line as shown in the task output. */
    val commandLine: String get() = "vfox ${arguments.joinToString(" ")}"
}

/** A vfox invocation failed. */
open class VfoxException(
    message: String,
    /** The arguments vfox was started with. */
    val arguments: List<String> = emptyList(),
    /** The exit code, if the process ran at all. */
    val exitCode: Int? = null,
    val stdout: String = "",
    val stderr: String = "",
) : Exception(message)

/** The vfox executable could not be started. */
class VfoxNotFoundException(
    /** The path or name that could not be executed. */
    val executable: String,
    message: String,
) : VfoxException(message)

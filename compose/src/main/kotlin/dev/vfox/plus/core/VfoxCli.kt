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

import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

/** The name of the vfox executable, looked up in `PATH` when no path is known. */
const val VFOX_EXECUTABLE_NAME = "vfox"

/**
 * Variables that tie a process to an activated shell session. They are removed
 * so that a GUI started from an activated terminal does not act on that
 * terminal's session.
 */
val SESSION_VARIABLES = setOf("__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH", "__VFOX_INITIALIZED")

/** How vfox processes are launched. */
data class VfoxLaunchConfig(
    /** Path to the vfox executable, or a bare name to look up in `PATH`. */
    val executable: String = VFOX_EXECUTABLE_NAME,
    /** The environment inherited from the application. */
    val baseEnvironment: Map<String, String> = emptyMap(),
    /** Variables added by the user, such as mirrors; they override [baseEnvironment]. */
    val extraEnvironment: Map<String, String> = emptyMap(),
    /**
     * Working directory of the commands that are not about a project. vfox reads
     * a project `.vfox.toml` (and legacy files such as `.nvmrc`) from its
     * working directory, so global queries run where none can be: in `/`.
     */
    val neutralDirectory: String = "/",
) {
    /** The complete environment of a vfox child process. */
    fun environment(hookShell: Boolean = false): Map<String, String> {
        val environment = (baseEnvironment + extraEnvironment).filterKeys { it !in SESSION_VARIABLES }.toMutableMap()
        // Disable colours; vfox (through pterm) honours https://no-color.org.
        environment["NO_COLOR"] = "1"
        if (hookShell) environment["__VFOX_SHELL"] = "bash"
        return environment
    }
}

private val namePattern = Regex("^[A-Za-z0-9][A-Za-z0-9._+-]*$")
private val configKeyPattern = Regex("^[A-Za-z][A-Za-z0-9]*(\\.[A-Za-z][A-Za-z0-9]*)*$")

/** Whether [name] is accepted as a plugin name or alias. */
fun isValidPluginName(name: String): Boolean = namePattern.matches(name)

/**
 * Drives the vfox CLI. Read-only queries run directly and are parsed into
 * models; state-changing operations are returned as [VfoxCommand] values, so
 * that the caller can queue them, stream their output and retry them. Every
 * command line built here is documented in docs/vfox-cli-contract.md.
 */
class VfoxCli(
    private val runner: ProcessRunner,
    /** How vfox is launched; replacing it affects the following invocations. */
    @Volatile var launchConfig: VfoxLaunchConfig = VfoxLaunchConfig(),
) {
    /** The version of vfox, for example `1.0.12`. */
    suspend fun version(): String {
        val arguments = listOf("--version")
        val output = query(arguments)
        return parseVfoxVersion("${output.stdout}\n${output.stderr}")
            ?: throw VfoxException("Unrecognised version output", arguments, output.exitCode, output.stdout, output.stderr)
    }

    /** All installed plugins with their installed and current versions (`vfox list` and `vfox current`). */
    suspend fun installedSdks(): List<InstalledSdk> =
        coroutineScope {
            val list = async { query(listOf("list"), allowFailure = true) }
            val currentOutput = async { query(listOf("current")) }
            val current = parseCurrent(currentOutput.await().stdout)
            val listOutput = list.await()
            if (listOutput.exitCode != 0) {
                // `vfox list` fails when no plugin is installed; `vfox current`
                // then prints nothing.
                if (current.isEmpty()) return@coroutineScope emptyList()
                throw failure(listOf("list"), listOutput)
            }
            mergeInstalledSdks(parseInstalledTree(listOutput.stdout), current)
        }

    /** Metadata of the installed plugin [name]. */
    suspend fun pluginInfo(name: String): PluginInfo {
        val arguments = listOf("info", checkName(name), "--format", PLUGIN_INFO_TEMPLATE)
        val output = query(arguments)
        return parsePluginInfo(output.stdout)
            ?: throw VfoxException("Unrecognised plugin information", arguments, output.exitCode, output.stdout, output.stderr)
    }

    /** The directory of an installed version, or `null` if it is not installed. */
    suspend fun installPath(
        name: String,
        version: String,
    ): String? = parseInstallPath(query(listOf("info", "${checkName(name)}@${checkVersion(version)}")).stdout)

    /** Versions of [name] that can be installed, newest first; may take a while. */
    suspend fun searchVersions(name: String): List<AvailableVersion> = parseSearch(query(listOf("search", checkName(name))).stdout)

    /** Plugins listed in the vfox registry. */
    suspend fun availablePlugins(): List<AvailablePlugin> = parseAvailablePlugins(query(listOf("available")).stdout)

    /** The vfox configuration as `key = value` entries. */
    suspend fun configEntries(): List<ConfigEntry> = parseConfigList(query(listOf("config", "--list")).stdout)

    /** Sets the configuration [key] to [value]. */
    suspend fun setConfig(
        key: String,
        value: String,
    ) {
        query(listOf("config", checkConfigKey(key), value))
    }

    /** Resets the configuration [key] to its default value. */
    suspend fun unsetConfig(key: String) {
        query(listOf("config", "--unset", checkConfigKey(key)))
    }

    /** Installs [version] of [name]. */
    fun install(
        name: String,
        version: String,
    ) = VfoxCommand(
        VfoxOperation.INSTALL,
        listOf("install", "--yes", "${checkName(name)}@${checkVersion(version)}"),
        sdk = name,
        version = version,
    )

    /** Uninstalls [version] of [name]. */
    fun uninstall(
        name: String,
        version: String,
    ) = VfoxCommand(
        VfoxOperation.UNINSTALL,
        listOf("uninstall", "${checkName(name)}@${checkVersion(version)}"),
        sdk = name,
        version = version,
    )

    /** Makes [version] of [name] the current version in [scope]; the project scope needs [projectDirectory]. */
    fun use(
        name: String,
        version: String,
        scope: UseScope = UseScope.GLOBAL,
        projectDirectory: String? = null,
    ) = VfoxCommand(
        VfoxOperation.USE,
        listOf("use", scopeFlag(scope), "${checkName(name)}@${checkVersion(version)}"),
        sdk = name,
        version = version,
        workingDirectory = scopeDirectory(scope, projectDirectory),
        requiresHookEnvironment = true,
    )

    /** Removes the current version of [name] from [scope]. */
    fun unuse(
        name: String,
        scope: UseScope = UseScope.GLOBAL,
        projectDirectory: String? = null,
    ) = VfoxCommand(
        VfoxOperation.UNUSE,
        listOf("unuse", scopeFlag(scope), checkName(name)),
        sdk = name,
        workingDirectory = scopeDirectory(scope, projectDirectory),
    )

    /**
     * Adds the plugin [name] from the registry, or from [source] (a URL or a
     * local path) when given; [alias] installs the plugin under another name.
     */
    fun addPlugin(
        name: String,
        source: String? = null,
        alias: String? = null,
    ): VfoxCommand {
        val trimmedSource = source?.trim()?.ifEmpty { null }
        val trimmedAlias = alias?.trim()?.ifEmpty { null }
        val trimmedName = name.trim().ifEmpty { null }
        require(trimmedName != null || trimmedSource != null) { "A plugin name or a source is required" }
        val arguments = mutableListOf("add")
        if (trimmedSource != null) arguments += listOf("--source", trimmedSource)
        if (trimmedAlias != null) arguments += listOf("--alias", checkName(trimmedAlias))
        if (trimmedName != null) arguments += checkName(trimmedName)
        return VfoxCommand(VfoxOperation.ADD_PLUGIN, arguments, sdk = trimmedAlias ?: trimmedName)
    }

    /** Removes the plugin [name] together with all of its installed versions. */
    fun removePlugin(name: String) = VfoxCommand(VfoxOperation.REMOVE_PLUGIN, listOf("remove", "--yes", checkName(name)), sdk = name)

    /** Updates the plugin [name] to its latest version. */
    fun updatePlugin(name: String) = VfoxCommand(VfoxOperation.UPDATE_PLUGIN, listOf("update", checkName(name)), sdk = name)

    /** Updates every installed plugin. */
    fun updateAllPlugins() = VfoxCommand(VfoxOperation.UPDATE_ALL_PLUGINS, listOf("update", "--all"))

    /** Starts [command]; throws [VfoxNotFoundException] if vfox cannot be started. */
    fun start(
        command: VfoxCommand,
        listener: ProcessListener,
    ): RunningProcess {
        val config = launchConfig
        return try {
            runner.start(
                config.executable,
                command.arguments,
                ProcessOptions(
                    workingDirectory = command.workingDirectory ?: config.neutralDirectory,
                    environment = config.environment(hookShell = command.requiresHookEnvironment),
                ),
                listener,
            )
        } catch (error: ProcessStartException) {
            throw VfoxNotFoundException(config.executable, error.message ?: "")
        }
    }

    private suspend fun query(
        arguments: List<String>,
        allowFailure: Boolean = false,
    ): ProcessOutput {
        val config = launchConfig
        val output =
            try {
                runner.run(config.executable, arguments, ProcessOptions(config.neutralDirectory, config.environment()))
            } catch (error: ProcessStartException) {
                throw VfoxNotFoundException(config.executable, error.message ?: "")
            }
        if (output.exitCode != 0 && !allowFailure) throw failure(arguments, output)
        return output
    }

    private fun failure(
        arguments: List<String>,
        output: ProcessOutput,
    ): VfoxException {
        val summary = summarizeError(output.stdout, output.stderr)
        return VfoxException(
            summary.ifEmpty { "vfox exited with code ${output.exitCode}" },
            arguments,
            output.exitCode,
            output.stdout,
            output.stderr,
        )
    }

    private fun scopeFlag(scope: UseScope) = if (scope == UseScope.GLOBAL) "--global" else "--project"

    private fun scopeDirectory(
        scope: UseScope,
        projectDirectory: String?,
    ): String? {
        if (scope == UseScope.GLOBAL) return null
        return requireNotNull(projectDirectory?.trim()?.ifEmpty { null }) { "A project directory is required for the project scope" }
    }

    // Arguments never pass through a shell, so the checks only guard against
    // option injection and typos.
    private fun checkName(name: String): String {
        require(namePattern.matches(name)) { "Not a valid plugin name: $name" }
        return name
    }

    private fun checkVersion(version: String): String {
        require(version.isNotEmpty() && !version.startsWith("-") && version.none { it.isWhitespace() || it == '@' }) {
            "Not a valid version: $version"
        }
        return version
    }

    private fun checkConfigKey(key: String): String {
        require(configKeyPattern.matches(key)) { "Not a valid configuration key: $key" }
        return key
    }
}

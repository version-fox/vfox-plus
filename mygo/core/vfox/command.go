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

package vfox

import "strings"

// VfoxOperation is the kind of state-changing operation the application
// performs:
//
//   - OperationInstall: `vfox install --yes <name>@<version>`
//   - OperationUninstall: `vfox uninstall <name>@<version>`
//   - OperationUse: `vfox use --global|--project <name>@<version>`
//   - OperationUnuse: `vfox unuse --global|--project <name>`
//   - OperationAddPlugin: `vfox add [--source <url>] [--alias <alias>] <name>`
//   - OperationRemovePlugin: `vfox remove --yes <name>`
//   - OperationUpdatePlugin: `vfox update <name>`
//   - OperationUpdateAllPlugins: `vfox update --all`
type VfoxOperation string

// The state-changing operations.
const (
	OperationInstall          VfoxOperation = "install"
	OperationUninstall        VfoxOperation = "uninstall"
	OperationUse              VfoxOperation = "use"
	OperationUnuse            VfoxOperation = "unuse"
	OperationAddPlugin        VfoxOperation = "addPlugin"
	OperationRemovePlugin     VfoxOperation = "removePlugin"
	OperationUpdatePlugin     VfoxOperation = "updatePlugin"
	OperationUpdateAllPlugins VfoxOperation = "updateAllPlugins"
)

// VfoxCommand is a state-changing vfox invocation.
//
// Commands are values: they are built by Client, queued and executed by the
// task queue, and can be retried by running them again.
type VfoxCommand struct {
	// What the command does.
	Operation VfoxOperation
	// Arguments passed to the vfox executable.
	Arguments []string
	// The plugin the command acts on, if any.
	Sdk string
	// The SDK version the command acts on, if any.
	Version string
	// Directory to run in. Empty means a neutral directory, so that no
	// project configuration is picked up.
	WorkingDirectory string
	// Whether vfox must believe that it runs inside an activated shell.
	//
	// `vfox use` refuses to run on Unix without shell integration, but with
	// `--global` or `--project` it only edits `.vfox.toml` and symlinks,
	// which is exactly what a GUI needs. See docs/en/dev/vfox-cli-contract.md.
	RequiresHookEnvironment bool
}

// Subject is the SDK and version as `name@version`, or just the name.
func (c VfoxCommand) Subject() string {
	if c.Sdk != "" && c.Version != "" {
		return c.Sdk + "@" + c.Version
	}
	return c.Sdk
}

// CommandLine is the command line, as shown in the task output.
func (c VfoxCommand) CommandLine() string {
	return "vfox " + strings.Join(c.Arguments, " ")
}

// VfoxError reports that a vfox invocation failed.
type VfoxError struct {
	Message string
	// The arguments vfox was started with.
	Arguments []string
	// The exit code, or -1 if the process never ran.
	ExitCode int
	// Standard output of the failed process.
	Stdout string
	// Standard error of the failed process.
	Stderr string
}

func (e *VfoxError) Error() string { return e.Message }

// VfoxNotFoundError reports that the vfox executable could not be started.
type VfoxNotFoundError struct {
	// The path or name that could not be executed.
	Executable string
	Err        error
}

// Error is the message of the operating system, which names the executable.
func (e *VfoxNotFoundError) Error() string { return e.Err.Error() }

func (e *VfoxNotFoundError) Unwrap() error { return e.Err }

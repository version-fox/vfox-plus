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

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"sync"

	core "github.com/version-fox/vfox-plus/gio/core"
)

// VfoxExecutableName is the name of the vfox executable, looked up in `PATH`
// when no path is known.
const VfoxExecutableName = "vfox"

// SessionVariables tie a process to an activated shell session. They are
// removed so that a GUI started from an activated terminal does not act on
// that terminal's session.
var SessionVariables = []string{"__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH", "__VFOX_INITIALIZED"}

var (
	namePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	configKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$`)
	versionForbidden = regexp.MustCompile(`[\s@]`)
)

// VfoxLaunchConfig is how vfox processes are launched.
type VfoxLaunchConfig struct {
	// Path to the vfox executable, or a bare name to look up in `PATH`.
	Executable string
	// The environment inherited from the application.
	BaseEnvironment map[string]string
	// Variables added by the user, for example download mirrors such as
	// `VFOX_NODEJS_MIRROR`. They override BaseEnvironment.
	ExtraEnvironment map[string]string
	// Working directory for commands that are not tied to a project.
	//
	// vfox reads a project `.vfox.toml` (and legacy files such as `.nvmrc`)
	// from its working directory, so global queries must run somewhere that
	// cannot contain one. The filesystem root is used by default.
	NeutralDirectory string
}

// DefaultLaunchConfig creates a launch configuration; zero values take the
// defaults.
func DefaultLaunchConfig() VfoxLaunchConfig {
	return VfoxLaunchConfig{Executable: VfoxExecutableName, NeutralDirectory: "/"}
}

// VfoxEnvironment is the complete environment of a vfox child process.
func VfoxEnvironment(config VfoxLaunchConfig, hookShell bool) map[string]string {
	environment := make(map[string]string, len(config.BaseEnvironment)+len(config.ExtraEnvironment)+2)
	session := make(map[string]bool, len(SessionVariables))
	for _, key := range SessionVariables {
		session[key] = true
	}
	for key, value := range config.BaseEnvironment {
		if !session[key] {
			environment[key] = value
		}
	}
	for key, value := range config.ExtraEnvironment {
		if !session[key] {
			environment[key] = value
		}
	}
	// Disable colours; vfox (through pterm) honours https://no-color.org.
	environment["NO_COLOR"] = "1"
	if hookShell {
		environment["__VFOX_SHELL"] = "bash"
	}
	return environment
}

// IsValidPluginName reports whether name is accepted as a plugin name or alias.
func IsValidPluginName(name string) bool {
	return namePattern.MatchString(name)
}

// Client drives the vfox CLI.
//
// Read-only queries are executed directly and parsed into models.
// State-changing operations are returned as VfoxCommand values instead, so
// that the caller can queue them, stream their output and retry them.
//
// Every command line built here is documented in
// docs/vfox-cli-contract.md. A Client is safe for use from several
// goroutines.
type Client struct {
	runner core.ProcessRunner
	mu     sync.RWMutex
	config VfoxLaunchConfig
}

// NewClient creates a client on runner with config.
func NewClient(runner core.ProcessRunner, config VfoxLaunchConfig) *Client {
	return &Client{runner: runner, config: cloneLaunchConfig(config)}
}

// LaunchConfig returns how vfox is launched. Its maps are shared with the
// client: change a copy, and hand it to SetLaunchConfig.
func (c *Client) LaunchConfig() VfoxLaunchConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// SetLaunchConfig changes how vfox is launched, for example after the user
// picked a different executable. It affects subsequent invocations only.
func (c *Client) SetLaunchConfig(config VfoxLaunchConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cloneLaunchConfig(config)
}

func cloneLaunchConfig(config VfoxLaunchConfig) VfoxLaunchConfig {
	config.BaseEnvironment = maps.Clone(config.BaseEnvironment)
	config.ExtraEnvironment = maps.Clone(config.ExtraEnvironment)
	return config
}

// Version returns the version of vfox, for example `1.0.12`.
func (c *Client) Version() (string, error) {
	args := []string{"--version"}
	output, err := c.query(args, false)
	if err != nil {
		return "", err
	}
	if version := ParseVfoxVersion(output.Stdout + "\n" + output.Stderr); version != "" {
		return version, nil
	}
	return "", &VfoxError{Message: "Unrecognised version output", Arguments: args, Stdout: output.Stdout, Stderr: output.Stderr}
}

// InstalledSdks returns all installed plugins with their installed and
// current versions.
//
// It runs `vfox list` and `vfox current` and merges their results.
func (c *Client) InstalledSdks() ([]core.InstalledSdk, error) {
	type result struct {
		output core.ProcessOutput
		err    error
	}
	listCh := make(chan result, 1)
	currentCh := make(chan result, 1)
	go func() {
		output, err := c.query([]string{"list"}, true)
		listCh <- result{output, err}
	}()
	go func() {
		output, err := c.query([]string{"current"}, false)
		currentCh <- result{output, err}
	}()
	list := <-listCh
	currentOutput := <-currentCh
	if currentOutput.err != nil {
		return nil, currentOutput.err
	}
	current := ParseCurrent(currentOutput.output.Stdout)
	if list.err != nil {
		return nil, list.err
	}
	if list.output.ExitCode != 0 {
		// `vfox list` fails with "you don't have any sdk installed yet" when
		// no plugin is installed; `vfox current` then prints nothing.
		if len(current) == 0 {
			return nil, nil
		}
		return nil, c.failure([]string{"list"}, list.output)
	}
	return MergeInstalledSdks(ParseInstalledTree(list.output.Stdout), current), nil
}

// PluginInfo returns metadata of the installed plugin name.
func (c *Client) PluginInfo(name string) (core.PluginInfo, error) {
	var info core.PluginInfo
	checked, err := checkName(name)
	if err != nil {
		return info, err
	}
	args := []string{"info", checked, "--format", PluginInfoTemplate}
	output, err := c.query(args, false)
	if err != nil {
		return info, err
	}
	info, ok := ParsePluginInfo(output.Stdout)
	if !ok {
		return info, &VfoxError{Message: "Unrecognised plugin information", Arguments: args, Stdout: output.Stdout, Stderr: output.Stderr}
	}
	return info, nil
}

// InstallPath returns the directory of an installed version, or false if it
// is not installed.
func (c *Client) InstallPath(name, version string) (string, bool, error) {
	checkedName, err := checkName(name)
	if err != nil {
		return "", false, err
	}
	checkedVersion, err := checkVersion(version)
	if err != nil {
		return "", false, err
	}
	output, err := c.query([]string{"info", checkedName + "@" + checkedVersion}, false)
	if err != nil {
		return "", false, err
	}
	path, ok := ParseInstallPath(output.Stdout)
	return path, ok, nil
}

// SearchVersions returns versions of name that can be installed, newest
// first.
//
// It may take a while: plugins fetch the list from the network unless vfox
// has it cached (see the `cache.availableHookDuration` setting).
func (c *Client) SearchVersions(name string) ([]core.AvailableVersion, error) {
	checked, err := checkName(name)
	if err != nil {
		return nil, err
	}
	output, err := c.query([]string{"search", checked}, false)
	if err != nil {
		return nil, err
	}
	return ParseSearch(output.Stdout), nil
}

// AvailablePlugins returns plugins listed in the vfox registry.
func (c *Client) AvailablePlugins() ([]core.AvailablePlugin, error) {
	output, err := c.query([]string{"available"}, false)
	if err != nil {
		return nil, err
	}
	return ParseAvailablePlugins(output.Stdout), nil
}

// ConfigEntries returns the vfox configuration as `key = value` entries.
func (c *Client) ConfigEntries() ([]core.ConfigEntry, error) {
	output, err := c.query([]string{"config", "--list"}, false)
	if err != nil {
		return nil, err
	}
	return ParseConfigList(output.Stdout), nil
}

// SetConfig sets the configuration key to value.
func (c *Client) SetConfig(key, value string) error {
	checked, err := checkConfigKey(key)
	if err != nil {
		return err
	}
	_, err = c.query([]string{"config", checked, value}, false)
	return err
}

// UnsetConfig resets the configuration key to its default value.
func (c *Client) UnsetConfig(key string) error {
	checked, err := checkConfigKey(key)
	if err != nil {
		return err
	}
	_, err = c.query([]string{"config", "--unset", checked}, false)
	return err
}

// Install installs version of name.
func (c *Client) Install(name, version string) (VfoxCommand, error) {
	checkedName, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	checkedVersion, err := checkVersion(version)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{
		Operation: OperationInstall,
		Arguments: []string{"install", "--yes", checkedName + "@" + checkedVersion},
		Sdk:       name,
		Version:   version,
	}, nil
}

// Uninstall uninstalls version of name.
func (c *Client) Uninstall(name, version string) (VfoxCommand, error) {
	checkedName, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	checkedVersion, err := checkVersion(version)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{
		Operation: OperationUninstall,
		Arguments: []string{"uninstall", checkedName + "@" + checkedVersion},
		Sdk:       name,
		Version:   version,
	}, nil
}

// Use makes version of name the current version in scope.
//
// projectDirectory is required for the project scope.
func (c *Client) Use(name, version string, scope core.UseScope, projectDirectory string) (VfoxCommand, error) {
	checkedName, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	checkedVersion, err := checkVersion(version)
	if err != nil {
		return VfoxCommand{}, err
	}
	directory, err := scopeDirectory(scope, projectDirectory)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{
		Operation:               OperationUse,
		Arguments:               []string{"use", scopeFlag(scope), checkedName + "@" + checkedVersion},
		Sdk:                     name,
		Version:                 version,
		WorkingDirectory:        directory,
		RequiresHookEnvironment: true,
	}, nil
}

// Unuse removes the current version of name from scope.
func (c *Client) Unuse(name string, scope core.UseScope, projectDirectory string) (VfoxCommand, error) {
	checked, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	directory, err := scopeDirectory(scope, projectDirectory)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{
		Operation:        OperationUnuse,
		Arguments:        []string{"unuse", scopeFlag(scope), checked},
		Sdk:              name,
		WorkingDirectory: directory,
	}, nil
}

// AddPlugin adds the plugin name from the registry, or from source (a URL or
// a local path) when given. Alias installs the plugin under another name.
func (c *Client) AddPlugin(name, source, alias string) (VfoxCommand, error) {
	trimmedSource := blankToNil(source)
	trimmedAlias := blankToNil(alias)
	trimmedName := blankToNil(name)
	if trimmedName == "" && trimmedSource == "" {
		return VfoxCommand{}, errors.New("A plugin name or a source is required") //nolint:staticcheck // Shown to the user as it is.
	}
	args := []string{"add"}
	if trimmedSource != "" {
		args = append(args, "--source", trimmedSource)
	}
	var sdk string
	if trimmedAlias != "" {
		checked, err := checkName(trimmedAlias)
		if err != nil {
			return VfoxCommand{}, err
		}
		args = append(args, "--alias", checked)
		sdk = trimmedAlias
	}
	if trimmedName != "" {
		checked, err := checkName(trimmedName)
		if err != nil {
			return VfoxCommand{}, err
		}
		args = append(args, checked)
		if sdk == "" {
			sdk = trimmedName
		}
	}
	return VfoxCommand{Operation: OperationAddPlugin, Arguments: args, Sdk: sdk}, nil
}

// RemovePlugin removes the plugin name together with all of its installed
// versions.
func (c *Client) RemovePlugin(name string) (VfoxCommand, error) {
	checked, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{Operation: OperationRemovePlugin, Arguments: []string{"remove", "--yes", checked}, Sdk: name}, nil
}

// UpdatePlugin updates the plugin name to its latest version.
func (c *Client) UpdatePlugin(name string) (VfoxCommand, error) {
	checked, err := checkName(name)
	if err != nil {
		return VfoxCommand{}, err
	}
	return VfoxCommand{Operation: OperationUpdatePlugin, Arguments: []string{"update", checked}, Sdk: name}, nil
}

// UpdateAllPlugins updates every installed plugin.
func (c *Client) UpdateAllPlugins() VfoxCommand {
	return VfoxCommand{Operation: OperationUpdateAllPlugins, Arguments: []string{"update", "--all"}}
}

// Start starts command and returns the running process.
//
// It returns a *VfoxNotFoundError if vfox cannot be started.
func (c *Client) Start(command VfoxCommand, listener core.ProcessListener) (core.RunningProcess, error) {
	config := c.LaunchConfig()
	workingDirectory := command.WorkingDirectory
	if workingDirectory == "" {
		workingDirectory = config.NeutralDirectory
	}
	process, err := c.runner.Start(config.Executable, command.Arguments, core.ProcessOptions{
		WorkingDirectory: workingDirectory,
		Environment:      VfoxEnvironment(config, command.RequiresHookEnvironment),
	}, listener)
	if err != nil {
		var startErr *core.ProcessStartError
		if errors.As(err, &startErr) {
			return nil, &VfoxNotFoundError{Executable: config.Executable, Err: err}
		}
		return nil, err
	}
	return process, nil
}

func (c *Client) query(args []string, allowFailure bool) (core.ProcessOutput, error) {
	config := c.LaunchConfig()
	output, err := c.runner.Run(config.Executable, args, core.ProcessOptions{
		WorkingDirectory: config.NeutralDirectory,
		Environment:      VfoxEnvironment(config, false),
	})
	if err != nil {
		var startErr *core.ProcessStartError
		if errors.As(err, &startErr) {
			return core.ProcessOutput{}, &VfoxNotFoundError{Executable: config.Executable, Err: err}
		}
		return core.ProcessOutput{}, err
	}
	if output.ExitCode != 0 && !allowFailure {
		return core.ProcessOutput{}, c.failure(args, output)
	}
	return output, nil
}

func (c *Client) failure(args []string, output core.ProcessOutput) *VfoxError {
	summary := SummarizeError(output.Stdout, output.Stderr)
	if summary == "" {
		summary = "vfox exited with code " + itoa(output.ExitCode)
	}
	return &VfoxError{Message: summary, Arguments: args, ExitCode: output.ExitCode, Stdout: output.Stdout, Stderr: output.Stderr}
}

func scopeFlag(scope core.UseScope) string {
	if scope == core.UseScopeGlobal {
		return "--global"
	}
	return "--project"
}

func scopeDirectory(scope core.UseScope, projectDirectory string) (string, error) {
	if scope == core.UseScopeGlobal {
		return "", nil
	}
	if directory := blankToNil(projectDirectory); directory != "" {
		return directory, nil
	}
	return "", errors.New("A project directory is required for the project scope") //nolint:staticcheck // Shown to the user as it is.
}

// checkName rejects names that vfox could mistake for flags or that contain
// separators. Arguments never pass through a shell, so this only guards
// against option injection and typos.
func checkName(name string) (string, error) {
	if !IsValidPluginName(name) {
		return "", fmt.Errorf("Not a valid plugin name: %s", name) //nolint:staticcheck // Shown to the user as it is.
	}
	return name, nil
}

func checkVersion(version string) (string, error) {
	if version == "" || strings.HasPrefix(version, "-") || versionForbidden.MatchString(version) {
		return "", fmt.Errorf("Not a valid version: %s", version) //nolint:staticcheck // Shown to the user as it is.
	}
	return version, nil
}

func checkConfigKey(key string) (string, error) {
	if !configKeyPattern.MatchString(key) {
		return "", fmt.Errorf("Not a valid configuration key: %s", key) //nolint:staticcheck // Shown to the user as it is.
	}
	return key, nil
}

func blankToNil(value string) string {
	return strings.TrimSpace(value)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

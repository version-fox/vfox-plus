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
	"slices"
	"strings"
)

// ActivationKind says how an Activation is carried out.
type ActivationKind string

// The kinds of activations.
const (
	// ActivationStartup: the command is a line for the startup file of the shell.
	ActivationStartup ActivationKind = "startup"
	// ActivationOnce: the command is run once in the shell, and changes its startup
	// file itself.
	ActivationOnce ActivationKind = "once"
	// ActivationScript: the command shows a directory, into which the script at URL
	// goes.
	ActivationScript ActivationKind = "script"
)

// Activation is how a shell comes to use the versions that vfox selects, as the
// quick start of vfox says it (<https://vfox.dev/guides/quick-start.html>).
type Activation struct {
	// Shell is the name that `vfox activate` knows the shell by.
	Shell string `json:"shell"`
	// Name is what the shell is called.
	Name string `json:"name"`
	// Kind says what the command is for.
	Kind ActivationKind `json:"kind"`
	// File is the startup file of an ActivationStartup.
	File string `json:"file,omitempty"`
	// Command is the line or the command.
	Command string `json:"command"`
	// URL is the script of an ActivationScript.
	URL string `json:"url,omitempty"`
}

// activations are the shells that `vfox activate` knows, by name.
var activations = map[string]Activation{
	"bash": {Shell: "bash", Name: "Bash", Kind: ActivationStartup, File: "~/.bashrc", Command: `eval "$(vfox activate bash)"`},
	"zsh":  {Shell: "zsh", Name: "Zsh", Kind: ActivationStartup, File: "~/.zshrc", Command: `eval "$(vfox activate zsh)"`},
	"fish": {Shell: "fish", Name: "fish", Kind: ActivationStartup, File: "~/.config/fish/config.fish", Command: "vfox activate fish | source"},
	"pwsh": {Shell: "pwsh", Name: "PowerShell", Kind: ActivationStartup, File: "$PROFILE", Command: `Invoke-Expression "$(vfox activate pwsh)"`},
	"nushell": {
		Shell: "nushell", Name: "Nushell", Kind: ActivationOnce,
		Command: "vfox activate nushell $nu.default-config-dir | save --append $nu.config-path",
	},
	"clink": {
		Shell: "clink", Name: "Clink", Kind: ActivationScript, Command: "clink info | findstr scripts",
		URL: "https://github.com/version-fox/vfox/blob/main/internal/shell/clink_vfox.lua",
	},
}

// Activations are the ways to activate vfox in the shells of an operating system,
// as runtime.GOOS names it. The first is the one to offer: the shell that the
// environment names in SHELL, or the usual one of the system (PowerShell on
// Windows, Zsh on macOS and Bash elsewhere).
func Activations(goos string, environment map[string]string) []Activation {
	var shells []string
	switch goos {
	case "windows":
		shells = []string{"pwsh", "bash", "clink", "nushell"}
	case "darwin":
		shells = []string{"zsh", "bash", "fish", "nushell", "pwsh"}
	default:
		shells = []string{"bash", "zsh", "fish", "nushell", "pwsh"}
	}
	if own := shellName(environment["SHELL"]); slices.Contains(shells, own) {
		shells = slices.Insert(slices.DeleteFunc(shells, func(shell string) bool { return shell == own }), 0, own)
	}
	result := make([]Activation, len(shells))
	for i, shell := range shells {
		result[i] = activations[shell]
	}
	return result
}

// shellName is the name that `vfox activate` knows the shell at path by, such as
// `/usr/bin/nu` for nushell, or what the file is called for any other.
func shellName(path string) string {
	name := path[strings.LastIndexAny(path, `/\`)+1:]
	name = strings.TrimSuffix(strings.ToLower(name), ".exe")
	if name == "nu" {
		return "nushell"
	}
	return name
}

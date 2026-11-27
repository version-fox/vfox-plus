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

package vfox_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

func shells(activations []vfox.Activation) []string {
	var result []string
	for _, activation := range activations {
		result = append(result, activation.Shell)
	}
	return result
}

func TestActivationsOfferTheShellOfThePersonFirst(t *testing.T) {
	for _, test := range []struct {
		goos, shell string
		want        []string
	}{
		{"linux", "", []string{"bash", "zsh", "fish", "nushell", "pwsh"}},
		{"linux", "/usr/bin/zsh", []string{"zsh", "bash", "fish", "nushell", "pwsh"}},
		{"linux", "/usr/local/bin/fish", []string{"fish", "bash", "zsh", "nushell", "pwsh"}},
		{"linux", "/home/user/.cargo/bin/nu", []string{"nushell", "bash", "zsh", "fish", "pwsh"}},
		{"linux", "/bin/dash", []string{"bash", "zsh", "fish", "nushell", "pwsh"}},
		{"freebsd", "", []string{"bash", "zsh", "fish", "nushell", "pwsh"}},
		{"darwin", "", []string{"zsh", "bash", "fish", "nushell", "pwsh"}},
		{"darwin", "/bin/bash", []string{"bash", "zsh", "fish", "nushell", "pwsh"}},
		{"windows", "", []string{"pwsh", "bash", "clink", "nushell"}},
		{"windows", `C:\Program Files\Git\usr\bin\bash.exe`, []string{"bash", "pwsh", "clink", "nushell"}},
		{"windows", "/usr/bin/fish", []string{"pwsh", "bash", "clink", "nushell"}},
	} {
		environment := map[string]string{"SHELL": test.shell}
		if got := shells(vfox.Activations(test.goos, environment)); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s with SHELL=%q: got %v, want %v", test.goos, test.shell, got, test.want)
		}
	}
}

// Every activation says what its kind needs, and the commands are those of the quick
// start of vfox.
func TestActivationsAreComplete(t *testing.T) {
	seen := map[string]vfox.Activation{}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, activation := range vfox.Activations(goos, nil) {
			seen[activation.Shell] = activation
		}
	}
	for shell, activation := range seen {
		if activation.Name == "" || activation.Command == "" {
			t.Errorf("%s: %#v lacks a name or a command", shell, activation)
		}
		if (activation.Kind == vfox.ActivationStartup) != (activation.File != "") {
			t.Errorf("%s: only a line for a startup file names one: %#v", shell, activation)
		}
		if (activation.Kind == vfox.ActivationScript) != strings.HasPrefix(activation.URL, "https://") {
			t.Errorf("%s: only a script has an address: %#v", shell, activation)
		}
		if activation.Kind != vfox.ActivationScript && !strings.Contains(activation.Command, "vfox activate "+shell) {
			t.Errorf("%s: %q activates another shell", shell, activation.Command)
		}
	}
	if got := seen["pwsh"].Command; got != `Invoke-Expression "$(vfox activate pwsh)"` {
		t.Errorf("got %q", got)
	}
}

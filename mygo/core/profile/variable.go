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

package profile

import (
	"fmt"
	"net/url"
	"strings"
)

// Reason is why the value of a variable was refused.
type Reason int

// The reasons.
const (
	// ReasonAddress: the value is not an address that starts with http:// or
	// https://. vfox takes any other value of such a variable for a file on
	// disk, and the download stops with an error that names the variable.
	ReasonAddress Reason = iota + 1
	// ReasonChoice: the value is none of the values the variable takes.
	ReasonChoice
)

// ValueError reports a value that a variable does not take.
type ValueError struct {
	Key    string
	Reason Reason
}

func (e *ValueError) Error() string {
	if e.Reason == ReasonChoice {
		return fmt.Sprintf("%s takes one of its listed values", e.Key)
	}
	return fmt.Sprintf("%s must be an address that starts with http:// or https://", e.Key)
}

// Validate checks a value of the variable. An empty value is fine: it takes the
// variable away, and the plugin uses its default.
func (v Variable) Validate(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if v.Kind == VariableChoice {
		for _, preset := range v.Presets {
			if preset.Value == value {
				return nil
			}
		}
		return &ValueError{Key: v.Key, Reason: ReasonChoice}
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return &ValueError{Key: v.Key, Reason: ReasonAddress}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &ValueError{Key: v.Key, Reason: ReasonAddress}
	}
	return nil
}

// Variable returns the variable of the profile with the given name.
func (p Profile) Variable(key string) (Variable, bool) {
	for _, variable := range p.Variables {
		if variable.Key == key {
			return variable, true
		}
	}
	return Variable{}, false
}

// PresetFor returns the preset that has the value, if there is one.
func (v Variable) PresetFor(value string) (Preset, bool) {
	value = strings.TrimSpace(value)
	for _, preset := range v.Presets {
		if preset.Value == value {
			return preset, true
		}
	}
	return Preset{}, false
}

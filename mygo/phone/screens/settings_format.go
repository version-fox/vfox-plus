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

package screens

import (
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The bounds of the port of the connection of other devices, as the web's field
// takes them.
const (
	settingsMinPort = 1024
	settingsMaxPort = 65535
)

// settingsParseEnvironment reads the extra environment of the settings field: one
// KEY=value per line, split at the first "=". The key loses its spaces at both ends,
// and the value is kept as it is. A line without "=", or with an empty key, is
// skipped. A variable that is written twice keeps its last value.
func settingsParseEnvironment(text string) map[string]string {
	env := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		i := strings.IndexByte(line, '=')
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		if key == "" {
			continue
		}
		env[key] = line[i+1:]
	}
	return env
}

// settingsFormatEnvironment writes the extra environment as the field shows it: one
// KEY=value per line, in the order of the keys. It is the inverse of
// settingsParseEnvironment for the keys and values it can hold.
func settingsFormatEnvironment(env map[string]string) string {
	keys := slices.Sorted(maps.Keys(env))
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+env[key])
	}
	return strings.Join(lines, "\n")
}

// settingsParsePort reads the port that the field holds. It is a whole number from
// settingsMinPort to settingsMaxPort, with the spaces at its ends ignored.
func settingsParsePort(text string) (int, bool) {
	port, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || port < settingsMinPort || port > settingsMaxPort {
		return 0, false
	}
	return port, true
}

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

import "strings"

// Parts is a version name taken apart.
type Parts struct {
	// Version is the name without the distribution or the architecture.
	Version string
	// Source names the distribution the version is of, when the plugin has
	// distributions.
	Source Text
	// SourceValue is the short name of the distribution, such as `tem`.
	SourceValue string
	// Flavors say what else the name tells: `JavaFX`, `x64`, `OpenHarmony`.
	Flavors []string
}

// Describe takes a version name apart the way the plugin puts it together. It
// reports false for a plugin whose names have no structure that is read.
func (p Profile) Describe(name string) (Parts, bool) {
	switch p.Naming {
	case NamingJava:
		return describeJava(name), true
	case NamingFlutter:
		return describeFlutter(name), true
	default:
		return Parts{}, false
	}
}

// describeJava reads `<version>[-<distribution>][-fx]`. A name without a
// distribution is OpenJDK from jdk.java.net, which the plugin names without a
// suffix; `28-ea+19` is such a name, its `-ea` is no distribution.
func describeJava(name string) Parts {
	parts := Parts{Version: name}
	rest := name
	if trimmed, ok := strings.CutSuffix(rest, "-fx"); ok {
		parts.Flavors = append(parts.Flavors, "JavaFX")
		rest = trimmed
	}
	if before, suffix, found := cutLast(rest, "-"); found {
		if choice, ok := javaDistribution(suffix); ok {
			parts.Source, parts.SourceValue = choice.Name, choice.Value
			rest = before
		}
	}
	if parts.SourceValue == "" {
		if choice, ok := javaDistribution(javaOpenJDK); ok {
			parts.Source, parts.SourceValue = choice.Name, choice.Value
		}
	}
	parts.Version = rest
	return parts
}

// describeFlutter reads `<version>[-<architecture>]`, and the `-ohos-` or
// `+ohos-` of an OpenHarmony build, which has no architecture.
func describeFlutter(name string) Parts {
	parts := Parts{Version: name}
	if strings.Contains(name, "-ohos-") || strings.Contains(name, "+ohos-") {
		parts.Flavors = append(parts.Flavors, "OpenHarmony")
		return parts
	}
	if before, suffix, found := cutLast(name, "-"); found {
		for _, architecture := range flutterArchitectures {
			if strings.EqualFold(suffix, architecture) {
				parts.Version = before
				parts.Flavors = append(parts.Flavors, architecture)
				break
			}
		}
	}
	return parts
}

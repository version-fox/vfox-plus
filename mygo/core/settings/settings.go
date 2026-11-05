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

package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
)

// Preferences of the application itself (not of vfox).
//
// The JSON keys are part of the specification in
// docs/en/dev/architecture.md#settings.

// ThemePreference is the light or dark appearance; System follows the desktop.
type ThemePreference string

// The theme preferences.
const (
	ThemeSystem ThemePreference = "system"
	ThemeLight  ThemePreference = "light"
	ThemeDark   ThemePreference = "dark"
)

// LanguagePreference is the user's language choice; System follows the
// desktop locale.
type LanguagePreference string

// The language preferences.
const (
	LanguageSystem            LanguagePreference = "system"
	LanguageEnglish           LanguagePreference = "english"
	LanguageSimplifiedChinese LanguagePreference = "simplifiedChinese"
)

// LanguageCode returns the language code of the translation catalogue, or
// empty for System.
func (p LanguagePreference) LanguageCode() string {
	switch p {
	case LanguageEnglish:
		return "en"
	case LanguageSimplifiedChinese:
		return "zh"
	default:
		return ""
	}
}

// DefaultSeedColor is the brand colour used when the desktop accent colour is
// not used or not available: the orange of the vfox logo.
const DefaultSeedColor = 0xFFF28C1B

// AppSettings are the preferences of the application, persisted as JSON.
type AppSettings struct {
	// Light or dark appearance.
	Theme ThemePreference
	// Whether to derive the colour scheme from the desktop accent colour.
	UseSystemColor bool
	// ARGB seed of the colour scheme when UseSystemColor is off.
	SeedColor uint32
	// The user interface language.
	Language LanguagePreference
	// Path of the vfox executable chosen by the user, or empty to detect it.
	VfoxExecutable string
	// Additional environment variables for vfox, such as mirror settings.
	ExtraEnvironment map[string]string
	// Whether the navigation rail shows labels next to the icons.
	RailExpanded bool
	// Whether the application serves its web interface on this machine while it
	// runs: browsers on this machine pair with it through a loopback server.
	// Off by default.
	WebInterface bool
	// Whether closing the window keeps the application running without it, with
	// the web interface still served, instead of quitting. It only means something
	// while the web interface is on.
	Lightweight bool
	// Whether other devices on the network, such as a phone, may connect to the
	// web interface too, over HTTPS. Off by default, and only while the web
	// interface is on.
	WebRemote bool
	// The port that other devices connect to, or 0 for DefaultWebPort. A port
	// that is not between MinWebPort and MaxWebPort is treated as 0.
	WebPort int
	// The PEM files of the certificate and of the private key that other devices
	// get over HTTPS, or empty for a certificate that the application makes for
	// itself and signs itself.
	WebCertificate, WebKey string
	// What the user chose to list for the plugins whose search takes an
	// argument, such as the distribution of Java: the plugin name, then the
	// value of the choice (`tem`). A plugin that lists its default has no entry.
	Searches map[string]string
	// Keys of the settings file that this version does not know, kept as
	// they are so that settings written by a newer version survive.
	Unknown map[string]interface{}
}

// The port that other devices connect to, when the user did not choose one, and
// the ports that they may choose: those above the ports of the system, which an
// application without privileges cannot listen on.
const (
	DefaultWebPort = 8369
	MinWebPort     = 1024
	MaxWebPort     = 65535
)

// ValidWebPort reports whether port is one that the web interface may listen
// on for other devices.
func ValidWebPort(port int) bool { return port >= MinWebPort && port <= MaxWebPort }

// EffectiveWebPort is the port that other devices connect to.
func (s AppSettings) EffectiveWebPort() int {
	if ValidWebPort(s.WebPort) {
		return s.WebPort
	}
	return DefaultWebPort
}

// DefaultSettings returns the settings of a fresh installation.
func DefaultSettings() AppSettings {
	return AppSettings{
		Theme:            ThemeSystem,
		UseSystemColor:   true,
		SeedColor:        DefaultSeedColor,
		Language:         LanguageSystem,
		ExtraEnvironment: map[string]string{},
		Searches:         map[string]string{},
		Unknown:          map[string]interface{}{},
	}
}

// knownKeys are the keys that this version reads. remoteAccess is the old name
// of webInterface: it is read when webInterface is absent, and no longer written.
var knownKeys = map[string]bool{
	"theme": true, "useSystemColor": true, "seedColor": true, "language": true,
	"vfoxExecutable": true, "extraEnvironment": true, "railExpanded": true,
	"webInterface": true, "lightweight": true, "webRemote": true, "webPort": true,
	"webCertificate": true, "webKey": true, "remoteAccess": true, "searches": true,
}

// Clone returns a copy that shares no map with s, so that it can be modified.
func (s AppSettings) Clone() AppSettings {
	s.ExtraEnvironment = maps.Clone(s.ExtraEnvironment)
	s.Searches = maps.Clone(s.Searches)
	s.Unknown = maps.Clone(s.Unknown)
	return s
}

// SettingsFromJSON restores settings from decoded JSON, ignoring malformed
// values.
func SettingsFromJSON(data interface{}) AppSettings {
	settings := DefaultSettings()
	settings.ExtraEnvironment = map[string]string{}
	settings.Searches = map[string]string{}
	settings.Unknown = map[string]interface{}{}
	object, ok := data.(map[string]interface{})
	if !ok {
		return settings
	}
	if theme, ok := object["theme"].(string); ok {
		switch ThemePreference(theme) {
		case ThemeSystem, ThemeLight, ThemeDark:
			settings.Theme = ThemePreference(theme)
		}
	}
	if use, ok := object["useSystemColor"].(bool); ok {
		settings.UseSystemColor = use
	}
	// JSON numbers decode as float64; a boolean is not a colour.
	if seed, ok := object["seedColor"].(float64); ok && seed == float64(uint32(seed)) && seed >= 0 {
		settings.SeedColor = uint32(seed)
	}
	if language, ok := object["language"].(string); ok {
		switch LanguagePreference(language) {
		case LanguageSystem, LanguageEnglish, LanguageSimplifiedChinese:
			settings.Language = LanguagePreference(language)
		}
	}
	if executable, ok := object["vfoxExecutable"].(string); ok && executable != "" {
		settings.VfoxExecutable = executable
	}
	if environment, ok := object["extraEnvironment"].(map[string]interface{}); ok {
		for key, value := range environment {
			if text, ok := value.(string); ok {
				settings.ExtraEnvironment[key] = text
			}
		}
	}
	if searches, ok := object["searches"].(map[string]interface{}); ok {
		for plugin, value := range searches {
			if text, ok := value.(string); ok && text != "" {
				settings.Searches[plugin] = text
			}
		}
	}
	if expanded, ok := object["railExpanded"].(bool); ok && expanded {
		settings.RailExpanded = true
	}
	if web, ok := object["webInterface"].(bool); ok {
		settings.WebInterface = web
	} else if legacy, ok := object["remoteAccess"].(bool); ok {
		settings.WebInterface = legacy
	}
	if lightweight, ok := object["lightweight"].(bool); ok {
		settings.Lightweight = lightweight
	}
	if remote, ok := object["webRemote"].(bool); ok {
		settings.WebRemote = remote
	}
	// JSON numbers decode as float64.
	if port, ok := object["webPort"].(float64); ok && port == float64(int(port)) && ValidWebPort(int(port)) {
		settings.WebPort = int(port)
	}
	if certificate, ok := object["webCertificate"].(string); ok {
		settings.WebCertificate = certificate
	}
	if key, ok := object["webKey"].(string); ok {
		settings.WebKey = key
	}
	for key, value := range object {
		if !knownKeys[key] {
			settings.Unknown[key] = value
		}
	}
	return settings
}

// settingsJSON is the serialised form, with the keys in the order of the
// specification. encoding/json sorts map keys, so the order is written by
// hand.
func settingsJSON(settings AppSettings) string {
	environment, err := json.Marshal(settings.ExtraEnvironment)
	if err != nil {
		environment = []byte("{}")
	}
	unknown, err := json.Marshal(settings.Unknown)
	if err != nil || string(unknown) == "null" {
		unknown = []byte("{}")
	}
	var out strings.Builder
	out.WriteString("{")
	fmt.Fprintf(&out, "%q:%q,", "theme", string(settings.Theme))
	fmt.Fprintf(&out, "%q:%v,", "useSystemColor", settings.UseSystemColor)
	fmt.Fprintf(&out, "%q:%d,", "seedColor", settings.SeedColor)
	fmt.Fprintf(&out, "%q:%q", "language", string(settings.Language))
	if settings.VfoxExecutable != "" {
		quoted, _ := json.Marshal(settings.VfoxExecutable)
		fmt.Fprintf(&out, ",%q:%s", "vfoxExecutable", quoted)
	}
	fmt.Fprintf(&out, ",%q:%s", "extraEnvironment", environment)
	fmt.Fprintf(&out, ",%q:%v", "railExpanded", settings.RailExpanded)
	fmt.Fprintf(&out, ",%q:%v", "webInterface", settings.WebInterface)
	fmt.Fprintf(&out, ",%q:%v", "lightweight", settings.Lightweight)
	fmt.Fprintf(&out, ",%q:%v", "webRemote", settings.WebRemote)
	if ValidWebPort(settings.WebPort) {
		fmt.Fprintf(&out, ",%q:%d", "webPort", settings.WebPort)
	}
	for _, field := range []struct{ key, value string }{{"webCertificate", settings.WebCertificate}, {"webKey", settings.WebKey}} {
		if field.value != "" {
			quoted, _ := json.Marshal(field.value)
			fmt.Fprintf(&out, ",%q:%s", field.key, quoted)
		}
	}
	searches, err := json.Marshal(settings.Searches)
	if err != nil || string(searches) == "null" {
		searches = []byte("{}")
	}
	fmt.Fprintf(&out, ",%q:%s", "searches", searches)
	for _, key := range sortedKeys(settings.Unknown) {
		quoted, err := json.Marshal(settings.Unknown[key])
		if err != nil {
			continue
		}
		name, _ := json.Marshal(key)
		fmt.Fprintf(&out, ",%s:%s", name, quoted)
	}
	_ = unknown
	out.WriteString("}")
	return out.String()
}

func sortedKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// MarshalJSON serialises the settings, with the keys in the order of the
// specification.
func (s AppSettings) MarshalJSON() ([]byte, error) {
	return []byte(settingsJSON(s)), nil
}

// Equal reports whether two settings are equal, comparing environments by
// value. The unknown keys compare by their JSON encoding.
func (s AppSettings) Equal(other AppSettings) bool {
	if s.Theme != other.Theme || s.UseSystemColor != other.UseSystemColor ||
		s.SeedColor != other.SeedColor || s.Language != other.Language ||
		s.VfoxExecutable != other.VfoxExecutable || s.RailExpanded != other.RailExpanded ||
		s.WebInterface != other.WebInterface || s.Lightweight != other.Lightweight ||
		s.WebRemote != other.WebRemote || s.EffectiveWebPort() != other.EffectiveWebPort() ||
		s.WebCertificate != other.WebCertificate || s.WebKey != other.WebKey ||
		len(s.ExtraEnvironment) != len(other.ExtraEnvironment) || len(s.Searches) != len(other.Searches) {
		return false
	}
	for key, value := range s.ExtraEnvironment {
		if other.ExtraEnvironment[key] != value {
			return false
		}
	}
	for key, value := range s.Searches {
		if other.Searches[key] != value {
			return false
		}
	}
	a, errA := json.Marshal(s.Unknown)
	b, errB := json.Marshal(other.Unknown)
	return errA == nil && errB == nil && string(a) == string(b)
}

// ParsedEnvironment is the result of ParseEnvironmentLines.
type ParsedEnvironment struct {
	// The variables of the valid lines.
	Variables map[string]string
	// The 1-based numbers of the lines that could not be parsed.
	InvalidLines []int
}

func isEnvironmentKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		ok := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9'
		if !ok {
			return false
		}
	}
	return true
}

// ParseEnvironmentLines parses `KEY=VALUE` lines into environment variables.
//
// Blank lines and lines starting with `#` are ignored. Keys must be valid
// POSIX variable names.
func ParseEnvironmentLines(text string) ParsedEnvironment {
	result := ParsedEnvironment{Variables: map[string]string{}}
	for index, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || !isEnvironmentKey(key) {
			result.InvalidLines = append(result.InvalidLines, index+1)
			continue
		}
		result.Variables[key] = strings.TrimSpace(value)
	}
	return result
}

// FormatEnvironmentLines formats environment variables as `KEY=VALUE` lines,
// sorted by key.
func FormatEnvironmentLines(variables map[string]string) string {
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+variables[key])
	}
	return strings.Join(lines, "\n")
}

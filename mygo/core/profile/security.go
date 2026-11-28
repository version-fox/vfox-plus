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

// What a profile may not say. A profile is data that can come from a plugin or
// from the network, and the application puts parts of it on command lines, into
// the environment of vfox and behind links, so every file is held to these
// rules, the application's own too.

package profile

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// The sizes that a profile file and its parts may have.
const (
	maxFile    = 256 << 10
	maxKey     = 128
	maxValue   = 2048
	maxName    = 128
	maxText    = 4000
	maxPattern = 512
)

var variableKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// reservedVariables are variables that change what a program that vfox or a
// plugin runs does, beyond the settings of a plugin: where programs are found
// and what they load, the shell, the trust in servers, compilers and build
// tools, and credentials. A profile may not offer them; the user can set them
// among the environment variables of the settings. Their names are compared in
// upper case, as Windows does.
var reservedVariables = map[string]bool{
	"PATH": true, "PATHEXT": true, "HOME": true, "USERPROFILE": true, "HOMEDRIVE": true, "HOMEPATH": true,
	"APPDATA": true, "LOCALAPPDATA": true, "PROGRAMDATA": true, "SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true,
	"PSMODULEPATH": true, "SHELL": true, "ENV": true, "BASH_ENV": true, "ZDOTDIR": true, "IFS": true,
	"PROMPT_COMMAND": true, "PS4": true, "SHELLOPTS": true, "BASHOPTS": true, "TMPDIR": true, "TEMP": true, "TMP": true,
	"XDG_CONFIG_HOME": true, "XDG_DATA_HOME": true, "XDG_CACHE_HOME": true, "XDG_RUNTIME_DIR": true,
	"VFOX_HOME":    true,
	"NODE_OPTIONS": true, "NODE_PATH": true, "NODE_EXTRA_CA_CERTS": true, "NODE_TLS_REJECT_UNAUTHORIZED": true,
	"JAVA_TOOL_OPTIONS": true, "_JAVA_OPTIONS": true, "JDK_JAVA_OPTIONS": true, "CLASSPATH": true,
	"PYTHONPATH": true, "PYTHONHOME": true, "PYTHONSTARTUP": true, "PERL5LIB": true, "PERL5OPT": true,
	"RUBYOPT": true, "RUBYLIB": true, "GEM_HOME": true, "GEM_PATH": true,
	"GOFLAGS": true, "GOSUMDB": true, "GONOSUMDB": true, "GONOSUMCHECK": true, "GOINSECURE": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "FTP_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
	"SSL_CERT_FILE": true, "SSL_CERT_DIR": true, "CURL_CA_BUNDLE": true, "REQUESTS_CA_BUNDLE": true,
	"SSH_AUTH_SOCK": true, "SSH_ASKPASS": true, "SUDO_ASKPASS": true, "GITHUB_TOKEN": true, "GH_TOKEN": true,
	"CC": true, "CXX": true, "CPP": true, "LD": true, "AR": true, "MAKE": true, "MAKEFLAGS": true, "MFLAGS": true,
	"CFLAGS": true, "CXXFLAGS": true, "CPPFLAGS": true, "LDFLAGS": true,
	"EDITOR": true, "VISUAL": true, "PAGER": true, "BROWSER": true,
}

// reservedPrefixes start the names of variables that are reserved too: the
// dynamic loader, the session of vfox, git and npm.
var reservedPrefixes = []string{"LD_", "DYLD_", "__VFOX_", "GIT_", "BASH_FUNC_", "NPM_CONFIG_"}

// IsReservedVariable reports whether a profile may not offer the variable.
func IsReservedVariable(key string) bool {
	upper := strings.ToUpper(key)
	if reservedVariables[upper] {
		return true
	}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

// variableKeyProblem says why a profile may not offer the variable, empty when
// it may.
func variableKeyProblem(key string) string {
	switch {
	case len(key) > maxKey || !variableKey.MatchString(key):
		return "is not the name of an environment variable"
	case IsReservedVariable(key):
		return "is reserved: a profile may not set it"
	}
	return ""
}

// isPlainValue reports whether value is one line of text that fits.
func isPlainValue(value string) bool {
	return len(value) <= maxValue && !strings.ContainsFunc(value, unicode.IsControl)
}

// isArgument reports whether value can stand on a command line of vfox as one
// argument: it does not start like an option, and has no space.
func isArgument(value string) bool {
	return value != "" && len(value) <= maxName && !strings.HasPrefix(value, "-") &&
		!strings.ContainsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
}

// isSecureAddress reports whether address is an https address with a host.
func isSecureAddress(address string) bool {
	if len(address) > maxValue || strings.ContainsFunc(address, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return false
	}
	parsed, err := url.Parse(address)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

var pluginName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

// isPluginName reports whether name can name a plugin, and a file of its
// profile.
func isPluginName(name string) bool {
	return pluginName.MatchString(name) && !strings.Contains(name, "..")
}

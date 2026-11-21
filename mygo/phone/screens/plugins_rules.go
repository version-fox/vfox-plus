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

// The rules of the plugin tab, as pure functions: how the source of a plugin is
// shown, which of its addresses may be opened, and which plugins a search and a
// filter keep, as the web's plugins.ts and links.ts decide them.

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// The filters of the plugin registry.
const (
	pluginFilterAll       = "all"
	pluginFilterOfficial  = "official"
	pluginFilterCommunity = "community"
)

// The patterns of pluginShortSource, which are the web's.
var (
	pluginGitHub       = regexp.MustCompile(`(?i)^https?://github\.com/([^/?#]+)/([^/?#]+)`)
	pluginGitSuffix    = regexp.MustCompile(`(?i)\.git$`)
	pluginSchemePrefix = regexp.MustCompile(`(?i)^https?://`)
)

// pluginShortSource is the source of a plugin as its owner names it: github.com's
// owner and repository for a GitHub homepage, such as "owner/repo", and the address
// without its scheme and its last slash for any other.
func pluginShortSource(homepage string) string {
	if match := pluginGitHub.FindStringSubmatch(homepage); match != nil {
		return match[1] + "/" + pluginGitSuffix.ReplaceAllString(match[2], "")
	}
	return strings.TrimSuffix(pluginSchemePrefix.ReplaceAllString(homepage, ""), "/")
}

// pluginWebAddress returns the homepage as an address to open, and false when it is
// not an absolute http or https address. A plugin names its homepage in its own
// metadata, so any other scheme, such as javascript: or file:, must never be opened.
func pluginWebAddress(homepage string) (string, bool) {
	address, err := url.Parse(availTrim(homepage))
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.Host == "" {
		return "", false
	}
	return address.String(), true
}

// pluginMatches reports whether a plugin of the registry is kept by the filter, which
// is all, official or community, and by the search q, which is in lower case: the
// name or the homepage must contain it.
func pluginMatches(plugin remote.AvailablePlugin, filter, q string) bool {
	switch filter {
	case pluginFilterOfficial:
		if !plugin.Official {
			return false
		}
	case pluginFilterCommunity:
		if plugin.Official {
			return false
		}
	}
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(plugin.Name), q) ||
		strings.Contains(strings.ToLower(plugin.Homepage), q)
}

// pluginInstalled returns the names of the plugins that are installed, as the
// inventory has them.
func pluginInstalled(installed []remote.InstalledSdk) map[string]bool {
	names := make(map[string]bool, len(installed))
	for _, sdk := range installed {
		names[sdk.Name] = true
	}
	return names
}

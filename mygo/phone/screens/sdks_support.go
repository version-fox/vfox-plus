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

// The helpers that the SDK tab and the SDK page share: the states of the
// inventory, the requests that change the computer, and the texts of the plugins
// and of their errors.

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// sdkPageSize is how many rows of a long list the screen builds at first: the
// plugins, the versions, the variables and the aliases. The rest come with a button
// that shows more, so that the screen holds no more than a page of them.
const sdkPageSize = 50

// sdkChinese reports whether the interface is in Chinese, which picks the texts of
// the plugins (remote.Text) in the language of the interface.
func sdkChinese() bool {
	return i18n.Current() == i18n.Chinese
}

// sdkText is the text of a plugin's message in the language of the interface.
func sdkText(text remote.Text) string {
	return text.In(sdkChinese())
}

// sdkJoin joins the texts that are not empty with sep, as the web interface does
// for its lines of details.
func sdkJoin(sep string, texts ...string) string {
	kept := make([]string, 0, len(texts))
	for _, text := range texts {
		if text != "" {
			kept = append(kept, text)
		}
	}
	return strings.Join(kept, sep)
}

// sdkFindInstalled returns the plugin called name among the installed ones, and
// false when there is none.
func sdkFindInstalled(installed []remote.InstalledSdk, name string) (remote.InstalledSdk, bool) {
	for _, sdk := range installed {
		if sdk.Name == name {
			return sdk, true
		}
	}
	return remote.InstalledSdk{}, false
}

// sdkConfigValue returns the value of a key of the vfox configuration, or "" when
// the key is not set, or when the configuration has not loaded.
func sdkConfigValue(inv remote.Inventory, key string) string {
	for _, entry := range inv.Config.Value {
		if entry.Key == key {
			return entry.Value
		}
	}
	return ""
}

// sdkPartsLine is the line that tells where an installed version comes from: the
// name of its source, then its flavours, separated as the web interface writes them.
func sdkPartsLine(parts remote.VersionParts, chinese bool) string {
	var names []string
	if parts.Source != nil {
		names = append(names, parts.Source.Name.In(chinese))
	}
	names = append(names, parts.Flavors...)
	return sdkJoin(" · ", names...)
}

// sdkAliasesOf names the aliases of a version of the plugin name, as "@j17, @lts".
// An alias names the version when its target is name@version, which vfox writes,
// or the version itself.
func sdkAliasesOf(aliases []remote.VersionAlias, name, version string) string {
	var named []string
	for _, alias := range aliases {
		if alias.Sdk == name && (alias.Target == name+"@"+version || alias.Target == version) {
			named = append(named, "@"+alias.Alias)
		}
	}
	return strings.Join(named, ", ")
}

// sdkWebAddress returns the address when it is an absolute http or https address,
// and false for anything else. A plugin names its homepage and its documents in its
// own metadata, so an address of another scheme, such as file: or javascript:, must
// not open (the web interface checks the same, in links.ts).
func sdkWebAddress(raw string) (string, bool) {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.Host == "" {
		return "", false
	}
	return address.String(), true
}

// sdkErrorText says why a load or a change failed: the answer of the computer when
// it gave one, and otherwise what kind of failure it was, in the words of the app.
func sdkErrorText(err error) string {
	var api *remote.APIError
	if errors.As(err, &api) && api.Message != "" {
		return api.Message
	}
	if text := noticeText(app.KindOf(err)); text != "" {
		return text
	}
	return i18n.Text("errOther")
}

// sdkInventory draws what a tab of an SDK screen shows of the inventory, and calls
// build with the inventory once it has something to show. Until the computer has
// answered, the tab shows a spinner; when the computer could not answer, it shows
// why, with a button that asks again. The installed plugins are a section of the
// inventory that loads on its own, and it is shown the same way while it loads or
// fails and has no plugin yet.
func (r *Root) sdkInventory(k kit.Ctx, build func(inv remote.Inventory)) {
	res := r.service.Inventory()
	inv, loaded := res.Value()
	switch {
	case !loaded && res.Err() != nil:
		r.sdkFailed(k, sdkErrorText(res.Err()))
	case !loaded:
		sdkLoading(k)
	case len(inv.Installed.Value) == 0 && inv.Installed.Loading:
		sdkLoading(k)
	case len(inv.Installed.Value) == 0 && inv.Installed.Error != "":
		r.sdkFailed(k, inv.Installed.Error)
	default:
		build(inv)
	}
}

// sdkLoading is the spinner of a tab whose data is on its way.
func sdkLoading(k kit.Ctx) {
	kit.LoadingMessage(k, i18n.Text("loading"), i18n.Text("loading"))
}

// sdkFailed is the failure of a tab: the reason, and a button that asks the computer
// again, which marks every resource out of date.
func (r *Root) sdkFailed(k kit.Ctx, reason string) {
	kit.StatusMessage(k, kit.StatusSpec{
		Icon:    "error",
		IsError: true,
		Title:   i18n.Text("failed"),
		Body:    reason,
		Actions: []func(){func() {
			kit.Button(k, kit.ButtonSpec{
				ID:      "sdkRetry",
				Kind:    kit.TonalButton,
				Label:   i18n.Text("retry"),
				OnClick: r.service.Refresh,
			})
		}},
	})
}

// sdkShowMore is the button that shows the next rows of a long list: at most
// sdkPageSize of the remaining ones, which it names. limit is how many rows the list
// shows; the caller builds the button only when some rows are left. id identifies the
// button among the others of its parent.
func sdkShowMore(k kit.Ctx, id string, remaining int, limit *int) {
	count := min(remaining, sdkPageSize)
	kit.Button(k, kit.ButtonSpec{
		ID:      id,
		Kind:    kit.TextButton,
		Label:   strings.ReplaceAll(i18n.Text("showMore"), "{n}", strconv.Itoa(count)),
		OnClick: func() { *limit += sdkPageSize },
	})
}

// sdkRun starts a task on the computer, which runs it in the background. Its
// progress is on the tasks tab, so the snack bar says where it is, instead of the
// tab opening as the web interface does.
func (r *Root) sdkRun(request remote.TaskRequest) {
	r.Do(func(ctx context.Context) error {
		_, err := r.service.StartTask(ctx, request)
		return err
	}, "sdkQueued")
}

// sdkPutConfig sets a key of the vfox configuration. The inventory that shows the
// configuration is marked out of date by the change itself, so nothing else is shown.
func (r *Root) sdkPutConfig(key, value string) {
	r.Do(func(ctx context.Context) error {
		return r.service.PutConfig(ctx, key, value)
	}, "")
}

// sdkDefaultVersion is the version that a new alias names: the version in use, or
// else the first installed one, as the web interface chooses it. It is empty when no
// version is installed.
func (r *Root) sdkDefaultVersion(name string) string {
	inv, _ := r.service.Inventory().Value()
	sdk, ok := sdkFindInstalled(inv.Installed.Value, name)
	switch {
	case ok && sdk.HasCurrent:
		return sdk.Current
	case ok && len(sdk.Versions) > 0:
		return sdk.Versions[0]
	}
	return ""
}

// sdkOpenLink opens a link that a plugin names, in the browser. A link that is not
// http or https is refused with a notice, as on the web.
func (r *Root) sdkOpenLink(services ui.Services, raw string) {
	address, ok := sdkWebAddress(raw)
	if !ok {
		r.env.ShowSnackbar(kit.Snack{Text: i18n.Text("linkRefused")})
		return
	}
	services.OpenURL(address)
}

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

package ui

import (
	"context"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/update"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

func TestUpdateCheckDownloadAndMirror(t *testing.T) {
	oldCheck, oldDownload := checkRelease, downloadRelease
	defer func() { checkRelease, downloadRelease = oldCheck, oldDownload }()
	checkRelease = func(context.Context) (update.Release, error) {
		return update.Release{
			Tag:  "v9.9.9",
			Page: "https://github.com/version-fox/vfox-plus/releases/tag/v9.9.9",
			Assets: []update.Asset{
				{Name: "vfox-plus_9.9.9_amd64.deb", URL: "https://github.com/version-fox/vfox-plus/releases/download/v9.9.9/vfox-plus_9.9.9_amd64.deb"},
			},
		}, nil
	}
	downloadRelease = func(context.Context, string, string, func(float64)) (string, error) {
		return "/tmp/vfox-plus_9.9.9_amd64.deb", nil
	}
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	ta.button("Settings")
	ta.mustText("Application updates")
	ta.button("Check for updates")
	ta.until("the available update", func() bool { return ta.text("Update 9.9.9 available") })
	ta.button("Download")
	ta.until("the downloaded path", func() bool { return ta.text("Downloaded to /tmp/vfox-plus_9.9.9_amd64.deb") })
	// The mirror prefix is stored in the settings.
	ta.app.saveUpdateMirror("https://gh-proxy.org/")
	if got := ta.app.model.Settings.Settings().UpdateMirror; got != "https://gh-proxy.org/" {
		t.Errorf("mirror = %q", got)
	}
}

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
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/update"
)

// platformAsset is the release asset for this platform, if there is one.
func platformAsset(release update.Release) (update.Asset, bool) {
	return update.PlatformAsset(release, "", "")
}

// checkRelease asks for the newest release. It is a variable so that tests
// can stub the network.
var checkRelease = func(ctx context.Context) (update.Release, error) {
	return update.Check(ctx, nil, "")
}

// downloadRelease fetches url through the mirror. It is a variable so that
// tests can stub the network.
var downloadRelease = func(ctx context.Context, url, mirror string, progress func(float64)) (string, error) {
	return update.Download(ctx, nil, url, mirror, "", progress)
}
func (a *App) updateMirrorInitial() {
	if !a.pages.updateMirrorReady {
		a.pages.updateMirrorReady = true
		a.pages.updateMirrorText = a.model.Settings.Settings().UpdateMirror
	}
}

// checkUpdate asks the GitHub API for the newest release, unless a check is
// already running. Silent reports only failures to callers that ask loudly:
// the automatic check on startup never nags, the button does.
func (a *App) checkUpdate(silent bool) {
	if a.pages.updateChecking || a.pages.updateDownloading {
		return
	}
	a.pages.updateChecking = true
	a.pages.updateError = ""
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		release, err := checkRelease(ctx)
		a.post(func() {
			a.pages.updateChecking = false
			a.pages.updateChecked = true
			if err != nil {
				a.pages.updateError = describeError(err)
				if !silent {
					a.notify(describeError(err))
				}
				return
			}
			a.pages.updateRelease = release
			a.pages.updateHasRelease = true
			a.pages.updateError = ""
			if !silent {
				if release.NewerThan(Version) {
					a.notify(a.T("updateAvailable", "version", release.Version()))
				} else {
					a.notify(a.T("updateUpToDate"))
				}
			}
		})
	}()
}

// autoCheckUpdate is the silent check on startup: it fills the settings page
// when the user opens it, without nagging.
func (a *App) autoCheckUpdate() {
	if a.pages.updateChecked || a.pages.updateChecking {
		return
	}
	a.checkUpdate(true)
}

// saveUpdateMirror stores the mirror prefix for GitHub downloads.
func (a *App) saveUpdateMirror(text string) {
	value := strings.TrimSpace(text)
	a.pages.updateMirrorText = value
	a.updateSettings(func(s *settings.AppSettings) { s.UpdateMirror = value })
	a.notify(a.T("settingsSaved"))
}

// downloadUpdate fetches the asset for this platform, through the mirror
// when one is set, and remembers where it landed. Nothing installs itself:
// the .deb is left for dpkg and the .msixbundle for the installer.
func (a *App) downloadUpdate() {
	if a.pages.updateDownloading || !a.pages.updateHasRelease {
		return
	}
	asset, ok := update.PlatformAsset(a.pages.updateRelease, "", "")
	if !ok {
		a.notify(a.T("updateNoAsset"))
		return
	}
	a.pages.updateDownloading = true
	a.pages.updateProgress = -1
	a.pages.updatePath = ""
	a.invalidate()
	mirror := a.model.Settings.Settings().UpdateMirror
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		path, err := downloadRelease(ctx, asset.URL, mirror, func(fraction float64) {
			a.post(func() {
				a.pages.updateProgress = fraction
			})
		})
		a.post(func() {
			a.pages.updateDownloading = false
			a.pages.updateProgress = -1
			if err != nil {
				a.notify(describeError(err))
				return
			}
			a.pages.updatePath = path
		})
	}()
}

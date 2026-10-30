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

package webheap

import (
	"errors"
	"os"
	"os/exec"

	"github.com/chromedp/chromedp"
)

// macChrome is where Chrome lives on macOS, which has no google-chrome in PATH.
const macChrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// findChrome returns the browser to start: the one that was named, then
// $CHROME_BIN, then the first of the usual names in PATH. (chromedp looks for
// a browser itself when it is not told, but it does not know the name that
// Chrome for Testing gives its headless build, chrome-headless-shell.)
func findChrome(named string) (string, error) {
	if named == "" {
		named = os.Getenv("CHROME_BIN")
	}
	if named != "" {
		return named, nil
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome-headless-shell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	if info, err := os.Stat(macChrome); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		return macChrome, nil
	}
	return "", errors.New("no Chrome found (set -chrome or CHROME_BIN)")
}

// chromeOptions are the options of a throw-away headless browser: its own empty
// profile (chromedp makes one and removes it), no network of its own, no
// proxy, and the window size that was asked for. They are the whole list: the
// defaults of chromedp, which are made for scraping, are not used, so that the
// pages look the way that the screenshots of the documentation always showed them.
func chromeOptions(binary string, width, height int) []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(binary),
		chromedp.Flag("headless", "new"),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("no-proxy-server", true),
		chromedp.WindowSize(width, height),
		chromedp.NoSandbox,
	}
}

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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// devToolsLine is what Chrome prints to its standard error once it listens.
var devToolsLine = regexp.MustCompile(`DevTools listening on (ws://\S+)`)

// macChrome is where Chrome lives on macOS, which has no google-chrome in PATH.
const macChrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// findChrome returns the browser to start: the one that was named, then
// $CHROME_BIN, then the first of the usual names in PATH.
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

// chromeArgs is the command line of a throw-away headless browser: its own
// empty profile, no network of its own, and a debugging port that the system
// picks (Chrome prints it).
func chromeArgs(width, height int, profile string) []string {
	return []string{
		"--headless=new",
		"--remote-debugging-port=0",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-default-apps",
		"--mute-audio",
		"--hide-scrollbars",
		"--no-proxy-server",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		"--user-data-dir=" + profile,
		"--no-sandbox",
		"about:blank",
	}
}

// stderrWatcher reads what Chrome writes to its standard error: it finds the
// DevTools address and keeps the last bytes for the error message of a Chrome
// that exits early.
type stderrWatcher struct {
	mu       sync.Mutex
	line     []byte
	tail     []byte
	endpoint chan string
}

const tailSize = 500

func (w *stderrWatcher) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tail = append(w.tail, p...)
	if len(w.tail) > tailSize {
		w.tail = w.tail[len(w.tail)-tailSize:]
	}
	w.line = append(w.line, p...)
	for {
		end := bytes.IndexByte(w.line, '\n')
		if end < 0 {
			break
		}
		if match := devToolsLine.FindSubmatch(w.line[:end]); match != nil {
			select {
			case w.endpoint <- string(match[1]):
			default:
			}
		}
		w.line = w.line[end+1:]
	}
	if len(w.line) > 1<<20 { // A line that never ends is not worth keeping.
		w.line = nil
	}
	return len(p), nil
}

func (w *stderrWatcher) lastOutput() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(string(w.tail))
}

// browser is a running headless Chrome.
type browser struct {
	cmd      *exec.Cmd
	exited   chan struct{} // closed once the process has been reaped
	endpoint string        // the ws:// address of the browser target
}

// startBrowser starts Chrome and waits until it listens for the DevTools.
func startBrowser(ctx context.Context, binary string, args []string) (*browser, error) {
	watcher := &stderrWatcher{endpoint: make(chan string, 1)}
	command := exec.Command(binary, args...)
	command.Stderr = watcher // Wait finishes the copying before it returns.
	isolate(command)
	if err := command.Start(); err != nil {
		return nil, err
	}
	chrome := &browser{cmd: command, exited: make(chan struct{})}
	go func() {
		_ = command.Wait()
		close(chrome.exited)
	}()
	select {
	case chrome.endpoint = <-watcher.endpoint:
		return chrome, nil
	case <-chrome.exited:
		return nil, fmt.Errorf("Chrome exited before it listened: %s", watcher.lastOutput())
	case <-time.After(60 * time.Second):
		chrome.stop()
		return nil, errors.New("Chrome did not print a DevTools address in 60 seconds")
	case <-ctx.Done():
		chrome.stop()
		return nil, ctx.Err()
	}
}

// stop kills the browser and its helper processes, and waits for them.
func (b *browser) stop() {
	killTree(b.cmd)
	select {
	case <-b.exited:
	case <-time.After(10 * time.Second):
	}
}

// devToolsTarget is one entry of /json/list.
type devToolsTarget struct {
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// choosePage picks the page to drive: one that shows something, else the
// first page.
func choosePage(targets []devToolsTarget) string {
	fallback := ""
	for _, target := range targets {
		if target.Type != "page" || target.WebSocketDebuggerURL == "" {
			continue
		}
		if target.URL != "about:blank" {
			return target.WebSocketDebuggerURL
		}
		if fallback == "" {
			fallback = target.WebSocketDebuggerURL
		}
	}
	return fallback
}

// pageSocket returns the address of the first page target of the browser. A
// browser that has just started may not list its page yet, so it asks again for
// a few seconds.
func pageSocket(ctx context.Context, endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	list := "http://" + parsed.Host + "/json/list"
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	lastErr := errors.New("the browser lists no page")
	for {
		targets, err := fetchTargets(ctx, client, list)
		if err == nil {
			if socket := choosePage(targets); socket != "" {
				return socket, nil
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return "", lastErr
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func fetchTargets(ctx context.Context, client *http.Client, address string) ([]devToolsTarget, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", address, response.Status)
	}
	var targets []devToolsTarget
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return nil, err
	}
	return targets, nil
}

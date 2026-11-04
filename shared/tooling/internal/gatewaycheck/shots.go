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

package gatewaycheck

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/chrome"
)

// `vfxtool gateway-shots` takes the pictures of the pages of the gateway for the documentation. It does what
// `gateway-check` does, with a browser in the place of the person's: the end-to-end build of the application with
// the web interface and "other devices" switched on, the program of the gateway, and a headless Chrome that pairs
// itself with the gateway, pastes the link of the application, compares the fingerprint and uses the interface.
// The pictures are what that Chrome shows at each step.

const shotsUsage = "Usage: gateway-shots --binary FILE --output DIR [--chrome FILE] <application command...>"

// documentationAddress is the address of the computer in the pictures of the documentation: the address that
// the end-to-end build of the application is told to give (VFOX_PLUS_E2E_ADDRESSES, the same as the
// desktop pictures of `vfxtool sandbox screenshots` show) and that the gateway really reaches the application
// at, so the computer has to have it: the command asks for a network namespace of its own that gives it to its
// loopback interface (see requireAddress). It is an address of the kind that a home network has, the same at
// every run, and the address of no machine.
const documentationAddress = "192.168.1.20"

// The sizes of the pictures: the pages of the gateway are a column of 40 rem, and the interface of the
// application that it shows is a window of a desktop.
const (
	pageWidth  = 760
	pageMargin = 40 // pixels under the last card of a page.
	appWidth   = 1280
	appHeight  = 800
)

// How long the run may take, how long a page may take to show what a step waits for, and how long the
// interface of the application is given to draw itself after it has said that it is ready.
const (
	shotsTimeout = 5 * time.Minute
	pageWait     = 30 * time.Second
	appSettle    = 500 * time.Millisecond
)

// shotPrefix begins the name of every picture: gateway-intro.png, gateway-home.png, ...
const shotPrefix = "gateway-"

// namespaceAdvice is what a person who lacks the address of the documentation runs: a network namespace of its own,
// in which they are root, with the address on its loopback interface.
const namespaceAdvice = "unshare -rn sh -c 'ip link set lo up && ip addr add " + documentationAddress +
	"/32 dev lo && xvfb-run -a dbus-run-session -- vfxtool gateway-shots ...'"

// interfaceAddrs is the addresses of the computer; the tests give another list.
var interfaceAddrs = net.InterfaceAddrs

// requireAddress sees that the computer has the address that the pictures show. The application would say that
// it has it all the same, and the gateway would not reach it there: the picture of the page that asks to
// compare the fingerprint would be a page that says that the application cannot be reached.
func requireAddress(address string) error {
	addresses, err := interfaceAddrs()
	if err != nil {
		return fmt.Errorf("reading the addresses of this computer: %w", err)
	}
	for _, found := range addresses {
		if network, ok := found.(*net.IPNet); ok && network.IP.String() == address {
			return nil
		}
	}
	return fmt.Errorf("this computer has no address %s, which the pictures show and which the gateway reaches the application at: "+
		"run the command in a network namespace that has it, as root of its own user namespace:\n  %s", address, namespaceAdvice)
}

// Shots implements `gateway-shots --binary FILE --output DIR [--chrome FILE] <application command...>`. The
// application command is the end-to-end build of the application, which needs a display that the caller
// provides (xvfb-run), as it does for `gateway-check`.
func Shots(args []string) error {
	flags := flag.NewFlagSet("gateway-shots", flag.ContinueOnError)
	binaryFlag := flags.String("binary", "", "the program of the gateway (go build ./cmd/vfox-plus-gateway in mygo); it listens on port 8080 of this computer")
	outputFlag := flags.String("output", "", "the directory that gets the pictures")
	chromeFlag := flags.String("chrome", "", "the Chrome to drive (default: $CHROME_BIN, then one in PATH)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	command := flags.Args()
	if *binaryFlag == "" || *outputFlag == "" || len(command) == 0 {
		return errors.New(shotsUsage)
	}
	if err := requireAddress(documentationAddress); err != nil {
		return err
	}
	browser, err := chrome.Find(*chromeFlag)
	if err != nil {
		return err
	}
	program, err := filepath.Abs(*binaryFlag)
	if err != nil {
		return err
	}
	output, err := filepath.Abs(*outputFlag)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "vfox-plus-gateway-shots-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	taker := &shots{
		check:   &check{program: &binary{path: program, home: filepath.Join(dir, "gateway-home")}, linkFile: filepath.Join(dir, "link.txt")},
		browser: browser, output: output,
	}
	return appsession.RunScenario(appsession.Config{
		Name: "gateway-shots", Command: command, Fixtures: appsession.FixturesDir(root),
		// The settings that a person switches on: the web interface and other devices, on the port that vfox-plus
		// has until it is changed, which the pictures show.
		Settings: map[string]any{"webInterface": true, "webRemote": true},
		Env:      []string{"VFOX_PLUS_E2E_LINK=" + taker.linkFile, "VFOX_PLUS_E2E_ADDRESSES=" + documentationAddress},
	}, "gateway-shots", taker.scenario)
}

// shots is one run: the application and the gateway of a check, and the browser that takes the pictures.
type shots struct {
	*check
	browser string // the Chrome to drive.
	output  string // where the pictures go.
}

// scenario starts the application and the gateway the way the check does, and takes the pictures.
func (c *shots) scenario(s *appsession.Session) {
	c.Session = s
	defer func() {
		recovered := recover()
		if recovered != nil && c.gateway != nil {
			appsession.Report("the gateway printed:\n%s", strings.TrimSpace(c.gateway.output()))
		}
		c.cleanup()
		if recovered != nil {
			panic(recovered)
		}
	}()
	c.pairWithTheApplication()
	c.Check(len(c.web.Addresses) == 1 && c.web.Addresses[0] == documentationAddress,
		"the application names the address of the documentation, and no other, for the devices that connect to it")
	c.startTheGateway()
	remote, err := c.app.remoteLink(documentationAddress)
	if err != nil {
		c.Fail("the application made no link for other devices: %v", err)
	}
	c.Step("take the pictures")
	if err := c.take(remote); err != nil {
		c.Fail("%v", err)
	}
}

// take drives a headless Chrome through the pages that a person sees, in the order that the user guide
// tells them: a browser that is not paired, a browser that is, the link pasted, the fingerprint to compare,
// the interface of the application in both its languages, and the page of a gateway that is paired.
func (c *shots) take(remote string) error {
	ctx, cancel := context.WithTimeout(context.Background(), shotsTimeout)
	defer cancel()
	tab, closeBrowser := openBrowser(ctx, c.browser)
	defer closeBrowser()
	p := &pages{tab: tab, base: "https://" + c.gateway.address(), dir: c.output, wait: pageWait}

	says := func(text string) string { return fmt.Sprintf("document.body.innerText.includes(%q)", text) }
	return p.run(
		// A browser that is not paired is told how to pair and which fingerprint to compare. The interface of the application
		// chooses its language from what the browser keeps for the address of the gateway: English first.
		p.visit("/"), p.until("the page of the gateway", says("This is the gateway of vfox+")+" && "+says(c.printed)), p.language("en"), p.shot("intro"),
		// The link that the gateway printed pairs the browser, once, and takes it to the page of a gateway that is paired
		// with nothing.
		p.visit(c.path), p.until("the form for the link", `!!document.getElementById("link")`), p.shot("home"),
		p.fill("#link", remote), p.shot("link"),
		// The gateway shows the fingerprint of the application, which is the one that the settings show.
		p.press(`form[action="/gateway/connect"] button`),
		p.until("the page that asks to compare the fingerprint", `!!document.querySelector("input[name=fingerprint]") && `+says(c.web.Fingerprint)), p.shot("trust"),
		// What is trusted is the interface of the application, which a window of a desktop shows whole, in English and then
		// in Simplified Chinese.
		p.press(`form[action="/gateway/trust"] button`),
		p.size(appWidth, appHeight), p.until("the interface of the application", says("Plugins in use")), p.pause(appSettle), p.window("app"),
		p.language("zh"), p.reload(), p.until("the interface of the application in Chinese", says("在用的插件")), p.pause(appSettle), p.window("zh-app"),
		// The page of a gateway that is paired.
		p.visit("/"), p.until("the page of a gateway that is paired", says("Paired with vfox+ at")+" && "+says(c.web.Fingerprint)), p.shot("paired"),
	)
}

// openBrowser starts the throw-away Chrome that takes the pictures, and gives its tab. The gateway serves a
// certificate that nobody signed, which a person trusts by its fingerprint, and which the first picture
// shows next to the one that the gateway printed: the browser is not asked to verify it.
func openBrowser(ctx context.Context, binary string) (tab context.Context, closeBrowser func()) {
	options := chrome.Options(binary, pageWidth, appHeight)
	options = append(options, chromedp.Flag("ignore-certificate-errors", true))
	allocator, stopAllocator := chromedp.NewExecAllocator(ctx, options...)
	tab, closeTab := chromedp.NewContext(allocator)
	return tab, func() {
		closeTab()
		stopAllocator()
	}
}

// pages is the tab of the browser and what the steps of the pictures do to it.
type pages struct {
	tab  context.Context
	base string        // the address of the gateway, with no path.
	dir  string        // where the pictures go.
	wait time.Duration // how long a page may take to show what a step waits for.
}

// A step is one thing that a person does, or one picture that they take.
type step func() error

// run does the steps one after the other, and stops at the first that fails.
func (p *pages) run(steps ...step) error {
	for _, do := range steps {
		if err := do(); err != nil {
			return err
		}
	}
	return nil
}

// evaluate runs a JavaScript expression in the page and decodes its value into T.
func evaluate[T any](p *pages, expression string) (T, error) {
	return chromedp.Run(p.tab, chromedp.Evaluate[T](expression))
}

// visit opens an address of the gateway, following what it redirects to.
func (p *pages) visit(path string) step {
	return func() error {
		if err := chromedp.Do(p.tab, chromedp.Navigate(p.base+path)); err != nil {
			return fmt.Errorf("cannot open %s: %w", path, err)
		}
		return nil
	}
}

// until waits for a JavaScript expression to be true: the page shows what the next step needs. An evaluation
// that fails, as one does while a page is loading, is tried again.
func (p *pages) until(what, expression string) step {
	return func() error {
		for deadline := time.Now().Add(p.wait); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
			if ready, err := evaluate[bool](p, expression); err == nil && ready {
				return nil
			}
		}
		return fmt.Errorf("timed out waiting for %s", what)
	}
}

// pause waits a moment, for what the page draws after it has said that it is ready.
func (p *pages) pause(d time.Duration) step {
	return func() error {
		time.Sleep(d)
		return nil
	}
}

// act runs a JavaScript expression that does something to an element and says whether the element is there.
func (p *pages) act(what, expression string) error {
	found, err := evaluate[bool](p, expression)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if !found {
		return fmt.Errorf("%s: the page has no such element", what)
	}
	return nil
}

// fill types a text into a field, as pasting it does.
func (p *pages) fill(selector, text string) step {
	expression := fmt.Sprintf(`(() => { const field = document.querySelector(%q); if (!field) return false;
		field.focus(); field.value = %q; field.dispatchEvent(new Event("input", {bubbles: true})); return true; })()`, selector, text)
	return func() error { return p.act("fill "+selector, expression) }
}

// language chooses the language of the interface of the application the way its settings do for a browser: it keeps
// the choice for the address that it is at. The page has to be loaded again for the choice to show.
func (p *pages) language(code string) step {
	expression := fmt.Sprintf(`(() => { localStorage.setItem("vfox-plus-lang", %q); return true; })()`, code)
	return func() error { return p.act("choose the language "+code, expression) }
}

// reload loads the page again.
func (p *pages) reload() step {
	return func() error { return chromedp.Do(p.tab, chromedp.Reload()) }
}

// press clicks a button. The page that the click leads to is waited for by the next step.
func (p *pages) press(selector string) step {
	expression := fmt.Sprintf(`(() => { const button = document.querySelector(%q); if (!button) return false; button.click(); return true; })()`, selector)
	return func() error { return p.act("press "+selector, expression) }
}

// size makes the window as big as a desktop's, or as a column.
func (p *pages) size(width, height int) step {
	return func() error {
		return chromedp.Do(p.tab, chromedp.EmulateViewport(int64(width), int64(height)))
	}
}

// shot takes the picture of a page of the gateway: as high as the page, its content and a margin, with no
// empty space under it.
func (p *pages) shot(name string) step {
	return func() error {
		height, err := evaluate[float64](p, `Math.ceil(document.querySelector("main").getBoundingClientRect().bottom + window.scrollY)`)
		if err != nil {
			return fmt.Errorf("measuring the page %s: %w", name, err)
		}
		if err := chromedp.Do(p.tab, chromedp.EmulateViewport(pageWidth, int64(math.Ceil(height))+pageMargin)); err != nil {
			return err
		}
		return p.capture(name)
	}
}

// window takes the picture of the window as it is.
func (p *pages) window(name string) step {
	return func() error { return p.capture(name) }
}

// capture writes the picture of the page to the directory.
func (p *pages) capture(name string) error {
	image, err := chromedp.Run(p.tab, chromedp.CaptureScreenshot())
	if err != nil {
		return fmt.Errorf("taking the picture %s: %w", name, err)
	}
	file := filepath.Join(p.dir, shotPrefix+name+".png")
	if err := os.WriteFile(file, image, 0o644); err != nil { //nolint:gosec // A picture for the documentation.
		return err
	}
	appsession.Report("took %s", file)
	return nil
}

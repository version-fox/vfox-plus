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

// Package gatewaycheck implements `vfxtool gateway-check`: it checks the gateway (mygo/gateway, the
// program of the Docker image vfox-plus-gateway) as the people who use it use it, against the
// application itself.
//
// It starts the end-to-end build of the application with the web interface and "other devices" switched
// on, in the way a person switches them on in the settings, and starts the gateway: the Docker image (a
// container with a volume of its own, stopped the way docker stop stops it) or the program. Then it is
// the person at a browser: it pairs the browser with the link that the gateway prints, pastes a link
// for other devices that the application makes, compares the fingerprint of the certificate that the
// gateway shows with the one that the application shows, and uses the application through the gateway
// (its page, its API, its events, a task). It restarts the gateway, to see that it is paired still,
// and ends the application, to see what the gateway says then.
//
// The application is a desktop application: the command that starts it needs a display, which the
// caller provides (xvfb-run).
package gatewaycheck

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

const usage = "Usage: gateway-check (--image NAME | --binary FILE) <application command...>"

// Run implements `gateway-check (--image NAME | --binary FILE) <application command...>`. The application
// command is the end-to-end build of the application.
func Run(args []string) error {
	flags := flag.NewFlagSet("gateway-check", flag.ContinueOnError)
	imageFlag := flags.String("image", "", "check the Docker image with this name (docker build -f mygo/Dockerfile -t NAME mygo)")
	binaryFlag := flags.String("binary", "", "check the program instead; it listens on port 8080 of this computer")
	if err := flags.Parse(args); err != nil {
		return err
	}
	command := flags.Args()
	if (*imageFlag == "") == (*binaryFlag == "") || len(command) == 0 {
		return errors.New(usage)
	}
	dir, err := os.MkdirTemp("", "vfox-plus-gateway-check-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	var gateways program
	if *imageFlag != "" {
		gateways = newImage(*imageFlag)
	} else {
		path, err := filepath.Abs(*binaryFlag)
		if err != nil {
			return err
		}
		gateways = &binary{path: path, home: filepath.Join(dir, "gateway-home")}
	}
	defer gateways.cleanup()

	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	port, err := freePort()
	if err != nil {
		return err
	}
	check := &check{program: gateways, linkFile: filepath.Join(dir, "link.txt")}
	return appsession.RunScenario(appsession.Config{
		Name: "gateway", Command: command, Fixtures: appsession.FixturesDir(root),
		// The settings that a person switches on: the web interface, and other devices on a port of their own.
		Settings: map[string]any{"webInterface": true, "webRemote": true, "webPort": port},
		Env:      []string{"VFOX_PLUS_E2E_LINK=" + check.linkFile},
	}, "gateway", check.scenario)
}

// freePort is a port that nothing listens on now.
func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// How long what the scenario waits for may take.
const (
	startWait   = 60 * time.Second
	eventWait   = 15 * time.Second
	taskWait    = 60 * time.Second
	exitWait    = 30 * time.Second
	stopWithin  = 5 * time.Second // docker stop waits ten seconds before it kills; the gateway needs a second.
	linkTimeout = 60 * time.Second
)

// check is one run of the scenario.
type check struct {
	*appsession.Session
	program  program
	linkFile string

	app *site     // a browser that is paired with the application itself, on this computer.
	web webStatus // what the application says of the servers for other devices.

	gateway gateway
	browser *site        // a browser that is paired with the gateway.
	printed string       // the fingerprint that the gateway printed.
	path    string       // the pairing link of a browser that it printed, without the address.
	link    string       // the link of the application that the gateway was paired with.
	events  *eventStream // the events of the application that a browser reads through the gateway.
}

// scenario is the person at the browser. When it fails, what the gateway printed says why as often as the
// application does.
func (c *check) scenario(s *appsession.Session) {
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
	c.startTheGateway()
	c.showTheIntro()
	c.pairABrowser()
	c.pairTheGateway()
	c.turnAwayStrangers()
	c.useTheApplication()
	c.restartTheGateway()
	c.endTheApplication()
	c.stopTheGateway()
}

// cleanup lets go of what the scenario holds.
func (c *check) cleanup() {
	if c.events != nil {
		c.events.close()
		c.events = nil
	}
	if c.gateway != nil {
		_, _ = c.gateway.stop()
		c.gateway = nil
	}
}

// pinned is a browser at the gateway that trusts the certificate with the fingerprint that it printed, and no other.
func (c *check) pinned(address, fingerprint string) *site {
	browser, err := newPinnedSite(address, fingerprint)
	if err != nil {
		c.Fail("a browser cannot trust the certificate of the gateway: %v", err)
	}
	return browser
}

// expect fails the scenario unless the request was answered with the status.
func (c *check) expect(answer reply, err error, status int, what string) {
	if err != nil {
		c.Fail("%s: %v", what, err)
	}
	if answer.status != status {
		c.Fail("%s: answered %d, want %d: %s", what, answer.status, status, excerpt(answer.body))
	}
}

// expectText fails the scenario unless the answer says what is wanted.
func (c *check) expectText(answer reply, want, what string) {
	if !strings.Contains(answer.text(), want) {
		c.Fail("%s: the answer does not say %q: %s", what, want, excerpt(answer.text()))
	}
}

// pairWithTheApplication pairs a browser with the application, as the settings do for the browser of
// the computer, and learns what the application says of the connections of other devices.
func (c *check) pairWithTheApplication() {
	c.Step("pair a browser with the application")
	link, err := waitForFile(c.linkFile, linkTimeout)
	if err != nil {
		c.Fail("the application made no link for a browser: %v", err)
	}
	if c.app, err = pairPlain(link); err != nil {
		c.Fail("the application did not pair a browser: %v", err)
	}
	c.WaitFor("the application to listen for other devices", startWait, func() bool {
		web, err := c.app.webStatus()
		if err != nil || !web.Listening {
			return false
		}
		c.web = web
		return true
	})
	c.Check(c.web.Remote && c.web.SelfSigned && c.web.Fingerprint != "", "the application listens with a certificate of its own, whose fingerprint it shows")
	c.Check(len(c.web.Addresses) > 0, "the application names an address of this computer that other devices connect to")
}

// startTheGateway starts the gateway and reads from its output what a person reads there.
func (c *check) startTheGateway() {
	c.Step("start the gateway")
	var err error
	if c.gateway, err = c.program.start(); err != nil {
		c.Fail("the gateway does not start: %v", err)
	}
	if c.path, c.printed, err = waitForBanner(c.gateway, startWait); err != nil {
		c.Fail("%v", err)
	}
	c.Check(strings.Contains(c.gateway.output(), "Not paired with a vfox+ yet"), "the gateway says that it is paired with no vfox+")
	// A browser that trusts the certificate that has the fingerprint that was printed, and none other: the
	// first request fails unless the certificate that the gateway serves is that one.
	c.browser = c.pinned(c.gateway.address(), c.printed)
}

// showTheIntro sees what a browser that is not paired with the gateway is shown first: the page of the
// gateway, which says how to pair, and the fingerprint of its certificate.
func (c *check) showTheIntro() {
	c.Step("show the page of the gateway to a browser that is not paired")
	intro, err := c.browser.get("/")
	c.expect(intro, err, 200, "the page of the gateway")
	c.expectText(intro, "This is the gateway of vfox+", "the page of the gateway")
	c.expectText(intro, c.printed, "the page of the gateway shows the fingerprint to compare")
}

// turnAwayStrangers sees that a browser that is not paired with the gateway gets nothing of the
// application once the gateway is paired with it: not its page, not its API, not its events, and not
// with the session that the application gave another browser.
func (c *check) turnAwayStrangers() {
	c.Step("turn away a browser that is not paired, now that the gateway is paired with the application")
	stranger := c.pinned(c.gateway.address(), c.printed)
	page, err := stranger.get("/app")
	c.expect(page, err, 302, "the interface")
	c.Check(page.header.Get("Location") == "/", "the interface sends a browser that is not paired to the page of the gateway")
	for _, path := range []string{"/api/inventory", "/api/events", "/api/web", "/api/tasks", "/api/token"} {
		denied, err := stranger.get(path)
		c.expect(denied, err, 401, path)
		c.Check(!strings.Contains(denied.body, "nodejs"), path+" shows nothing of the application")
	}
	started, err := stranger.postJSON("/api/tasks", `{"operation":"install","sdk":"nodejs","version":"24.11.0"}`)
	c.expect(started, err, 401, "starting a task")
	forged := c.pinned(c.gateway.address(), c.printed)
	forged.cookie = "vfox-plus-gateway=" + strings.TrimPrefix(c.app.cookie, sessionCookie+"=")
	denied, err := forged.get("/api/inventory")
	c.expect(denied, err, 401, "the inventory, with the session of the application as the cookie of the gateway")
}

// pairABrowser pairs the browser with the link that the gateway printed, once.
func (c *check) pairABrowser() {
	c.Step("pair a browser with the gateway")
	paired, err := c.browser.get(c.path)
	c.expect(paired, err, 303, "the pairing link of the gateway")
	c.Check(paired.header.Get("Location") == "/", "a gateway that is paired with nothing takes the browser to its page")
	again, err := c.pinned(c.gateway.address(), c.printed).get(c.path)
	c.expect(again, err, 403, "the pairing link of the gateway, a second time")
	home, err := c.browser.get("/")
	c.expect(home, err, 200, "the page of the gateway, for a browser that is paired")
	c.expectText(home, "not paired with a vfox+ yet", "the page of the gateway")
}

// pairTheGateway pastes a link for other devices that the application makes, compares the fingerprint
// of its certificate and trusts it.
func (c *check) pairTheGateway() {
	c.Step("pair the gateway with the application, comparing the fingerprint")
	made, err := c.app.remoteLink(c.web.Addresses[0])
	if err != nil {
		c.Fail("the application made no link for other devices: %v", err)
	}
	// The address that the gateway reaches the application at is not always the one the application
	// names: a container reaches the computer through the address of its gateway. The gateway says when
	// it cannot reach one, and the next is tried.
	var shown reply
	var tried []string
	reached := ""
	for _, host := range c.program.hosts(c.web.Addresses) {
		c.link = withHost(made, host)
		answer, err := c.browser.postForm("/gateway/connect", "link", c.link)
		if err != nil {
			c.Fail("the page of the gateway: %v", err)
		}
		if answer.status == 502 {
			tried = append(tried, host+": "+firstAlert(answer.text()))
			continue
		}
		c.expect(answer, nil, 200, "pasting the link for other devices")
		shown, reached = answer, host
		break
	}
	if shown.status == 0 {
		c.Fail("the gateway reaches the application at none of its addresses:\n  %s", strings.Join(tried, "\n  "))
	}
	appsession.Report("the gateway reaches the application at %s", reached)
	// What the person compares: the fingerprint that the gateway shows is the one that the application
	// shows in its settings, in the same form.
	c.expectText(shown, "not signed by an authority", "the page that asks for the comparison")
	c.expectText(shown, c.web.Fingerprint, "the page that asks for the comparison shows the fingerprint of the application")
	home, err := c.browser.get("/")
	c.expect(home, err, 200, "the page of the gateway")
	c.expectText(home, "not paired with a vfox+ yet", "the gateway trusts nothing before the comparison")

	trusted, err := c.browser.postForm("/gateway/trust", "link", c.link, "fingerprint", c.web.Fingerprint)
	c.expect(trusted, err, 303, "trusting the certificate that was compared")
	c.Check(trusted.header.Get("Location") == "/app", "the gateway takes the browser to the interface")
	home, err = c.browser.get("/")
	c.expect(home, err, 200, "the page of the gateway")
	c.expectText(home, "Paired with vfox+ at", "the page of the gateway, once paired")
	c.expectText(home, c.web.Fingerprint, "the page of the gateway names the certificate it trusts")
}

// useTheApplication uses the application through the gateway: its page, its API, its events and a
// task, which changes something in the application.
func (c *check) useTheApplication() {
	c.Step("use the application through the gateway")
	page, err := c.browser.get("/app")
	c.expect(page, err, 200, "the interface")
	own, err := c.app.get("/app")
	c.expect(own, err, 200, "the interface of the application itself")
	c.Check(page.body == own.body, "the page that the gateway serves is the page of the application, byte for byte")
	c.Check(strings.Contains(page.body, "EventSource"), "the page listens for the events of the application")

	inventory, err := c.browser.get("/api/inventory")
	c.expect(inventory, err, 200, "the inventory")
	c.expectText(inventory, "nodejs", "the inventory of the application")
	manifest, err := c.pinned(c.gateway.address(), c.printed).get("/manifest.webmanifest")
	c.expect(manifest, err, 200, "the manifest, which a browser fetches without its cookie")
	c.expectText(manifest, "vfox+", "the manifest")

	c.events, err = c.browser.openEvents()
	if err != nil {
		c.Fail("the events: %v", err)
	}
	if err := c.events.wait("hello", eventWait); err != nil {
		c.Fail("the events come as they happen, not when the gateway has a buffer full: %v", err)
	}

	// A change that the browser makes, as the page makes it: with the origin of the gateway, which the
	// application would take for another site if the gateway passed it on.
	refused, err := c.browser.postJSONFrom("https://evil.example", "/api/tasks", `{"operation":"install","sdk":"nodejs","version":"24.11.0"}`)
	c.expect(refused, err, 403, "a request that comes from another site")
	started, err := c.browser.postJSON("/api/tasks", `{"operation":"install","sdk":"nodejs","version":"24.11.0"}`)
	c.expect(started, err, 201, "installing a version")
	task, err := taskOf(started.body)
	if err != nil {
		c.Fail("the task that was started: %v", err)
	}
	if err := c.events.wait("task", eventWait); err != nil {
		c.Fail("the application reports the task: %v", err)
	}
	c.WaitFor("the task to finish", taskWait, func() bool {
		tasks, err := c.browser.get("/api/tasks")
		if err != nil || tasks.status != 200 {
			return false
		}
		status, ok := statusOf(tasks.body, task)
		if ok && status == "failed" {
			c.Fail("the task failed: %s", excerpt(tasks.body))
		}
		return ok && status == "succeeded"
	})
}

// restartTheGateway stops it with an open connection of events, as docker stop does while a browser has
// the interface open, and starts it again: it is paired still, with the certificate it had.
func (c *check) restartTheGateway() {
	c.Step("stop the gateway with the interface open, and start it again")
	took, err := c.gateway.stop()
	c.gateway = nil
	if err != nil {
		c.Fail("the gateway does not end well: %v", err)
	}
	appsession.Report("the gateway stopped in %v with the events open", took.Round(10*time.Millisecond))
	c.Check(took < stopWithin, fmt.Sprintf("the gateway stops within %v even though a browser holds the events open", stopWithin))
	c.events.close()
	c.events = nil

	if c.gateway, err = c.program.start(); err != nil {
		c.Fail("the gateway does not start again: %v", err)
	}
	path, printed, err := waitForBanner(c.gateway, startWait)
	if err != nil {
		c.Fail("%v", err)
	}
	c.Check(printed == c.printed, "the gateway keeps its certificate between runs: a browser that trusted it is not asked again")
	c.Check(strings.Contains(c.gateway.output(), "Paired with vfox+ at"), "the gateway says that it is paired with the application")
	// The browsers are not remembered, the pairing with the application is.
	c.browser = c.pinned(c.gateway.address(), printed)
	paired, err := c.browser.get(path)
	c.expect(paired, err, 303, "the pairing link of the gateway that was started again")
	c.Check(paired.header.Get("Location") == "/app", "a gateway that is paired takes the browser straight to the interface")
	inventory, err := c.browser.get("/api/inventory")
	c.expect(inventory, err, 200, "the inventory, through the session the gateway kept")
	c.expectText(inventory, "nodejs", "the inventory")
}

// endTheApplication quits the application from the web interface, as a person does, and sees what the
// gateway says when there is nothing to show.
func (c *check) endTheApplication() {
	c.Step("end the application")
	quitting, err := c.browser.postJSON("/api/quit", `{}`)
	c.expect(quitting, err, 202, "quitting the application through the gateway")
	c.Check(c.WaitExit(exitWait), "the application ends when it is asked to quit")

	gone, err := c.browser.get("/api/inventory")
	c.expect(gone, err, 502, "the inventory of an application that has ended")
	c.expectText(gone, "Cannot reach vfox+", "the answer for the interface")
	page, err := c.browser.get("/app")
	c.expect(page, err, 502, "the interface of an application that has ended")
	c.expectText(page, "cannot be reached", "the page that says the application is gone")
	home, err := c.browser.get("/")
	c.expect(home, err, 200, "the page of the gateway")
	c.expectText(home, "Paired with vfox+ at", "the gateway keeps its pairing for the application that comes back")
}

// stopTheGateway stops it as docker stop does, and sees that it ends well.
func (c *check) stopTheGateway() {
	c.Step("stop the gateway")
	took, err := c.gateway.stop()
	c.gateway = nil
	if err != nil {
		c.Fail("the gateway does not end well: %v", err)
	}
	appsession.Report("the gateway stopped in %v", took.Round(10*time.Millisecond))
	c.Check(took < stopWithin, fmt.Sprintf("the gateway stops within %v", stopWithin))
}

// excerpt shortens the body of an answer for a message.
func excerpt(text string) string {
	text = strings.TrimSpace(text)
	const limit = 400
	if len(text) > limit {
		return text[:limit] + "…"
	}
	return text
}

// firstAlert is what the page of the gateway says went wrong: the text of its alert.
func firstAlert(page string) string {
	_, rest, found := strings.Cut(page, `role="alert"`)
	if !found {
		return excerpt(page)
	}
	_, rest, _ = strings.Cut(rest, "<p>")
	text, _, _ := strings.Cut(rest, "</p>")
	return strings.TrimSpace(text)
}

// withHost is the link with another host and the same port, path and token.
func withHost(link, host string) string {
	u, err := parseURL(link)
	if err != nil {
		return link
	}
	u.Host = net.JoinHostPort(host, u.Port())
	return u.String()
}

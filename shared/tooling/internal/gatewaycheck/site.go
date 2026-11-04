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
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// What the checks of this package do is what a person does at a browser: they talk to a server, which is
// either the application on this computer, in plain HTTP on the loopback address, or the gateway, in
// HTTPS with a certificate that nobody signed and that the person trusts by its fingerprint.

// requestTimeout is how long a request may take: the inventory waits for vfox.
const requestTimeout = 60 * time.Second

// sessionCookie is the cookie that the servers of the application give a browser that is paired.
const sessionCookie = "vfox-plus-session"

// site is a server and a browser at it.
type site struct {
	base   string
	client *http.Client
	// cookie is sent by hand to a server in plain HTTP, whose cookie is Secure, which a jar does not send
	// over HTTP.
	cookie string
}

// reply is an answer, read to the end.
type reply struct {
	status int
	header http.Header
	body   string
}

// text is the body as a person reads it: the page of the gateway escapes what the program says, and a +
// in "vfox+" is &#43; in it.
func (r reply) text() string { return html.UnescapeString(r.body) }

// noRedirect makes a client show the redirects instead of following them: where a server sends a
// browser is what is checked.
func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// fingerprintOf is the SHA-256 fingerprint of a certificate in the form that OpenSSL and the settings of the
// application show: pairs of capital hexadecimal digits with colons. It is computed here and not asked of the
// program that is checked, so that what it prints is checked too.
func fingerprintOf(certificate []byte) string {
	sum := sha256.Sum256(certificate)
	pairs := make([]string, len(sum))
	for i, b := range sum {
		pairs[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(pairs, ":")
}

// newPinnedSite is a browser at the gateway (host:port) that trusts the certificate with the fingerprint, and
// no other: a person compared it with the one that the gateway printed.
func newPinnedSite(address, fingerprint string) *site {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err) // cookiejar.New fails only for options, which it is given none.
	}
	return &site{
		base: "https://" + address,
		client: &http.Client{
			Jar: jar, Timeout: requestTimeout, CheckRedirect: noRedirect,
			Transport: &http.Transport{
				Proxy: nil,
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, //nolint:gosec // Not verified by a chain: by the fingerprint, just below.
					VerifyConnection: func(connection tls.ConnectionState) error {
						if len(connection.PeerCertificates) == 0 {
							return errors.New("the server presented no certificate")
						}
						if got := fingerprintOf(connection.PeerCertificates[0].Raw); got != fingerprint {
							return fmt.Errorf("the certificate has the fingerprint %s, not the %s that was trusted", got, fingerprint)
						}
						return nil
					},
				},
			},
		},
	}
}

// parseURL is url.Parse for a link that was made by the program that is checked.
func parseURL(link string) (*url.URL, error) { return url.Parse(strings.TrimSpace(link)) }

// pairPlain pairs a browser with the application, in plain HTTP, by a link that it made for this computer.
func pairPlain(link string) (*site, error) {
	u, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	s := &site{
		base:   u.Scheme + "://" + u.Host,
		client: &http.Client{Timeout: requestTimeout, CheckRedirect: noRedirect, Transport: &http.Transport{Proxy: nil}},
	}
	response, err := s.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusSeeOther {
		return nil, fmt.Errorf("the pairing link answered %s, want 303", response.Status)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == sessionCookie && cookie.Value != "" {
			s.cookie = sessionCookie + "=" + cookie.Value
			return s, nil
		}
	}
	return nil, errors.New("the pairing link gave the browser no session")
}

// do sends a request and reads the answer to the end.
func (s *site) do(method, path string, body io.Reader, headers map[string]string) (reply, error) {
	request, err := http.NewRequest(method, s.base+path, body)
	if err != nil {
		return reply{}, err
	}
	if s.cookie != "" {
		request.Header.Set("Cookie", s.cookie)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return reply{}, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	return reply{status: response.StatusCode, header: response.Header, body: string(data)}, err
}

func (s *site) get(path string) (reply, error) { return s.do(http.MethodGet, path, nil, nil) }

// postForm posts the fields (name, value, name, value…) of a form of the page of the gateway, from its origin.
func (s *site) postForm(path string, fields ...string) (reply, error) {
	form := url.Values{}
	for i := 0; i+1 < len(fields); i += 2 {
		form.Set(fields[i], fields[i+1])
	}
	return s.do(http.MethodPost, path, strings.NewReader(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded", "Origin": s.base,
	})
}

// postJSON posts JSON as the web interface does, from the origin of the server.
func (s *site) postJSON(path, body string) (reply, error) { return s.postJSONFrom(s.base, path, body) }

// postJSONFrom posts JSON from the origin of a page.
func (s *site) postJSONFrom(origin, path, body string) (reply, error) {
	return s.do(http.MethodPost, path, strings.NewReader(body), map[string]string{
		"Content-Type": "application/json", "Origin": origin,
	})
}

// webStatus is what the application says of the servers for other devices (GET /api/web).
type webStatus struct {
	Remote      bool     `json:"remote"`
	Port        int      `json:"port"`
	Listening   bool     `json:"listening"`
	Addresses   []string `json:"addresses"`
	Fingerprint string   `json:"fingerprint"`
	SelfSigned  bool     `json:"selfSigned"`
	Error       string   `json:"error"`
}

func (s *site) webStatus() (webStatus, error) {
	answer, err := s.get("/api/web")
	if err != nil {
		return webStatus{}, err
	}
	if answer.status != http.StatusOK {
		return webStatus{}, fmt.Errorf("answered %d: %s", answer.status, excerpt(answer.body))
	}
	var status webStatus
	if err := json.Unmarshal([]byte(answer.body), &status); err != nil {
		return webStatus{}, err
	}
	return status, nil
}

// remoteLink asks the application for a link that pairs a device on the network, as its settings do for a
// phone, with host as the address of the computer in it.
func (s *site) remoteLink(host string) (string, error) {
	request, err := json.Marshal(map[string]string{"host": host})
	if err != nil {
		return "", err
	}
	answer, err := s.postJSON("/api/web/pairing", string(request))
	if err != nil {
		return "", err
	}
	if answer.status != http.StatusOK {
		return "", fmt.Errorf("answered %d: %s", answer.status, excerpt(answer.body))
	}
	var made struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(answer.body), &made); err != nil || made.URL == "" {
		return "", fmt.Errorf("no link in the answer: %s", excerpt(answer.body))
	}
	if !strings.HasPrefix(made.URL, "https://") {
		return "", fmt.Errorf("the link for other devices is not https: %s", made.URL)
	}
	return made.URL, nil
}

// waitForFile waits for a file to appear and returns what it holds, a line.
func waitForFile(path string, timeout time.Duration) (string, error) {
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if content, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(content)) != "" { //nolint:gosec // The file of this run.
			return strings.TrimSpace(string(content)), nil
		}
	}
	return "", fmt.Errorf("%s did not appear in %v", path, timeout)
}

// taskOf is the id of the task that the application started.
func taskOf(body string) (int, error) {
	var task struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &task); err != nil || task.ID == 0 {
		return 0, fmt.Errorf("no task in the answer: %s", excerpt(body))
	}
	return task.ID, nil
}

// statusOf is the status of a task in the list that the application answers with.
func statusOf(body string, id int) (string, bool) {
	var tasks []struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &tasks); err != nil {
		return "", false
	}
	for _, task := range tasks {
		if task.ID == id {
			return task.Status, true
		}
	}
	return "", false
}

// eventStream is the events of the application (GET /api/events), which the page listens to and which
// never end.
type eventStream struct {
	events chan string
	cancel context.CancelFunc
	body   io.Closer
}

// openEvents opens the events, as the page does when it opens.
func (s *site) openEvents() (*eventStream, error) {
	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+"/api/events", nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if s.cookie != "" {
		request.Header.Set("Cookie", s.cookie)
	}
	// The same browser, without the limit of time that a request has: the events end when the page does.
	stream := &http.Client{Transport: s.client.Transport, Jar: s.client.Jar, CheckRedirect: noRedirect}
	response, err := stream.Do(request)
	if err != nil {
		cancel()
		return nil, err
	}
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		_ = response.Body.Close()
		cancel()
		return nil, fmt.Errorf("answered %s with %q, not events", response.Status, response.Header.Get("Content-Type"))
	}
	opened := &eventStream{events: make(chan string, 256), cancel: cancel, body: response.Body}
	go func() {
		defer close(opened.events)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			kind, found := strings.CutPrefix(scanner.Text(), "event: ")
			if !found {
				continue
			}
			select {
			case opened.events <- kind:
			case <-ctx.Done():
				return
			}
		}
	}()
	return opened, nil
}

// wait waits for an event of the kind, and lets the others go by.
func (e *eventStream) wait(kind string, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case got, open := <-e.events:
			if !open {
				return errors.New("the events ended")
			}
			if got == kind {
				return nil
			}
		case <-deadline.C:
			return fmt.Errorf("no %q event came in %v", kind, timeout)
		}
	}
}

// close lets go of the connection.
func (e *eventStream) close() {
	e.cancel()
	_ = e.body.Close()
}

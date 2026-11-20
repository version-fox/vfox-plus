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

// Package phonelink implements `vfxtool phone-link`: it gives the phone of the emulator a pairing
// link for the end-to-end build of the desktop application. The build writes a link to its loopback
// server into the file that VFOX_PLUS_E2E_LINK names, when it starts. The tool pairs with that server
// (which sets the session cookie), allows other devices (PUT /api/web, which starts the server on
// the port of the settings), waits until the server listens, and asks for a pairing link for the
// address at which the emulator reaches the computer (POST /api/web/pairing). The link goes into the
// file that --output names, with the permissions of a secret. CI runs it in the emulator job of
// .github/workflows/phone.yml, with the application started with VFOX_PLUS_E2E_ADDRESSES set to that
// address.
//
// The loopback server is plain HTTP and its cookie is Secure, which a cookie jar does not send back
// over plain HTTP, so the cookie is passed on by hand, as measurelightweight does.
package phonelink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// cookieName is the name of the session cookie of the pairing server.
const cookieName = "vfox-plus-session"

// usage is the command line of phone-link, as its errors print it.
const usage = "Usage: phone-link --loopback-link-file FILE --host ADDRESS --output FILE [--wait SECONDS]"

// pollInterval is how often the tool looks for the link file and for the listening server.
const pollInterval = 200 * time.Millisecond

// requestTimeout bounds one request to the pairing server.
const requestTimeout = 10 * time.Second

// Run implements `phone-link --loopback-link-file FILE --host ADDRESS --output FILE [--wait SECONDS]`.
func Run(args []string) error {
	flags := flag.NewFlagSet("phone-link", flag.ContinueOnError)
	loopbackFile := flags.String("loopback-link-file", "", "the file that the end-to-end build writes its loopback link to (VFOX_PLUS_E2E_LINK)")
	host := flags.String("host", "", "the address at which the phone reaches the computer, such as 10.0.2.2 for the emulator, which the end-to-end build lists (VFOX_PLUS_E2E_ADDRESSES)")
	output := flags.String("output", "", "the file receiving the pairing link for the phone")
	wait := flags.Int("wait", 120, "seconds to wait for the end-to-end build to write its link, and for the server to listen")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *loopbackFile == "" || *host == "" || *output == "" || *wait <= 0 {
		return errors.New(usage)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*wait)*time.Second)
	defer cancel()
	loopback, err := waitForLink(ctx, *loopbackFile)
	if err != nil {
		return err
	}
	link, err := PairingLink(ctx, loopback, *host)
	if err != nil {
		return err
	}
	return writeSecret(*output, link+"\n")
}

// waitForLink waits until the file holds a link, and returns the link.
func waitForLink(ctx context.Context, path string) (string, error) {
	for {
		if data, err := os.ReadFile(path); err == nil {
			if link := strings.TrimSpace(string(data)); link != "" {
				return link, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("the end-to-end build wrote no link to %s", path)
		case <-time.After(pollInterval):
		}
	}
}

// PairingLink pairs with the loopback server of the end-to-end build, allows other devices, and
// returns a pairing link for host. The loopback link must be the one of this computer: plain HTTP
// on the loopback interface, as the build writes it.
func PairingLink(ctx context.Context, loopback, host string) (string, error) {
	base, token, err := splitLoopbackLink(loopback)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	session, err := pair(ctx, client, base, token)
	if err != nil {
		return "", err
	}
	if err := allowOtherDevices(ctx, client, base, session); err != nil {
		return "", err
	}
	if err := waitForListening(ctx, client, base, session); err != nil {
		return "", err
	}
	return pairingURL(ctx, client, base, session, host)
}

// splitLoopbackLink returns the base URL of a loopback link, and its token.
func splitLoopbackLink(link string) (base, token string, err error) {
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "http" || !isLoopback(u.Hostname()) || !strings.HasPrefix(u.Path, "/auth/connect/") {
		// The link is not named: it is a secret until it is used.
		return "", "", errors.New("the end-to-end build did not write a loopback pairing link (http://127.0.0.1:<port>/auth/connect/<token>)")
	}
	return u.Scheme + "://" + u.Host, strings.TrimPrefix(u.Path, "/auth/connect/"), nil
}

// isLoopback reports whether host is the loopback interface.
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// pair uses the one-time token of the loopback link, and returns the session cookie it sets.
func pair(ctx context.Context, client *http.Client, base, token string) (string, error) {
	response, err := do(ctx, client, http.MethodGet, base+"/auth/connect/"+token, "", "")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("the loopback link did not pair: %s", response.Status)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == cookieName && cookie.Value != "" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("the pairing set no %s cookie", cookieName)
}

// allowOtherDevices turns on the access of other devices, which starts their server.
func allowOtherDevices(ctx context.Context, client *http.Client, base, session string) error {
	response, err := do(ctx, client, http.MethodPut, base+"/api/web", session, `{"remote":true}`)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("allowing other devices: %s", response.Status)
	}
	return nil
}

// waitForListening waits until the server for other devices listens.
func waitForListening(ctx context.Context, client *http.Client, base, session string) error {
	for {
		listening, err := webListening(ctx, client, base, session)
		if err == nil && listening {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the server for other devices does not listen: %v", ctxOrErr(ctx, err))
		case <-time.After(pollInterval):
		}
	}
}

// webListening reports the "listening" of the settings of other devices.
func webListening(ctx context.Context, client *http.Client, base, session string) (bool, error) {
	response, err := do(ctx, client, http.MethodGet, base+"/api/web", session, "")
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("reading the settings of other devices: %s", response.Status)
	}
	var state struct {
		Listening bool `json:"listening"`
	}
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		return false, err
	}
	return state.Listening, nil
}

// pairingURL asks the server for a pairing link for host, and checks that it is the address of the
// server that the phone reaches.
func pairingURL(ctx context.Context, client *http.Client, base, session, host string) (string, error) {
	body, err := json.Marshal(map[string]string{"host": host})
	if err != nil {
		return "", err
	}
	response, err := do(ctx, client, http.MethodPost, base+"/api/web/pairing", session, string(body))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("asking for a pairing link: %s", response.Status)
	}
	var answer struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&answer); err != nil {
		return "", err
	}
	if !strings.HasPrefix(answer.URL, "https://"+host+":") || !strings.Contains(answer.URL, "/auth/connect/") {
		return "", fmt.Errorf("the pairing link is not for %s", host)
	}
	return answer.URL, nil
}

// do sends one request, with the session cookie passed on by hand when there is one, and a JSON
// body when there is one.
func do(ctx context.Context, client *http.Client, method, target, session, body string) (*http.Response, error) {
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, err
	}
	if session != "" {
		request.Header.Set("Cookie", cookieName+"="+session)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		// The URL is named in no error: a pairing link is a secret until it is used.
		return nil, fmt.Errorf("%s %s: %w", method, pathOf(target), unwrapURLError(err))
	}
	return response, nil
}

// pathOf is the path of a URL, without the token of a pairing link.
func pathOf(target string) string {
	if i := strings.Index(target, "/auth/connect/"); i >= 0 {
		return "/auth/connect/<token>"
	}
	u, err := url.Parse(target)
	if err != nil {
		return "<unparsed>"
	}
	return u.Path
}

// unwrapURLError returns the error of the transport without the URL that it names.
func unwrapURLError(err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) {
		return urlError.Err
	}
	return err
}

// ctxOrErr is the error of a poll, or the reason the context ended, whichever says more.
func ctxOrErr(ctx context.Context, err error) error {
	if err != nil {
		return err
	}
	return ctx.Err()
}

// writeSecret writes text to path, readable by the owner only: a pairing link is a secret until
// it is used.
func writeSecret(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil // The modes of Windows are not the ones of the file system of Unix.
	}
	return os.Chmod(path, 0o600)
}

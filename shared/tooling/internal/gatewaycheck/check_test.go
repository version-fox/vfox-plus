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
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

func TestTheCheckNeedsOneOfTheImageAndTheProgramAndTheApplication(t *testing.T) {
	for name, args := range map[string][]string{
		"nothing":                     {},
		"the application only":        {"app"},
		"an image without the app":    {"--image", "vfox-plus-gateway"},
		"a program without the app":   {"--binary", "gateway"},
		"both an image and a program": {"--image", "x", "--binary", "y", "app"},
	} {
		if err := Run(args); err == nil || !strings.Contains(err.Error(), "Usage:") {
			t.Errorf("%s: %v, want the usage", name, err)
		}
	}
}

func TestALinkIsMovedToAnotherHostOfTheSameApplication(t *testing.T) {
	for _, c := range []struct{ link, host, want string }{
		{"https://192.168.1.20:8369/auth/connect/abc", "host.docker.internal", "https://host.docker.internal:8369/auth/connect/abc"},
		{"https://192.168.1.20:8369/auth/connect/abc", "10.0.2.2", "https://10.0.2.2:8369/auth/connect/abc"},
		{"https://192.168.1.20:8369/auth/connect/abc", "fd00::1", "https://[fd00::1]:8369/auth/connect/abc"},
		{"https://[fd00::1]:8369/auth/connect/abc", "192.168.1.20", "https://192.168.1.20:8369/auth/connect/abc"},
		{"https://192.168.1.20:8369/auth/connect/abc?x=1", "10.0.2.2", "https://10.0.2.2:8369/auth/connect/abc?x=1"},
	} {
		if got := withHost(c.link, c.host); got != c.want {
			t.Errorf("withHost(%q, %q) = %q, want %q", c.link, c.host, got, c.want)
		}
	}
	if got := withHost("://", "x"); got != "://" {
		t.Errorf("a link that is not one is left as it is: %q", got)
	}
}

func TestWhatThePageOfTheGatewayCompainsOfIsReadFromItsAlert(t *testing.T) {
	page := `<div class="card error" role="alert"><p>Cannot reach vfox+ at 10.1.2.3:8369.</p></div><div class="card"><p>Other text</p>`
	if got := firstAlert(page); got != "Cannot reach vfox+ at 10.1.2.3:8369." {
		t.Errorf("firstAlert = %q", got)
	}
	if got := firstAlert("<p>no alert here</p>"); got != "<p>no alert here</p>" {
		t.Errorf("a page without an alert is shown as it is: %q", got)
	}
}

func TestAnAnswerIsShortenedForAMessage(t *testing.T) {
	if got := excerpt("  short \n"); got != "short" {
		t.Errorf("excerpt = %q", got)
	}
	long := strings.Repeat("x", 1000)
	if got := excerpt(long); len(got) > 410 || !strings.HasSuffix(got, "…") {
		t.Errorf("excerpt of %d bytes is %d bytes: %.20s", len(long), len(got), got)
	}
}

func TestAFreePortIsFree(t *testing.T) {
	port, err := freePort()
	if err != nil || port < 1024 {
		t.Fatalf("freePort = %d, %v", port, err)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("the port is not free: %v", err)
	}
	_ = listener.Close()
}

// failure is the message of the failure of a check, which ends the scenario with a panic.
func failure(t *testing.T, check func()) (message string) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			message = fmt.Sprint(recovered)
		}
	}()
	check()
	return ""
}

func TestWhatTheCheckExpectsOfAnAnswerIsSaidWhenItIsNot(t *testing.T) {
	c := &check{Session: &appsession.Session{}}
	good := reply{status: 200, body: "all is well, vfox&#43;"}
	if got := failure(t, func() { c.expect(good, nil, 200, "the page") }); got != "" {
		t.Errorf("an answer that is expected failed the check: %s", got)
	}
	if got := failure(t, func() { c.expectText(good, "vfox+", "the page") }); got != "" {
		t.Errorf("a text that is there failed the check: %s", got)
	}
	if got := failure(t, func() { c.expect(reply{status: 403, body: "no"}, nil, 200, "the page") }); !strings.Contains(got, "the page: answered 403, want 200: no") {
		t.Errorf("a status that is not expected: %q", got)
	}
	if got := failure(t, func() { c.expect(reply{}, errors.New("refused"), 200, "the page") }); !strings.Contains(got, "the page: refused") {
		t.Errorf("a request that failed: %q", got)
	}
	if got := failure(t, func() { c.expectText(good, "missing", "the page") }); !strings.Contains(got, `the page: the answer does not say "missing"`) {
		t.Errorf("a text that is not there: %q", got)
	}
}

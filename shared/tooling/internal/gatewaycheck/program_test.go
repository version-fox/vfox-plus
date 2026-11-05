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
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// fakeGatewayEnv makes the test program a gateway: the tests run it as the program that is checked, as the
// tests of os/exec do, and it prints what the gateway prints and ends when it is told to.
const (
	fakeGatewayEnv = "GATEWAYCHECK_FAKE_GATEWAY"
	fakeExitEnv    = "GATEWAYCHECK_FAKE_EXIT"
)

// fakeFingerprint is the fingerprint that the fake gateway prints.
var fakeFingerprint = strings.TrimSuffix(strings.Repeat("AB:", 32), ":")

func TestMain(m *testing.M) {
	if os.Getenv(fakeGatewayEnv) != "" {
		fakeGateway()
		return
	}
	os.Exit(m.Run())
}

func fakeGateway() {
	fmt.Println("vfox+ gateway")
	fmt.Printf("Serving over HTTPS on port 8080. Certificate fingerprint, to compare with what the browser shows:\n  %s\n", fakeFingerprint)
	fmt.Println("Not paired with a vfox+ yet: pair it on its page, with a pairing link from the settings of vfox+.")
	fmt.Println("Pair a browser with this link (it works once and expires in 5 minutes), with localhost replaced by the address that reaches the gateway:")
	fmt.Println("  https://localhost:8080/auth/connect/ABCDEFGHIJKLMNOP")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	code, _ := strconv.Atoi(os.Getenv(fakeExitEnv))
	os.Exit(code)
}

func TestTheLinkAndTheFingerprintAreReadFromWhatTheGatewayPrints(t *testing.T) {
	banner := "vfox+ gateway\n" +
		"Serving over HTTPS on port 8080. Certificate fingerprint, to compare with what the browser shows:\n  " + fakeFingerprint + "\n" +
		"Pair a browser with this link (it works once and expires in 5 minutes), with localhost replaced by the address that reaches the gateway:\n" +
		"  https://localhost:8080/auth/connect/NNI65VXA2BMH6GRNW23EBA3PRB\n"
	path, fingerprint, err := parseBanner(banner)
	if err != nil || path != "/auth/connect/NNI65VXA2BMH6GRNW23EBA3PRB" || fingerprint != fakeFingerprint {
		t.Errorf("parseBanner = %q, %q, %v", path, fingerprint, err)
	}
	for name, text := range map[string]string{
		"nothing":                 "",
		"no link":                 "  " + fakeFingerprint + "\n",
		"no fingerprint":          "https://localhost:8080/auth/connect/ABC\n",
		"a fingerprint in a line": "the fingerprint is " + fakeFingerprint + "\nhttps://localhost:8080/auth/connect/ABC\n",
		"a short fingerprint":     "  AB:CD:EF\nhttps://localhost:8080/auth/connect/ABC\n",
		"another address":         "  " + fakeFingerprint + "\nhttps://example.com:8080/auth/connect/ABC\n",
	} {
		if _, _, err := parseBanner(text); err == nil {
			t.Errorf("%s: a banner was read", name)
		}
	}
}

// printing is a gateway that prints what it is told to, when it is told to.
type printing struct {
	mu   sync.Mutex
	text string
}

func (p *printing) print(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.text = text
}

func (p *printing) address() string { return "127.0.0.1:1" }
func (p *printing) output() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.text
}
func (p *printing) stop() (time.Duration, error) { return 0, nil }

func TestTheBannerIsWaitedFor(t *testing.T) {
	late := &printing{}
	go func() {
		time.Sleep(250 * time.Millisecond)
		late.print("  " + fakeFingerprint + "\nhttps://localhost:8080/auth/connect/ABCDEFGH\n")
	}()
	if path, _, err := waitForBanner(late, 5*time.Second); err != nil || path != "/auth/connect/ABCDEFGH" {
		t.Errorf("waitForBanner = %q, %v", path, err)
	}
	_, _, err := waitForBanner(&printing{text: "starting\n"}, 150*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "no link to pair a browser with") || !strings.Contains(err.Error(), "starting") {
		t.Errorf("a gateway that prints no link: %v", err)
	}
}

func TestTheHostsAtWhichAGatewayReachesTheApplication(t *testing.T) {
	addresses := []string{"192.168.1.20", "fd00::1"}
	if got := (&binary{}).hosts(addresses); strings.Join(got, ",") != "192.168.1.20,fd00::1" {
		t.Errorf("the program is on the computer of the application: %v", got)
	}
	got := (&image{}).hosts(addresses)
	if strings.Join(got, ",") != "192.168.1.20,fd00::1,host.docker.internal,10.0.2.2" {
		t.Errorf("a container reaches the computer through the network of its own: %v", got)
	}
	if len(addresses) != 2 {
		t.Error("the addresses of the application were changed")
	}
}

// the gateway needs port 8080, which the test cannot have taken from somebody else.
func portIsFree(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", ":8080") //nolint:gosec // To see that the port of the gateway is free.
	if err != nil {
		t.Skipf("port 8080 is in use: %v", err)
	}
	_ = listener.Close()
}

func TestAProgramIsStartedWithAHomeOfItsOwnAndStoppedAsDockerStopsIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a program on Windows is not stopped by SIGTERM")
	}
	portIsFree(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeGatewayEnv, "1")
	home := filepath.Join(t.TempDir(), "home")
	started, err := (&binary{path: self, home: home}).start()
	if err != nil {
		t.Fatal(err)
	}
	if started.address() != "127.0.0.1:8080" {
		t.Errorf("address = %s", started.address())
	}
	path, fingerprint, err := waitForBanner(started, 10*time.Second)
	if err != nil || path != "/auth/connect/ABCDEFGHIJKLMNOP" || fingerprint != fakeFingerprint {
		t.Fatalf("banner = %q, %q, %v", path, fingerprint, err)
	}
	took, err := started.stop()
	if err != nil || took > 5*time.Second {
		t.Errorf("stop = %v, %v; want a program that ends well and at once", took, err)
	}
	if logs, _ := filepath.Glob(filepath.Join(home, "gateway-*.log")); len(logs) != 1 {
		t.Errorf("the output of the program is kept in its home: %v", logs)
	}
}

func TestAProgramThatEndsBadlyIsSaidToHave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a program on Windows is not stopped by SIGTERM")
	}
	portIsFree(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeGatewayEnv, "1")
	t.Setenv(fakeExitEnv, "3")
	started, err := (&binary{path: self, home: t.TempDir()}).start()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := waitForBanner(started, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := started.stop(); err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("stop = %v, want the exit status", err)
	}
}

func TestAProgramThatIsNotThereDoesNotStart(t *testing.T) {
	portIsFree(t)
	if _, err := (&binary{path: filepath.Join(t.TempDir(), "nothing"), home: t.TempDir()}).start(); err == nil {
		t.Error("a program that does not exist was started")
	}
	// The home of the program cannot be made in a place that is a file.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&binary{path: file, home: filepath.Join(file, "home")}).start(); err == nil {
		t.Error("a home was made inside a file")
	}
}

func TestAProgramThatNeedsPort8080IsNotStartedWhereSomebodyHasIt(t *testing.T) {
	listener, err := net.Listen("tcp", ":8080") //nolint:gosec // To take the port of the gateway.
	if err != nil {
		t.Skipf("port 8080 is in use already: %v", err)
	}
	defer func() { _ = listener.Close() }()
	if _, err := (&binary{path: "unused", home: t.TempDir()}).start(); err == nil || !strings.Contains(err.Error(), "port 8080") {
		t.Errorf("start = %v, want it to say that port 8080 is taken", err)
	}
}

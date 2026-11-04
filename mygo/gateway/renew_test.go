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

package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// waitUntil polls a condition that the test cannot wait for any other way.
func waitUntil(t *testing.T, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestARenewedSessionLastsPastTheEndOfTheOneItReplaced(t *testing.T) {
	// The server of the instance has the time of its own, and a session of it lasts as long as it is told: long
	// enough for the margins below, which are about a second each, to hold on a runner that is slow.
	const lifetime = 2 * time.Second
	instance := startInstanceFor(t, lifetime)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	paired := time.Now()
	first := g.current().session

	time.Sleep(time.Second)
	if err := g.renew(context.Background()); err != nil {
		t.Fatal(err)
	}
	renewed := time.Now() // The new session was made before this, so it ends before renewed plus its lifetime.
	if second := g.current().session; second == "" || second == first {
		t.Fatalf("the session was not replaced: %q then %q", first, second)
	}
	if !g.Status().RenewedAt.After(paired) {
		t.Errorf("RenewedAt = %v, want after %v", g.Status().RenewedAt, paired)
	}
	if remembered := loadState(g.options.Dir); remembered.Session != g.current().session {
		t.Errorf("the file remembers %q, the gateway holds %q", remembered.Session, g.current().session)
	}

	// The session that the gateway began with is over by now, and the interface is served still.
	time.Sleep(time.Until(paired.Add(lifetime + 200*time.Millisecond)))
	if status, _ := b.get("/api/inventory"); status != http.StatusOK {
		t.Errorf("after the first session ended: %d, want the interface, through the renewed session", status)
	}
	// And the renewed one ends in its turn, as the instance has it: the test is not a gateway that never ends.
	time.Sleep(time.Until(renewed.Add(lifetime + 200*time.Millisecond)))
	if status, _ := b.get("/api/inventory"); status != http.StatusUnauthorized {
		t.Errorf("after the renewed session ended: %d, want 401", status)
	}
	if g.Status().Connected {
		t.Error("the gateway does not know that its session ended")
	}
}

func TestRenewingFindsOutThatTheInstanceHasEndedTheSession(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())
	instance.endSessions()

	if err := g.renew(context.Background()); !errors.Is(err, errSessionEnded) {
		t.Fatalf("renew = %v, want %v", err, errSessionEnded)
	}
	if g.Status().Connected || g.Status().Address == "" {
		t.Errorf("status = %+v, want the instance kept and the session gone", g.Status())
	}
	if err := g.renew(context.Background()); !errors.Is(err, errNotConnected) {
		t.Errorf("a gateway without a session: renew = %v, want %v", err, errNotConnected)
	}
}

func TestAnInstanceThatCannotBeReachedKeepsTheSessionOfTheGateway(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())
	instance.server.Close()

	err := g.renew(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cannot reach vfox+") {
		t.Fatalf("renew = %v, want the instance to be unreachable", err)
	}
	if !g.Status().Connected {
		t.Error("the session was dropped because the instance was out of reach: it may be back before the session ends")
	}
}

func TestRenewingNeedsAnInstanceThatAnswersLikeOne(t *testing.T) {
	for name, answer := range map[string]http.HandlerFunc{
		"an error":        func(w http.ResponseWriter, r *http.Request) { http.Error(w, "broken", http.StatusInternalServerError) },
		"nothing of JSON": func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"a link of nowhere": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url": "https://x/elsewhere/abcdef0123456789"}`))
		},
		"a token of another kind": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url": "https://x/auth/connect/../../etc"}`))
		},
	} {
		fake := httptest.NewServer(answer)
		t.Cleanup(fake.Close)
		up := newUpstream(mustURL(t, fake.URL), "", nil).withSession("session", time.Now())
		if _, err := up.mint(context.Background()); err == nil {
			t.Errorf("%s: a link was taken", name)
		}
	}
}

func TestARenewalThatFinishesAfterTheGatewayWasPairedAgainLeavesTheNewPairingAlone(t *testing.T) {
	first := startFakeInstance(t)
	second := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, first.link())
	release := first.holdTokens()
	defer release()

	renewed := make(chan error, 1)
	go func() { renewed <- g.renew(context.Background()) }()
	waitUntil(t, "the renewal to ask the first instance for a link", func() bool { return len(first.requests("/api/token")) == 1 })

	// While it waits, the person pairs the gateway with another instance.
	pairGateway(t, b, second.link())
	release()
	if err := <-renewed; err != nil {
		t.Fatal(err)
	}
	if status := g.Status(); !status.Connected || status.Address != strings.TrimPrefix(second.server.URL, "http://") {
		t.Errorf("status = %+v, want the second instance: a renewal of the first has no right to undo that", status)
	}
	if status, _ := b.get("/api/echo"); status != http.StatusOK {
		t.Errorf("the second instance is not served: %d", status)
	}
	if len(second.requests("/api/echo")) != 1 || len(first.requests("/api/echo")) != 0 {
		t.Error("the request went to the wrong instance")
	}
}

func TestKeepPairedRenewsAtOnceAndAgainUntilItIsToldToStop(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{RenewEvery: 30 * time.Millisecond})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())

	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		g.keepPaired(ctx)
		close(done)
	}()
	waitUntil(t, "three renewals", func() bool { return len(instance.requests("/api/token")) >= 3 })
	stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("keepPaired does not stop")
	}
	count := len(instance.requests("/api/token"))
	time.Sleep(100 * time.Millisecond)
	if more := len(instance.requests("/api/token")); more != count {
		t.Errorf("the gateway renews after it was stopped: %d then %d", count, more)
	}
	if !g.Status().Connected {
		t.Error("the gateway lost its session by renewing it")
	}
}

func TestKeepPairedLeavesAGatewayThatIsPairedWithNothingAlone(t *testing.T) {
	g, _ := serveGateway(t, Options{RenewEvery: 10 * time.Millisecond})
	ctx, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stop()
	done := make(chan struct{})
	go func() {
		g.keepPaired(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("keepPaired does not stop")
	}
}

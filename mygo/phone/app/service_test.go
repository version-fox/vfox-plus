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

package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/phone/internal/phonetest"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// stores returns the stores of a service, in files of the test.
func stores(t *testing.T) (store.ServerStore, store.SessionStore) {
	t.Helper()
	dir := t.TempDir()
	return store.NewFileServerStore(filepath.Join(dir, "server.json")),
		store.NewFileSessionStore(filepath.Join(dir, "session.json"))
}

// newTestService returns a service over the stores, which retries in 10 ms.
func newTestService(t *testing.T, servers store.ServerStore, sessions store.SessionStore) *Service {
	t.Helper()
	svc, err := NewService(Options{Servers: servers, Sessions: sessions, RetryDelay: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// savedComputer saves the record of d, and a session that d accepts.
func savedComputer(t *testing.T, d *desktop) (store.ServerStore, store.SessionStore) {
	t.Helper()
	servers, sessions := stores(t)
	if err := servers.Save(d.record()); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(d.issueSession("saved-session")); err != nil {
		t.Fatal(err)
	}
	return servers, sessions
}

// paired returns a service paired with d, which has not been run.
func paired(t *testing.T, d *desktop) *Service {
	t.Helper()
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	if err := svc.Pair(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	return svc
}

// startRun runs the service until the test ends, and waits for the loop to end.
func startRun(t *testing.T, svc *Service) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// settle reads a resource the way a frame does, until its load ends.
func settle[T any](t *testing.T, read func() Resource[T]) Resource[T] {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		r := read()
		if !r.Loading() {
			return r
		}
		if time.Now().After(deadline) {
			t.Fatal("a load did not end")
		}
		time.Sleep(time.Millisecond)
	}
}

// loaded settles a resource and returns its value, which must have loaded.
func loaded[T any](t *testing.T, read func() Resource[T]) T {
	t.Helper()
	r := settle(t, read)
	v, ok := r.Value()
	if r.Err() != nil || !ok {
		t.Fatalf("the load failed: %v (loaded %v)", r.Err(), ok)
	}
	return v
}

// waitFor polls ok until it holds, or fails the test.
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// wrongFingerprint is a fingerprint of the right shape that no certificate has.
var wrongFingerprint = strings.TrimSuffix(strings.Repeat("00:", 32), ":")

// anotherRecord is the record of d with the certificate of another computer: the one
// that the phone paired with is not the one that answers.
func anotherRecord(t *testing.T, d *desktop) store.Record {
	t.Helper()
	other := phonetest.Another(t)
	record := d.record()
	record.Fingerprint = pairing.FormatFingerprint(other)
	record.Certificate = store.EncodeCertificate(other)
	return record
}

func TestAServiceWithNothingSavedIsUnpairedAndMakesNoRequest(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	if got := svc.Status(); got.Session != Unpaired || got.Server != (store.Record{}) || got.Cause != nil {
		t.Errorf("status %+v, want unpaired with nothing saved", got)
	}
	if r := svc.Inventory(); r.Phase() != Empty || r.Loading() {
		t.Errorf("a read with no computer is %v, want empty and not loading", r.Phase())
	}
	if n := d.count("/api/inventory"); n != 0 {
		t.Errorf("%d requests with no computer, want none", n)
	}
}

func TestASavedComputerWithItsSessionIsCheckedWhenItRuns(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := savedComputer(t, d)
	svc := newTestService(t, servers, sessions)
	if got := svc.Status().Session; got != Connecting {
		t.Fatalf("session %v at start, want connecting", got)
	}
	startRun(t, svc)
	waitFor(t, "the computer to accept the session", func() bool { return svc.Status().Session == Paired })
	if n := d.count("/api/settings"); n != 1 {
		t.Errorf("the check made %d settings requests, want 1", n)
	}
	if got := svc.Status().Server.Base; got != d.base.String() {
		t.Errorf("the saved computer is %q, want %q", got, d.base.String())
	}
}

func TestASavedComputerThatRefusesItsSessionNeedsPairing(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := savedComputer(t, d)
	if err := sessions.Save("stale-session"); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, servers, sessions)
	startRun(t, svc)
	waitFor(t, "the phone to need pairing", func() bool { return svc.Status().Session == NeedsPairing })
	if got := KindOf(svc.Status().Cause); got != KindUnpaired {
		t.Errorf("cause kind %v, want unpaired", got)
	}
	if rec, err := servers.Load(); err != nil || rec.Base != d.base.String() {
		t.Errorf("the saved computer after the refusal: %+v, %v; want it kept", rec, err)
	}
	if r := svc.Tasks(); r.Loading() {
		t.Error("a read after the refusal sent a request")
	}
}

func TestASavedRecordWithoutASessionNeedsPairingAtOnce(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	if err := servers.Save(d.record()); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, servers, sessions)
	if got := svc.Status().Session; got != NeedsPairing {
		t.Errorf("session %v, want needs pairing", got)
	}
	startRun(t, svc)
	time.Sleep(20 * time.Millisecond)
	if n := d.count("/api/settings"); n != 0 {
		t.Errorf("%d requests for a phone without a session, want none", n)
	}
}

func TestASessionWithoutItsComputerIsForgotten(t *testing.T) {
	servers, sessions := stores(t)
	if err := sessions.Save("orphan"); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, servers, sessions)
	if got := svc.Status().Session; got != Unpaired {
		t.Errorf("session %v, want unpaired", got)
	}
	if _, err := sessions.Load(); !errors.Is(err, store.ErrNoSession) {
		t.Errorf("the orphan session is still saved: %v", err)
	}
}

func TestAChangedCertificateIsReportedAndNotTrusted(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	if err := servers.Save(anotherRecord(t, d)); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(d.issueSession("saved-session")); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, servers, sessions)
	startRun(t, svc)
	waitFor(t, "the certificate to be reported", func() bool { return svc.Status().Session == CertificateChanged })
	if !errors.Is(svc.Status().Cause, remote.ErrCertificateChanged) {
		t.Errorf("cause %v, want the certificate change", svc.Status().Cause)
	}
	if n := d.count("/api/settings"); n != 0 {
		t.Errorf("%d requests reached a computer whose certificate changed, want none", n)
	}
	if _, err := sessions.Load(); err != nil {
		t.Errorf("the session was dropped with the certificate: %v", err)
	}
}

func TestProbeReadsTheFingerprintBeforeAnyTrust(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	if candidate.Fingerprint != d.fingerprint || candidate.Link.Token != link.Token {
		t.Errorf("candidate %+v, want the fingerprint %q and the link", candidate, d.fingerprint)
	}
	if !bytes.Equal(candidate.Certificate, d.certificate) || candidate.SystemTrusted {
		t.Errorf("the candidate does not carry the self-signed certificate that was presented: trusted %t", candidate.SystemTrusted)
	}
	if got := svc.Status().Session; got != Unpaired {
		t.Errorf("probing made the phone %v; nothing is trusted before the user confirms", got)
	}
	if rec, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("probing saved %+v, %v", rec, err)
	}
}

func TestPairingSavesTheComputerAndItsSession(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	if err := svc.Pair(t.Context(), candidate); err != nil {
		t.Fatalf("Pair: %v", err)
	}
	status := svc.Status()
	if status.Session != Paired || status.Server.Fingerprint != d.fingerprint || status.Server.Base != d.base.String() {
		t.Errorf("status after pairing %+v", status)
	}
	if rec, err := servers.Load(); err != nil || rec != d.pairedRecord() {
		t.Errorf("saved record %+v, %v; want the one that pins the certificate that was compared: %+v", rec, err, d.pairedRecord())
	}
	if session, err := sessions.Load(); err != nil || session == "" {
		t.Errorf("saved session %q, %v", session, err)
	}
	if inventory := loaded(t, svc.Inventory); !inventory.Complete {
		t.Error("the inventory of the paired computer is not complete")
	}
}

func TestAPairingLinkWorksOnce(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	if err := svc.Pair(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	err := svc.Pair(t.Context(), candidate)
	if KindOf(err) != KindPairingRefused {
		t.Errorf("the second pairing with one link: %v, want it refused", err)
	}
	if got := svc.Status().Session; got != Paired {
		t.Errorf("the refused pairing changed the session to %v", got)
	}
}

func TestAPairingWithAnotherCertificateSavesNothing(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	// The user compared another certificate than the one that the computer presents now.
	candidate.Certificate = phonetest.Another(t)
	candidate.Fingerprint = pairing.FormatFingerprint(candidate.Certificate)
	err := svc.Pair(t.Context(), candidate)
	if KindOf(err) != KindCertificateChanged {
		t.Errorf("pairing with another certificate: %v, want the change refused", err)
	}
	if got := svc.Status().Session; got != Unpaired {
		t.Errorf("session %v after a refused certificate, want unpaired", got)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("a record was saved: %v", err)
	}
}

// TestACandidateWhoseCertificateIsNotTheOneComparedIsRefusedBeforeAnyConnection checks that
// the fingerprint that the user compared and the certificate that is pinned are one: a
// candidate whose certificate has another fingerprint is refused, and the computer is not
// asked for anything.
func TestACandidateWhoseCertificateIsNotTheOneComparedIsRefusedBeforeAnyConnection(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	candidate.Fingerprint = wrongFingerprint
	if err := svc.Pair(t.Context(), candidate); err == nil {
		t.Fatal("a candidate whose certificate is not the one that was compared was paired")
	}
	if n := d.count("/auth/connect/" + link.Token); n != 0 {
		t.Errorf("the computer was asked %d times for a pairing with a certificate that was not compared", n)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("a record was saved: %v", err)
	}
}

// TestACertificateTheSystemVouchesForIsNotPinned checks the other kind of server: when the
// roots of the system vouch for the certificate, nothing is pinned, and the roots go on
// deciding, so that a renewal of the certificate needs no pairing. The test server is
// self-signed, so here the roots do not vouch for it: the pairing is refused as a
// certificate that is not the paired one, which shows that the certificate was not pinned.
func TestACertificateTheSystemVouchesForIsNotPinned(t *testing.T) {
	d := newDesktop(t)
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	candidate.SystemTrusted = true
	err := svc.Pair(t.Context(), candidate)
	if KindOf(err) != KindCertificateChanged {
		t.Fatalf("pairing without a pin with a certificate that the roots do not vouch for: %v, want the certificate refused", err)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("a record was saved: %v", err)
	}
}

// TestARecordWithADamagedCertificateIsNotConnectedAndIsReplacedByAPairing checks that a
// record whose certificate is not what the phone saved is reported as damaged, with no
// connection made in its name, and that the next pairing replaces it.
func TestARecordWithADamagedCertificateIsNotConnectedAndIsReplacedByAPairing(t *testing.T) {
	d := newDesktop(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	content := `{"name": "desktop", "base": "` + d.base.String() + `", "fingerprint": "` + d.fingerprint + `", "certificate": "not base64 !!"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	servers, sessions := store.NewFileServerStore(path), store.NewFileSessionStore(filepath.Join(dir, "session.json"))
	if err := sessions.Save(d.issueSession("saved-session")); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, servers, sessions)
	if got := svc.Status(); got.Session != Unpaired || !errors.Is(got.Cause, store.ErrCorrupt) {
		t.Fatalf("status %+v, want unpaired with the damaged record as its cause", got)
	}
	startRun(t, svc)
	time.Sleep(20 * time.Millisecond)
	if n := d.count("/api/settings"); n != 0 {
		t.Errorf("%d requests were made in the name of a damaged record, want none", n)
	}
	link := d.issueLink()
	candidate := loaded(t, func() Resource[Candidate] { return svc.Probe(link) })
	if err := svc.Pair(t.Context(), candidate); err != nil {
		t.Fatalf("the pairing after a damaged record: %v", err)
	}
	if rec, err := servers.Load(); err != nil || rec != d.pairedRecord() {
		t.Errorf("the pairing did not replace the damaged record: %+v, %v", rec, err)
	}
}

func TestDisconnectForgetsTheComputerAndItsSession(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	if err := svc.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got := svc.Status(); got.Session != Unpaired || got.Server != (store.Record{}) {
		t.Errorf("status after disconnecting %+v", got)
	}
	servers, sessions := svc.servers, svc.sessions
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("the record is still saved: %v", err)
	}
	if _, err := sessions.Load(); !errors.Is(err, store.ErrNoSession) {
		t.Errorf("the session is still saved: %v", err)
	}
	if r := svc.Settings(); r.Phase() != Empty {
		t.Errorf("a read after disconnecting is %v, want empty", r.Phase())
	}
}

func TestLazyReadsLoadOnceAndShareTheLoad(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	release := d.hold("GET", "/api/inventory")
	for range 5 {
		if !svc.Inventory().Loading() {
			t.Fatal("a read that finds the inventory due does not show it loading")
		}
	}
	waitFor(t, "the request to arrive", func() bool { return d.count("/api/inventory") == 1 })
	time.Sleep(20 * time.Millisecond)
	if n := d.count("/api/inventory"); n != 1 {
		t.Errorf("%d requests while the inventory loads, want one shared load", n)
	}
	release()
	loaded(t, svc.Inventory)
	for range 5 {
		svc.Inventory()
	}
	time.Sleep(20 * time.Millisecond)
	if n := d.count("/api/inventory"); n != 1 {
		t.Errorf("%d requests after the load, want the value kept", n)
	}
}

func TestAFailedLoadIsNotRepeatedByReadsAndRefreshRetriesIt(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	d.fail("GET", "/api/tasks", 500)
	r := settle(t, svc.Tasks)
	if r.Phase() != Failed || KindOf(r.Err()) != KindServer {
		t.Fatalf("the tasks are %v with %v; want a failed load of a server error", r.Phase(), r.Err())
	}
	for range 5 {
		svc.Tasks()
	}
	time.Sleep(30 * time.Millisecond)
	if n := d.count("/api/tasks"); n != 1 {
		t.Errorf("%d requests for a failed load, want one", n)
	}
	d.fail("GET", "/api/tasks", 0)
	svc.Refresh()
	if tasks := loaded(t, svc.Tasks); len(tasks) != 1 {
		t.Errorf("after the refresh the tasks are %d, want the one task", len(tasks))
	}
	if n := d.count("/api/tasks"); n != 2 {
		t.Errorf("%d requests after a refresh, want two", n)
	}
}

func TestAMissingTaskIsNotFoundAndKeepsThePairing(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	r := settle(t, func() Resource[remote.Task] { return svc.Task(99) })
	if KindOf(r.Err()) != KindNotFound {
		t.Errorf("task 99: %v, want not found", r.Err())
	}
	if got := svc.Status().Session; got != Paired {
		t.Errorf("a missing task changed the session to %v", got)
	}
}

func TestMutationsChangeTheComputerAndMarkWhatTheyChange(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	ctx := t.Context()

	task, err := svc.StartTask(ctx, remote.TaskRequest{Operation: remote.RequestInstall, Sdk: "nodejs", Version: "20.19.0"})
	if err != nil {
		t.Fatalf("StartTask: %v", err)
	}
	listed := loaded(t, svc.Tasks)
	if !slices.ContainsFunc(listed, func(x remote.Task) bool { return x.ID == task.ID && x.Status == remote.TaskQueued }) {
		t.Errorf("the tasks after StartTask are %+v, want the queued task", listed)
	}

	if got := loaded(t, func() Resource[remote.Task] { return svc.Task(task.ID) }); got.Status != remote.TaskQueued {
		t.Fatalf("the task before the cancel is %q", got.Status)
	}
	if err := svc.CancelTask(ctx, task.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if got := loaded(t, func() Resource[remote.Task] { return svc.Task(task.ID) }); got.Status != remote.TaskCancelled {
		t.Errorf("the task after the cancel is %q, want cancelled", got.Status)
	}

	if err := svc.RetryTask(ctx, task.ID); err != nil {
		t.Fatalf("RetryTask: %v", err)
	}
	if got := loaded(t, func() Resource[remote.Task] { return svc.Task(task.ID) }); got.Status != remote.TaskQueued {
		t.Errorf("the task after the retry is %q, want queued", got.Status)
	}

	if err := svc.ClearFinished(ctx); err != nil {
		t.Fatalf("ClearFinished: %v", err)
	}
	for _, x := range loaded(t, svc.Tasks) {
		if x.ID == 1 {
			t.Error("the finished task is still listed after ClearFinished")
		}
	}

	if err := svc.PutConfig(ctx, "proxy.url", "http://proxy.example:8080"); err != nil {
		t.Fatalf("PutConfig: %v", err)
	}
	if !configHas(loaded(t, svc.Inventory), "proxy.url", "http://proxy.example:8080") {
		t.Error("the configuration after PutConfig lacks the new value")
	}
	if err := svc.DeleteConfig(ctx, "proxy.url"); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}
	if configHas(loaded(t, svc.Inventory), "proxy.url", "http://proxy.example:8080") {
		t.Error("the configuration after DeleteConfig still has the value")
	}

	if got := loaded(t, func() Resource[*remote.Profile] { return svc.Profile("java") }); got.Search.Current != "lts" {
		t.Fatalf("the search before SetSearch is %q", got.Search.Current)
	}
	if got := loaded(t, func() Resource[[]remote.AvailableVersion] { return svc.Versions("java", "") }); got[0].Note != "search lts" {
		t.Fatalf("the versions before SetSearch say %q", got[0].Note)
	}
	if err := svc.SetSearch(ctx, "java", "all"); err != nil {
		t.Fatalf("SetSearch: %v", err)
	}
	if got := loaded(t, func() Resource[*remote.Profile] { return svc.Profile("java") }); got.Search.Current != "all" {
		t.Errorf("the search after SetSearch is %q, want all", got.Search.Current)
	}
	if got := loaded(t, func() Resource[[]remote.AvailableVersion] { return svc.Versions("java", "") }); got[0].Note != "search all" {
		t.Errorf("the versions after SetSearch say %q, want the new search", got[0].Note)
	}

	if err := svc.SetVariable(ctx, "nodejs", "VFOX_MIRROR", "https://mirror.example"); err != nil {
		t.Fatalf("SetVariable: %v", err)
	}
	if got := loaded(t, func() Resource[*remote.Profile] { return svc.Profile("nodejs") }); got.Variables[0].Value != "https://mirror.example" {
		t.Errorf("the variable after SetVariable is %q", got.Variables[0].Value)
	}

	added, err := svc.AddPlugin(ctx, "https://example.test/plugin.lua", "")
	if err != nil {
		t.Fatalf("AddPlugin: %v", err)
	}
	if listed := loaded(t, svc.Tasks); !slices.ContainsFunc(listed, func(x remote.Task) bool { return x.ID == added.ID }) {
		t.Errorf("the tasks after AddPlugin lack task %d", added.ID)
	}
}

// configHas reports whether the configuration of inv has key with value.
func configHas(inv remote.Inventory, key, value string) bool {
	for _, entry := range inv.Config.Value {
		if entry.Key == key && entry.Value == value {
			return true
		}
	}
	return false
}

func TestPutSettingsAndPutWebSetTheirAnswersWithoutAReload(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	ctx := t.Context()
	if got := loaded(t, svc.Settings); got.Language != remote.LanguageSystem {
		t.Fatalf("the language before the change is %q", got.Language)
	}
	english := remote.LanguageEnglish
	settings, err := svc.PutSettings(ctx, remote.SettingsPatch{Language: &english})
	if err != nil || settings.Language != remote.LanguageEnglish {
		t.Fatalf("PutSettings = %+v, %v", settings, err)
	}
	before := d.count("/api/settings")
	if got := svc.Settings(); got.Phase() != Ready {
		v, _ := got.Value()
		t.Errorf("settings after PutSettings are %v with %+v, want the answer as they are", got.Phase(), v)
	} else if v, _ := got.Value(); v.Language != remote.LanguageEnglish {
		t.Errorf("settings after PutSettings say %q", v.Language)
	}
	time.Sleep(10 * time.Millisecond)
	if n := d.count("/api/settings"); n != before {
		t.Errorf("PutSettings made the phone read the settings again (%d requests, was %d)", n, before)
	}

	port := 9000
	state, err := svc.PutWeb(ctx, remote.WebPatch{Port: &port})
	if err != nil || state.Port != port {
		t.Fatalf("PutWeb = %+v, %v", state, err)
	}
	if got, _ := svc.Web().Value(); got.Port != port {
		t.Errorf("web after PutWeb has port %d, want %d", got.Port, port)
	}
}

// dirtyNames returns the names of the resources that are out of date, sorted.
func dirtyNames(svc *Service) []string {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	var names []string
	add := func(name string, e bool) {
		if e {
			names = append(names, name)
		}
	}
	add("settings", svc.settings.dirty)
	add("web", svc.web.dirty)
	add("inventory", svc.inventory.dirty)
	add("tasks", svc.tasks.dirty)
	for id, e := range svc.task.entries {
		add("task:"+strconv.Itoa(id), e.dirty)
	}
	for id, e := range svc.logs.entries {
		add("log:"+strconv.Itoa(id), e.dirty)
	}
	for key, e := range svc.doctors.entries {
		add("doctor:"+key.sdk+"|"+key.version, e.dirty)
	}
	for name, e := range svc.profiles.entries {
		add("profile:"+name, e.dirty)
	}
	for key, e := range svc.versions.entries {
		add("versions:"+key.name+"|"+key.search, e.dirty)
	}
	for name, e := range svc.aliases.entries {
		add("aliases:"+name, e.dirty)
	}
	for name, e := range svc.infos.entries {
		add("info:"+name, e.dirty)
	}
	slices.Sort(names)
	return names
}

func TestEventsMarkOnlyWhatTheyName(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	startRun(t, svc)
	waitFor(t, "the event stream", func() bool { return d.openStreams() == 1 })
	time.Sleep(50 * time.Millisecond) // the hello of the stream has come before the loads below

	reloadAll := func() {
		t.Helper()
		loaded(t, svc.Tasks)
		loaded(t, func() Resource[remote.Task] { return svc.Task(1) })
		loaded(t, func() Resource[string] { return svc.TaskLog(1) })
		loaded(t, svc.Inventory)
		loaded(t, svc.Web)
		loaded(t, svc.Settings)
		loaded(t, func() Resource[remote.DoctorReport] { return svc.Doctor("", "") })
		loaded(t, func() Resource[*remote.Profile] { return svc.Profile("nodejs") })
		loaded(t, func() Resource[[]remote.AvailableVersion] { return svc.Versions("nodejs", "") })
		loaded(t, func() Resource[[]remote.VersionAlias] { return svc.Aliases("nodejs") })
		loaded(t, func() Resource[remote.PluginInfo] { return svc.Info("nodejs") })
	}
	reloadAll()
	all := []string{
		"aliases:nodejs", "doctor:|", "info:nodejs", "inventory", "log:1",
		"profile:nodejs", "settings", "task:1", "tasks", "versions:nodejs|", "web",
	}
	for _, tt := range []struct {
		kind string
		want []string
	}{
		{remote.EventTask, []string{"log:1", "task:1", "tasks"}},
		{remote.EventInventory, []string{"aliases:nodejs", "doctor:|", "info:nodejs", "inventory", "profile:nodejs", "versions:nodejs|"}},
		{remote.EventVfox, []string{"doctor:|", "inventory"}},
		{remote.EventWeb, []string{"web"}},
		{remote.EventHello, all},
	} {
		reloadAll()
		if got := dirtyNames(svc); len(got) != 0 {
			t.Fatalf("before %q the resources %v are out of date", tt.kind, got)
		}
		d.emit(tt.kind)
		waitFor(t, "the event "+tt.kind, func() bool { return len(dirtyNames(svc)) > 0 })
		if got := dirtyNames(svc); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("event %q marks %v, want %v", tt.kind, got, tt.want)
		}
	}
}

func TestAStreamThatBreaksIsOpenedAgain(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	startRun(t, svc)
	waitFor(t, "the event stream", func() bool { return d.openStreams() == 1 })
	d.breakStreams()
	waitFor(t, "a second stream", func() bool { return d.openedStreams() >= 2 })
	waitFor(t, "the phone paired again", func() bool { return svc.Status().Session == Paired })
}

func TestAnUnauthorizedAnswerEndsTheSession(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	startRun(t, svc)
	waitFor(t, "the event stream", func() bool { return d.openStreams() == 1 })
	d.revoke()
	r := settle(t, svc.Inventory)
	if KindOf(r.Err()) != KindUnpaired {
		t.Fatalf("the inventory after revoking the session: %v", r.Err())
	}
	if got := svc.Status().Session; got != NeedsPairing {
		t.Errorf("session %v after the refusal, want needs pairing", got)
	}
	waitFor(t, "the stream to end", func() bool { return d.openStreams() == 0 })
	opened := d.openedStreams()
	time.Sleep(30 * time.Millisecond)
	if n := d.openedStreams(); n != opened {
		t.Errorf("the stream was opened again (%d, was %d) after the session ended", n, opened)
	}
	if _, err := svc.servers.Load(); err != nil {
		t.Errorf("the saved computer was lost: %v", err)
	}
}

func TestSuspendEndsTheStreamAndTheCallsInFlight(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	startRun(t, svc)
	waitFor(t, "the event stream", func() bool { return d.openStreams() == 1 })

	release := d.hold("GET", "/api/tasks")
	_ = svc.Tasks()
	waitFor(t, "the request to arrive", func() bool { return d.count("/api/tasks") == 1 })

	svc.Suspend()
	waitFor(t, "the stream to end", func() bool { return d.openStreams() == 0 })
	waitFor(t, "the call to end", func() bool { return !svc.Tasks().Loading() })
	if r := svc.Tasks(); r.Err() != nil || r.Phase() != Empty {
		t.Errorf("the cancelled load is %v with %v; want empty, without an error", r.Phase(), r.Err())
	}
	time.Sleep(20 * time.Millisecond)
	if n := d.count("/api/tasks"); n != 1 {
		t.Errorf("%d requests while in the background, want none new", n)
	}
	if n := d.openedStreams(); n != 1 {
		t.Errorf("%d streams opened, want the one before the suspension", n)
	}

	release()
	svc.Resume()
	waitFor(t, "the stream to open again", func() bool { return d.openStreams() == 1 })
	if tasks := loaded(t, svc.Tasks); len(tasks) != 1 {
		t.Errorf("the tasks after Resume are %d, want the one task", len(tasks))
	}
	// The hello of the new stream may mark the tasks out of date again while the load
	// runs, which loads them once more; at least one load after Resume is what the
	// test is about.
	if n := d.count("/api/tasks"); n < 2 {
		t.Errorf("%d requests for the tasks after Resume, want the load again", n)
	}
}

func TestDoReportsItsErrorsAsNotices(t *testing.T) {
	servers, sessions := stores(t)
	svc := newTestService(t, servers, sessions)
	svc.Do(func(ctx context.Context) error { return nil })
	svc.Do(func(ctx context.Context) error { return context.Canceled })
	svc.Do(func(ctx context.Context) error { return remote.ErrNotFound })

	var notices []Notice
	waitFor(t, "the notice", func() bool {
		notices = append(notices, svc.TakeNotices()...)
		return len(notices) > 0
	})
	time.Sleep(10 * time.Millisecond)
	notices = append(notices, svc.TakeNotices()...)
	if len(notices) != 1 || notices[0].Kind != KindNotFound || !errors.Is(notices[0].Err, remote.ErrNotFound) {
		t.Errorf("notices %+v, want the one not-found error", notices)
	}
}

func TestTheNotifyFunctionIsCalledWhenALoadEnds(t *testing.T) {
	d := newDesktop(t)
	svc := paired(t, d)
	frames := make(chan struct{}, 8)
	svc.SetNotify(func() {
		select {
		case frames <- struct{}{}:
		default:
		}
	})
	_ = svc.Inventory()
	select {
	case <-frames:
	case <-time.After(5 * time.Second):
		t.Fatal("the window was not asked for a frame when the load ended")
	}
}

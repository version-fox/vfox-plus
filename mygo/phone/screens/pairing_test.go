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

package screens

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// pairTestToken is the one-time token of the fake computer's pairing link.
const pairTestToken = "good-token"

// pairTestSession is the session that the fake computer gives a phone that pairs.
const pairTestSession = "session-of-the-test"

// pairTestInventory is the inventory of the fake computer: vfox 0.6.0, with Go in use.
const pairTestInventory = `{"vfox":{"availability":"ready","version":"0.6.0"},` +
	`"installed":{"loading":false,"value":[{"Name":"go","Versions":["1.22.0"],"Current":"1.22.0","HasCurrent":true},` +
	`{"Name":"node","Versions":["20.1.0"],"Current":"","HasCurrent":false}]},` +
	`"registry":{"loading":false,"value":[]},"config":{"loading":false,"value":[]},"complete":true}`

// pairDesktop is a computer for the tests: an HTTPS server that answers the requests of
// the phone with the answers the test sets, and records the tasks the phone starts.
type pairDesktop struct {
	server      *httptest.Server
	fingerprint string

	mu sync.Mutex
	// used says that the pairing link was exchanged; the link is one-time.
	used bool
	// connects counts the requests of pairing links.
	connects int
	// inventory is the body of GET /api/inventory, and inventoryStatus its status when it
	// is not 200.
	inventory       string
	inventoryStatus int
	// doctor is the body of GET /api/doctor.
	doctor string
	// tasks are the requests of POST /api/tasks, in order.
	tasks []remote.TaskRequest
}

// newPairDesktop starts a computer that answers on the loopback address.
func newPairDesktop(t *testing.T) *pairDesktop {
	t.Helper()
	fd := &pairDesktop{inventory: pairTestInventory, doctor: `{"ok":true,"findings":[]}`}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/connect/", fd.connect)
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		pairWriteJSON(w, `{"vfoxExecutable":"","extraEnvironment":{},"language":"system","updateMirror":""}`)
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		// A stream that stays open, as the computer keeps it.
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	})
	mux.HandleFunc("/api/inventory", fd.serveInventory)
	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) {
		fd.mu.Lock()
		body := fd.doctor
		fd.mu.Unlock()
		pairWriteJSON(w, body)
	})
	mux.HandleFunc("/api/tasks", fd.createTask)
	fd.server = newTLSServer(t, mux)
	fd.fingerprint = pairing.FormatFingerprint(fd.server.Certificate().Raw)
	return fd
}

// deepLink is the deep link of the pairing page that carries the pairing link of the
// computer, with the token.
func (fd *pairDesktop) deepLink(token string) string {
	return "vfoxplus://pair?link=" + url.QueryEscape(fd.server.URL+"/auth/connect/"+token)
}

// connect answers a pairing link: the one-time token is exchanged for a session.
func (fd *pairDesktop) connect(w http.ResponseWriter, r *http.Request) {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	fd.connects++
	if strings.TrimPrefix(r.URL.Path, "/auth/connect/") != pairTestToken || fd.used {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"error":"the pairing link is invalid, already used or expired"}`)
		return
	}
	fd.used = true
	http.SetCookie(w, &http.Cookie{Name: "vfox-plus-session", Value: pairTestSession, Path: "/", HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}

// serveInventory answers the inventory, or the status the test set for it.
func (fd *pairDesktop) serveInventory(w http.ResponseWriter, r *http.Request) {
	fd.mu.Lock()
	body, status := fd.inventory, fd.inventoryStatus
	fd.mu.Unlock()
	if status != 0 {
		http.Error(w, "vfox did not answer", status)
		return
	}
	pairWriteJSON(w, body)
}

// createTask records the task the phone starts, and answers it as queued.
func (fd *pairDesktop) createTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request remote.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fd.mu.Lock()
	fd.tasks = append(fd.tasks, request)
	fd.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprint(w, `{"id":1,"operation":"doctorAdopt","status":"queued","commandLine":"","createdAt":"","log":[],"droppedLines":0}`)
}

// tasksStarted returns the tasks the phone started, in order.
func (fd *pairDesktop) tasksStarted() []remote.TaskRequest {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	return append([]remote.TaskRequest(nil), fd.tasks...)
}

// pairWriteJSON answers body as JSON.
func pairWriteJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, body)
}

// pairSavedRoot returns the window of a phone that has saved the computer at base, with
// the fingerprint given. The phone is paired when session is set, and otherwise the
// saved computer needs pairing. The configuration directory is the test's own.
func pairSavedRoot(t *testing.T, base, fingerprint string, session bool) *Root {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config := filepath.Join(dir, configDirName)
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	record, err := json.Marshal(store.Record{Name: "127.0.0.1", Base: base, Fingerprint: fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, serverFile), record, 0o600); err != nil {
		t.Fatal(err)
	}
	if session {
		saved := fmt.Sprintf(`{"session":%q}`, pairTestSession)
		if err := os.WriteFile(filepath.Join(config, sessionFile), []byte(saved), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// pairStartService runs the connection of the phone, as the app does, until the test ends.
func pairStartService(t *testing.T, root *Root) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go root.Service().Run(ctx)
}

// pairWaitFor draws the frames until the frame shows text. It fails when the text does not
// come within a few seconds: the reads and the calls run in the background.
func pairWaitFor(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !hasText(tt, text) {
		if time.Now().After(deadline) {
			t.Fatalf("the frame does not show %q; it shows %q", text, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
	}
}

// pairWaitUntil draws the frames until ok reports true, and fails at the deadline.
func pairWaitUntil(t *testing.T, tt *ui.Tester, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen; the frame shows %q", what, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
	}
}

// pairSavePicture writes the frame to the directory named by the VFOXPLUS_SHOTS variable,
// for the people who read the screens; without the variable nothing is written.
func pairSavePicture(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("VFOXPLUS_SHOTS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The transitions of the dialogs and of the labels end after a moment: the picture
	// is of the settled screen.
	tt.Advance(2 * time.Second)
	tt.Frame()
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

// pairCaptureStdout redirects the standard output to a buffer until the test ends, and
// returns a function that says what has been written so far.
func pairCaptureStdout(t *testing.T) func() string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	var (
		mu  sync.Mutex
		out strings.Builder
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	saved := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = saved
		_ = writer.Close()
		<-done
		_ = reader.Close()
	})
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		return out.String()
	}
}

// TestTheDeepLinkOpensTheFingerprintDialog checks that a deep link fills the field,
// probes the computer and asks for the fingerprint, and does not pair by itself.
func TestTheDeepLinkOpensTheFingerprintDialog(t *testing.T) {
	fd := newPairDesktop(t)
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.OpenURL(fd.deepLink(pairTestToken))
	pairWaitFor(t, tt, "Trust and pair")
	if !hasText(tt, "Trust this computer?") {
		t.Errorf("the dialog has no title; the frame shows %q", tt.Texts())
	}
	if !hasText(tt, fd.fingerprint) {
		t.Errorf("the dialog does not show the fingerprint %s; it shows %q", fd.fingerprint, tt.Texts())
	}
	if want := fd.server.URL + "/auth/connect/" + pairTestToken; root.linkText != want {
		t.Errorf("the field holds %q, want the pairing link %q", root.linkText, want)
	}
	pairSavePicture(t, tt, "pair-confirm")
	if session := root.Service().Status().Session; session != app.Unpaired {
		t.Errorf("the deep link paired the phone: session %v", session)
	}
	fd.mu.Lock()
	connects := fd.connects
	fd.mu.Unlock()
	if connects != 0 {
		t.Errorf("the deep link exchanged the token %d times before the user confirmed", connects)
	}
}

// TestTrustAndPairPairsThePhoneAndPrintsTheLine checks the confirmation: the phone pairs
// once the user taps Trust and pair, goes to the home tab, and the standard output has
// the line of the pairing.
func TestTrustAndPairPairsThePhoneAndPrintsTheLine(t *testing.T) {
	fd := newPairDesktop(t)
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	out := pairCaptureStdout(t)
	root.OpenURL(fd.deepLink(pairTestToken))
	pairWaitFor(t, tt, "Trust and pair")
	if err := tt.Click("Trust and pair"); err != nil {
		t.Fatalf("Click(Trust and pair): %v", err)
	}
	pairWaitUntil(t, tt, "the pairing", func() bool { return root.Service().Status().Session == app.Paired })
	pairWaitUntil(t, tt, "the line of the pairing", func() bool {
		return strings.Contains(out(), "vfoxplus: paired")
	})
	want := "vfoxplus: paired host=127.0.0.1 fingerprint=" + fd.fingerprint + "\n"
	if got := out(); !strings.Contains(got, want) {
		t.Errorf("the standard output is %q, want the line %q", got, want)
	}
	tt.Frame()
	if root.App().Screen() != app.ScreenTabs {
		t.Errorf("after pairing the window shows the pairing page, not the tabs")
	}
	if hasText(tt, "Trust and pair") {
		t.Errorf("the dialog is still open after pairing")
	}
	pairSavePicture(t, tt, "pair-home-after")
}

// TestAFailedProbeSaysWhyInTheDialog checks that a computer that does not answer is
// said so in the dialog, which offers no way to trust it.
func TestAFailedProbeSaysWhyInTheDialog(t *testing.T) {
	fd := newPairDesktop(t)
	closed := fd.server.URL
	fd.server.Close()
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.OpenURL("vfoxplus://pair?link=" + url.QueryEscape(closed+"/auth/connect/"+pairTestToken))
	pairWaitFor(t, tt, noticeText(app.KindUnreachable))
	if hasText(tt, "Trust and pair") {
		t.Error("a computer that did not answer can be trusted")
	}
	pairSavePicture(t, tt, "pair-error")
}

// TestAPairingLinkThatIsNotOneSaysSoInTheField checks the validation of the field.
func TestAPairingLinkThatIsNotOneSaysSoInTheField(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.linkText = "not a link"
	tt.Frame()
	if !hasText(tt, noticeText(app.KindNotLink)) {
		t.Errorf("the field does not say that the text is not a link; the frame shows %q", tt.Texts())
	}
	if hasText(tt, "Trust and pair") {
		t.Error("a text that is not a link opened the dialog")
	}
	pairSavePicture(t, tt, "pair-link-invalid")
}

// TestPasteFillsTheFieldFromTheClipboard checks the paste entry, with a clipboard that
// holds a link and with one that holds no text.
func TestPasteFillsTheFieldFromTheClipboard(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	link := "https://192.168.1.10:8369/auth/connect/abc123"
	tt.SetClipboard(link)
	tt.Frame()
	if err := tt.Click("Paste"); err != nil {
		t.Fatalf("Click(Paste): %v", err)
	}
	tt.Frame()
	if root.linkText != link {
		t.Errorf("the field holds %q after the paste, want %q", root.linkText, link)
	}
	tt.SetClipboard("")
	if err := tt.Click("Paste"); err != nil {
		t.Fatalf("Click(Paste) with an empty clipboard: %v", err)
	}
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "pairClipboardEmpty"))
}

// TestThePhotoEntryDegradesWithoutACamera checks the photo entry where no camera can be
// used, as on a desktop, where the window has no camera and the permission is not asked.
func TestThePhotoEntryDegradesWithoutACamera(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	if err := tt.Click("Photo"); err != nil {
		t.Fatalf("Click(Photo): %v", err)
	}
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "pairCameraUnavailable"))
	pairSavePicture(t, tt, "pair-photo-unavailable")
}

// TestTheUnpairedPageHasItsEntries checks the first screen: its lead, the field, the
// three ways to get a link, the button that connects and the way to learn how.
func TestTheUnpairedPageHasItsEntries(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	for _, text := range []string{"This phone", "Paste", "Photo", "Picture", "Connect", "How to pair"} {
		if !hasText(tt, text) {
			t.Errorf("the pairing page has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	pairSavePicture(t, tt, "pair-unpaired")
}

// TestAPairedPhoneCanPairANewLinkAboveTheTabs checks that the page of a new link keeps
// the session, and shows the form, above the tabs.
func TestAPairedPhoneCanPairANewLinkAboveTheTabs(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, true)
	pairStartService(t, root)
	tt := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, tt, "the connection", func() bool { return root.Service().Status().Session == app.Paired })
	if !root.App().Push(app.Page{Kind: app.PagePair}) {
		t.Fatal("Push refused the pairing page")
	}
	settle(tt)
	if !hasText(tt, "Paste") || !hasText(tt, "Connect") {
		t.Errorf("the page of a new link shows %q; want the pairing form", tt.Texts())
	}
	if root.Service().Status().Session != app.Paired {
		t.Errorf("the session of the phone is %v while a new link is paired", root.Service().Status().Session)
	}
	pairSavePicture(t, tt, "pair-new-link")
}

// TestTheDeepLinkOfAPairedPhoneOpensThePairingPage checks that a deep link opens the
// pairing page above the tabs of a paired phone, and asks for the fingerprint there.
func TestTheDeepLinkOfAPairedPhoneOpensThePairingPage(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, true)
	pairStartService(t, root)
	tt := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, tt, "the connection", func() bool { return root.Service().Status().Session == app.Paired })
	root.OpenURL(fd.deepLink(pairTestToken))
	pairWaitFor(t, tt, "Trust and pair")
	if top, ok := root.App().Top(); !ok || top.Kind != app.PagePair {
		t.Errorf("the top page is %+v, %v; want the pairing page", top, ok)
	}
	if root.Service().Status().Session != app.Paired {
		t.Errorf("the deep link changed the session to %v", root.Service().Status().Session)
	}
}

// TestTheHowToPairDialogExplainsTheSteps checks the explanation of the pairing, which
// the page offers under its button.
func TestTheHowToPairDialogExplainsTheSteps(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	if err := tt.Click("How to pair"); err != nil {
		t.Fatalf("Click(How to pair): %v", err)
	}
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "pairHowTitle"))
	pairSavePicture(t, tt, "pair-how")
}

// TestACameraWithoutPermissionAsksForIt checks the dialog that a phone whose camera is
// denied shows: why the camera is needed, and the settings where the user allows it.
func TestACameraWithoutPermissionAsksForIt(t *testing.T) {
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.pairCameraDenied()
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "pairCameraTitle"))
	for _, text := range []string{i18n.TextIn(i18n.English, "openSettings"), "Cancel"} {
		if !hasText(tt, text) {
			t.Errorf("the dialog has no %q; the frame shows %q", text, tt.Texts())
		}
	}
}

// pairPictureFile writes data as a file of the test's directory, and returns its path.
func pairPictureFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "picture")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestAPictureOfACodeGivesItsLinkAndIsRemoved checks the reading of a picture: a picture
// of a pairing code gives its link, and the copy of the picture is removed afterwards.
func TestAPictureOfACodeGivesItsLinkAndIsRemoved(t *testing.T) {
	link := "https://192.168.1.10:8369/auth/connect/Ab-9_z"
	code, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, code.Image(256)); err != nil {
		t.Fatal(err)
	}
	path := pairPictureFile(t, buf.Bytes())
	got, note := pairReadCode(path)
	if got != link || note != "" {
		t.Errorf("pairReadCode = %q, %q; want the link and no note", got, note)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the copy of the picture was kept: %v", err)
	}
}

// TestAPictureThatHoldsNoCodeSaysWhy checks the notes of the pictures that give no link:
// a picture without a code, a file that is not a picture, and a copy that is missing.
func TestAPictureThatHoldsNoCodeSaysWhy(t *testing.T) {
	var blank bytes.Buffer
	if err := png.Encode(&blank, image.NewGray(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		path string
		want string
	}{
		{"a picture without a code", pairPictureFile(t, blank.Bytes()), "pairNoQRCode"},
		{"a file that is not a picture", pairPictureFile(t, []byte("not a picture")), "pairNotPicture"},
		{"a copy that is missing", filepath.Join(t.TempDir(), "missing"), "pairPhotoFailed"},
	}
	for _, c := range cases {
		link, note := pairReadCode(c.path)
		if link != "" || note != c.want {
			t.Errorf("%s: pairReadCode = %q, %q; want no link and the note %q", c.name, link, note, c.want)
		}
	}
}

// TestConnectAsksForTheFingerprintOfTheTypedLink checks the button that connects with the
// link of the field: it asks for the fingerprint of the computer, and does not pair.
func TestConnectAsksForTheFingerprintOfTheTypedLink(t *testing.T) {
	fd := newPairDesktop(t)
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.linkText = fd.server.URL + "/auth/connect/" + pairTestToken
	tt.Frame()
	if err := tt.Click("Connect"); err != nil {
		t.Fatalf("Click(Connect): %v", err)
	}
	pairWaitFor(t, tt, fd.fingerprint)
	if !hasText(tt, "Trust and pair") {
		t.Errorf("the dialog has no way to trust the computer; the frame shows %q", tt.Texts())
	}
	if root.Service().Status().Session != app.Unpaired {
		t.Errorf("connecting paired the phone before the confirmation: %v", root.Service().Status().Session)
	}
}

// TestARefusedLinkSaysSoInTheDialog checks a pairing that the computer refuses, because
// the link was used before: the dialog says why, and the phone stays unpaired.
func TestARefusedLinkSaysSoInTheDialog(t *testing.T) {
	fd := newPairDesktop(t)
	fd.used = true
	root := newRoot(t)
	tt := ui.NewTester(root.View, 480, 900)
	root.OpenURL(fd.deepLink(pairTestToken))
	pairWaitFor(t, tt, "Trust and pair")
	if err := tt.Click("Trust and pair"); err != nil {
		t.Fatalf("Click(Trust and pair): %v", err)
	}
	pairWaitFor(t, tt, noticeText(app.KindPairingRefused))
	if hasText(tt, "Trust and pair") {
		t.Error("the dialog still offers to trust after the link was refused")
	}
	if root.Service().Status().Session != app.Unpaired {
		t.Errorf("a refused link paired the phone: %v", root.Service().Status().Session)
	}
	pairSavePicture(t, tt, "pair-refused")
}

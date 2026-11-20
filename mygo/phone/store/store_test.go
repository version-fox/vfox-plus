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

package store_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

const (
	sampleSession = "9f2c0d7e5b1a4c3d8e6f7a8b9c0d1e2f"
	sampleFinger  = "BA:78:16:BF:8F:01:CF:EA:41:41:40:DE:5D:AE:22:23:B0:03:61:A3:96:17:7A:9C:B4:10:FF:61:F2:00:15:AD"
)

func sampleRecord() store.Record {
	return store.Record{Name: "Study PC", Base: "https://192.168.1.10:8369", Fingerprint: sampleFinger}
}

// requirePrivate checks that the file at path can be read and written by its
// owner only. Windows has no such mode, so there the check is skipped.
func requirePrivate(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("%s has mode %o, want 600", path, mode)
	}
}

func TestServerRecordRoundTripsAndIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	servers := store.NewFileServerStore(path)
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Fatalf("Load before Save = %v, want ErrNoRecord", err)
	}
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	got, err := servers.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != sampleRecord() {
		t.Fatalf("Load = %+v, want %+v", got, sampleRecord())
	}
	requirePrivate(t, path)

	// The record is plain JSON with the three fields.
	// The file is in the test's own temporary directory.
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"name"`, `"base"`, `"fingerprint"`} {
		if !bytes.Contains(content, []byte(key)) {
			t.Errorf("the record file lacks %s: %s", key, content)
		}
	}
}

func TestAPinnedCertificateIsSavedWithTheRecordAndReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	servers := store.NewFileServerStore(path)
	der := []byte{0x30, 0x82, 0x01, 0x0a, 0xde, 0xad, 0xbe, 0xef} // Not a certificate: the store keeps bytes.
	record := sampleRecord()
	record.Certificate = store.EncodeCertificate(der)
	if err := servers.Save(record); err != nil {
		t.Fatal(err)
	}
	got, err := servers.Load()
	if err != nil || got != record {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, record)
	}
	pinned, err := got.PinnedCertificate()
	if err != nil || !bytes.Equal(pinned, der) {
		t.Fatalf("PinnedCertificate = %x, %v; want %x", pinned, err, der)
	}
	// The certificate is public, so it is in the record file, under its own key.
	// The file is in the test's own temporary directory.
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte(`"certificate"`)) {
		t.Errorf("the record file lacks the certificate: %s", content)
	}
}

func TestARecordThatPinsNothingSaysSoAndLeavesOutTheKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	servers := store.NewFileServerStore(path)
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	got, err := servers.Load()
	if err != nil {
		t.Fatal(err)
	}
	if pinned, err := got.PinnedCertificate(); err != nil || pinned != nil {
		t.Errorf("PinnedCertificate of a record without one = %x, %v; want nil", pinned, err)
	}
	// The file is in the test's own temporary directory.
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(content, []byte(`"certificate"`)) {
		t.Errorf("a record that pins nothing has the key of the certificate: %s", content)
	}
	if store.EncodeCertificate(nil) != "" {
		t.Error("EncodeCertificate of nothing is not empty")
	}
}

func TestACertificateThatIsNotBase64IsDamagedAndIsNotSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	servers := store.NewFileServerStore(path)
	bad := sampleRecord()
	bad.Certificate = "not base64 !!"
	if err := servers.Save(bad); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("Save of a record with a damaged certificate = %v, want ErrCorrupt", err)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("a record that was refused is saved: %v", err)
	}
	// A file that holds one is damaged, and the next Save replaces it.
	if err := os.WriteFile(path, []byte(`{"name": "Study PC", "base": "https://192.168.1.10:8369", "certificate": "not base64 !!"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("a record with a damaged certificate: Load = %v, want ErrCorrupt", err)
	}
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	if _, err := servers.Load(); err != nil {
		t.Errorf("Save did not replace the damaged record: %v", err)
	}
}

func TestSaveReplacesTheRecordAndForgetRemovesIt(t *testing.T) {
	servers := store.NewFileServerStore(filepath.Join(t.TempDir(), "server.json"))
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	other := store.Record{Name: "Lab", Base: "https://lab.example:8369"}
	if err := servers.Save(other); err != nil {
		t.Fatal(err)
	}
	if got, err := servers.Load(); err != nil || got != other {
		t.Fatalf("Load after a second Save = %+v, %v; want %+v", got, err, other)
	}
	if err := servers.Forget(); err != nil {
		t.Fatal(err)
	}
	if _, err := servers.Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Fatalf("Load after Forget = %v, want ErrNoRecord", err)
	}
	if err := servers.Forget(); err != nil {
		t.Fatalf("a second Forget = %v, want nil", err)
	}
}

func TestSessionRoundTripsAndIsPrivateAndApartFromTheRecord(t *testing.T) {
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server.json")
	sessionPath := filepath.Join(dir, "session.json")
	servers := store.NewFileServerStore(serverPath)
	sessions := store.NewFileSessionStore(sessionPath)
	if _, err := sessions.Load(); !errors.Is(err, store.ErrNoSession) {
		t.Fatalf("Load before Save = %v, want ErrNoSession", err)
	}
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(sampleSession); err != nil {
		t.Fatal(err)
	}
	got, err := sessions.Load()
	if err != nil || got != sampleSession {
		t.Fatalf("Load = %q, %v; want %q", got, err, sampleSession)
	}
	requirePrivate(t, sessionPath)

	// The credential is never written into the record, and the record survives
	// the session being forgotten.
	// The file is in the test's own temporary directory.
	record, err := os.ReadFile(filepath.Clean(serverPath))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(record), sampleSession) {
		t.Fatalf("the server record holds the session: %s", record)
	}
	if err := sessions.Forget(); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Load(); !errors.Is(err, store.ErrNoSession) {
		t.Fatalf("Load after Forget = %v, want ErrNoSession", err)
	}
	if _, err := servers.Load(); err != nil {
		t.Fatalf("forgetting the session removed the record: %v", err)
	}
}

func TestSaveRejectsWhatCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	if err := store.NewFileServerStore(filepath.Join(dir, "server.json")).Save(store.Record{Name: "no address"}); err == nil {
		t.Error("a record without an address was saved")
	}
	if err := store.NewFileSessionStore(filepath.Join(dir, "session.json")).Save(""); err == nil {
		t.Error("an empty session was saved")
	}
}

func TestEmptyFilesAreNothingSaved(t *testing.T) {
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server.json")
	sessionPath := filepath.Join(dir, "session.json")
	for _, path := range []string{serverPath, sessionPath} {
		if err := os.WriteFile(path, []byte(" \n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.NewFileServerStore(serverPath).Load(); !errors.Is(err, store.ErrNoRecord) {
		t.Errorf("an empty record file: Load = %v, want ErrNoRecord", err)
	}
	if _, err := store.NewFileSessionStore(sessionPath).Load(); !errors.Is(err, store.ErrNoSession) {
		t.Errorf("an empty session file: Load = %v, want ErrNoSession", err)
	}
}

func TestDamagedFilesAreReportedAndReplacedBySave(t *testing.T) {
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server.json")
	sessionPath := filepath.Join(dir, "session.json")
	damaged := map[string]string{
		serverPath:  `{"name": "Study PC", "base": `,
		sessionPath: `{"session": `,
	}
	for path, content := range damaged {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.NewFileServerStore(serverPath).Load(); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("a broken record: Load = %v, want ErrCorrupt", err)
	}
	if _, err := store.NewFileSessionStore(sessionPath).Load(); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("a broken session: Load = %v, want ErrCorrupt", err)
	}

	// A record without an address, and a session that is an empty string, are
	// damaged too.
	if err := os.WriteFile(serverPath, []byte(`{"name": "no address"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewFileServerStore(serverPath).Load(); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("a record without an address: Load = %v, want ErrCorrupt", err)
	}
	if err := os.WriteFile(sessionPath, []byte(`{"session": ""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewFileSessionStore(sessionPath).Load(); !errors.Is(err, store.ErrCorrupt) {
		t.Errorf("an empty session: Load = %v, want ErrCorrupt", err)
	}

	// Save writes over the damage.
	servers := store.NewFileServerStore(serverPath)
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	if got, err := servers.Load(); err != nil || got != sampleRecord() {
		t.Fatalf("Load after Save over damage = %+v, %v", got, err)
	}
}

func TestSaveCreatesTheDirectoryAndLeavesNoTemporaryFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "phone")
	servers := store.NewFileServerStore(filepath.Join(dir, "server.json"))
	if err := servers.Save(sampleRecord()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "server.json" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("the directory holds %v, want only server.json", names)
	}
}

func TestUnreadableFileIsAnErrorNotNothingSaved(t *testing.T) {
	path := t.TempDir() // a directory cannot be read as a file
	_, err := store.NewFileServerStore(path).Load()
	if err == nil || errors.Is(err, store.ErrNoRecord) || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load of a directory = %v, want a read error", err)
	}
}

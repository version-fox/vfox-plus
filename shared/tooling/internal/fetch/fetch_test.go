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

package fetch

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const testName = "pkg_1.0_amd64.deb"

var testContent = func() []byte {
	content := make([]byte, 300_000)
	rand.New(rand.NewSource(7)).Read(content)
	return content
}()

var testSHA256 = "SHA256:" + sha256Hex(testContent)
var testWrong = func() []byte {
	wrong := append([]byte(nil), testContent...)
	wrong[1000] ^= 0xFF
	return wrong
}()

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// mirror is an HTTP server that has the file; the Range header of each
// request is in requests.
type mirror struct {
	server  *httptest.Server
	mu      sync.Mutex
	ranges  []string
	content []byte
	status  int
	rate    int
	ignore  bool
	cut     int
}

func newMirror(t *testing.T, content []byte, options ...func(*mirror)) *mirror {
	t.Helper()
	m := &mirror{content: content, cut: -1}
	for _, option := range options {
		option(m)
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.server.Close)
	return m
}

func withStatus(status int) func(*mirror) {
	return func(m *mirror) { m.status = status }
}

func withRate(rate int) func(*mirror) {
	return func(m *mirror) { m.rate = rate }
}

func withIgnoreRange() func(*mirror) {
	return func(m *mirror) { m.ignore = true }
}

func withCutAfter(cut int) func(*mirror) {
	return func(m *mirror) { m.cut = cut }
}

func (m *mirror) url() string { return m.server.URL + "/ubuntu/pool/" + testName }

func (m *mirror) requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.ranges...)
}

func (m *mirror) serve(w http.ResponseWriter, r *http.Request) {
	rangeHeader := r.Header.Get("Range")
	m.mu.Lock()
	// Only GETs resume a download; the HEAD go-getter sends first to learn
	// whether ranges are supported is not part of the download.
	if r.Method != http.MethodHead {
		m.ranges = append(m.ranges, rangeHeader)
	}
	m.mu.Unlock()
	status := m.status
	if r.URL.Path != "/ubuntu/pool/"+testName {
		status = http.StatusNotFound
	}
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	data := m.content
	start := 0
	if rangeHeader != "" && !m.ignore {
		match := regexp.MustCompile(`^bytes=(\d+)-`).FindStringSubmatch(rangeHeader)
		start, _ = strconv.Atoi(match[1])
		if start >= len(data) {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(data)))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(data)-1, len(data)))
		w.Header().Set("Content-Length", strconv.Itoa(len(data)-start))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		// A mirror that ignores ranges still answers the whole file.
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
	}
	body := data[start:]
	if r.Method == http.MethodHead {
		// Like a real mirror, answer a HEAD with the headers only; the
		// handler must not stream the body, which the server discards.
		return
	}
	flusher, _ := w.(http.Flusher)
	chunk := 65536
	if m.rate > 0 {
		chunk = m.rate / 20
		if chunk < 1 {
			chunk = 1
		}
	}
	for sent := 0; sent < len(body); {
		if m.cut >= 0 && sent >= m.cut {
			// Cut the connection: fewer bytes than announced.
			return
		}
		part := body[sent:]
		if len(part) > chunk {
			part = part[:chunk]
		}
		if m.cut >= 0 && sent+len(part) > m.cut {
			part = part[:m.cut-sent]
		}
		if _, err := w.Write(part); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
		sent += len(part)
		if m.rate > 0 {
			time.Sleep(time.Duration(len(part)) * time.Second / time.Duration(m.rate))
		}
	}
}

type fixture struct {
	directory string
	output    *bytes.Buffer
	oldStdout io.Writer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "partial"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := &fixture{directory: directory, output: &bytes.Buffer{}}
	f.oldStdout = stdout
	stdout = f.output
	t.Cleanup(func() { stdout = f.oldStdout })
	return f
}

func (f *fixture) run(t *testing.T, addresses []string, options ...runOption) error {
	t.Helper()
	config := runConfig{seconds: 60, minSpeed: "150000", speedTime: "1", hash: testSHA256, size: len(testContent)}
	for _, option := range options {
		option(&config)
	}
	t.Setenv("APT_DEADLINE", strconv.FormatInt(time.Now().Unix()+config.seconds, 10))
	t.Setenv("APT_MIN_SPEED", config.minSpeed)
	t.Setenv("APT_SPEED_TIME", config.speedTime)
	args := append([]string{f.directory, strconv.Itoa(config.size), testName, config.hash}, addresses...)
	return Run(args)
}

type runConfig struct {
	seconds   int64
	minSpeed  string
	speedTime string
	hash      string
	size      int
}

type runOption func(*runConfig)

func withSeconds(seconds int64) runOption {
	return func(c *runConfig) { c.seconds = seconds }
}

func withMinSpeed(minSpeed, speedTime string) runOption {
	return func(c *runConfig) { c.minSpeed, c.speedTime = minSpeed, speedTime }
}

func withHash(hash string) runOption {
	return func(c *runConfig) { c.hash = hash }
}

func (f *fixture) archive(t *testing.T) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(f.directory, testName))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func (f *fixture) assertFetched(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run: %v, output: %s", err, f.output.String())
	}
	if content := f.archive(t); !bytes.Equal(content, testContent) {
		t.Fatal("archive does not hold the content")
	}
	if _, err := os.Stat(filepath.Join(f.directory, "partial", testName)); !os.IsNotExist(err) {
		t.Fatal("nothing may be left in partial/")
	}
}

func TestRunDownloadsAndMovesOutOfPartial(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if ranges := m.requests(); len(ranges) != 1 || ranges[0] != "" {
		t.Fatalf("ranges = %q", ranges)
	}
	if f.output.String() != "" {
		t.Fatalf("output = %q", f.output.String())
	}
}

func TestRunContinuesPartial(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, "partial", testName), testContent[:100_000], 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if ranges := m.requests(); len(ranges) != 1 || ranges[0] != "bytes=100000-" {
		t.Fatalf("ranges = %q", ranges)
	}
}

func TestRunContinuesAfterCut(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent, withCutAfter(100_000))
	f.assertFetched(t, f.run(t, []string{m.url()}))
	want := []string{"", "bytes=100000-", "bytes=200000-"}
	if ranges := m.requests(); !equalStrings(ranges, want) {
		t.Fatalf("ranges = %q, want %q", ranges, want)
	}
}

func TestRunCompletePartialNeedsNoRequest(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, "partial", testName), testContent, 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if ranges := m.requests(); len(ranges) != 0 {
		t.Fatalf("ranges = %q", ranges)
	}
}

func TestRunOversizePartialStartsAgain(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, "partial", testName), append(testContent, []byte("more")...), 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if ranges := m.requests(); len(ranges) != 1 || ranges[0] != "" {
		t.Fatalf("ranges = %q", ranges)
	}
}

func TestRunServerWithoutRangeSupport(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent, withIgnoreRange())
	if err := os.WriteFile(filepath.Join(f.directory, "partial", testName), testContent[:100_000], 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	want := []string{"bytes=100000-", ""}
	if ranges := m.requests(); !equalStrings(ranges, want) {
		t.Fatalf("ranges = %q, want %q", ranges, want)
	}
}

func TestRunNextMirrorOn404(t *testing.T) {
	f := newFixture(t)
	first := newMirror(t, testContent, withStatus(http.StatusNotFound))
	second := newMirror(t, testContent)
	f.assertFetched(t, f.run(t, []string{first.url(), second.url()}))
	if n, m := len(first.requests()), len(second.requests()); n != 1 || m != 1 {
		t.Fatalf("requests = (%d, %d)", n, m)
	}
}

func TestRunSlowMirrorThenContinue(t *testing.T) {
	f := newFixture(t)
	first := newMirror(t, testContent, withRate(30_000))
	second := newMirror(t, testContent)
	err := f.run(t, []string{first.url(), second.url()})
	f.assertFetched(t, err)
	if n := len(first.requests()); n != 1 {
		t.Fatalf("first requests = %d", n)
	}
	ranges := second.requests()
	if len(ranges) != 1 {
		t.Fatalf("second requests = %d", len(ranges))
	}
	if matched, _ := regexp.MatchString(`^bytes=[1-9]\d*-$`, ranges[0]); !matched {
		t.Fatalf("second range = %q, want a continuation", ranges[0])
	}
	matched, _ := regexp.MatchString(
		testName+`: \d+ of \d+ bytes, \d+ bytes a second from 127\.0\.0\.1:\d+: curl: \(28\)`,
		f.output.String())
	if !matched {
		t.Fatalf("output = %q", f.output.String())
	}
}

func TestRunBestMirrorWhenAllSlow(t *testing.T) {
	f := newFixture(t)
	slow := newMirror(t, testContent, withRate(30_000))
	faster := newMirror(t, testContent, withRate(100_000))
	f.assertFetched(t, f.run(t, []string{slow.url(), faster.url()}))
	if n := len(slow.requests()); n != 1 {
		t.Fatalf("slower mirror asked %d times, want 1", n)
	}
	if n := len(faster.requests()); n != 2 {
		t.Fatalf("better mirror asked %d times, want 2", n)
	}
}

func TestRunGivesUpWhenNoMirrorHasIt(t *testing.T) {
	f := newFixture(t)
	first := newMirror(t, testContent, withStatus(http.StatusNotFound))
	second := newMirror(t, testContent, withStatus(http.StatusNotFound))
	err := f.run(t, []string{first.url(), second.url()}, withSeconds(120))
	if err == nil || !strings.Contains(err.Error(), "no mirror sends it") {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(f.directory, testName)); !os.IsNotExist(statErr) {
		t.Fatal("no archive may be moved")
	}
	if n, m := len(first.requests()), len(second.requests()); n != 3 || m != 3 {
		t.Fatalf("requests = (%d, %d), want three rounds", n, m)
	}
}

func TestRunKeepsDeadlinePartial(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent, withRate(50_000))
	err := f.run(t, []string{m.url()}, withSeconds(2), withMinSpeed("1000", "10"))
	if err == nil {
		t.Fatal("want a deadline error")
	}
	if _, statErr := os.Stat(filepath.Join(f.directory, testName)); !os.IsNotExist(statErr) {
		t.Fatal("no archive may be moved")
	}
	info, statErr := os.Stat(filepath.Join(f.directory, "partial", testName))
	if statErr != nil {
		t.Fatal("what came must be kept")
	}
	if info.Size() <= 0 || info.Size() >= int64(len(testContent)) {
		t.Fatalf("partial size = %d", info.Size())
	}
}

func TestRunChecksHashBeforeMoving(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent[:], withCutAfter(-1))
	m.content = testWrong
	err := f.run(t, []string{m.url()})
	if err == nil || !strings.Contains(err.Error(), "no mirror sends the right file") {
		t.Fatalf("err = %v", err)
	}
	for _, path := range []string{testName, filepath.Join("partial", testName)} {
		if _, statErr := os.Stat(filepath.Join(f.directory, path)); !os.IsNotExist(statErr) {
			t.Fatalf("%s must not exist", path)
		}
	}
	if n := len(m.requests()); n != 3 {
		t.Fatalf("requests = %d, want three downloads", n)
	}
}

func TestRunTriesAnotherMirrorAfterWrongFile(t *testing.T) {
	f := newFixture(t)
	wrong := newMirror(t, testWrong)
	right := newMirror(t, testContent)
	err := f.run(t, []string{wrong.url(), right.url()})
	f.assertFetched(t, err)
	if ranges := right.requests(); len(ranges) != 1 || ranges[0] != "" {
		t.Fatalf("right ranges = %q, want all of the file again", ranges)
	}
	if !strings.Contains(f.output.String(), "the hash is not "+testSHA256) {
		t.Fatalf("output = %q", f.output.String())
	}
}

func TestRunKeepsGoodFinal(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, testName), testContent, 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if n := len(m.requests()); n != 0 {
		t.Fatalf("requests = %d", n)
	}
	if f.output.String() != "" {
		t.Fatalf("output = %q", f.output.String())
	}
}

func TestRunReplacesBadFinal(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, testName), testWrong, 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if n := len(m.requests()); n != 1 {
		t.Fatalf("requests = %d", n)
	}
	if !strings.Contains(f.output.String(), "is not the one that apt expects; removed") {
		t.Fatalf("output = %q", f.output.String())
	}
}

func TestRunReplacesShortFinal(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	if err := os.WriteFile(filepath.Join(f.directory, testName), testContent[:1000], 0o644); err != nil {
		t.Fatal(err)
	}
	f.assertFetched(t, f.run(t, []string{m.url()}))
	if n := len(m.requests()); n != 1 {
		t.Fatalf("requests = %d", n)
	}
}

func TestRunHashTypes(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	sum := md5.Sum(testContent)
	hash := "MD5Sum:" + hex.EncodeToString(sum[:])
	f.assertFetched(t, f.run(t, []string{m.url()}, withHash(hash)))
}

func TestRunUnknownHashTypeFails(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent)
	err := f.run(t, []string{m.url()}, withHash("CRC32:abcdef"))
	if err == nil {
		t.Fatal("want an error")
	}
	if _, statErr := os.Stat(filepath.Join(f.directory, testName)); !os.IsNotExist(statErr) {
		t.Fatal("no archive may be moved")
	}
}

func TestFetchOneServerIgnoresRange(t *testing.T) {
	f := newFixture(t)
	m := newMirror(t, testContent, withIgnoreRange())
	partial := filepath.Join(f.directory, "partial", testName)
	if err := os.WriteFile(partial, testContent[:100_000], 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Unix() + 60
	speed, message, rangeIgnored := fetchOne(&http.Client{}, m.url(), partial, 100_000, testSHA256, 150000, 1, deadline)
	if !rangeIgnored {
		t.Fatalf("rangeIgnored = false, message = %q", message)
	}
	if !strings.Contains(message, "curl: (33)") {
		t.Fatalf("message = %q, want the curl (33) report", message)
	}
	if speed != 0 {
		t.Fatalf("speed = %v, want 0 for a discarded download", speed)
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("the appended partial must be discarded, not kept for a checksum failure")
	}
	// Only the resumed GET happened; the restart is the caller's next round.
	if ranges := m.requests(); !equalStrings(ranges, []string{"bytes=100000-"}) {
		t.Fatalf("ranges = %q", ranges)
	}
}

func TestGetterChecksumParam(t *testing.T) {
	cases := []struct {
		hash string
		want string
		ok   bool
	}{
		{hash: "SHA256:abcdef", want: "sha256:abcdef", ok: true},
		{hash: "MD5Sum:abcdef", want: "md5:abcdef", ok: true},
		{hash: "SHA1:abcdef", want: "sha1:abcdef", ok: true},
		{hash: "SHA512:abcdef", want: "sha512:abcdef", ok: true},
		{hash: "CRC32:abcdef", ok: false},
		{hash: "SHA256", ok: false},
		{hash: "SHA256:", ok: false},
		{hash: "SHA256:a:b", ok: false},
		{hash: "SHA256:xyz", ok: false},
	}
	for _, tc := range cases {
		if got, ok := getterChecksumParam(tc.hash); got != tc.want || ok != tc.ok {
			t.Errorf("getterChecksumParam(%q) = (%q, %v), want (%q, %v)", tc.hash, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run([]string{"/tmp"}); err == nil || !strings.Contains(err.Error(), "Usage: fetch") {
		t.Fatalf("err = %v", err)
	}
	t.Setenv("APT_DEADLINE", strconv.FormatInt(time.Now().Unix()+60, 10))
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "Usage: fetch") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckHash(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, testContent, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(testContent)
	if !checkHash(file, "MD5Sum:"+hex.EncodeToString(sum[:])) {
		t.Error("MD5Sum must match")
	}
	if !checkHash(file, testSHA256) {
		t.Error("SHA256 must match")
	}
	if checkHash(file, "SHA256:abcdef") {
		t.Error("wrong digest must not match")
	}
	if checkHash(file, "CRC32:abcdef") {
		t.Error("unknown type must not match")
	}
	if checkHash(file, "SHA256") {
		t.Error("missing digest must not match")
	}
	if checkHash(filepath.Join(t.TempDir(), "missing"), testSHA256) {
		t.Error("missing file must not match")
	}
}

func TestHostOf(t *testing.T) {
	if got := hostOf("http://mirror.example:8080/ubuntu/file.deb"); got != "mirror.example:8080" {
		t.Fatalf("got %q", got)
	}
	if got := hostOf("http://127.0.0.1:1234/x"); got != "127.0.0.1:1234" {
		t.Fatalf("got %q", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

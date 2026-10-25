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

// Package fetch makes one archive of a package available for installpkgs,
// which starts several of these at the same time:
//
//	fetch <archive directory> <size> <file name> <hash> <address>...
//
// It is a port of .github/actions/install-linux-packages/fetch.sh to Go. The
// hash is the one that apt prints, TYPE:digest (SHA256:... with
// -o Acquire::ForceHash=SHA256). The addresses are those of the file on the
// mirrors, in the order to try them. The download goes to partial/ first,
// where apt keeps what it is downloading, and continues from what is there.
//
// The single-URL transfer, resume and hash check included, is delegated to
// go-getter's HttpGetter.GetFile through a Client; the mirror loop, the
// deadline handling, the 3-round give-up, the partial-file protocol and the
// speed-limit dropping of slow mirrors stay here.
//
// A connection that gets less than APT_MIN_SPEED bytes a second (262144 by
// default) for APT_SPEED_TIME seconds (10) is dropped, and the download
// continues from the next mirror. When every mirror has been slow in turn, the
// best of them continues, and only a stalled connection is dropped.
// APT_DEADLINE, in seconds since the epoch, is the time when the download ends.
package fetch

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	getter "github.com/hashicorp/go-getter/v2"
)

// stdout receives the progress messages. It is a variable so that tests can
// capture them.
var stdout io.Writer = os.Stdout

// Run fetches one archive. args are <archive directory> <size> <file name>
// <hash> <address>..., as described above.
func Run(args []string) error {
	if len(args) < 5 {
		return errors.New("Usage: fetch <archive directory> <size> <file name> <hash> <address...>")
	}
	directory, sizeText, name, hashText := args[0], args[1], args[2], args[3]
	original := append([]string(nil), args[4:]...)
	size, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil || size < 0 {
		return fmt.Errorf("fetch: invalid size %q", sizeText)
	}
	deadlineText := os.Getenv("APT_DEADLINE")
	if deadlineText == "" {
		return errors.New("fetch: APT_DEADLINE is the time when the download ends, in seconds since the epoch")
	}
	deadline, err := strconv.ParseInt(deadlineText, 10, 64)
	if err != nil {
		return fmt.Errorf("fetch: invalid APT_DEADLINE %q", deadlineText)
	}
	minimum, err := envWithDefault("APT_MIN_SPEED", 262144)
	if err != nil {
		return fmt.Errorf("fetch: invalid APT_MIN_SPEED: %v", err)
	}
	window, err := envWithDefault("APT_SPEED_TIME", 10)
	if err != nil {
		return fmt.Errorf("fetch: invalid APT_SPEED_TIME: %v", err)
	}

	final := filepath.Join(directory, name)
	partial := filepath.Join(directory, "partial", name)

	if info, err := os.Stat(final); err == nil {
		if info.Size() == size && checkHash(final, hashText) {
			return nil
		}
		fmt.Fprintf(stdout, "%s: the archive in the directory is not the one that apt expects; removed\n", name)
		os.Remove(final)
	}

	// The mirrors keep what the runners list; without a proxy between them.
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}}

	addresses := append([]string(nil), original...)
	var fruitless, wrong int
	var last string
	for {
		if have(partial) == size {
			if checkHash(partial, hashText) {
				break
			}
			fmt.Fprintf(stdout, "%s: the hash is not %s; starting again\n", name, hashText)
			os.Remove(partial)
			wrong++
			if wrong >= 3 {
				return fmt.Errorf("%s: no mirror sends the right file", name)
			}
			// The mirror that completed it is the last one to try again.
			addresses = withoutPlusLast(original, last)
		}
		// More than the file has is not a part of it.
		if have(partial) > size {
			os.Remove(partial)
		}
		start := have(partial)
		best := ""
		var bestGain int64
		for _, address := range addresses {
			before := have(partial)
			if before >= size {
				break
			}
			if deadline-time.Now().Unix() <= 0 {
				return errors.New("fetch: deadline exceeded")
			}
			last = address
			speed, message, rangeIgnored := fetchOne(client, address, partial, before, hashText, minimum, window, deadline)
			// A server that does not continue a download sends all of the
			// file, which is not taken (curl would say 33): start again.
			if rangeIgnored {
				os.Remove(partial)
			}
			if gain := have(partial) - before; gain > bestGain {
				best, bestGain = address, gain
			}
			if have(partial) != size {
				fmt.Fprintf(stdout, "%s: %d of %d bytes, %d bytes a second from %s: %s\n",
					name, have(partial), size, int64(speed), hostOf(address), message)
			}
		}
		// All of it came, to be checked at the top.
		if have(partial) == size {
			continue
		}
		if have(partial) > start {
			fruitless = 0
		} else {
			fruitless++
			if fruitless >= 3 {
				return fmt.Errorf("%s: no mirror sends it", name)
			}
		}
		// Every mirror was slow in turn, or failed: go on with the best of them.
		minimum = 1000
		if best != "" {
			addresses = append([]string{best}, without(original, best)...)
		}
		time.Sleep(time.Second)
	}
	return os.Rename(partial, final)
}

// withoutPlusLast returns the original addresses without last, with last at
// the end (unless it is empty).
func withoutPlusLast(original []string, last string) []string {
	rest := without(original, last)
	if last != "" {
		rest = append(rest, last)
	}
	return rest
}

// without returns the addresses of original except the one to move.
func without(original []string, move string) []string {
	rest := make([]string, 0, len(original))
	for _, address := range original {
		if address != move {
			rest = append(rest, address)
		}
	}
	return rest
}

// envWithDefault reads an integer variable that is empty when unset, like the
// shell's ${APT_MIN_SPEED:-262144}.
func envWithDefault(name string, def int64) (int64, error) {
	text := os.Getenv(name)
	if text == "" {
		return def, nil
	}
	return strconv.ParseInt(text, 10, 64)
}

// have is the size of what has been downloaded.
func have(partial string) int64 {
	info, err := os.Stat(partial)
	if err != nil {
		return 0
	}
	return info.Size()
}

// getterChecksumParam maps our TYPE:digest hash to the type:value form that
// go-getter accepts, or false when the type is not one go-getter verifies.
// apt prints MD5Sum where go-getter wants md5; the SHA names only differ in
// case.
func getterChecksumParam(hashText string) (string, bool) {
	parts := strings.SplitN(hashText, ":", 2)
	if len(parts) != 2 || parts[1] == "" || strings.Contains(parts[1], ":") {
		return "", false
	}
	var checksumType string
	switch strings.ToLower(parts[0]) {
	case "md5", "md5sum":
		checksumType = "md5"
	case "sha1":
		checksumType = "sha1"
	case "sha256":
		checksumType = "sha256"
	case "sha512":
		checksumType = "sha512"
	default:
		return "", false
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", false
	}
	return checksumType + ":" + parts[1], true
}

// checkHash reports whether the file has the hash that apt printed,
// TYPE:digest, where TYPE names the tool the shell used. The digest itself
// is verified with go-getter's FileChecksum, which accepts the same
// type:value form.
func checkHash(path, hashText string) bool {
	param, ok := getterChecksumParam(hashText)
	if !ok {
		return false
	}
	parts := strings.SplitN(param, ":", 2)
	var h hash.Hash
	switch parts[0] {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return false
	}
	value, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	checksum := &getter.FileChecksum{Type: parts[0], Hash: h, Value: value}
	return checksum.Checksum(path) == nil
}

// hostOf strips the scheme and the path, for the progress message.
func hostOf(address string) string {
	host := address
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	return host
}

// statusRecorder remembers the Range header and the status of the last GET
// that went through it, so that fetchOne can tell a resumed download the
// server ignored from an ordinary one. go-getter's HttpGetter.GetFile does
// not report the status itself: a 200 to a resumed download looks like a
// success, while the whole file was appended to the partial.
type statusRecorder struct {
	mu     sync.Mutex
	sent   bool
	header string
	status int
}

// recordingTransport passes requests to base, noting the GETs for the
// recorder.
type recordingTransport struct {
	base http.RoundTripper
	rec  *statusRecorder
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err == nil && req.Method == http.MethodGet {
		t.rec.mu.Lock()
		t.rec.sent = true
		t.rec.header = req.Header.Get("Range")
		t.rec.status = resp.StatusCode
		t.rec.mu.Unlock()
	}
	return resp, err
}

func (r *statusRecorder) lastGet() (sent bool, header string, status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sent, r.header, r.status
}

// errSlowTransfer aborts a download that stays below the configured speed.
var errSlowTransfer = errors.New("operation too slow")

// speedTracker is a go-getter ProgressTracker that drops a connection that
// gets less than minimum bytes a second for window seconds, like curl's
// --speed-limit and --speed-time. It cancels the download when the average
// speed over the window is below the minimum.
type speedTracker struct {
	minimum, window int64
	cancel          context.CancelFunc

	mu    sync.Mutex
	slow  bool
	total int64
}

func (t *speedTracker) TrackProgress(src string, current, totalSize int64, stream io.ReadCloser) io.ReadCloser {
	if t.minimum <= 0 || t.window <= 0 {
		return stream
	}
	watch := &speedWatch{tracker: t, src: stream, started: time.Now()}
	watch.start()
	return watch
}

func (t *speedTracker) isSlow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.slow
}

func (t *speedTracker) markSlow() {
	t.mu.Lock()
	t.slow = true
	t.mu.Unlock()
}

func (t *speedTracker) add(n int64) {
	t.mu.Lock()
	t.total += n
	t.mu.Unlock()
}

func (t *speedTracker) bytes() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total
}

type sample struct {
	at time.Time
	n  int64
}

// speedWatch counts the downloaded bytes for the ticker started below.
type speedWatch struct {
	tracker *speedTracker
	src     io.ReadCloser
	started time.Time

	mu      sync.Mutex
	samples []sample
	done    chan struct{}
	once    sync.Once
}

func (w *speedWatch) start() {
	w.done = make(chan struct{})
	w.samples = []sample{{at: w.started}}
	window := time.Duration(w.tracker.window) * time.Second
	ticker := time.NewTicker(200 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case now := <-ticker.C:
				if now.Sub(w.started) < window {
					continue
				}
				cutoff := now.Add(-window)
				w.mu.Lock()
				base := w.samples[0].n
				for _, s := range w.samples {
					if !s.at.Before(cutoff) {
						base = s.n
						break
					}
				}
				total := w.samples[len(w.samples)-1].n
				w.mu.Unlock()
				if total-base < w.tracker.minimum*w.tracker.window {
					w.tracker.markSlow()
					w.tracker.cancel()
					return
				}
			}
		}
	}()
}

func (w *speedWatch) Read(p []byte) (int, error) {
	if w.tracker.isSlow() {
		return 0, errSlowTransfer
	}
	n, err := w.src.Read(p)
	if n > 0 {
		w.tracker.add(int64(n))
		total := w.tracker.bytes()
		w.mu.Lock()
		w.samples = append(w.samples, sample{at: time.Now(), n: total})
		w.mu.Unlock()
	}
	return n, err
}

func (w *speedWatch) Close() error {
	w.once.Do(func() { close(w.done) })
	return w.src.Close()
}

// fetchOne downloads from one address, continuing from have bytes, honouring
// the deadline. The single-URL transfer, resume and all, is delegated to
// go-getter's HttpGetter.GetFile through a Client, with our TYPE:digest hash
// passed through as the checksum go-getter accepts. It returns the average
// speed in bytes a second, a one-line message in the style of curl for the
// progress report, and whether the server ignored the Range request by sending
// all of the file.
func fetchOne(client *http.Client, address, partial string, have int64, hashText string, minimum, window, deadline int64) (speed float64, message string, rangeIgnored bool) {
	start := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(deadline, 0))
	defer cancel()
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	recorder := &statusRecorder{}
	limited := *client
	limited.Transport = &recordingTransport{base: base, rec: recorder}
	tracker := &speedTracker{minimum: minimum, window: window, cancel: cancel}
	downloader := &getter.HttpGetter{Client: &limited, XTerraformGetDisabled: true}
	transfer := &getter.Client{
		Getters:       []getter.Getter{downloader},
		Decompressors: map[string]getter.Decompressor{},
	}
	src := address
	if param, ok := getterChecksumParam(hashText); ok {
		sep := "?"
		if strings.Contains(address, "?") {
			sep = "&"
		}
		src += sep + "checksum=" + param
	}
	_, err := transfer.Get(ctx, &getter.Request{
		Src:              src,
		Dst:              partial,
		GetMode:          getter.ModeFile,
		ProgressListener: tracker,
	})
	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		speed = float64(tracker.bytes()) / elapsed
	}
	// A server that does not continue a download sends all of the file, which
	// go-getter appends to the partial (curl would say 33): discard it and
	// report it, so that the download starts again from the next mirror.
	if sent, header, status := recorder.lastGet(); sent && have > 0 && header != "" && status == http.StatusOK {
		os.Remove(partial)
		return 0, "curl: (33) HTTP server doesn't seem to support byte ranges", true
	}
	if tracker.isSlow() {
		return speed, fmt.Sprintf("curl: (28) Operation too slow. Less than %d bytes/sec transferred the last %d seconds", minimum, window), false
	}
	if err != nil {
		var mismatch *getter.ChecksumError
		if errors.As(err, &mismatch) {
			// go-getter checked the hash and it does not match. Leave the
			// file for the gate at the top of the loop, which reports it
			// ("the hash is not ...") and counts the round, exactly as when
			// the hand-rolled download failed its own check.
			return speed, "", false
		}
		if sent, _, status := recorder.lastGet(); sent && status != http.StatusOK && status != http.StatusPartialContent {
			return speed, fmt.Sprintf("curl: (22) The requested URL returned error: %d", status), false
		} else if !sent {
			return speed, "curl: (7) " + oneLine(failReason(ctx, err)), false
		}
		return speed, "curl: (18) " + oneLine(failReason(ctx, err)), false
	}
	return speed, "", false
}

// failReason prefers the deadline over the error of a cancelled request.
func failReason(ctx context.Context, err error) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "operation timed out"
	}
	return err.Error()
}

// oneLine keeps the message on the one line of the progress report.
func oneLine(message string) string {
	return strings.ReplaceAll(strings.TrimSpace(message), "\n", " ")
}

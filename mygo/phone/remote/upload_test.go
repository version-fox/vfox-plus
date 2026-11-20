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

package remote

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// upload is what the test server saw of one request to add a plugin.
type upload struct {
	method        string
	contentType   string
	contentLength int64
	// bodyLength is the number of bytes that arrived, which must be what the request declared.
	bodyLength int64
	// fields holds the values of the plain fields, such as alias.
	fields map[string]string
	// fileField, fileName and fileSize describe the part that holds the file; head is its first bytes.
	fileField string
	fileName  string
	fileSize  int64
	head      string
	// parseError is why the body is not a form, empty when it is one.
	parseError string
}

// uploads is a plain HTTP server of the loopback interface that reads the forms that
// are posted to /api/plugins as the server of the desktop does, and keeps what it saw.
type uploads struct {
	mu   sync.Mutex
	seen []upload
}

func (u *uploads) all() []upload {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]upload(nil), u.seen...)
}

// serveUploads starts the server, and returns the address of it and what it saw.
func serveUploads(t *testing.T) (*url.URL, *uploads) {
	t.Helper()
	seen := &uploads{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := upload{method: r.Method, contentType: r.Header.Get("Content-Type"), contentLength: r.ContentLength, fields: map[string]string{}}
		counter := &countingReader{reader: r.Body}
		mediaType, params, err := mime.ParseMediaType(got.contentType)
		if err != nil || mediaType != "multipart/form-data" {
			got.parseError = "the body is not multipart/form-data"
		} else {
			form := multipart.NewReader(counter, params["boundary"])
			for {
				part, err := form.NextPart()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					got.parseError = err.Error()
					break
				}
				if part.FileName() == "" {
					value, _ := io.ReadAll(part)
					got.fields[part.FormName()] = string(value)
					continue
				}
				got.fileField, got.fileName = part.FormName(), part.FileName()
				head := make([]byte, 64)
				n, _ := io.ReadFull(part, head)
				got.head = string(head[:n])
				rest, _ := io.Copy(io.Discard, part)
				got.fileSize = int64(n) + rest
			}
		}
		_, _ = io.Copy(io.Discard, r.Body)
		got.bodyLength = counter.n
		seen.mu.Lock()
		seen.seen = append(seen.seen, got)
		seen.mu.Unlock()
		if got.parseError != "" {
			http.Error(w, "Cannot read the upload.", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":7,"operation":"addPlugin","status":"queued","commandLine":"vfox add","log":[],"droppedLines":0}`)
	}))
	t.Cleanup(server.Close)
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return base, seen
}

// countingReader counts the bytes that were read.
type countingReader struct {
	reader io.Reader
	n      int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.n += int64(n)
	return n, err
}

// zeros is a reader of any number of zero bytes, so that a test sends a file of a size
// that it could not keep in memory cheaply.
type zeros struct{ left int64 }

func (z *zeros) Read(p []byte) (int, error) {
	if z.left <= 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > z.left {
		n = z.left
	}
	clear(p[:n])
	z.left -= n
	return int(n), nil
}

func TestUploadPluginSendsTheFormThatTheWebInterfaceSends(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "session")
	if err != nil {
		t.Fatal(err)
	}
	content := "-- the plugin\nPLUGIN = {name = 'demo'}\n"
	task, err := client.UploadPlugin(t.Context(), "demo.lua", strings.NewReader(content), int64(len(content)), "demo")
	if err != nil {
		t.Fatalf("UploadPlugin: %v", err)
	}
	if task.ID != 7 || task.Operation != OperationAddPlugin || task.Status != TaskQueued {
		t.Errorf("the task is %+v, want the one that the server answered", task)
	}
	got := seen.all()
	if len(got) != 1 {
		t.Fatalf("the server saw %d requests, want 1", len(got))
	}
	u := got[0]
	if u.method != http.MethodPost || u.parseError != "" {
		t.Fatalf("the request: method %q, body error %q", u.method, u.parseError)
	}
	// The form is what a browser sends: the file in the field file, and the alias in the field alias.
	if u.fileField != "file" || u.fileName != "demo.lua" || u.fileSize != int64(len(content)) || u.head != content {
		t.Errorf("the file: field %q, name %q, %d bytes, starting %q; want file, demo.lua and the content", u.fileField, u.fileName, u.fileSize, u.head)
	}
	if u.fields["alias"] != "demo" {
		t.Errorf("the alias is %q, want demo", u.fields["alias"])
	}
	// The length is declared, as a browser declares it, and it is the length that arrived: a server
	// behind a proxy that does not take a chunked body takes this one.
	if u.contentLength <= 0 || u.contentLength != u.bodyLength {
		t.Errorf("the request declares %d bytes and sent %d", u.contentLength, u.bodyLength)
	}
}

func TestUploadPluginWithoutAnAliasSendsNone(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.UploadPlugin(t.Context(), "demo.zip", strings.NewReader("PK"), 2, ""); err != nil {
		t.Fatalf("UploadPlugin: %v", err)
	}
	u := seen.all()[0]
	if _, ok := u.fields["alias"]; ok || u.fileName != "demo.zip" || u.head != "PK" {
		t.Errorf("the form has fields %v and the file %q %q; want the file only", u.fields, u.fileName, u.head)
	}
}

// TestUploadPluginSendsTheNameOfTheFileAlone checks that the server hears of the name of the
// file and not of where the phone keeps it: the pick of the system has a path in the cache of
// the app, and a file of a computer has its own folders.
func TestUploadPluginSendsTheNameOfTheFileAlone(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"/data/user/0/dev.vfox.vfoxplus/cache/picked/123/node.ZIP": "node.ZIP",
		`C:\Users\me\Downloads\golang.lua`:                         "golang.lua",
		"plain.lua":                                                "plain.lua",
	} {
		if _, err := client.UploadPlugin(t.Context(), name, strings.NewReader("x"), 1, ""); err != nil {
			t.Fatalf("UploadPlugin(%q): %v", name, err)
		}
		got := seen.all()
		if last := got[len(got)-1]; last.fileName != want {
			t.Errorf("UploadPlugin(%q) sent the name %q, want %q", name, last.fileName, want)
		}
	}
}

func TestUploadPluginRefusesWhatIsNotAPluginFileBeforeSendingAnything(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "plugin", "notes.txt", "plugin.zip.exe", "archive.tar.gz", "x\r\nX-Injected: yes.zip", "a\x00b.lua"} {
		_, err := client.UploadPlugin(t.Context(), name, strings.NewReader("x"), 1, "")
		if !errors.Is(err, ErrNotPluginFile) {
			t.Errorf("UploadPlugin(%q) = %v, want ErrNotPluginFile", name, err)
		}
	}
	if got := seen.all(); len(got) != 0 {
		t.Errorf("the server saw %d requests for files that are not plugins", len(got))
	}
}

func TestUploadPluginRefusesAFileThatTheServerWouldNotTake(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int64{MaxPluginFile + 1, 1 << 40, -1} {
		_, err := client.UploadPlugin(t.Context(), "big.zip", &zeros{left: 1}, size, "")
		if !errors.Is(err, ErrPluginFileTooLarge) {
			t.Errorf("a file of %d bytes: %v, want ErrPluginFileTooLarge", size, err)
		}
	}
	if got := seen.all(); len(got) != 0 {
		t.Errorf("the server saw %d requests for files that it would not take", len(got))
	}
}

// TestUploadPluginSendsTheLargestFile checks that a file of the greatest size goes through,
// and that it is sent as it is read: the reader makes it up as the transport asks for it.
func TestUploadPluginSendsTheLargestFile(t *testing.T) {
	base, seen := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.UploadPlugin(t.Context(), "largest.zip", &zeros{left: MaxPluginFile}, MaxPluginFile, "big"); err != nil {
		t.Fatalf("UploadPlugin of the largest file: %v", err)
	}
	u := seen.all()[0]
	if u.fileSize != MaxPluginFile || u.contentLength != u.bodyLength {
		t.Errorf("the server got a file of %d bytes in a body of %d, declared %d; want %d", u.fileSize, u.bodyLength, u.contentLength, int64(MaxPluginFile))
	}
}

// TestUploadPluginFailsWhenTheFileIsNotTheSizeThatWasSaid checks that a file that changed
// under the upload is an error, and not a plugin that was cut short or padded: the length
// that the request declares is the length of the form.
func TestUploadPluginFailsWhenTheFileIsNotTheSizeThatWasSaid(t *testing.T) {
	base, _ := serveUploads(t)
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for name, size := range map[string]int64{"the file shrank": 100, "the file grew": 5} {
		content := bytes.Repeat([]byte("x"), 12)
		if name == "the file shrank" {
			content = content[:6]
		}
		if _, err := client.UploadPlugin(t.Context(), "demo.lua", bytes.NewReader(content), size, ""); err == nil {
			t.Errorf("%s: the upload succeeded", name)
		}
	}
}

func TestUploadPluginAnswersLikeEveryCall(t *testing.T) {
	refuse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Error(w, "An uploaded file or a source URL is required.", http.StatusBadRequest)
	}))
	t.Cleanup(refuse.Close)
	base, err := url.Parse(refuse.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(base, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.UploadPlugin(t.Context(), "demo.lua", strings.NewReader("x"), 1, "")
	var api *APIError
	if !errors.As(err, &api) || api.Status != http.StatusBadRequest || !strings.Contains(api.Message, "required") {
		t.Errorf("a refused upload: %v, want the 400 answer with its text", err)
	}
}

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

package memoryreport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v93/github"
)

// repositoryAt returns the repository as the server at address serves it.
func repositoryAt(t *testing.T, address string) *Repository {
	t.Helper()
	repo, err := NewRepository(testRepository, "test-token", github.WithURLs(&address, nil))
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestNewRepository(t *testing.T) {
	t.Run("names the repository", func(t *testing.T) {
		repo, err := NewRepository(testRepository, "test-token")
		if err != nil {
			t.Fatal(err)
		}
		if repo.Owner != "version-fox" || repo.Name != "vfox-plus" || repo.FullName() != testRepository {
			t.Fatalf("repo = %+v", repo)
		}
	})
	t.Run("needs a token", func(t *testing.T) {
		if _, err := NewRepository(testRepository, ""); err == nil || !strings.Contains(err.Error(), "GH_TOKEN") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("needs OWNER/NAME", func(t *testing.T) {
		for _, name := range []string{"", "not a repository", "owner", "owner/", "/name"} {
			if _, err := NewRepository(name, "test-token"); err == nil {
				t.Errorf("NewRepository(%q) = nil", name)
			}
		}
	})
}

// roundTripFunc stubs the network of a transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// reply is what the network answers with: a status or an error.
type reply struct {
	status int
	err    error
}

// outcome is what a request met on its way through a retryTransport.
type outcome struct {
	response *http.Response
	err      error
	// bodies is what the network received, by attempt.
	bodies []string
	// pauses are the waits between the attempts.
	pauses []time.Duration
}

// send sends the request through a retryTransport whose network answers with
// the replies, one for each attempt (the last one repeats).
func send(t *testing.T, request *http.Request, replies ...reply) outcome {
	t.Helper()
	var result outcome
	old := retryWait
	retryWait = func(attempt int) time.Duration {
		result.pauses = append(result.pauses, backoff(attempt))
		return 0
	}
	t.Cleanup(func() { retryWait = old })
	network := roundTripFunc(func(sent *http.Request) (*http.Response, error) {
		var body []byte
		if sent.Body != nil {
			var err error
			if body, err = io.ReadAll(sent.Body); err != nil {
				t.Error(err)
			}
			_ = sent.Body.Close()
		}
		result.bodies = append(result.bodies, string(body))
		next := replies[min(len(result.bodies), len(replies))-1]
		if next.err != nil {
			return nil, next.err
		}
		return &http.Response{StatusCode: next.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: sent}, nil
	})
	result.response, result.err = retryTransport{base: network}.RoundTrip(request)
	return result
}

func newRequest(t *testing.T, method string, body io.Reader) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, "https://api.example/repos/o/r", body)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestRetryTransport(t *testing.T) {
	t.Run("sends again after a failure of the server", func(t *testing.T) {
		got := send(t, newRequest(t, http.MethodGet, nil), reply{status: 502}, reply{status: 200})
		if got.err != nil || got.response.StatusCode != 200 {
			t.Fatalf("got %+v", got)
		}
		if len(got.bodies) != 2 || !slices.Equal(got.pauses, []time.Duration{2 * time.Second}) {
			t.Fatalf("attempts = %d, pauses = %v", len(got.bodies), got.pauses)
		}
	})
	t.Run("sends again after a broken connection", func(t *testing.T) {
		for _, message := range []string{"read tcp: connection reset by peer", "unexpected EOF", "dial tcp: i/o timeout", "net/http: TLS handshake timeout"} {
			got := send(t, newRequest(t, http.MethodGet, nil), reply{err: errors.New(message)}, reply{status: 200})
			if got.err != nil || got.response.StatusCode != 200 || len(got.bodies) != 2 {
				t.Errorf("%s: got %+v", message, got)
			}
		}
	})
	t.Run("gives up after three attempts, waiting 2 seconds and then 4", func(t *testing.T) {
		got := send(t, newRequest(t, http.MethodGet, nil), reply{status: 503})
		// The last answer is passed on, for the client to turn into an error.
		if got.err != nil || got.response.StatusCode != 503 {
			t.Fatalf("got %+v", got)
		}
		if len(got.bodies) != 3 || !slices.Equal(got.pauses, []time.Duration{2 * time.Second, 4 * time.Second}) {
			t.Fatalf("attempts = %d, pauses = %v", len(got.bodies), got.pauses)
		}
		got = send(t, newRequest(t, http.MethodGet, nil), reply{err: errors.New("connection reset by peer")})
		if got.err == nil || len(got.bodies) != 3 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("passes on a refusal and a rate limit", func(t *testing.T) {
		for _, status := range []int{400, 403, 404, 429} {
			got := send(t, newRequest(t, http.MethodGet, nil), reply{status: status})
			if got.err != nil || got.response.StatusCode != status || len(got.bodies) != 1 || len(got.pauses) != 0 {
				t.Errorf("%d: got %+v", status, got)
			}
		}
	})
	t.Run("passes on an error that is no trouble of the connection", func(t *testing.T) {
		got := send(t, newRequest(t, http.MethodGet, nil), reply{err: errors.New("x509: certificate signed by unknown authority")})
		if got.err == nil || len(got.bodies) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("sends a POST once", func(t *testing.T) {
		// After a lost response, a second POST would write the comment twice.
		got := send(t, newRequest(t, http.MethodPost, strings.NewReader(`{"body":"x"}`)), reply{status: 502}, reply{status: 201})
		if got.err != nil || got.response.StatusCode != 502 || len(got.bodies) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("sends the body again", func(t *testing.T) {
		got := send(t, newRequest(t, http.MethodPatch, strings.NewReader(`{"body":"x"}`)), reply{status: 500}, reply{status: 200})
		if got.err != nil || got.response.StatusCode != 200 {
			t.Fatalf("got %+v", got)
		}
		if !slices.Equal(got.bodies, []string{`{"body":"x"}`, `{"body":"x"}`}) {
			t.Fatalf("bodies = %q", got.bodies)
		}
	})
	t.Run("sends once a body that is gone after the first attempt", func(t *testing.T) {
		request := newRequest(t, http.MethodPatch, io.NopCloser(strings.NewReader(`{"body":"x"}`)))
		got := send(t, request, reply{status: 500}, reply{status: 200})
		if got.err != nil || got.response.StatusCode != 500 || len(got.bodies) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("does not send again for a request that is over", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		request := newRequest(t, http.MethodGet, nil).WithContext(ctx)
		cancel()
		got := send(t, request, reply{status: 502}, reply{status: 200})
		if got.response.StatusCode != 502 || len(got.bodies) != 1 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("stops waiting when the request is over", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		old := retryWait
		retryWait = func(int) time.Duration {
			cancel()
			return time.Hour
		}
		defer func() { retryWait = old }()
		var attempts int
		transport := retryTransport{base: roundTripFunc(func(sent *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{StatusCode: 502, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		// The context is alive when the first answer comes, and over during the wait.
		request := newRequest(t, http.MethodGet, nil).WithContext(ctx)
		_, err := transport.RoundTrip(request)
		if !errors.Is(err, context.Canceled) || attempts != 1 {
			t.Fatalf("err = %v, attempts = %d", err, attempts)
		}
	})
}

func TestBackoff(t *testing.T) {
	if backoff(1) != 2*time.Second || backoff(2) != 4*time.Second {
		t.Fatalf("backoff = %v, %v", backoff(1), backoff(2))
	}
}

// The client of the repository sends again through the transport, and gives
// the message that the server sent with a refusal.
func TestRepositoryClient(t *testing.T) {
	serve := func(t *testing.T, handler func(requests int32, w http.ResponseWriter, r *http.Request)) (*Repository, *atomic.Int32) {
		t.Helper()
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(requests.Add(1), w, r)
		}))
		t.Cleanup(server.Close)
		return repositoryAt(t, server.URL), &requests
	}
	t.Run("sends again after a failure of the server", func(t *testing.T) {
		repo, requests := serve(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
			if n < 3 {
				writeJSON(w, http.StatusBadGateway, map[string]any{"message": "Bad Gateway"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"default_branch": "main"})
		})
		info, _, err := repo.Client.Repositories.Get(context.Background(), repo.Owner, repo.Name)
		if err != nil || info.GetDefaultBranch() != "main" || requests.Load() != 3 {
			t.Fatalf("info = %v, err = %v, requests = %d", info, err, requests.Load())
		}
	})
	t.Run("says what the server said when it keeps failing", func(t *testing.T) {
		repo, requests := serve(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"message": "Service Unavailable"})
		})
		_, _, err := repo.Client.Repositories.Get(context.Background(), repo.Owner, repo.Name)
		if err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") || requests.Load() != 3 {
			t.Fatalf("err = %v, requests = %d", err, requests.Load())
		}
	})
	t.Run("writes a comment once", func(t *testing.T) {
		repo, requests := serve(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusBadGateway, map[string]any{"message": "Bad Gateway"})
		})
		_, _, err := repo.Client.Issues.CreateComment(context.Background(), repo.Owner, repo.Name, 7, github.IssueCommentRequest{Body: "x"})
		if err == nil || requests.Load() != 1 {
			t.Fatalf("err = %v, requests = %d", err, requests.Load())
		}
	})
	t.Run("sends the body of an update again", func(t *testing.T) {
		var bodies []string
		repo, _ := serve(t, func(n int32, w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			if n == 1 {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Server Error"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id": 31, "body": "updated"})
		})
		comment, _, err := repo.Client.Issues.UpdateComment(context.Background(), repo.Owner, repo.Name, 31, github.IssueCommentRequest{Body: "updated"})
		if err != nil || comment.GetBody() != "updated" {
			t.Fatalf("comment = %v, err = %v", comment, err)
		}
		if len(bodies) != 2 || bodies[0] != bodies[1] || !strings.Contains(bodies[0], `"body":"updated"`) {
			t.Fatalf("bodies = %q", bodies)
		}
	})
	t.Run("does not wait out a rate limit", func(t *testing.T) {
		repo, requests := serve(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "4102444800")
			writeJSON(w, http.StatusForbidden, map[string]any{"message": "API rate limit exceeded for 127.0.0.1."})
		})
		_, _, err := repo.Client.Repositories.Get(context.Background(), repo.Owner, repo.Name)
		var limit *github.RateLimitError
		if !errors.As(err, &limit) || requests.Load() != 1 {
			t.Fatalf("err = %v, requests = %d", err, requests.Load())
		}
	})
	t.Run("says that the token cannot write", func(t *testing.T) {
		// Run turns this message into a warning instead of a failure.
		repo, _ := serve(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusForbidden, map[string]any{"message": "Resource not accessible by integration"})
		})
		_, err := UpsertComment(context.Background(), repo, 7, Marker+"\nnew")
		if err == nil || !strings.Contains(err.Error(), "Resource not accessible by integration") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestDownloadArtifact(t *testing.T) {
	archive := makeZip(map[string]string{"memory.txt": defaultMemoryText()})
	// both is an API that points at a file store, which is another server.
	both := func(t *testing.T, store http.HandlerFunc) (*Repository, *http.Header) {
		t.Helper()
		storage := httptest.NewServer(store)
		t.Cleanup(storage.Close)
		var seen http.Header
		api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = r.Header.Clone()
			w.Header().Set("Location", storage.URL+"/42.zip?sig=secret")
			w.WriteHeader(http.StatusFound)
		}))
		t.Cleanup(api.Close)
		return repositoryAt(t, api.URL), &seen
	}
	t.Run("follows the address of the file, without the token", func(t *testing.T) {
		var asked http.Header
		repo, seen := both(t, func(w http.ResponseWriter, r *http.Request) {
			asked = r.Header.Clone()
			_, _ = w.Write(archive)
		})
		data, err := repo.DownloadArtifact(context.Background(), 42)
		if err != nil || !bytes.Equal(data, archive) {
			t.Fatalf("data = %d bytes, err = %v", len(data), err)
		}
		if seen.Get("Authorization") != "Bearer test-token" {
			t.Errorf("the API got %q", seen.Get("Authorization"))
		}
		if asked.Get("Authorization") != "" {
			t.Errorf("the file store got %q", asked.Get("Authorization"))
		}
	})
	t.Run("refuses a file that is too large", func(t *testing.T) {
		repo, _ := both(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), MaxArtifactBytes+1))
		})
		if _, err := repo.DownloadArtifact(context.Background(), 42); err == nil || !strings.Contains(err.Error(), "more than") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("takes a file of the largest size", func(t *testing.T) {
		repo, _ := both(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), MaxArtifactBytes))
		})
		data, err := repo.DownloadArtifact(context.Background(), 42)
		if err != nil || len(data) != MaxArtifactBytes {
			t.Fatalf("data = %d bytes, err = %v", len(data), err)
		}
	})
	t.Run("sends again after a failure of the file store, and says its host and not its address", func(t *testing.T) {
		var requests atomic.Int32
		repo, _ := both(t, func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		})
		_, err := repo.DownloadArtifact(context.Background(), 42)
		if err == nil || requests.Load() != 3 {
			t.Fatalf("err = %v, requests = %d", err, requests.Load())
		}
		if !strings.Contains(err.Error(), "500 Internal Server Error") || strings.Contains(err.Error(), "42.zip") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("fails when the API does not point at a file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusGone, map[string]any{"message": "Artifact has expired"})
		}))
		defer server.Close()
		if _, err := repositoryAt(t, server.URL).DownloadArtifact(context.Background(), 42); err == nil {
			t.Fatal("want an error")
		}
	})
}

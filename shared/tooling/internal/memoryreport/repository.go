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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
)

// requestTimeout bounds a request to GitHub with its attempts and the pauses
// between them, and the two requests that download an artifact.
const requestTimeout = time.Minute

// Repository is a repository on GitHub with the clients that read it and write
// the comment on its pull requests.
type Repository struct {
	Owner string
	Name  string
	// Client is the REST API of GitHub.
	Client *github.Client
	// Downloads fetches what the API refers to at another address, such as the
	// zip of an artifact. It sends no credentials there.
	Downloads *http.Client
}

// NewRepository returns repository (OWNER/NAME) as the owner of token sees it.
// options adjust the client of the API, such as github.WithURLs to talk to
// another server.
func NewRepository(repository, token string, options ...github.ClientOptionsFunc) (*Repository, error) {
	owner, name, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("repository must be OWNER/NAME, got %q", repository)
	}
	if token == "" {
		return nil, errors.New("GH_TOKEN is empty: the memory report needs a token to read the API of GitHub")
	}
	transport := retryTransport{base: http.DefaultTransport}
	options = append([]github.ClientOptionsFunc{
		github.WithAuthToken(token),
		github.WithTransport(transport),
		github.WithTimeout(requestTimeout),
	}, options...)
	client, err := github.NewClient(options...)
	if err != nil {
		return nil, err
	}
	return &Repository{
		Owner:     owner,
		Name:      name,
		Client:    client,
		Downloads: &http.Client{Transport: transport, Timeout: requestTimeout},
	}, nil
}

// FullName returns OWNER/NAME.
func (r *Repository) FullName() string { return r.Owner + "/" + r.Name }

// DownloadArtifact returns the zip of an artifact. The API answers with the
// address of the file, which the second request fetches. It refuses a file of
// more than MaxArtifactBytes, which the metadata of the artifact (checked by
// the caller) does not promise.
func (r *Repository) DownloadArtifact(ctx context.Context, id int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	target, _, err := r.Client.Actions.DownloadArtifact(ctx, r.Owner, r.Name, id, 0)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := r.Downloads.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		// The address carries a signature, so only its host is worth saying.
		return nil, fmt.Errorf("download from %s: %s", target.Host, response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxArtifactBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxArtifactBytes {
		return nil, fmt.Errorf("download from %s: more than %d bytes", target.Host, MaxArtifactBytes)
	}
	return data, nil
}

// transient matches the errors of a connection that are worth trying again.
var transient = regexp.MustCompile(`(?i)timeout|timed out|connection reset|EOF`)

// retryAttempts is how often a request is tried.
const retryAttempts = 3

// retryWait is the pause after the attempt that failed. It is a variable so
// that tests do not wait.
var retryWait = backoff

// backoff doubles the pause after each attempt: 2 seconds, then 4.
func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * time.Second }

// retryTransport sends a request again when the server fails (HTTP 5xx) or the
// connection breaks, as the gh CLI did that this tool used to run. A POST is
// sent once: after a lost response, a second one would write the comment twice.
// Rate limits (403 and 429) are passed on, so that the run reports them instead
// of waiting them out.
type retryTransport struct {
	base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t retryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	hasBody := request.Body != nil && request.Body != http.NoBody
	attempts := retryAttempts
	if request.Method == http.MethodPost || (hasBody && request.GetBody == nil) {
		attempts = 1
	}
	for attempt := 1; ; attempt++ {
		sent := request
		if attempt > 1 && hasBody {
			body, err := request.GetBody()
			if err != nil {
				return nil, err
			}
			sent = request.Clone(request.Context())
			sent.Body = body
		}
		response, err := t.base.RoundTrip(sent)
		if attempt == attempts || request.Context().Err() != nil || !retryable(response, err) {
			return response, err
		}
		if response != nil {
			_ = response.Body.Close()
		}
		select {
		case <-request.Context().Done():
			return nil, request.Context().Err()
		case <-time.After(retryWait(attempt)):
		}
	}
}

// retryable reports whether a failed request is worth sending again.
func retryable(response *http.Response, err error) bool {
	if err != nil {
		return transient.MatchString(err.Error())
	}
	return response.StatusCode >= http.StatusInternalServerError
}

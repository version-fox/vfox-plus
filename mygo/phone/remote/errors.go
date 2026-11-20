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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// The errors that the callers test with errors.Is. An *APIError matches the
// ones that its status stands for; see APIError.Is.
var (
	// ErrUnpaired reports that the server does not accept the session of this
	// phone (HTTP 401): the phone must pair again.
	ErrUnpaired = errors.New("the server does not accept this phone; pair it again")
	// ErrNotFound reports that the server has no such item (HTTP 404).
	ErrNotFound = errors.New("the server has no such item")
	// ErrUnavailable reports that vfox failed or did not answer in time (HTTP 502
	// or 504).
	ErrUnavailable = errors.New("vfox failed or did not answer in time")
	// ErrPairingRefused reports that the server refused a pairing link: it is
	// invalid, already used or expired (HTTP 403 to the pairing link).
	ErrPairingRefused = errors.New("the pairing link is invalid, already used or expired")
	// ErrCertificateChanged reports that the server presents a certificate other
	// than the one whose fingerprint the user confirmed. The connection is
	// refused; nothing is trusted automatically.
	ErrCertificateChanged = errors.New("the server's certificate is not the one that was paired")
	// ErrNotPluginFile reports a file that is not a plugin that vfox can add: its
	// name does not end in .zip or .lua. Nothing was sent.
	ErrNotPluginFile = errors.New("a plugin file is a .zip archive or a .lua script")
	// ErrPluginFileTooLarge reports a plugin file that the server would not take:
	// it is larger than MaxPluginFile. Nothing was sent.
	ErrPluginFileTooLarge = errors.New("the plugin file is larger than the server takes")
	// ErrFileChanged reports a plugin file that did not have the size that it was said
	// to have when it was read to be sent: it changed under the upload. The form was not
	// completed, so the server added nothing.
	ErrFileChanged = errors.New("the plugin file changed while it was sent")
)

// maxMessageRunes bounds the message kept from an answer of the server, which
// may be a page of HTML when it is not the API that answered.
const maxMessageRunes = 500

// APIError is an answer of the server with a status that is not 2xx.
type APIError struct {
	// Status is the HTTP status code.
	Status int
	// Message is the text of the answer: the "error" field of a JSON answer,
	// otherwise the body as it is.
	Message string
}

// Error describes the answer.
func (e *APIError) Error() string {
	return fmt.Sprintf("the server answered %d: %s", e.Status, e.Message)
}

// Is reports whether target is one of the sentinel errors that the status of e
// stands for: 401 is ErrUnpaired, 404 is ErrNotFound and 502 or 504 is
// ErrUnavailable. A 403 is not ErrPairingRefused by itself, because the server
// also refuses other requests with 403; the pairing call maps it instead.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnpaired:
		return e.Status == http.StatusUnauthorized
	case ErrNotFound:
		return e.Status == http.StatusNotFound
	case ErrUnavailable:
		return e.Status == http.StatusBadGateway || e.Status == http.StatusGatewayTimeout
	}
	return false
}

// newAPIError makes the error of an answer with status and body.
func newAPIError(status int, body []byte) *APIError {
	message := string(body)
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		message = payload.Error
	}
	runes := []rune(message)
	if len(runes) > maxMessageRunes {
		runes = runes[:maxMessageRunes]
	}
	return &APIError{Status: status, Message: strings.TrimSpace(string(runes))}
}

// requestError describes a request that did not get an answer. The address of
// the request is left out: the URL of a pairing link carries its one-time token.
// The cause stays in the chain, so that errors.Is still finds, for example,
// ErrCertificateChanged or context.Canceled.
func requestError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return fmt.Errorf("remote: the request failed: %w", err)
}

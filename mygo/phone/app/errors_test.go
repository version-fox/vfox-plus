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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// TestAFailureOfAFileIsNotTheSilenceOfTheComputer checks that the error of a file that cannot
// be opened is not taken for a network that is down: the system error inside it (ENOENT on a
// computer that runs Linux, for example) implements net.Error, and a request that fails to read
// its body wraps the error of the file in the error of the request.
func TestAFailureOfAFileIsNotTheSilenceOfTheComputer(t *testing.T) {
	_, err := os.Open(filepath.Join(t.TempDir(), "missing.zip"))
	if err == nil {
		t.Fatal("a file that is not there was opened")
	}
	if got := KindOf(err); got != KindOther {
		t.Errorf("KindOf(the error of a file that is not there) = %v, want KindOther", got)
	}
	wrapped := &url.Error{Op: "Post", URL: "https://192.168.1.10:8369/api/plugins", Err: err}
	if got := KindOf(wrapped); got != KindOther {
		t.Errorf("KindOf(a request that failed to read its file) = %v, want KindOther", got)
	}
	// The failures of the connection that have no file in them are still the silence of the computer.
	dial := &url.Error{Op: "Post", URL: "https://192.168.1.10:8369/api/plugins", Err: &net.OpError{Op: "dial", Err: errors.New("no route")}}
	if got := KindOf(dial); got != KindUnreachable {
		t.Errorf("KindOf(a connection that failed) = %v, want KindUnreachable", got)
	}
}

func TestKindOfNamesWhatTheUserCanActOn(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want Kind
	}{
		{"nothing", nil, KindNone},
		{"cancelled", context.Canceled, KindCanceled},
		{"not a link", fmt.Errorf("paste: %w", pairing.ErrNotPairingLink), KindNotLink},
		{"not a plugin file", fmt.Errorf("upload: %w", remote.ErrNotPluginFile), KindNotPluginFile},
		{"a plugin file that is too large", fmt.Errorf("upload: %w", remote.ErrPluginFileTooLarge), KindFileTooLarge},
		{"refused link", fmt.Errorf("%w: %w", remote.ErrPairingRefused, &remote.APIError{Status: 403}), KindPairingRefused},
		{"unpaired", remote.ErrUnpaired, KindUnpaired},
		{"unpaired answer", &remote.APIError{Status: 401}, KindUnpaired},
		{"not found", &remote.APIError{Status: 404}, KindNotFound},
		{"server", &remote.APIError{Status: 500}, KindServer},
		{"unavailable", &remote.APIError{Status: 504}, KindUnavailable},
		{"rejected", &remote.APIError{Status: 400}, KindRejected},
		{"certificate", fmt.Errorf("remote: the request failed: %w",
			&net.OpError{Op: "remote", Err: remote.ErrCertificateChanged}), KindCertificateChanged},
		{"network", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, KindUnreachable},
		{"a file of the phone", fmt.Errorf("%w: %w", ErrFileUnreadable, errors.New("gone")), KindFileUnreadable},
		{"a file that changed while it was sent", &url.Error{Op: "Post", Err: remote.ErrFileChanged}, KindFileUnreadable},
		{"end of stream", fmt.Errorf("read: %w", io.ErrUnexpectedEOF), KindUnreachable},
		{"other", errors.New("something"), KindOther},
	} {
		if got := KindOf(tt.err); got != tt.want {
			t.Errorf("KindOf(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

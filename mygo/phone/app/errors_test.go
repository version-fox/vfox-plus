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
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

func TestKindOfNamesWhatTheUserCanActOn(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want Kind
	}{
		{"nothing", nil, KindNone},
		{"cancelled", context.Canceled, KindCanceled},
		{"not a link", fmt.Errorf("paste: %w", pairing.ErrNotPairingLink), KindNotLink},
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
		{"end of stream", fmt.Errorf("read: %w", io.ErrUnexpectedEOF), KindUnreachable},
		{"other", errors.New("something"), KindOther},
	} {
		if got := KindOf(tt.err); got != tt.want {
			t.Errorf("KindOf(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

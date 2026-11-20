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
	"io"
	"io/fs"
	"net"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// ErrFileUnreadable is a file of the phone that cannot be read: the copy that the picker
// of the system made is gone, or it is not a file.
var ErrFileUnreadable = errors.New("the file cannot be read")

// Kind tells the errors of the service apart for the user interface, which says
// them in its own words. The service gives the kind only; the text is the window's.
type Kind int

// The kinds of error.
const (
	// KindNone is no error.
	KindNone Kind = iota
	// KindCanceled is a call that ended because the app went to the background. It
	// is not an error to show.
	KindCanceled
	// KindNotLink is text that is not a pairing link.
	KindNotLink
	// KindNotPluginFile is a file that is not a plugin that vfox adds: not a .zip
	// archive and not a .lua script.
	KindNotPluginFile
	// KindFileTooLarge is a plugin file that the computer would not take.
	KindFileTooLarge
	// KindFileUnreadable is a file of the phone that cannot be read.
	KindFileUnreadable
	// KindPairingRefused is a pairing link that is invalid, used or expired.
	KindPairingRefused
	// KindUnpaired is a computer that does not accept the phone: pair it again.
	KindUnpaired
	// KindCertificateChanged is a computer whose certificate is not the paired one.
	KindCertificateChanged
	// KindUnreachable is a computer that did not answer on the network.
	KindUnreachable
	// KindUnavailable is a computer whose vfox failed or did not answer in time.
	KindUnavailable
	// KindNotFound is an item that the computer does not have.
	KindNotFound
	// KindRejected is a request that the computer refused (a status 4xx).
	KindRejected
	// KindServer is a request that the computer failed to answer (a status 5xx).
	KindServer
	// KindOther is any other error.
	KindOther
)

// KindOf returns the kind of err. The checks that match a sentinel come first: the
// certificate error, for example, is also a network error.
func KindOf(err error) Kind {
	var api *remote.APIError
	switch {
	case err == nil:
		return KindNone
	case errors.Is(err, context.Canceled):
		return KindCanceled
	case errors.Is(err, pairing.ErrNotPairingLink):
		return KindNotLink
	case errors.Is(err, ErrFileUnreadable):
		return KindFileUnreadable
	case errors.Is(err, remote.ErrNotPluginFile):
		return KindNotPluginFile
	case errors.Is(err, remote.ErrPluginFileTooLarge):
		return KindFileTooLarge
	case errors.Is(err, remote.ErrPairingRefused):
		return KindPairingRefused
	case errors.Is(err, remote.ErrCertificateChanged):
		return KindCertificateChanged
	case errors.Is(err, remote.ErrUnpaired):
		return KindUnpaired
	case errors.Is(err, remote.ErrNotFound):
		return KindNotFound
	case errors.Is(err, remote.ErrUnavailable):
		return KindUnavailable
	case errors.As(err, &api) && api.Status >= 500:
		return KindServer
	case errors.As(err, &api):
		return KindRejected
	case isNetwork(err):
		return KindUnreachable
	}
	return KindOther
}

// isNetwork reports whether err is a failure of the connection, which the
// computer did not answer. A failure of a file is not: its system error (a file that is
// not there) implements net.Error as well, and it does not say that the network is down.
func isNetwork(err error) bool {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded)
}

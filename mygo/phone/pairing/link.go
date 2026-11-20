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

// Package pairing reads what the phone needs to pair with a desktop server: the
// pairing link as it is pasted or read from a QR code, the fingerprint of the
// server's certificate in the form the desktop application shows it, and the
// QR code of a picture.
package pairing

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// connectPath is the path of a pairing link; the one-time token follows it.
const connectPath = "/auth/connect/"

// tokenPattern is the shape of a one-time token. The desktop makes tokens from
// hexadecimal digits; the pattern also takes letters, '-' and '_', so that a
// change of the token format does not break pairing. A slash is never part of
// a token, so a token is always one path segment.
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// ErrNotPairingLink reports text that is not a pairing link. The error that
// wraps it says why; the messages never repeat the text, because a link holds a
// one-time token.
var ErrNotPairingLink = errors.New("not a pairing link")

// Link is a parsed pairing link.
type Link struct {
	// Base is the address of the server: https://host[:port], with no path.
	Base *url.URL
	// Token is the one-time token that the server exchanges for a session.
	Token string
}

// ValidToken reports whether token has the shape of a one-time token.
func ValidToken(token string) bool {
	return tokenPattern.MatchString(token)
}

// ParseLink parses a pairing link of the form https://host[:port]/auth/connect/<token>,
// as the desktop application shows it. Surrounding white space is ignored.
//
// Only https is accepted: a server that other devices reach speaks https. A bare
// address is accepted only together with the pairing path, such as
// 192.168.1.10:8369/auth/connect/<token>, and is then read as https. An address
// without that path is rejected, so that the phone never guesses a token. Other
// paths, query strings, fragments, user information and tokens with a slash are
// rejected too.
func ParseLink(text string) (Link, error) {
	link := strings.TrimSpace(text)
	if !strings.Contains(link, "://") {
		if !strings.Contains(link, connectPath) {
			return Link{}, fmt.Errorf("%w: it has no scheme and no %s path", ErrNotPairingLink, connectPath)
		}
		link = "https://" + link
	}
	u, err := url.Parse(link)
	if err != nil {
		return Link{}, fmt.Errorf("%w: it does not parse as a URL", ErrNotPairingLink)
	}
	switch {
	case u.Scheme != "https":
		return Link{}, fmt.Errorf("%w: the link must start with https://", ErrNotPairingLink)
	case u.Hostname() == "":
		return Link{}, fmt.Errorf("%w: the link names no server", ErrNotPairingLink)
	case u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		return Link{}, fmt.Errorf("%w: the link has user information, a query or a fragment", ErrNotPairingLink)
	case !validPort(u):
		return Link{}, fmt.Errorf("%w: the port is not a number from 1 to 65535", ErrNotPairingLink)
	case !strings.HasPrefix(u.Path, connectPath):
		return Link{}, fmt.Errorf("%w: the path is not %s<token>", ErrNotPairingLink, connectPath)
	}
	token := strings.TrimPrefix(u.Path, connectPath)
	if !ValidToken(token) {
		return Link{}, fmt.Errorf("%w: the token is missing or has characters that a token does not have", ErrNotPairingLink)
	}
	return Link{Base: &url.URL{Scheme: "https", Host: u.Host}, Token: token}, nil
}

// validPort reports whether the port of u, when it has one, is a number from 1
// to 65535. A colon with no port after it is not valid.
func validPort(u *url.URL) bool {
	port := u.Port()
	if port == "" {
		return !strings.HasSuffix(u.Host, ":")
	}
	number, err := strconv.Atoi(port)
	return err == nil && number >= 1 && number <= 65535
}

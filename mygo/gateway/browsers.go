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

package gateway

import (
	"crypto/rand"
	"net/http"
	"sync"
	"time"
)

// The lifetimes of what pairs a browser with the gateway, as the servers of the application have
// them for the devices of the network.
const (
	// DefaultTokenTTL is how long the pairing link of a browser works.
	DefaultTokenTTL = 5 * time.Minute
	// DefaultSessionTTL is how long a browser stays paired.
	DefaultSessionTTL = 24 * time.Hour
)

// cookieName is the cookie of a browser that is paired with the gateway. It is not the cookie of
// the instances (vfox-plus-session): cookies belong to a host and not to a port, so the cookie of a
// gateway and that of an instance on the same host would be one.
const cookieName = "vfox-plus-gateway"

// browsers are the browsers that are paired with the gateway, and the one-time links that pair them.
type browsers struct {
	mu         sync.Mutex
	tokens     map[string]time.Time
	sessions   map[string]time.Time
	tokenTTL   time.Duration
	sessionTTL time.Duration
	// now is the time: time.Now, or a clock that the tests move when they say, so that no test waits for a
	// link to expire.
	now func() time.Time
}

func newBrowsers(tokenTTL, sessionTTL time.Duration, now func() time.Time) *browsers {
	if now == nil {
		now = time.Now
	}
	return &browsers{tokens: map[string]time.Time{}, sessions: map[string]time.Time{}, tokenTTL: tokenTTL, sessionTTL: sessionTTL, now: now}
}

// newToken makes a one-time token for a pairing link. The tokens that expired unused are swept along
// the way, so that links made for a long-lived gateway cannot grow the map without bound.
func (b *browsers) newToken() string {
	token := rand.Text() // 128 bits.
	now := b.now()
	b.mu.Lock()
	defer b.mu.Unlock()
	for known, expiry := range b.tokens {
		if !now.Before(expiry) {
			delete(b.tokens, known)
		}
	}
	b.tokens[token] = now.Add(b.tokenTTL)
	return token
}

// takeToken uses up a token, and says whether it was a valid one that had not been used or expired.
func (b *browsers) takeToken(token string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	expiry, ok := b.tokens[token]
	if !ok {
		return false
	}
	delete(b.tokens, token)
	return b.now().Before(expiry)
}

// newSession pairs a browser: it makes the value of its cookie.
func (b *browsers) newSession() string {
	session := rand.Text()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions[session] = b.now()
	return session
}

// paired says whether the request is of a browser that is paired with the gateway. The sessions
// that expired are dropped.
func (b *browsers) paired(request *http.Request) bool {
	cookie, err := request.Cookie(cookieName)
	if err != nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	created, ok := b.sessions[cookie.Value]
	if !ok {
		return false
	}
	if b.sessionTTL > 0 && b.now().Sub(created) > b.sessionTTL {
		delete(b.sessions, cookie.Value)
		return false
	}
	return true
}

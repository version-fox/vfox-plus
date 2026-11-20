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

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// entry is one resource of the cache: its value, and whether the value is out of
// date. Every field is guarded by the mutex of the Service.
type entry[T any] struct {
	res Resource[T]
	// dirty says that an event, a change or a Refresh made the value out of date,
	// so that the next read loads it again.
	dirty bool
	// gen counts the values that were set directly (see set). A load that began
	// before the last one does not overwrite it.
	gen uint64
	// read is the tick of the last read, which chooses the entry a cache forgets.
	read uint64
}

// load starts the load of e with fetch when e is due: it has no value and no error
// yet, or it is out of date. A load that is running is not started again, and a
// failed load is not started again until something marks it out of date; so a read
// on every frame never makes a request of its own. The caller holds s.mu.
//
// The load runs on a goroutine, with the request context of the Service, which
// ends when the app goes to the background. Its result is applied under s.mu, and
// the window is asked for a frame. client is the connection that fetch calls: an
// answer that says the computer no longer accepts it, or that its certificate
// changed, ends that connection (see observeLocked). client is nil for a call that
// needs no paired computer.
func load[T any](s *Service, client *remote.Client, e *entry[T], fetch func(ctx context.Context) (T, error)) {
	// Nothing loads while the app is in the background: the reads that come then
	// keep the entry as it is, and the first read after Resume loads it.
	if s.reqCtx.Err() != nil || e.res.Loading() || !(e.dirty || e.res.Phase() == Empty) {
		return
	}
	e.dirty = false
	gen := e.gen
	e.res.Begin()
	ctx := s.reqCtx
	go func() {
		value, err := fetch(ctx)
		s.mu.Lock()
		if e.gen == gen {
			switch {
			case err == nil:
				e.res.Succeed(value)
			case ctx.Err() != nil:
				// The app went to the background during the load. The value stays as
				// it was, and the next read loads it again.
				e.res.Cancel()
				e.dirty = true
			default:
				e.res.Fail(err)
				s.observeLocked(client, err)
			}
		}
		s.mu.Unlock()
		s.notify()
	}()
}

// set stores value as the current answer of e, without a load. A load that is
// running when it is set does not overwrite it. The caller holds s.mu.
func set[T any](e *entry[T], value T) {
	e.res.Succeed(value)
	e.dirty = false
	e.gen++
}

// cache is a set of entries by key. When it holds more than its limit, it forgets
// the entry that was read least recently, unless that entry is loading.
type cache[K comparable, T any] struct {
	limit   int
	tick    uint64
	entries map[K]*entry[T]
}

// newCache returns an empty cache that keeps limit entries.
func newCache[K comparable, T any](limit int) cache[K, T] {
	return cache[K, T]{limit: limit, entries: map[K]*entry[T]{}}
}

// at returns the entry of key, which it makes when there is none, and marks the
// entry as read now. The caller holds s.mu.
func (c *cache[K, T]) at(key K) *entry[T] {
	c.tick++
	e, ok := c.entries[key]
	if !ok {
		if len(c.entries) >= c.limit {
			c.forget()
		}
		e = &entry[T]{}
		c.entries[key] = e
	}
	e.read = c.tick
	return e
}

// forget drops the entry that was read least recently and is not loading.
func (c *cache[K, T]) forget() {
	var (
		victim K
		found  bool
	)
	for key, e := range c.entries {
		if e.res.Loading() {
			continue
		}
		if !found || e.read < c.entries[victim].read {
			victim, found = key, true
		}
	}
	if found {
		delete(c.entries, victim)
	}
}

// mark marks the entries whose key match reports as out of date. A nil match
// marks every entry.
func (c *cache[K, T]) mark(match func(K) bool) {
	for key, e := range c.entries {
		if match == nil || match(key) {
			e.dirty = true
		}
	}
}

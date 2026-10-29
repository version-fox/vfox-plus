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

package state

import "sync"

// AsyncResource is a value that is loaded asynchronously and can be reloaded.
//
// While a reload is in flight the previous Value stays available, so the
// user interface can keep showing stale data with a progress indicator
// instead of flashing an empty state. When loads overlap, only the latest
// one wins.
type AsyncResource[T any] struct {
	Notifier
	mu         sync.Mutex
	loader     func() (T, error)
	value      T
	loaded     bool
	err        error
	loading    bool
	generation uint64
	inFlight   chan struct{}
}

// NewAsyncResource creates a resource that loads with loader.
func NewAsyncResource[T any](loader func() (T, error)) *AsyncResource[T] {
	return &AsyncResource[T]{loader: loader}
}

// Value returns the last successfully loaded value.
func (r *AsyncResource[T]) Value() (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.value, r.loaded
}

// HasValue reports whether a value was loaded at least once.
func (r *AsyncResource[T]) HasValue() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loaded
}

// Error returns the error of the last load, or nil if it succeeded.
func (r *AsyncResource[T]) Error() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// IsLoading reports whether a load is in progress.
func (r *AsyncResource[T]) IsLoading() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loading
}

// Load loads the value, replacing any load that is still in progress. It
// returns a channel that is closed when this load settles; a replaced load
// settles immediately.
func (r *AsyncResource[T]) Load() <-chan struct{} {
	r.mu.Lock()
	r.generation++
	generation := r.generation
	r.loading = true
	done := make(chan struct{})
	r.inFlight = done
	r.mu.Unlock()
	r.NotifyListeners()
	go r.run(generation, done)
	return done
}

// EnsureLoaded loads the value unless it is already loaded or loading.
func (r *AsyncResource[T]) EnsureLoaded() <-chan struct{} {
	r.mu.Lock()
	inFlight := r.inFlight
	loaded := r.loaded
	loading := r.loading
	r.mu.Unlock()
	if loading && inFlight != nil {
		return inFlight
	}
	if loaded {
		done := make(chan struct{})
		close(done)
		return done
	}
	return r.Load()
}

// SetValue replaces the value without loading, for example after an
// optimistic update.
func (r *AsyncResource[T]) SetValue(value T) {
	r.mu.Lock()
	r.generation++
	r.value = value
	r.loaded = true
	r.err = nil
	r.loading = false
	r.mu.Unlock()
	r.NotifyListeners()
}

// Reset forgets the value and the error, and drops the load that is in
// progress, for a resource whose value no longer answers the question that is
// asked of it, such as the list of another distribution. Unlike Load it keeps
// nothing on show while the next load runs.
func (r *AsyncResource[T]) Reset() {
	var zero T
	r.mu.Lock()
	r.generation++
	r.value = zero
	r.loaded = false
	r.err = nil
	r.loading = false
	r.mu.Unlock()
	r.NotifyListeners()
}

func (r *AsyncResource[T]) run(generation uint64, done chan struct{}) {
	value, err := r.loader()
	r.mu.Lock()
	if generation != r.generation {
		r.mu.Unlock()
		close(done)
		return
	}
	if err != nil {
		r.err = err
	} else {
		r.value = value
		r.loaded = true
		r.err = nil
	}
	r.loading = false
	r.mu.Unlock()
	r.NotifyListeners()
	close(done)
}

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

// Resource is a value that the phone loads from the computer: the last value that
// loaded, the error of the last load, and whether a load is running. The zero
// Resource is empty and not loading.
//
// A Resource is not safe for concurrent use. The Service keeps its resources
// under its mutex, and the window reads copies of them, each a snapshot of one
// moment (see Service).
type Resource[T any] struct {
	value   T
	loaded  bool
	err     error
	loading bool
}

// Phase is the state of a Resource.
type Phase int

// The phases of a Resource.
const (
	// Empty means nothing has loaded, and nothing is loading.
	Empty Phase = iota
	// Loading means that the first load is running.
	Loading
	// Failed means that the last load failed. The value is the one loaded
	// before, if any.
	Failed
	// Ready means that a value has loaded, and no load has failed since. A
	// refresh that is running over the value is Ready too; see Refreshing.
	Ready
)

// Begin starts a load, or a refresh of a value that has loaded, which keeps the
// value until a load succeeds.
func (r *Resource[T]) Begin() {
	r.loading = true
}

// Succeed ends the running load with v.
func (r *Resource[T]) Succeed(v T) {
	r.value, r.loaded, r.err, r.loading = v, true, nil, false
}

// Fail ends the running load with err. The value that loaded before is kept.
func (r *Resource[T]) Fail(err error) {
	r.err, r.loading = err, false
}

// Cancel ends the running load without a result: the value and the error that
// were there stay. The Service cancels the loads that the app leaves behind when
// it goes to the background.
func (r *Resource[T]) Cancel() {
	r.loading = false
}

// Value returns the value that loaded, and false when none has.
func (r Resource[T]) Value() (T, bool) {
	return r.value, r.loaded
}

// Err returns the error of the last load, or nil.
func (r Resource[T]) Err() error {
	return r.err
}

// Loading reports whether a load is running.
func (r Resource[T]) Loading() bool {
	return r.loading
}

// Refreshing reports whether a load is running over a value that has loaded.
func (r Resource[T]) Refreshing() bool {
	return r.loading && r.loaded
}

// Phase returns the state of the resource.
func (r Resource[T]) Phase() Phase {
	switch {
	case r.loading && !r.loaded:
		return Loading
	case r.loading:
		return Ready
	case r.err != nil:
		return Failed
	case r.loaded:
		return Ready
	}
	return Empty
}

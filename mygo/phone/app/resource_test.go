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
	"errors"
	"testing"
)

func TestAZeroResourceIsEmptyAndNotLoading(t *testing.T) {
	var r Resource[int]
	if r.Phase() != Empty || r.Loading() || r.Refreshing() {
		t.Errorf("phase %v, loading %v; want empty and not loading", r.Phase(), r.Loading())
	}
	if _, ok := r.Value(); ok {
		t.Error("an empty resource has a value")
	}
}

func TestALoadRunsThenSucceeds(t *testing.T) {
	var r Resource[int]
	r.Begin()
	if r.Phase() != Loading || !r.Loading() || r.Refreshing() {
		t.Errorf("while the first load runs: phase %v", r.Phase())
	}
	r.Succeed(3)
	if v, ok := r.Value(); !ok || v != 3 || r.Phase() != Ready || r.Loading() || r.Err() != nil {
		t.Errorf("after the load: value %v %v, phase %v, err %v", v, ok, r.Phase(), r.Err())
	}
}

func TestAFailedLoadKeepsTheValueBeforeIt(t *testing.T) {
	boom := errors.New("boom")
	var r Resource[int]
	r.Begin()
	r.Succeed(3)
	r.Begin()
	r.Fail(boom)
	if r.Phase() != Failed || r.Loading() || !errors.Is(r.Err(), boom) {
		t.Errorf("after a failed refresh: phase %v, loading %v, err %v", r.Phase(), r.Loading(), r.Err())
	}
	if v, ok := r.Value(); !ok || v != 3 {
		t.Errorf("the value after a failure is %v, %v; want the 3 loaded before", v, ok)
	}
}

func TestARefreshKeepsTheValueWhileItRuns(t *testing.T) {
	var r Resource[string]
	r.Begin()
	r.Succeed("one")
	r.Begin()
	if r.Phase() != Ready || !r.Refreshing() {
		t.Errorf("while a refresh runs: phase %v, refreshing %v", r.Phase(), r.Refreshing())
	}
	if v, _ := r.Value(); v != "one" {
		t.Errorf("the value while refreshing is %q, want %q", v, "one")
	}
	r.Succeed("two")
	if v, _ := r.Value(); v != "two" || r.Refreshing() {
		t.Errorf("after the refresh: value %q, refreshing %v", v, r.Refreshing())
	}
}

func TestARetryAfterAFailureIsALoadAgain(t *testing.T) {
	var r Resource[int]
	r.Begin()
	r.Fail(errors.New("down"))
	r.Begin()
	if r.Phase() != Loading {
		t.Errorf("a retry of a first load is %v, want Loading", r.Phase())
	}
	r.Succeed(1)
	if r.Err() != nil || r.Phase() != Ready {
		t.Errorf("after the retry: err %v, phase %v", r.Err(), r.Phase())
	}
}

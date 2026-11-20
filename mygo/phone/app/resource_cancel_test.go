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

import "testing"

func TestACancelledLoadKeepsTheValueAndTheError(t *testing.T) {
	var r Resource[int]
	r.Begin()
	r.Succeed(5)
	r.Begin()
	r.Cancel()
	if v, ok := r.Value(); !ok || v != 5 || r.Loading() || r.Phase() != Ready {
		t.Errorf("after a cancelled refresh: value %v %v, loading %v, phase %v", v, ok, r.Loading(), r.Phase())
	}

	var first Resource[int]
	first.Begin()
	first.Cancel()
	if first.Phase() != Empty {
		t.Errorf("a cancelled first load is %v, want empty", first.Phase())
	}
}

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

package core

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestProcessLockSerializes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.lock")
	first := NewProcessLock(path)
	if err := first.Lock(); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	acquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		second := NewProcessLock(path)
		defer close(done)
		close(started)
		if err := second.Lock(); err != nil {
			t.Error(err)
			return
		}
		close(acquired)
		second.Unlock()
	}()
	<-started
	select {
	case <-acquired:
		t.Fatal("second lock acquired while the first is held")
	case <-time.After(200 * time.Millisecond):
	}
	first.Unlock()
	<-acquired
	<-done
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestProcessLockUnlockTwice(t *testing.T) {
	lock := NewProcessLock(filepath.Join(t.TempDir(), "tasks.lock"))
	lock.Unlock()
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	lock.Unlock()
	lock.Unlock()
}

func TestProcessLockConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.lock")
	var mu sync.Mutex
	inside := 0
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := NewProcessLock(path)
			if err := lock.Lock(); err != nil {
				t.Error(err)
				return
			}
			defer lock.Unlock()
			mu.Lock()
			inside++
			if inside != 1 {
				t.Error("two holders inside the lock")
			}
			mu.Unlock()
			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	wg.Wait()
}

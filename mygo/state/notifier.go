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

import (
	"sync"
	"sync/atomic"
)

// Listener is called when an observed object changed.
type Listener func()

// Unsubscribe stops a Listener.
type Unsubscribe func()

// Listenable is something whose changes can be observed.
type Listenable interface {
	// AddListener calls listener after every change; the result stops it.
	AddListener(listener Listener) Unsubscribe
}

// Notifier is a Listenable that notifies its listeners when told to. The zero
// value is ready to use.
type Notifier struct {
	mu        sync.Mutex
	next      int64
	listeners map[int64]Listener
	disposed  bool
}

// AddListener implements Listenable.
func (n *Notifier) AddListener(listener Listener) Unsubscribe {
	id := atomic.AddInt64(&n.next, 1)
	n.mu.Lock()
	if n.listeners == nil {
		n.listeners = make(map[int64]Listener)
	}
	n.listeners[id] = listener
	n.mu.Unlock()
	return func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		delete(n.listeners, id)
	}
}

// NotifyListeners calls every listener. It does nothing once disposed.
func (n *Notifier) NotifyListeners() {
	n.mu.Lock()
	if n.disposed {
		n.mu.Unlock()
		return
	}
	listeners := make([]Listener, 0, len(n.listeners))
	for _, listener := range n.listeners {
		listeners = append(listeners, listener)
	}
	n.mu.Unlock()
	for _, listener := range listeners {
		listener()
	}
}

// IsDisposed reports whether the object was disposed.
func (n *Notifier) IsDisposed() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.disposed
}

// Dispose releases the listeners; the object must not be used afterwards.
func (n *Notifier) Dispose() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.disposed = true
	n.listeners = nil
}

// ListenAll listens to several objects at once; the result stops it.
func ListenAll(listenables []Listenable, listener Listener) Unsubscribe {
	var stops []Unsubscribe
	for _, listenable := range listenables {
		if listenable != nil {
			stops = append(stops, listenable.AddListener(listener))
		}
	}
	return func() {
		for _, stop := range stops {
			stop()
		}
	}
}

// Emitter is a simple event source for values.
type Emitter[T any] struct {
	mu        sync.Mutex
	next      int64
	listeners map[int64]func(T)
}

// Listen calls listener with every emitted value; the result stops it.
func (e *Emitter[T]) Listen(listener func(T)) Unsubscribe {
	id := atomic.AddInt64(&e.next, 1)
	e.mu.Lock()
	if e.listeners == nil {
		e.listeners = make(map[int64]func(T))
	}
	e.listeners[id] = listener
	e.mu.Unlock()
	return func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		delete(e.listeners, id)
	}
}

// Emit emits value to every listener.
func (e *Emitter[T]) Emit(value T) {
	e.mu.Lock()
	listeners := make([]func(T), 0, len(e.listeners))
	for _, listener := range e.listeners {
		listeners = append(listeners, listener)
	}
	e.mu.Unlock()
	for _, listener := range listeners {
		listener(value)
	}
}

// Close removes every listener.
func (e *Emitter[T]) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners = nil
}

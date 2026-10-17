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

/** Observable state, the counterpart of Flutter's `ChangeNotifier`. */

/** Called when an observed object changed. */
export type Listener = () => void;

/** Something whose changes can be observed. */
export interface Listenable {
  /** Calls `listener` after every change; returns a function that stops it. */
  addListener(listener: Listener): () => void;
}

/** A {@link Listenable} that notifies its listeners when told to. */
export class ChangeNotifier implements Listenable {
  private readonly listeners = new Set<Listener>();
  private disposed = false;

  addListener(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  /** Whether the object was disposed. */
  get isDisposed(): boolean {
    return this.disposed;
  }

  /** Calls every listener. Does nothing once disposed. */
  protected notifyListeners(): void {
    if (this.disposed) {
      return;
    }
    for (const listener of [...this.listeners]) {
      listener();
    }
  }

  /** Releases the listeners; the object must not be used afterwards. */
  dispose(): void {
    this.disposed = true;
    this.listeners.clear();
  }
}

/** Listens to several objects at once; returns a function that stops it. */
export function listenAll(listenables: readonly Listenable[], listener: Listener): () => void {
  const stops = listenables.map((listenable) => listenable.addListener(listener));
  return () => stops.forEach((stop) => stop());
}

/** A simple event source for values. */
export class Emitter<T> {
  private readonly listeners = new Set<(value: T) => void>();

  /** Calls `listener` with every emitted value; returns a function that stops it. */
  listen(listener: (value: T) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  /** Emits `value` to every listener. */
  emit(value: T): void {
    for (const listener of [...this.listeners]) {
      listener(value);
    }
  }

  /** Removes every listener. */
  close(): void {
    this.listeners.clear();
  }
}

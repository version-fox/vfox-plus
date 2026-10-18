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

import { ChangeNotifier } from './notifier';

/**
 * A value that is loaded asynchronously and can be reloaded.
 *
 * While a reload is in flight the previous {@link value} stays available, so
 * the UI can keep showing stale data with a progress indicator instead of
 * flashing an empty state. When loads overlap, only the latest one wins.
 */
export class AsyncResource<T> extends ChangeNotifier {
  private currentValue: T | null = null;
  private loaded = false;
  private lastError: unknown = null;
  private loading = false;
  private generation = 0;
  private inFlight: Promise<void> | null = null;

  constructor(private readonly loader: () => Promise<T>) {
    super();
  }

  /** The last successfully loaded value. */
  get value(): T | null {
    return this.currentValue;
  }

  /** Whether a value was loaded at least once. */
  get hasValue(): boolean {
    return this.loaded;
  }

  /** The error of the last load, or `null` if it succeeded. */
  get error(): unknown {
    return this.lastError;
  }

  /** Whether a load is in progress. */
  get isLoading(): boolean {
    return this.loading;
  }

  /** Loads the value, replacing any load that is still in progress. */
  load(): Promise<void> {
    const generation = ++this.generation;
    this.loading = true;
    this.notifyListeners();
    this.inFlight = this.run(generation);
    return this.inFlight;
  }

  /** Loads the value unless it is already loaded or loading. */
  ensureLoaded(): Promise<void> {
    if (this.loading) {
      return this.inFlight ?? Promise.resolve();
    }
    return this.loaded ? Promise.resolve() : this.load();
  }

  /** Replaces the value without loading, for example after an optimistic update. */
  setValue(value: T): void {
    this.generation++;
    this.currentValue = value;
    this.loaded = true;
    this.lastError = null;
    this.loading = false;
    this.notifyListeners();
  }

  private async run(generation: number): Promise<void> {
    try {
      const value = await this.loader();
      if (generation !== this.generation) {
        return;
      }
      this.currentValue = value;
      this.loaded = true;
      this.lastError = null;
    } catch (error) {
      if (generation !== this.generation) {
        return;
      }
      this.lastError = error ?? new Error('Unknown error');
    }
    this.loading = false;
    this.notifyListeners();
  }
}

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

import type { ReactiveController, ReactiveControllerHost } from 'lit';

import type { Listenable } from '../state/notifier';

/**
 * Re-renders an element whenever one of the listenables it watches changes,
 * the counterpart of Flutter's `ListenableBuilder`.
 *
 * The listenables are computed again after every change, so an element can
 * watch objects that depend on its state, such as the task of a version.
 */
export class Watch implements ReactiveController {
  private stops: (() => void)[] = [];
  private watched: readonly Listenable[] = [];

  constructor(
    private readonly host: ReactiveControllerHost,
    private readonly listenables: () => readonly (Listenable | null | undefined)[],
  ) {
    host.addController(this);
  }

  hostConnected(): void {
    this.subscribe();
  }

  hostUpdate(): void {
    // The element may watch other objects now.
    const next = this.current();
    if (next.length !== this.watched.length || next.some((listenable, i) => listenable !== this.watched[i])) {
      this.subscribe();
    }
  }

  hostDisconnected(): void {
    this.unsubscribe();
  }

  private current(): Listenable[] {
    return this.listenables().filter(
      (listenable): listenable is Listenable => listenable !== null && listenable !== undefined,
    );
  }

  private subscribe(): void {
    this.unsubscribe();
    this.watched = this.current();
    this.stops = this.watched.map((listenable) =>
      listenable.addListener(() => {
        this.host.requestUpdate();
      }),
    );
  }

  private unsubscribe(): void {
    this.stops.forEach((stop) => stop());
    this.stops = [];
    this.watched = [];
  }
}

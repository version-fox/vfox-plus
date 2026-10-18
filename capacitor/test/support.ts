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

// Test doubles of the running application of ../web/src/ui/context.ts.

import type { Host } from '@web/core/host';
import type { AppSettings } from '@web/core/settings/app-settings';
import type { AppModel } from '@web/state/app-model';
import { LocaleController, setApp, ShellController } from '@web/ui/context';

/** The parts of the application that the tests use; the rest is left out. */
export interface FakeApp {
  readonly locale: LocaleController;
  readonly shell: ShellController;
  /** The settings that `model.settings.update` produced last. */
  settings: Partial<AppSettings>;
  readonly host: Partial<Host>;
}

/** Makes an application with `host` the running one, in English. */
export function startFakeApp(host: Partial<Host> = {}): FakeApp {
  const fake: FakeApp = {
    locale: new LocaleController(['en-US']),
    shell: new ShellController(),
    settings: {},
    host,
  };
  const model = {
    settings: {
      update: (change: (settings: AppSettings) => AppSettings) => {
        fake.settings = change(fake.settings as AppSettings);
        return Promise.resolve();
      },
    },
  };
  setApp({ model: model as unknown as AppModel, host: host as Host, shell: fake.shell, locale: fake.locale });
  return fake;
}

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

import type { Destination } from '@web/ui/context';
import type { IconName } from './icons';

/** The icon and the message key of the label of each destination. */
export const NAVIGATION: Readonly<Record<Destination, { readonly icon: IconName; readonly label: string }>> = {
  home: { icon: 'home', label: 'navHome' },
  sdks: { icon: 'sdks', label: 'navSdks' },
  plugins: { icon: 'plugins', label: 'navPlugins' },
  tasks: { icon: 'tasks', label: 'navTasks' },
  settings: { icon: 'settings', label: 'navSettings' },
};

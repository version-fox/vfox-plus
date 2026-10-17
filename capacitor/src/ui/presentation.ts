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

/**
 * What the pages show, apart from how they show it: constants and functions
 * with the same values and results as in the other implementations.
 */

/** The line that activates vfox in Bash. */
export const ACTIVATION_COMMAND = 'eval "$(vfox activate bash)"';

/** The narrowest SDKs page that shows the list and the detail side by side. */
export const TWO_PANE_WIDTH = 840;

/** Material 3 window size classes: compact windows are narrower than 600 dp. */
export const COMPACT_QUERY = '(max-width: 599.98px)';

/** The repository path of a homepage, such as `version-fox/vfox-nodejs`. */
export function repositoryPath(homepage: string): string {
  try {
    const segments = new URL(homepage).pathname.split('/').filter((segment) => segment !== '');
    return segments.length === 0 ? homepage : segments.join('/');
  } catch {
    return homepage;
  }
}

/** The configuration keys of vfox that get a dedicated editor. */
export const CONFIG_KEYS = {
  proxyEnable: 'proxy.enable',
  proxyUrl: 'proxy.url',
  registry: 'registry.address',
  storage: 'storage.sdkPath',
  legacyEnable: 'legacyVersionFile.enable',
  legacyStrategy: 'legacyVersionFile.strategy',
  cache: 'cache.availableHookDuration',
  httpTimeout: 'plugin.http.timeout',
  downloadTimeout: 'plugin.http.downloadTimeout',
  gitignore: 'gitignore.enable',
} as const;

/** The keys of {@link CONFIG_KEYS}; the other keys are listed as plain text fields. */
export const KNOWN_CONFIG_KEYS: ReadonlySet<string> = new Set(Object.values(CONFIG_KEYS));

/**
 * The shapes of the Material 3 Expressive shape library that look good as
 * decorative containers of small icons and initials, as named by `@m3e/web`.
 */
export const DECORATIVE_SHAPES = [
  '9-sided-cookie',
  '4-leaf-clover',
  'sunny',
  'soft-burst',
  '6-sided-cookie',
  'gem',
  'pentagon',
  'flower',
  'puffy-diamond',
  '4-sided-cookie',
  'arch',
  '8-leaf-clover',
] as const;

/** A shape name of `@m3e/web`. */
export type ShapeName = (typeof DECORATIVE_SHAPES)[number] | 'pill' | 'circle' | '12-sided-cookie';

/**
 * A deterministic, platform-independent string hash (FNV-1a, 32 bit) of the
 * UTF-16 code units of `text`, the same as in the other implementations.
 */
export function stableHash(text: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < text.length; i++) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  return hash >>> 0;
}

/** How the avatar of a plugin looks: its shape, its colour roles and its initial. */
export interface AvatarLook {
  readonly shape: ShapeName;
  /** The colour role of the shape, such as `secondary`. */
  readonly container: string;
  /** The colour role of the initial or the icon. */
  readonly content: string;
  readonly initial: string;
}

/**
 * The colour roles of avatars, as container and content. Selected list items
 * use the secondary container colour. Avatars avoid it, and the primary
 * container too, which some seeds (the vfox orange among them) make just as
 * light, so that they stay visible on selected items.
 */
const AVATAR_COLORS = [
  ['primary-fixed-dim', 'on-primary-fixed'],
  ['secondary', 'on-secondary'],
  ['tertiary-container', 'on-tertiary-container'],
] as const;

/** The avatar of the plugin `name`, which keeps its look on every run. */
export function avatarLook(name: string): AvatarLook {
  const hash = stableHash(name);
  const [container, content] = AVATAR_COLORS[hash % AVATAR_COLORS.length] ?? AVATAR_COLORS[0];
  const first = new Intl.Segmenter().segment(name)[Symbol.iterator]().next().value?.segment;
  return {
    shape: DECORATIVE_SHAPES[hash % DECORATIVE_SHAPES.length] ?? '9-sided-cookie',
    container,
    content,
    initial: (first ?? '?').toUpperCase(),
  };
}

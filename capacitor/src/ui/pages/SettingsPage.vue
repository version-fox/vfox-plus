<!--
  Copyright 2026 Han Li and contributors

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->

<!-- Settings of the application and of vfox. -->
<script setup lang="ts">
import { IonButton, IonContent, IonItem, IonLabel, IonList, IonTextarea, IonToggle } from '@ionic/vue';
import { ref } from 'vue';

import {
  formatEnvironmentLines,
  parseEnvironmentLines,
  type AppSettings,
  type LanguagePreference,
  type ThemePreference,
} from '@web/core/settings/app-settings';
import { app, LINKS } from '@web/ui/context';
import { licenseText } from '@web/ui/licenses';
import { hex, primaryOf, SEED_COLORS } from '@web/ui/theme';
import { APP_VERSION } from '@web/version';
import type { Choice } from '../components/choice';
import ChoiceGroup from '../components/ChoiceGroup.vue';
import ErrorMessage from '../components/ErrorMessage.vue';
import IconButton from '../components/IconButton.vue';
import LoadingMessage from '../components/LoadingMessage.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import PluginAvatar from '../components/PluginAvatar.vue';
import { showText } from '../overlays';
import { CONFIG_KEYS, KNOWN_CONFIG_KEYS } from '../presentation';
import ConfigChoiceRow from '../settings/ConfigChoiceRow.vue';
import ConfigSwitchRow from '../settings/ConfigSwitchRow.vue';
import ConfigTextRow from '../settings/ConfigTextRow.vue';
import { t, watched } from '../reactivity';
import { snackbar } from '../snackbar';

const { model, host } = app();
const vfox = () => watched(model.vfox);
const config = () => watched(model.vfoxConfig);
if (model.vfox.availability === 'ready') {
  void model.vfoxConfig.load();
}

/** The text of the environment field that has not been saved yet. */
const environmentText = ref<string | null>(null);
const environmentError = ref('');

function settings(): AppSettings {
  return watched(model.settings).settings;
}

function update(change: (settings: AppSettings) => AppSettings): void {
  void model.settings.update(change);
}

function themes(): Choice<ThemePreference>[] {
  return [
    ['system', t('themeSystem'), 'systemMode'],
    ['light', t('themeLight'), 'lightMode'],
    ['dark', t('themeDark'), 'darkMode'],
  ];
}

function languages(): Choice<LanguagePreference>[] {
  // Language names are always shown in their own language.
  return [
    ['system', t('languageSystem'), null],
    ['english', 'English', null],
    ['simplifiedChinese', '简体中文', null],
  ];
}

function dark(): boolean {
  return document.documentElement.style.colorScheme === 'dark';
}

function executableDetails(): string {
  const { executable, isCustomExecutable, version } = vfox();
  const source =
    executable === null
      ? t('settingsExecutableMissing')
      : isCustomExecutable
        ? t('settingsExecutableCustom')
        : t('settingsExecutableDetected');
  return [executable ?? 'vfox', source, ...(version === null ? [] : [t('settingsVfoxVersion', { version })])].join(
    ' · ',
  );
}

async function chooseExecutable(): Promise<void> {
  const path = await host.pickFile('/usr/local/bin');
  if (path !== null) {
    update((s) => ({ ...s, vfoxExecutable: path }));
  }
}

function environment(): string {
  return environmentText.value ?? formatEnvironmentLines(settings().extraEnvironment);
}

async function saveEnvironment(): Promise<void> {
  const parsed = parseEnvironmentLines(environment());
  if (parsed.invalidLines.length > 0) {
    environmentError.value = t('settingsEnvironmentInvalid', {
      count: parsed.invalidLines.length,
      lines: parsed.invalidLines.join(', '),
    });
    return;
  }
  environmentError.value = '';
  await model.settings.update((s) => ({ ...s, extraEnvironment: parsed.variables }));
  snackbar(t('settingsSaved'));
}

function otherKeys(): string[] {
  return (config().value ?? []).map((entry) => entry.key).filter((key) => !KNOWN_CONFIG_KEYS.has(key));
}

async function showLicenses(): Promise<void> {
  await showText(t('aboutLicenses'), await licenseText());
}
</script>

<template>
  <IonContent class="pane">
    <header class="page-header">
      <div class="titles">
        <h1 class="type-headline-medium-emphasized">{{ t('navSettings') }}</h1>
      </div>
    </header>
    <div class="content settings">
      <h2 class="group-header">{{ t('settingsAppearance') }}</h2>
      <IonList class="segmented group" lines="none">
        <IonItem class="setting">
          <MaterialIcon slot="start" name="darkMode" />
          <IonLabel
            ><div class="title type-body-large">{{ t('settingsTheme') }}</div></IonLabel
          >
          <ChoiceGroup
            slot="end"
            :choices="themes()"
            :selected="settings().theme"
            @choose="(theme) => update((s) => ({ ...s, theme }))"
          />
        </IonItem>
        <IonItem class="setting">
          <MaterialIcon slot="start" name="palette" />
          <IonToggle
            justify="space-between"
            :checked="settings().useSystemColor"
            @ion-change="update((s) => ({ ...s, useSystemColor: $event.detail.checked }))"
          >
            <span class="title type-body-large">{{ t('settingsSystemColor') }}</span>
          </IonToggle>
        </IonItem>
        <IonItem class="setting">
          <span slot="start" class="icon-space"></span>
          <div class="row-text">
            <div class="title type-body-large">{{ t('settingsColor') }}</div>
            <div class="swatches">
              <button
                v-for="(seed, index) in SEED_COLORS"
                :key="seed"
                class="swatch"
                :aria-label="t('settingsColorOption', { index: index + 1 })"
                :title="t('settingsColorOption', { index: index + 1 })"
                :aria-pressed="!settings().useSystemColor && settings().seedColor === seed ? 'true' : 'false'"
                :data-seed="hex(seed)"
                @click="update((s) => ({ ...s, seedColor: seed, useSystemColor: false }))"
              >
                <m3e-shape
                  :name="!settings().useSystemColor && settings().seedColor === seed ? 'flower' : 'circle'"
                  :style="{ '--m3e-shape-container-color': primaryOf(seed, dark()) }"
                />
                <MaterialIcon name="check" :style="{ color: dark() ? '#000' : '#fff' }" />
              </button>
            </div>
          </div>
        </IonItem>
        <IonItem class="setting">
          <MaterialIcon slot="start" name="language" />
          <IonLabel
            ><div class="title type-body-large">{{ t('settingsLanguage') }}</div></IonLabel
          >
          <ChoiceGroup
            slot="end"
            :choices="languages()"
            :selected="settings().language"
            @choose="(language) => update((s) => ({ ...s, language }))"
          />
        </IonItem>
      </IonList>

      <h2 class="group-header">{{ t('settingsVfox') }}</h2>
      <IonList class="segmented group" lines="none">
        <IonItem class="setting">
          <MaterialIcon slot="start" name="terminal" />
          <IonLabel>
            <div class="title type-body-large">{{ t('settingsExecutable') }}</div>
            <div class="supporting type-body-medium">{{ executableDetails() }}</div>
          </IonLabel>
          <div slot="end" class="trailing">
            <m3e-loading-indicator
              v-if="vfox().availability === 'checking'"
              class="small"
              :aria-label="t('vfoxCheckingTitle')"
            />
            <IconButton
              v-else-if="vfox().isCustomExecutable"
              icon="refresh"
              :label="t('settingsExecutableDetect')"
              @click="update((s) => ({ ...s, vfoxExecutable: null }))"
            />
            <IonButton color="tonal" @click="chooseExecutable">{{ t('actionBrowse') }}</IonButton>
          </div>
        </IonItem>
        <IonItem class="setting">
          <MaterialIcon slot="start" name="environment" />
          <div class="row-text">
            <div class="title type-body-large">{{ t('settingsEnvironment') }}</div>
            <div class="supporting type-body-medium">{{ t('settingsEnvironmentHelp') }}</div>
            <IonTextarea
              class="environment mono"
              fill="outline"
              :rows="3"
              :aria-label="t('settingsEnvironment')"
              placeholder="VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node"
              :value="environment()"
              :error-text="environmentError"
              :class="{ 'ion-invalid': environmentError !== '', 'ion-touched': environmentError !== '' }"
              @ion-input="environmentText = String($event.detail.value ?? '')"
            />
            <div class="save">
              <IonButton @click="saveEnvironment">{{ t('actionSave') }}</IonButton>
            </div>
          </div>
        </IonItem>
      </IonList>

      <h2 class="group-header">{{ t('settingsVfoxConfig') }}</h2>
      <p class="help type-body-small">{{ t('settingsVfoxConfigHelp') }}</p>
      <template v-if="vfox().availability === 'ready'">
        <template v-if="!config().hasValue">
          <LoadingMessage v-if="config().error === null" />
          <ErrorMessage
            v-else
            :heading="t('settingsVfoxConfigError')"
            :error="config().error"
            :retry="() => void model.vfoxConfig.load()"
          />
        </template>
        <template v-else>
          <IonList class="segmented group" lines="none">
            <ConfigSwitchRow :config-key="CONFIG_KEYS.proxyEnable" :title="t('configProxyEnable')" icon="proxy" />
            <ConfigTextRow
              :config-key="CONFIG_KEYS.proxyUrl"
              :title="t('configProxyUrl')"
              :icon="null"
              hint="http://127.0.0.1:7890"
            />
            <ConfigTextRow
              :config-key="CONFIG_KEYS.registry"
              :title="t('configRegistry')"
              icon="registry"
              :hint="t('configRegistryHint')"
            />
            <ConfigTextRow
              :config-key="CONFIG_KEYS.storage"
              :title="t('configStorage')"
              icon="storage"
              :hint="t('configStorageHint')"
              directory
            />
          </IonList>
          <IonList class="segmented group" lines="none">
            <ConfigSwitchRow
              :config-key="CONFIG_KEYS.legacyEnable"
              :title="t('configLegacyEnable')"
              icon="legacyFiles"
              :help="t('configLegacyEnableHelp')"
            />
            <ConfigChoiceRow
              :config-key="CONFIG_KEYS.legacyStrategy"
              :title="t('configLegacyStrategy')"
              :choices="[
                ['specified', t('strategySpecified')],
                ['latest_installed', t('strategyLatestInstalled')],
                ['latest_available', t('strategyLatestAvailable')],
              ]"
            />
            <ConfigSwitchRow
              :config-key="CONFIG_KEYS.gitignore"
              :title="t('configGitignore')"
              icon="gitignore"
              :help="t('configGitignoreHelp')"
            />
          </IonList>
          <IonList class="segmented group" lines="none">
            <ConfigTextRow
              :config-key="CONFIG_KEYS.cache"
              :title="t('configCache')"
              icon="cache"
              hint="12h"
              :help="t('configCacheHelp')"
            />
            <ConfigTextRow
              :config-key="CONFIG_KEYS.httpTimeout"
              :title="t('configHttpTimeout')"
              icon="timeout"
              hint="30s"
              :help="t('configTimeoutHelp')"
            />
            <ConfigTextRow
              :config-key="CONFIG_KEYS.downloadTimeout"
              :title="t('configDownloadTimeout')"
              :icon="null"
              hint="30m"
              :help="t('configTimeoutHelp')"
            />
          </IonList>
          <template v-if="otherKeys().length > 0">
            <h2 class="group-header">{{ t('configOther') }}</h2>
            <IonList class="segmented group" lines="none">
              <ConfigTextRow
                v-for="key in otherKeys()"
                :key="key"
                :config-key="key"
                :title="key"
                :icon="null"
                hint=""
              />
            </IonList>
          </template>
        </template>
      </template>

      <h2 class="group-header">{{ t('settingsAbout') }}</h2>
      <IonList class="segmented group" lines="none">
        <IonItem class="setting">
          <PluginAvatar slot="start" name="vfox+" icon="sdks" :size="40" />
          <IonLabel>
            <div class="title type-body-large">{{ t('appTitle') }}</div>
            <div class="supporting type-body-medium">
              {{ t('aboutVersion', { version: APP_VERSION }) }} · {{ t('aboutLicense') }} · {{ host.name }}
            </div>
          </IonLabel>
        </IonItem>
        <IonItem button :detail="false" class="setting" @click="host.openUrl(LINKS.repository)">
          <MaterialIcon slot="start" name="sourceCode" />
          <IonLabel>
            <div class="title type-body-large">{{ t('aboutSource') }}</div>
            <div class="supporting type-body-medium">{{ LINKS.repository }}</div>
          </IonLabel>
          <MaterialIcon slot="end" name="openInNew" />
        </IonItem>
        <IonItem button :detail="false" class="setting" @click="host.openUrl(LINKS.vfoxDocs)">
          <MaterialIcon slot="start" name="docs" />
          <IonLabel>
            <div class="title type-body-large">{{ t('aboutDocs') }}</div>
            <div class="supporting type-body-medium">{{ LINKS.vfoxDocs }}</div>
          </IonLabel>
          <MaterialIcon slot="end" name="openInNew" />
        </IonItem>
        <IonItem button :detail="false" class="setting" @click="showLicenses">
          <MaterialIcon slot="start" name="license" />
          <IonLabel
            ><div class="title type-body-large">{{ t('aboutLicenses') }}</div></IonLabel
          >
          <MaterialIcon slot="end" name="chevronRight" />
        </IonItem>
      </IonList>
    </div>
  </IonContent>
</template>

<style scoped>
.content {
  max-width: 840px;
  padding-bottom: var(--gap-xxl);
}
.group + .group {
  margin-top: var(--gap-l);
}
.row-text {
  flex: 1 1 auto;
  min-width: 0;
  padding: var(--gap-m) 0;
}
.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--gap-s);
  margin-top: var(--gap-s);
}
.swatch {
  position: relative;
  width: 40px;
  height: 40px;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
  display: grid;
  place-items: center;
}
.swatch:focus-visible {
  outline: 3px solid var(--md-sys-color-secondary);
  outline-offset: 2px;
  border-radius: 50%;
}
.swatch m3e-shape {
  position: absolute;
  inset: 0;
  --m3e-shape-size: 40px;
  --m3e-shape-transition: clip-path var(--md-sys-motion-spring-fast-spatial);
  transition: transform var(--md-sys-motion-spring-fast-spatial);
}
.swatch[aria-pressed='true'] m3e-shape {
  transform: rotate(34deg);
}
.swatch .material-icon {
  position: relative;
  --icon-size: 20px;
  opacity: 0;
  transition: opacity var(--md-sys-motion-spring-fast-effects);
}
.swatch[aria-pressed='true'] .material-icon {
  opacity: 1;
}
.environment {
  margin-top: var(--gap-m);
}
.save {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--gap-s);
}
.help {
  padding: 0 var(--gap-l) var(--gap-m);
  color: var(--md-sys-color-on-surface-variant);
}
.trailing {
  display: flex;
  align-items: center;
  gap: var(--gap-xs);
}
m3e-loading-indicator.small {
  --m3e-loading-indicator-size: 36px;
}
</style>

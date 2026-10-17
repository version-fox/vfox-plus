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

// The catalogues of shared/i18n, and how the front end formats them.

import { readdirSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

import { Localizations, messageKeys, resolveLanguage } from '../../src/core/i18n/localizations';
import { formatMessage, MessageFormatError, parseMessage, type MessagePart } from '../../src/core/i18n/message-format';
import { SHARED } from '../support/fakes';

type Catalogue = Record<string, unknown>;
const en = JSON.parse(readFileSync(`${SHARED}/i18n/app_en.arb`, 'utf8')) as Catalogue;
const zh = JSON.parse(readFileSync(`${SHARED}/i18n/app_zh.arb`, 'utf8')) as Catalogue;

function placeholders(parts: readonly MessagePart[]): Set<string> {
  const names = new Set<string>();
  for (const part of parts) {
    if (part.kind === 'argument') {
      names.add(part.name);
    } else if (part.kind === 'plural') {
      names.add(part.name);
      for (const nested of part.cases.values()) {
        placeholders(nested).forEach((name) => names.add(name));
      }
    }
  }
  return names;
}

describe('message format', () => {
  it('formats placeholders and plurals with exact matches first', () => {
    const parts = parseMessage('{count, plural, =0{none} =1{one {name}} other{{count} of {name}}}!');
    expect(formatMessage(parts, { count: 0, name: 'x' }, 'en')).toBe('none!');
    expect(formatMessage(parts, { count: 1, name: 'x' }, 'en')).toBe('one x!');
    expect(formatMessage(parts, { count: 7, name: 'x' }, 'en')).toBe('7 of x!');
  });

  it('chooses plural categories by language', () => {
    const parts = parseMessage('{n, plural, one{single} other{many}}');
    expect(formatMessage(parts, { n: 1 }, 'en')).toBe('single');
    expect(formatMessage(parts, { n: 1 }, 'zh')).toBe('many');
  });

  it('keeps unknown placeholders visible and apostrophes literal', () => {
    expect(formatMessage(parseMessage("don't {x}"), {}, 'en')).toBe("don't {x}");
  });

  it('rejects malformed messages', () => {
    expect(() => parseMessage('{a, select, x{y} other{z}}')).toThrow(MessageFormatError);
    expect(() => parseMessage('{a, plural, =1{y}}')).toThrow(MessageFormatError);
    expect(() => parseMessage('a } b')).toThrow(MessageFormatError);
    expect(() => parseMessage('{1}')).toThrow(MessageFormatError);
  });
});

describe('catalogues', () => {
  const englishKeys = messageKeys(en);

  it('are the only ones, and the template is English', () => {
    expect(readdirSync(`${SHARED}/i18n`).sort()).toEqual(['app_en.arb', 'app_zh.arb']);
    expect(en['@@locale']).toBe('en');
    expect(zh['@@locale']).toBe('zh');
  });

  it('have the same messages', () => {
    expect(messageKeys(zh).sort()).toEqual([...englishKeys].sort());
  });

  it('describe every message of the template', () => {
    for (const key of englishKeys) {
      expect((en[`@${key}`] as { description?: string } | undefined)?.description, key).toBeTruthy();
    }
  });

  it('parse, and keep every placeholder in translations', () => {
    for (const key of englishKeys) {
      const template = placeholders(parseMessage(en[key] as string));
      const translated = placeholders(parseMessage(zh[key] as string));
      for (const name of template) {
        expect(translated.has(name), `${key} lacks {${name}} in app_zh.arb`).toBe(true);
      }
      const declared = Object.keys(
        (en[`@${key}`] as { placeholders?: Record<string, unknown> } | undefined)?.placeholders ?? {},
      );
      expect([...template].sort(), key).toEqual(declared.sort());
    }
  });
});

describe('Localizations', () => {
  it('formats messages of each language and falls back to English', () => {
    const english = new Localizations('en', en, en);
    const chinese = new Localizations('zh', zh, en);
    expect(english.t('sdksVersionCount', { count: 2 })).toBe('2 versions installed');
    expect(english.t('sdksVersionCount', { count: 1 })).toBe('1 version installed');
    expect(chinese.t('sdksVersionCount', { count: 2 })).toBe('已安装 2 个版本');
    expect(chinese.t('taskInstall', { subject: 'maven@3.9.16' })).toBe('安装 maven@3.9.16');
    expect(new Localizations('zh', { navHome: '首页' }, en).t('navSdks')).toBe('SDKs');
    expect(english.t('noSuchKey')).toBe('noSuchKey');
    expect(english.languageTag).toBe('en');
    expect(chinese.languageTag).toBe('zh-Hans');
  });

  it('resolves the system language', () => {
    expect(resolveLanguage('system', ['zh-TW'])).toBe('zh');
    expect(resolveLanguage('system', ['zh_CN.UTF-8'])).toBe('zh');
    expect(resolveLanguage('system', ['de-DE', 'en-US'])).toBe('en');
    expect(resolveLanguage('system', ['fr'])).toBe('en');
    expect(resolveLanguage('english', ['zh-CN'])).toBe('en');
    expect(resolveLanguage('simplifiedChinese', ['en'])).toBe('zh');
  });
});

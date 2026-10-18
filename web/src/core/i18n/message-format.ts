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
 * The subset of ICU MessageFormat that the catalogues in `shared/i18n` use:
 * placeholders (`{name}`) and plurals with exact matches and plural
 * categories (`{count, plural, =0{…} =1{…} other{…}}`), which may contain
 * placeholders themselves. Apostrophes are literal, as in Flutter's
 * `gen-l10n` without `use-escaping`. See `docs/i18n.md`.
 */

/** A parsed message. */
export type MessagePart =
  | { readonly kind: 'text'; readonly text: string }
  | { readonly kind: 'argument'; readonly name: string }
  | {
      readonly kind: 'plural';
      readonly name: string;
      readonly cases: ReadonlyMap<string, readonly MessagePart[]>;
    };

/** A message could not be parsed. */
export class MessageFormatError extends Error {
  constructor(message: string, source: string) {
    super(`${message} in "${source}"`);
    this.name = 'MessageFormatError';
  }
}

/** Parses `source` into parts. */
export function parseMessage(source: string): MessagePart[] {
  const parser = new Parser(source);
  const parts = parser.parseParts();
  if (!parser.atEnd()) {
    throw new MessageFormatError('Unbalanced "}"', source);
  }
  return parts;
}

/** Arguments of a message, by placeholder name. */
export type MessageArguments = Readonly<Record<string, string | number>>;

/**
 * Formats parsed `parts` with `args`, choosing plural cases with the plural
 * rules of `locale`.
 */
export function formatMessage(parts: readonly MessagePart[], args: MessageArguments, locale: string): string {
  const rules = pluralRules(locale);
  const format = (items: readonly MessagePart[]): string =>
    items
      .map((part) => {
        switch (part.kind) {
          case 'text':
            return part.text;
          case 'argument':
            return String(args[part.name] ?? `{${part.name}}`);
          case 'plural': {
            const value = Number(args[part.name] ?? 0);
            const chosen =
              part.cases.get(`=${value}`) ?? part.cases.get(rules.select(value)) ?? part.cases.get('other') ?? [];
            return format(chosen);
          }
        }
      })
      .join('');
  return format(parts);
}

const rulesCache = new Map<string, Intl.PluralRules>();

function pluralRules(locale: string): Intl.PluralRules {
  let rules = rulesCache.get(locale);
  if (rules === undefined) {
    rules = new Intl.PluralRules(locale);
    rulesCache.set(locale, rules);
  }
  return rules;
}

class Parser {
  private position = 0;

  constructor(private readonly source: string) {}

  atEnd(): boolean {
    return this.position >= this.source.length;
  }

  /** Parses parts until the end or an unmatched `}`, which is not consumed. */
  parseParts(): MessagePart[] {
    const parts: MessagePart[] = [];
    let text = '';
    while (!this.atEnd()) {
      const char = this.source.charAt(this.position);
      if (char === '}') {
        break;
      }
      if (char === '{') {
        if (text !== '') {
          parts.push({ kind: 'text', text });
          text = '';
        }
        parts.push(this.parsePlaceholder());
      } else {
        text += char;
        this.position++;
      }
    }
    if (text !== '') {
      parts.push({ kind: 'text', text });
    }
    return parts;
  }

  private parsePlaceholder(): MessagePart {
    this.expect('{');
    const name = this.readIdentifier();
    this.skipSpaces();
    if (this.peek() === '}') {
      this.position++;
      return { kind: 'argument', name };
    }
    this.expect(',');
    this.skipSpaces();
    const type = this.readIdentifier();
    if (type !== 'plural') {
      throw new MessageFormatError(`Unsupported argument type "${type}"`, this.source);
    }
    this.skipSpaces();
    this.expect(',');
    const cases = new Map<string, MessagePart[]>();
    for (;;) {
      this.skipSpaces();
      if (this.peek() === '}') {
        this.position++;
        break;
      }
      const selector = this.readSelector();
      this.skipSpaces();
      this.expect('{');
      cases.set(selector, this.parseParts());
      this.expect('}');
    }
    if (!cases.has('other')) {
      throw new MessageFormatError('A plural needs an "other" case', this.source);
    }
    return { kind: 'plural', name, cases };
  }

  private readIdentifier(): string {
    this.skipSpaces();
    const match = /^[A-Za-z_][A-Za-z0-9_]*/.exec(this.source.slice(this.position));
    if (match === null) {
      throw new MessageFormatError('Expected a name', this.source);
    }
    this.position += match[0].length;
    return match[0];
  }

  private readSelector(): string {
    const match = /^(=\d+|[a-z]+)/.exec(this.source.slice(this.position));
    if (match === null) {
      throw new MessageFormatError('Expected a plural case', this.source);
    }
    this.position += match[0].length;
    return match[0];
  }

  private skipSpaces(): void {
    while (!this.atEnd() && /\s/.test(this.source[this.position] ?? '')) {
      this.position++;
    }
  }

  private peek(): string | undefined {
    return this.source[this.position];
  }

  private expect(char: string): void {
    if (this.peek() !== char) {
      throw new MessageFormatError(`Expected "${char}" at ${this.position}`, this.source);
    }
    this.position++;
  }
}

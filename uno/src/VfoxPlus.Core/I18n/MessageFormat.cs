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

using System.Globalization;

namespace VfoxPlus.Core.I18n;

/// <summary>A part of a parsed message.</summary>
public abstract record MessagePart;

/// <summary>Literal text.</summary>
/// <param name="Content">The text.</param>
public sealed record TextPart(string Content) : MessagePart;

/// <summary>A placeholder: <c>{name}</c>.</summary>
/// <param name="Name">The name of the argument.</param>
public sealed record ArgumentPart(string Name) : MessagePart;

/// <summary>A plural: <c>{name, plural, =0{…} other{…}}</c>.</summary>
/// <param name="Name">The name of the argument that decides.</param>
/// <param name="Cases">The cases by selector: <c>=0</c>, <c>one</c>, <c>other</c> and so on.</param>
public sealed record PluralPart(string Name, IReadOnlyDictionary<string, IReadOnlyList<MessagePart>> Cases) : MessagePart;

/// <summary>Reports that a message could not be parsed.</summary>
public sealed class MessageFormatException : Exception
{
    /// <summary>Initializes a new instance of the <see cref="MessageFormatException"/> class.</summary>
    /// <param name="message">What is wrong.</param>
    /// <param name="source">The message that could not be parsed.</param>
    public MessageFormatException(string message, string source)
        : base($"{message} in \"{source}\"")
    {
    }
}

/// <summary>
/// The subset of ICU MessageFormat that the catalogues in <c>shared/i18n</c> use: placeholders
/// (<c>{name}</c>) and plurals with exact matches and plural categories
/// (<c>{count, plural, =0{…} =1{…} other{…}}</c>), which may contain placeholders themselves. Apostrophes
/// are literal, as in Flutter's <c>gen-l10n</c> without <c>use-escaping</c>. See <c>docs/i18n.md</c>.
/// </summary>
public static class MessageFormat
{
    /// <summary>Parses a message into parts.</summary>
    /// <param name="source">The message.</param>
    /// <returns>The parts.</returns>
    /// <exception cref="MessageFormatException">The message is malformed.</exception>
    public static IReadOnlyList<MessagePart> Parse(string source)
    {
        Parser parser = new(source);
        List<MessagePart> parts = parser.ParseParts();
        return parser.AtEnd ? parts : throw new MessageFormatException("Unbalanced \"}\"", source);
    }

    /// <summary>
    /// Returns the CLDR plural category of a number in a language: English has <c>one</c> and
    /// <c>other</c>, Chinese only <c>other</c>.
    /// </summary>
    /// <param name="value">The number.</param>
    /// <param name="language">The language code.</param>
    /// <returns>The category.</returns>
    public static string PluralCategory(double value, string language) =>
        language == "en" && value == 1 ? "one" : "other";

    /// <summary>Formats parsed parts with arguments, choosing plural cases with the plural rules of a language.</summary>
    /// <param name="parts">The parts.</param>
    /// <param name="args">The arguments by placeholder name.</param>
    /// <param name="language">The language code.</param>
    /// <returns>The text.</returns>
    public static string Format(IReadOnlyList<MessagePart> parts, IReadOnlyDictionary<string, object?> args, string language)
    {
        System.Text.StringBuilder output = new();
        foreach (MessagePart part in parts)
        {
            switch (part)
            {
                case TextPart text:
                    output.Append(text.Content);
                    break;
                case ArgumentPart argument:
                    output.Append(args.TryGetValue(argument.Name, out object? value) ? FormatValue(value) : "{" + argument.Name + "}");
                    break;
                case PluralPart plural:
                    double number = NumberOf(args.GetValueOrDefault(plural.Name));
                    IReadOnlyList<MessagePart>? chosen = null;
                    if (number == Math.Floor(number) && Math.Abs(number) <= 1e15)
                    {
                        plural.Cases.TryGetValue("=" + ((long)number).ToString(CultureInfo.InvariantCulture), out chosen);
                    }

                    chosen ??= plural.Cases.GetValueOrDefault(PluralCategory(number, language)) ?? plural.Cases["other"];
                    output.Append(Format(chosen, args, language));
                    break;
            }
        }

        return output.ToString();
    }

    private static string FormatValue(object? value) => value switch
    {
        null => string.Empty,
        string text => text,
        double number when number == Math.Floor(number) && Math.Abs(number) <= 1e15 => ((long)number).ToString(CultureInfo.InvariantCulture),
        IFormattable formattable => formattable.ToString(null, CultureInfo.InvariantCulture),
        _ => value.ToString() ?? string.Empty,
    };

    private static double NumberOf(object? value) => value switch
    {
        int number => number,
        long number => number,
        double number => number,
        float number => number,
        string text when double.TryParse(text.Trim(), NumberStyles.Float, CultureInfo.InvariantCulture, out double number) => number,
        _ => 0,
    };

    private sealed class Parser(string source)
    {
        private int position;

        public bool AtEnd => position >= source.Length;

        // Parses parts until the end or an unmatched "}", which is not consumed.
        public List<MessagePart> ParseParts()
        {
            List<MessagePart> parts = [];
            System.Text.StringBuilder text = new();
            while (!AtEnd)
            {
                char c = source[position];
                if (c == '}')
                {
                    break;
                }

                if (c == '{')
                {
                    if (text.Length > 0)
                    {
                        parts.Add(new TextPart(text.ToString()));
                        text.Clear();
                    }

                    parts.Add(ParsePlaceholder());
                }
                else
                {
                    text.Append(c);
                    position++;
                }
            }

            if (text.Length > 0)
            {
                parts.Add(new TextPart(text.ToString()));
            }

            return parts;
        }

        private MessagePart ParsePlaceholder()
        {
            Expect('{');
            string name = ReadIdentifier();
            SkipSpaces();
            if (Peek() == '}')
            {
                position++;
                return new ArgumentPart(name);
            }

            Expect(',');
            string kind = ReadIdentifier();
            if (kind != "plural")
            {
                throw new MessageFormatException($"Unsupported argument type \"{kind}\"", source);
            }

            SkipSpaces();
            Expect(',');
            Dictionary<string, IReadOnlyList<MessagePart>> cases = [];
            while (true)
            {
                SkipSpaces();
                if (Peek() == '}')
                {
                    position++;
                    break;
                }

                string selector = ReadSelector();
                SkipSpaces();
                Expect('{');
                List<MessagePart> body = ParseParts();
                Expect('}');
                cases[selector] = body;
            }

            return cases.ContainsKey("other")
                ? new PluralPart(name, cases)
                : throw new MessageFormatException("A plural needs an \"other\" case", source);
        }

        private string ReadIdentifier()
        {
            SkipSpaces();
            int start = position;
            while (position < source.Length)
            {
                char c = source[position];
                if (c == '_' || c is >= 'a' and <= 'z' || c is >= 'A' and <= 'Z' || (position > start && c is >= '0' and <= '9'))
                {
                    position++;
                }
                else
                {
                    break;
                }
            }

            return position == start ? throw new MessageFormatException("Expected a name", source) : source[start..position];
        }

        private string ReadSelector()
        {
            SkipSpaces();
            int start = position;
            if (position < source.Length && source[position] == '=')
            {
                position++;
                int digits = position;
                while (position < source.Length && source[position] is >= '0' and <= '9')
                {
                    position++;
                }

                return position == digits ? throw new MessageFormatException("Expected a plural case", source) : source[start..position];
            }

            while (position < source.Length && source[position] is >= 'a' and <= 'z')
            {
                position++;
            }

            return position == start ? throw new MessageFormatException("Expected a plural case", source) : source[start..position];
        }

        private void SkipSpaces()
        {
            while (!AtEnd && source[position] is ' ' or '\t' or '\n' or '\r')
            {
                position++;
            }
        }

        private char Peek() => AtEnd ? '\0' : source[position];

        private void Expect(char c)
        {
            if (Peek() != c)
            {
                throw new MessageFormatException($"Expected \"{c}\" at {position}", source);
            }

            position++;
        }
    }
}

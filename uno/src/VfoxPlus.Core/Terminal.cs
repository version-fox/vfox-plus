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

using System.Text;
using System.Text.RegularExpressions;

namespace VfoxPlus.Core;

/// <summary>
/// Plain-text helpers for the output of terminal programs.
/// </summary>
public static partial class Terminal
{
    /// <summary>
    /// Removes the ANSI escape sequences from text: CSI sequences such as colours and cursor movement, OSC
    /// sequences such as hyperlinks, and lone two-byte escapes.
    /// </summary>
    /// <remarks>
    /// vfox honours <c>NO_COLOR</c>, but its download progress bar does not, so the output is always
    /// sanitised before it is parsed or shown.
    /// </remarks>
    /// <param name="text">The text.</param>
    /// <returns>The text without escape sequences.</returns>
    public static string StripAnsi(string text) => AnsiEscape().Replace(text, string.Empty);

    /// <summary>
    /// Splits terminal output into lines. Both <c>\n</c> and a lone <c>\r</c> end a line: progress bars
    /// redraw themselves by returning the carriage, and every redraw is treated as a new line.
    /// </summary>
    /// <param name="text">The output.</param>
    /// <returns>The non-empty lines.</returns>
    public static string[] SplitLines(string text) =>
        text.Split(['\n', '\r'], StringSplitOptions.RemoveEmptyEntries);

    [GeneratedRegex("\u001b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\u0007\u001b]*(?:\u0007|\u001b\\\\)|[@-Z\\\\-_])", RegexOptions.CultureInvariant)]
    private static partial Regex AnsiEscape();
}

/// <summary>
/// Turns chunks of raw process output into plain-text lines while the process runs.
/// </summary>
/// <remarks>
/// Bytes are decoded as UTF-8 (malformed sequences are replaced once as a whole, and a character split
/// between two chunks is kept together), lines end at <c>\n</c> and at <c>\r</c>, ANSI escape sequences are
/// removed and blank lines are dropped. The trailing partial line is emitted by <see cref="Close"/>.
/// </remarks>
public sealed class TerminalLineSplitter
{
    private readonly Decoder decoder = Encoding.UTF8.GetDecoder();
    private readonly StringBuilder text = new();

    /// <summary>Feeds a chunk of output.</summary>
    /// <param name="chunk">The bytes.</param>
    /// <returns>The lines the chunk completed.</returns>
    public IReadOnlyList<string> Push(ReadOnlySpan<byte> chunk)
    {
        Decode(chunk, flush: false);
        return Split();
    }

    /// <summary>
    /// Ends the output. A trailing partial character is replaced, as a decoder does at the end of the
    /// stream.
    /// </summary>
    /// <returns>The last, unterminated line, if there is one; and the lines still pending.</returns>
    public IReadOnlyList<string> Close()
    {
        Decode([], flush: true);
        List<string> lines = [.. Split()];
        lines.AddRange(CleanLine(text.ToString()));
        text.Clear();
        return lines;
    }

    private static string[] CleanLine(string line)
    {
        string trimmed = Terminal.StripAnsi(line).Trim();
        return trimmed.Length == 0 ? [] : [trimmed];
    }

    private void Decode(ReadOnlySpan<byte> bytes, bool flush)
    {
        char[] buffer = new char[decoder.GetCharCount(bytes, flush)];
        int count = decoder.GetChars(bytes, buffer, flush);
        text.Append(buffer, 0, count);
    }

    private List<string> Split()
    {
        List<string> lines = [];
        string content = text.ToString();
        int start = 0;
        for (int i = 0; i < content.Length; i++)
        {
            if (content[i] is '\n' or '\r')
            {
                lines.AddRange(CleanLine(content[start..i]));
                start = i + 1;
            }
        }

        text.Clear();
        text.Append(content, start, content.Length - start);
        return lines;
    }
}

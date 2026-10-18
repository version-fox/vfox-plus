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
using System.Text;
using VfoxPlus.Core.Processes;

namespace VfoxPlus.Tests.Support;

/// <summary>The captured vfox output in <c>shared/fixtures</c>.</summary>
internal static class Fixtures
{
    /// <summary>The vfox release whose captures the tests use by default.</summary>
    public const string Version = "1.0.13";

    /// <summary>
    /// Reads a capture, in which control characters are written as <c>\xHH</c> and backslashes as
    /// <c>\\</c>.
    /// </summary>
    /// <param name="path">The file.</param>
    /// <returns>The exact text that vfox printed.</returns>
    public static string ReadCapture(string path)
    {
        string text = File.ReadAllText(path);
        StringBuilder output = new(text.Length);
        for (int i = 0; i < text.Length; i++)
        {
            if (text[i] == '\\' && i + 1 < text.Length)
            {
                if (text[i + 1] == '\\')
                {
                    output.Append('\\');
                    i++;
                    continue;
                }

                if (text[i + 1] == 'x' && i + 3 < text.Length
                    && int.TryParse(text.AsSpan(i + 2, 2), NumberStyles.HexNumber, CultureInfo.InvariantCulture, out int code))
                {
                    output.Append((char)code);
                    i += 3;
                    continue;
                }
            }

            output.Append(text[i]);
        }

        return output.ToString();
    }

    /// <summary>Returns the result of a fixture.</summary>
    /// <param name="name">The name of the case.</param>
    /// <param name="exitCode">The exit code.</param>
    /// <returns>The output.</returns>
    public static ProcessOutput Output(string name, int exitCode = 0)
    {
        string baseName = Path.Combine(TestPaths.SharedDirectory, "fixtures", "vfox-" + Version, name);
        return new ProcessOutput(exitCode, ReadCapture(baseName + ".stdout"), ReadCapture(baseName + ".stderr"));
    }
}

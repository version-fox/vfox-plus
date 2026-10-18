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
using VfoxPlus.Core;
using Xunit;

namespace VfoxPlus.Tests.Core;

public class TerminalTests
{
    [Fact]
    public void StripAnsiRemovesColoursLinksAndCursorMoves()
    {
        Assert.Equal("red text", Terminal.StripAnsi("\u001b[31mred\u001b[0m text"));
        Assert.Equal("ab", Terminal.StripAnsi("a\u001b[2K\u001b[1Gb"));
        Assert.Equal("vfox", Terminal.StripAnsi("\u001b]8;;https://vfox.dev\u0007vfox\u001b]8;;\u0007"));
        Assert.Equal("vfox", Terminal.StripAnsi("\u001b]8;;https://vfox.dev\u001b\\vfox\u001b]8;;\u001b\\"));
        Assert.Equal("nothing to remove", Terminal.StripAnsi("nothing to remove"));
        Assert.Equal("ok link x", Terminal.StripAnsi("\u001b[1;32mok\u001b[0m \u001b]8;;https://x\u0007link\u001b]8;;\u0007 \u001bMx"));
    }

    [Fact]
    public void SplitLinesEndsLinesAtNewlinesAndCarriageReturns()
    {
        Assert.Equal(["one", "two", "three", "four"], Terminal.SplitLines("one\r\ntwo\rthree\n\nfour"));
        Assert.Equal(["a", "b", "c", "d"], Terminal.SplitLines("a\r\nb\rc\nd"));
    }

    [Fact]
    public void LineSplitterJoinsChunksAndCharactersSplitBetweenThem()
    {
        TerminalLineSplitter splitter = new();
        List<string> lines = [];

        // "é" is two bytes and "你" three; each is cut between two chunks.
        foreach (byte[] chunk in new[]
        {
            Encoding.Latin1.GetBytes("caf\u00c3"),
            Encoding.Latin1.GetBytes("\u00a9 \u00e4\u00bd"),
            Encoding.Latin1.GetBytes("\u00a0\nnext"),
            Encoding.Latin1.GetBytes(" line\r"),
            Encoding.Latin1.GetBytes("\u001b[32mgreen\u001b[0m\n"),
        })
        {
            lines.AddRange(splitter.Push(chunk));
        }

        lines.AddRange(splitter.Close());
        Assert.Equal(["café 你", "next line", "green"], lines);
    }

    [Fact]
    public void LineSplitterKeepsAPartialLineUntilItIsComplete()
    {
        TerminalLineSplitter splitter = new();
        Assert.Equal(["one"], splitter.Push("one\r\u001b[32mtw"u8));
        Assert.Empty(splitter.Push("o "u8));
        Assert.Equal(["two"], splitter.Push("\n \n"u8));
        Assert.Empty(splitter.Push("last"u8));
        Assert.Equal(["last"], splitter.Close());
        Assert.Empty(splitter.Close());
    }

    [Fact]
    public void LineSplitterTreatsRedrawsOfAProgressBarAsLines()
    {
        TerminalLineSplitter splitter = new();
        IReadOnlyList<string> lines = splitter.Push("\rDownloading...  10%\rDownloading...  20%\r"u8);
        Assert.Equal(["Downloading...  10%", "Downloading...  20%"], lines);
    }

    [Fact]
    public void LineSplitterReplacesMalformedBytesOnceForTheWholeSequence()
    {
        TerminalLineSplitter splitter = new();
        List<string> lines = [.. splitter.Push([(byte)'a', 0xff, (byte)'b', 0xe4, 0xbd, (byte)'\n'])];
        lines.AddRange(splitter.Close());
        Assert.Equal(["a\ufffdb\ufffd"], lines);

        TerminalLineSplitter truncated = new();
        truncated.Push([.. "end "u8, 0xe4, 0xbd]);
        Assert.Equal(["end \ufffd"], truncated.Close());
    }

    [Fact]
    public void VersionAdditionsAreShownWithTheirVersion()
    {
        Assert.Equal("npm 11.19.1", new VersionAddition("npm", "11.19.1").ToString());
        Assert.Equal("npm", new VersionAddition("npm", string.Empty).ToString());
    }
}

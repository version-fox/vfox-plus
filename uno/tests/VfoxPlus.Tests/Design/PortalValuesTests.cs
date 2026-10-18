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

using VfoxPlus.Design.Platform;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class PortalValuesTests
{
    [Fact]
    public void TheAccentColourOfThePortalIsConvertedToArgb()
    {
        Assert.Equal(0xFFFF0000u, PortalValues.ParseAccent([1, 0, 0]));
        Assert.Equal(0xFF000000u, PortalValues.ParseAccent([0, 0, 0]));
        Assert.Equal(0xFF808080u, PortalValues.ParseAccent([0.5, 0.5, 0.5]));
        Assert.Equal(0xFF3584E4u, PortalValues.ParseAccent([53 / 255.0, 132 / 255.0, 228 / 255.0]));
    }

    [Theory]
    [InlineData(-0.1, 0, 0)]
    [InlineData(0, 1.1, 0)]
    [InlineData(0, 0, double.NaN)]
    public void ComponentsOutsideTheRangeMeanThatTheDesktopHasNoAccentColour(double red, double green, double blue)
    {
        Assert.Null(PortalValues.ParseAccent([red, green, blue]));
    }

    [Fact]
    public void AnAccentColourNeedsThreeComponents()
    {
        Assert.Null(PortalValues.ParseAccent([]));
        Assert.Null(PortalValues.ParseAccent([1, 1]));
        Assert.Null(PortalValues.ParseAccent([1, 1, 1, 1]));
    }

    [Theory]
    [InlineData(0u, null)]
    [InlineData(1u, true)]
    [InlineData(2u, false)]
    [InlineData(3u, null)]
    public void TheColourSchemeSaysWhetherTheDesktopPrefersDark(uint scheme, bool? expected)
    {
        Assert.Equal(expected, PortalValues.ParseColorScheme(scheme));
    }
}

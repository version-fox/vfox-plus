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

using VfoxPlus.Design.Layout;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class PlacementTests
{
    private static readonly Box Anchor = new(100, 100, 40, 40);

    [Fact]
    public void APopupGoesBelowItsAnchorAlignedToItsStart()
    {
        Assert.Equal((100, 140), Placement.Place(800, 600, Anchor, 200, 100, alignEnd: false, margin: 8));
    }

    [Fact]
    public void APopupCanAlignWithTheEndOfItsAnchor()
    {
        Assert.Equal((8, 140), Placement.Place(800, 600, Anchor, 200, 100, alignEnd: true, margin: 8));
        Assert.Equal((8, 140), Placement.Place(800, 600, new Box(10, 100, 40, 40), 200, 100, alignEnd: true, margin: 8));
    }

    [Fact]
    public void APopupThatDoesNotFitBelowGoesAbove()
    {
        Assert.Equal((100, 400), Placement.Place(800, 600, new Box(100, 560, 40, 30), 200, 160, alignEnd: false, margin: 8));
    }

    [Fact]
    public void APopupThatFitsNowhereIsPushedIntoTheWindow()
    {
        Assert.Equal((592, 8), Placement.Place(800, 100, new Box(790, 40, 40, 40), 200, 90, alignEnd: false, margin: 8));
        Assert.Equal((8, 8), Placement.Place(100, 100, new Box(0, 0, 10, 10), 200, 200, alignEnd: false, margin: 8));
    }

    [Fact]
    public void ATooltipIsCentredBelowItsAnchorAndKeptInTheWindow()
    {
        Assert.Equal((90, 144), Placement.PlaceTooltip(800, 600, Anchor, 60, 24));
        Assert.Equal((8, 144), Placement.PlaceTooltip(800, 600, new Box(0, 100, 20, 40), 100, 24));
        Assert.Equal((792 - 100, 144), Placement.PlaceTooltip(800, 600, new Box(780, 100, 20, 40), 100, 24));
        Assert.Equal((90, 72), Placement.PlaceTooltip(800, 160, Anchor, 60, 24));
    }

    [Fact]
    public void ABoxHasItsFarEdges()
    {
        Assert.Equal(140, Anchor.Right);
        Assert.Equal(140, Anchor.Bottom);
    }
}

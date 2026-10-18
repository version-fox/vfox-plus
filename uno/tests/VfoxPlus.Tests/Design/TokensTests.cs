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

using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class TokensTests
{
    [Theory]
    [InlineData(0, WindowSizeClass.Compact)]
    [InlineData(599.9, WindowSizeClass.Compact)]
    [InlineData(600, WindowSizeClass.Medium)]
    [InlineData(839, WindowSizeClass.Medium)]
    [InlineData(840, WindowSizeClass.Expanded)]
    [InlineData(1199, WindowSizeClass.Expanded)]
    [InlineData(1200, WindowSizeClass.Large)]
    [InlineData(1599, WindowSizeClass.Large)]
    [InlineData(1600, WindowSizeClass.ExtraLarge)]
    public void WindowsAreClassifiedByWidth(double width, WindowSizeClass expected)
    {
        Assert.Equal(expected, WindowSizeClasses.FromWidth(width));
    }

    [Fact]
    public void TheTypeScaleHasTheMaterialSizesAndTheEmphasizedWeights()
    {
        Assert.Equal(new TextStyle(14, 20, 400), Typography.Of(TextRole.BodyMedium));
        Assert.Equal(new TextStyle(14, 20, 500), Typography.Emphasized(TextRole.BodyMedium));
        Assert.Equal(new TextStyle(22, 28, 400), Typography.Of(TextRole.TitleLarge));
        Assert.Equal(700, Typography.Emphasized(TextRole.TitleMedium).Weight);
        Assert.Equal(700, Typography.Emphasized(TextRole.LabelLarge).Weight);
        Assert.Equal(500, Typography.Emphasized(TextRole.HeadlineMedium).Weight);
        TextStyle larger = Typography.Of(TextRole.BodyMedium).WithSize(28);
        Assert.Equal(new TextStyle(28, 40, 400), larger);
        Assert.Equal(300, larger.WithWeight(300).Weight);
        foreach (TextRole role in Enum.GetValues<TextRole>())
        {
            Assert.True(Typography.Emphasized(role).Weight > Typography.Of(role).Weight, $"{role} gets heavier");
        }
    }

    [Fact]
    public void TheCornerScaleIsTheMaterialOne()
    {
        double[] scale = [Corners.None, Corners.ExtraSmall, Corners.Small, Corners.Medium, Corners.Large, Corners.LargeIncreased, Corners.ExtraLarge, Corners.ExtraLargeIncreased, Corners.ExtraExtraLarge];
        Assert.Equal([0, 4, 8, 12, 16, 20, 28, 32, 48], scale);
        Assert.True(Corners.Full > 1000);
        Assert.Equal([2, 4, 8, 12, 16, 24, 32], new[] { Gaps.Hairline, Gaps.Xs, Gaps.S, Gaps.M, Gaps.L, Gaps.Xl, Gaps.Xxl });
    }

    [Fact]
    public void AvatarsAreTheSameOnEveryRunAndInEveryImplementation()
    {
        // FNV-1a over the UTF-16 code units.
        Assert.Equal(0x811C9DC5u, Avatars.StableHash(string.Empty));
        Assert.Equal(0xE40C292Cu, Avatars.StableHash("a"));
        Assert.Equal(Avatars.StableHash("nodejs"), Avatars.StableHash("nodejs"));
        Assert.NotEqual(Avatars.StableHash("nodejs"), Avatars.StableHash("maven"));
        foreach (string name in new[] { "nodejs", "maven", "mvn2", "java", "python", "go", "中文" })
        {
            Assert.InRange(Avatars.ShapeIndex(name), 0, MaterialShapes.AvatarShapes.Count - 1);
            Assert.InRange(Avatars.ColorIndex(name), 0, 2);
        }
    }

    [Fact]
    public void TheInitialIsTheFirstCharacterInUpperCase()
    {
        Assert.Equal("N", Avatars.Initial("nodejs"));
        Assert.Equal("?", Avatars.Initial(string.Empty));
        Assert.Equal("É", Avatars.Initial("été"));
        Assert.Equal("\U0001D4D0", Avatars.Initial("\U0001D4D0b"));
    }
}

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

using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>
/// The avatar of a plugin: its initial on a shape picked by a hash of its name, in one of three colour pairs. Hovering
/// spins the shape by 30°.
/// </summary>
internal sealed class Avatar : Grid
{
    private readonly ShapeView shape;
    private readonly Animated spin;

    public Avatar(string name, double size = 40, string? symbol = null)
    {
        AppTheme theme = Ui.Theme;

        // Avatars avoid the secondary container, which marks selected list items, and the primary container, which
        // some seeds make just as light.
        (Brush background, Brush foreground) = Avatars.ColorIndex(name) switch
        {
            1 => (theme.Secondary, theme.OnSecondary),
            2 => (theme.TertiaryContainer, theme.OnTertiaryContainer),
            _ => (theme.PrimaryFixedDim, theme.OnPrimaryFixed),
        };
        shape = new ShapeView(MaterialShapes.AvatarShapes[Avatars.ShapeIndex(name)], size, background);
        Children.Add(shape);
        if (symbol is not null)
        {
            Children.Add(new M3Icon(symbol, size / 2, foreground) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center });
        }
        else
        {
            TextBlock initial = Ui.Text(Avatars.Initial(name), Typography.Emphasized(TextRole.TitleMedium).WithSize(size * 0.42).WithWeight(700), foreground);
            initial.HorizontalAlignment = HorizontalAlignment.Center;
            initial.VerticalAlignment = VerticalAlignment.Center;
            Children.Add(initial);
        }

        Width = size;
        Height = size;
        Background = theme.Transparent;
        spin = new Animated(MotionScheme.Expressive.FastSpatial, value => shape.Angle = value * 30);
        PointerEntered += (_, _) => spin.To(1);
        PointerExited += (_, _) => spin.To(0);
    }
}

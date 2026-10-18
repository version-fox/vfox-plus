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

using Microsoft.UI.Xaml.Automation;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Shapes;

namespace VfoxPlus.App.Controls;

/// <summary>
/// The Material 3 Expressive loading indicator: a shape that morphs into the next while it turns. It is drawn
/// while it is on screen, and costs no frames otherwise.
/// </summary>
internal sealed class LoadingIndicator : Grid
{
    private static readonly Lazy<IReadOnlyList<Morph>> Morphs = new(MakeMorphs);

    private readonly ShapeView shape;
    private readonly TimeSpan started = FrameClock.Instance.Now;
    private bool attached;

    public LoadingIndicator(double size = 48, bool contained = false, Brush? color = null, string name = "")
    {
        AppTheme theme = Ui.Theme;
        Width = size;
        Height = size;
        if (contained)
        {
            Children.Add(new Border { Background = theme.PrimaryContainer, CornerRadius = new CornerRadius(size / 2) });
        }

        double side = size * Progress.LoadingActiveScale;
        shape = new ShapeView(MaterialShapes.SoftBurst, side, color ?? (contained ? theme.OnPrimaryContainer : theme.Primary))
        {
            HorizontalAlignment = HorizontalAlignment.Center,
            VerticalAlignment = VerticalAlignment.Center,
        };
        Children.Add(shape);
        AutomationProperties.SetName(this, name);
        Draw(TimeSpan.Zero);
        Loaded += (_, _) =>
        {
            if (!attached)
            {
                attached = true;
                FrameClock.Instance.Run(OnFrame);
            }
        };
        Unloaded += (_, _) => attached = false;
    }

    private static IReadOnlyList<Morph> MakeMorphs()
    {
        RoundedPolygon[] sequence =
        [
            MaterialShapes.SoftBurst, MaterialShapes.Cookie9, MaterialShapes.Pentagon, MaterialShapes.Pill,
            MaterialShapes.Sunny, MaterialShapes.Cookie4, MaterialShapes.Oval,
        ];
        RoundedPolygon[] fitted = [.. sequence.Select(polygon => polygon.FitToCircle())];
        return [.. fitted.Select((polygon, i) => new Morph(polygon, fitted[(i + 1) % fitted.Length]))];
    }

    private bool OnFrame(double seconds)
    {
        if (!attached)
        {
            return false;
        }

        if (!FrameClock.Instance.ReducedMotion)
        {
            Draw(FrameClock.Instance.Now - started);
        }

        return true;
    }

    private void Draw(TimeSpan elapsed)
    {
        (int step, double progress, double rotation) = Progress.Loading(elapsed);
        IReadOnlyList<Morph> morphs = Morphs.Value;
        shape.SetOutline(morphs[step % morphs.Count].AsCubics(Math.Min(1, Math.Max(0, progress))));
        shape.Angle = rotation * 180 / Math.PI;
    }
}

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
using Windows.Foundation;

namespace VfoxPlus.App.Controls;

/// <summary>
/// A filled shape of the Material shape library, drawn in a square of a side, which can be turned and, for the
/// loading indicator, redrawn as a morph of two shapes.
/// </summary>
internal sealed class ShapeView : Grid
{
    private readonly Microsoft.UI.Xaml.Shapes.Path path = new() { Stretch = Stretch.None, IsHitTestVisible = false };
    private double size;
    private IReadOnlyList<Cubic> outline = [];

    public ShapeView(RoundedPolygon shape, double size, Brush fill)
    {
        Fill = fill;
        Size = size;
        Shape = shape;
        path.RenderTransformOrigin = new Point(0.5, 0.5);
        path.HorizontalAlignment = HorizontalAlignment.Center;
        path.VerticalAlignment = VerticalAlignment.Center;
        Children.Add(path);
        IsHitTestVisible = false;
    }

    /// <summary>Gets or sets the shape.</summary>
    public RoundedPolygon Shape
    {
        get => field!;
        set
        {
            field = value;
            SetOutline(value.Cubics);
        }
    }

    /// <summary>Gets or sets the side of the square the shape is drawn in, in dp.</summary>
    public double Size
    {
        get => size;
        set
        {
            size = value;
            Width = value;
            Height = value;
            path.Width = value;
            path.Height = value;
            if (outline.Count > 0)
            {
                SetOutline(outline);
            }
        }
    }

    /// <summary>Gets or sets the colour of the shape.</summary>
    public Brush Fill
    {
        get => path.Fill;
        set => path.Fill = value;
    }

    /// <summary>Gets or sets the angle in degrees that the shape is turned by around its centre.</summary>
    public double Angle
    {
        get => (path.RenderTransform as RotateTransform)?.Angle ?? 0;
        set => path.RenderTransform = new RotateTransform { Angle = value };
    }

    /// <summary>Draws an outline, which is a loop of curves in the unit square, in place of the shape.</summary>
    /// <param name="cubics">The curves.</param>
    public void SetOutline(IReadOnlyList<Cubic> cubics)
    {
        outline = cubics;
        PathFigure figure = new() { IsClosed = true, IsFilled = true, StartPoint = new Point(cubics[0].X0 * size, cubics[0].Y0 * size) };
        foreach (Cubic cubic in cubics)
        {
            figure.Segments.Add(new BezierSegment
            {
                Point1 = new Point(cubic.X1 * size, cubic.Y1 * size),
                Point2 = new Point(cubic.X2 * size, cubic.Y2 * size),
                Point3 = new Point(cubic.X3 * size, cubic.Y3 * size),
            });
        }

        path.Data = new PathGeometry { Figures = { figure } };
    }
}

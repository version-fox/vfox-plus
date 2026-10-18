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

using Windows.Foundation;

namespace VfoxPlus.App.Controls;

/// <summary>
/// Lays its children out in rows, and starts a new row when the next child does not fit: the tags of a header, the
/// swatches of the colours, the controls of a toolbar.
/// </summary>
internal sealed class M3WrapPanel : Panel
{
    private double spacing;
    private double runSpacing;
    private bool centerCross;
    private bool alignEnd;

    /// <summary>Gets or sets the space between the children of a row.</summary>
    public double Spacing
    {
        get => spacing;
        set
        {
            spacing = value;
            InvalidateMeasure();
        }
    }

    /// <summary>Gets or sets the space between the rows.</summary>
    public double RunSpacing
    {
        get => runSpacing;
        set
        {
            runSpacing = value;
            InvalidateMeasure();
        }
    }

    /// <summary>Gets or sets a value indicating whether the children of a row are centred on it, rather than at its top.</summary>
    public bool CenterCross
    {
        get => centerCross;
        set
        {
            centerCross = value;
            InvalidateArrange();
        }
    }

    /// <summary>Gets or sets a value indicating whether the rows end at the end of the space, rather than start at its start.</summary>
    public bool AlignEnd
    {
        get => alignEnd;
        set
        {
            alignEnd = value;
            InvalidateArrange();
        }
    }

    /// <inheritdoc/>
    protected override Size MeasureOverride(Size availableSize)
    {
        double width = availableSize.Width;
        foreach (UIElement child in Children)
        {
            child.Measure(new Size(width, availableSize.Height));
        }

        List<Row> rows = Split(width);
        double height = rows.Sum(row => row.Height) + (Math.Max(0, rows.Count - 1) * runSpacing);
        double widest = rows.Count == 0 ? 0 : rows.Max(row => row.Width);
        return new Size(double.IsInfinity(width) ? widest : Math.Min(widest, width), height);
    }

    /// <inheritdoc/>
    protected override Size ArrangeOverride(Size finalSize)
    {
        double y = 0;
        foreach (Row row in Split(finalSize.Width))
        {
            double x = alignEnd ? Math.Max(0, finalSize.Width - row.Width) : 0;
            foreach (UIElement child in row.Children)
            {
                Size size = child.DesiredSize;
                double top = centerCross ? y + ((row.Height - size.Height) / 2) : y;
                child.Arrange(new Rect(x, top, size.Width, size.Height));
                x += size.Width + spacing;
            }

            y += row.Height + runSpacing;
        }

        return finalSize;
    }

    private List<Row> Split(double width)
    {
        List<Row> rows = [];
        Row current = new();
        foreach (UIElement child in Children.Where(child => child.Visibility == Visibility.Visible))
        {
            Size size = child.DesiredSize;
            if (current.Children.Count > 0 && current.Width + spacing + size.Width > width)
            {
                rows.Add(current);
                current = new Row();
            }

            current.Width += (current.Children.Count > 0 ? spacing : 0) + size.Width;
            current.Height = Math.Max(current.Height, size.Height);
            current.Children.Add(child);
        }

        if (current.Children.Count > 0)
        {
            rows.Add(current);
        }

        return rows;
    }

    private sealed class Row
    {
        public List<UIElement> Children { get; } = [];

        public double Width { get; set; }

        public double Height { get; set; }
    }
}

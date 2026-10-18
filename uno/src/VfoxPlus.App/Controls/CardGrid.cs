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
/// A grid of cards of the same height: as many columns as fit, so that no card is wider than <see cref="MaxCardWidth"/>
/// (plus a little), and the cards share the width.
/// </summary>
internal sealed class CardGrid : Panel
{
    /// <summary>Gets or sets the width a card should not exceed, in dp.</summary>
    public double MaxCardWidth { get; set; } = 340;

    /// <summary>Gets or sets the height of a card, in dp.</summary>
    public double CardHeight { get; set; } = 152;

    /// <summary>Gets or sets the space between the cards, in dp.</summary>
    public double Gap { get; set; } = 12;

    /// <inheritdoc/>
    protected override Size MeasureOverride(Size availableSize)
    {
        int columns = Columns(availableSize.Width);
        double cell = CellWidth(availableSize.Width, columns);
        foreach (UIElement child in Children)
        {
            child.Measure(new Size(cell, CardHeight));
        }

        int rows = (Children.Count + columns - 1) / columns;
        double width = double.IsInfinity(availableSize.Width) ? (columns * cell) + ((columns - 1) * Gap) : availableSize.Width;
        return new Size(width, (rows * CardHeight) + (Math.Max(0, rows - 1) * Gap));
    }

    /// <inheritdoc/>
    protected override Size ArrangeOverride(Size finalSize)
    {
        int columns = Columns(finalSize.Width);
        double cell = CellWidth(finalSize.Width, columns);
        for (int i = 0; i < Children.Count; i++)
        {
            int row = i / columns;
            int column = i % columns;
            Children[i].Arrange(new Rect(column * (cell + Gap), row * (CardHeight + Gap), cell, CardHeight));
        }

        return finalSize;
    }

    private int Columns(double width) =>
        double.IsInfinity(width) || width <= 0 ? 1 : Math.Max(1, (int)Math.Ceiling(width / (MaxCardWidth + Gap)));

    private double CellWidth(double width, int columns) =>
        double.IsInfinity(width) ? MaxCardWidth : Math.Max(0, (width - (Gap * (columns - 1))) / columns);
}

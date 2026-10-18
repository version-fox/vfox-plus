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

namespace VfoxPlus.Design.Layout;

/// <summary>
/// A rectangle.
/// </summary>
/// <param name="X">The left edge.</param>
/// <param name="Y">The top edge.</param>
/// <param name="Width">The width.</param>
/// <param name="Height">The height.</param>
public readonly record struct Box(double X, double Y, double Width, double Height)
{
    /// <summary>Gets the right edge.</summary>
    public double Right => X + Width;

    /// <summary>Gets the bottom edge.</summary>
    public double Bottom => Y + Height;
}

/// <summary>Where popups go.</summary>
public static class Placement
{
    /// <summary>
    /// Puts a popup of a size next to an anchor: below it, aligned to its start edge, or its end edge with
    /// <paramref name="alignEnd"/>, and moved to stay inside the window, above the anchor if it does not fit below.
    /// </summary>
    /// <param name="windowWidth">The width of the window.</param>
    /// <param name="windowHeight">The height of the window.</param>
    /// <param name="anchor">Where the anchor is, in the coordinates of the window.</param>
    /// <param name="popupWidth">The width of the popup.</param>
    /// <param name="popupHeight">The height of the popup.</param>
    /// <param name="alignEnd">Whether the popup lines up with the end of the anchor.</param>
    /// <param name="margin">The space to keep between the popup and the edges of the window.</param>
    /// <returns>The top left corner of the popup.</returns>
    public static (double X, double Y) Place(double windowWidth, double windowHeight, Box anchor, double popupWidth, double popupHeight, bool alignEnd, double margin)
    {
        double x = alignEnd ? anchor.Right - popupWidth : anchor.X;
        double y = anchor.Bottom;
        if (y + popupHeight > windowHeight - margin)
        {
            double above = anchor.Y - popupHeight;
            y = above >= margin ? above : windowHeight - margin - popupHeight;
        }

        x = Math.Min(Math.Max(x, margin), Math.Max(margin, windowWidth - margin - popupWidth));
        y = Math.Max(y, margin);
        return (x, y);
    }

    /// <summary>
    /// Puts a tooltip of a size below an anchor, centred, and above it when it does not fit below, inside the window.
    /// </summary>
    /// <param name="windowWidth">The width of the window.</param>
    /// <param name="windowHeight">The height of the window.</param>
    /// <param name="anchor">Where the anchor is, in the coordinates of the window.</param>
    /// <param name="tooltipWidth">The width of the tooltip.</param>
    /// <param name="tooltipHeight">The height of the tooltip.</param>
    /// <returns>The top left corner of the tooltip.</returns>
    public static (double X, double Y) PlaceTooltip(double windowWidth, double windowHeight, Box anchor, double tooltipWidth, double tooltipHeight)
    {
        const double gap = 4;
        const double margin = 8;
        double x = anchor.X + (anchor.Width / 2) - (tooltipWidth / 2);
        double y = anchor.Bottom + gap;
        if (y + tooltipHeight > windowHeight)
        {
            y = anchor.Y - gap - tooltipHeight;
        }

        x = Math.Min(Math.Max(x, margin), Math.Max(margin, windowWidth - margin - tooltipWidth));
        return (x, Math.Max(y, 0));
    }
}

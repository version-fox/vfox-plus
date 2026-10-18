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

namespace VfoxPlus.App.E2E;

/// <summary>What a node of the tree of the user interface is, for the tests.</summary>
internal enum NodeRole
{
    /// <summary>A text.</summary>
    Text,

    /// <summary>A button, a tab, a destination of the navigation.</summary>
    Button,

    /// <summary>A switch.</summary>
    Switch,

    /// <summary>An item of a list.</summary>
    ListItem,

    /// <summary>An entry of a menu.</summary>
    MenuItem,

    /// <summary>A field that takes text.</summary>
    TextField,

    /// <summary>A dialog.</summary>
    Dialog,
}

/// <summary>
/// Something on the screen that a scenario can see and use: what a person would see of it, and what they could do.
/// </summary>
/// <param name="Role">What it is.</param>
/// <param name="Name">What it is called: its label, or its tooltip.</param>
/// <param name="Text">The text, of a text.</param>
/// <param name="Enabled">Whether it reacts.</param>
/// <param name="Rect">Where it is in the window.</param>
/// <param name="Click">What a click does, or <see langword="null"/> when it cannot be clicked.</param>
/// <param name="Fill">What typing into it does (the text, and whether to press Enter), or <see langword="null"/>.</param>
internal sealed record Node(NodeRole Role, string Name, string Text, bool Enabled, Rect Rect, Action? Click = null, Action<string, bool>? Fill = null)
{
    /// <summary>Determines whether the node lies inside another one.</summary>
    /// <param name="outer">The other node.</param>
    /// <returns><see langword="true"/> if it does.</returns>
    public bool IsInside(Node outer) =>
        Rect.Left >= outer.Rect.Left - 0.5 && Rect.Top >= outer.Rect.Top - 0.5
        && Rect.Right <= outer.Rect.Right + 0.5 && Rect.Bottom <= outer.Rect.Bottom + 0.5;
}

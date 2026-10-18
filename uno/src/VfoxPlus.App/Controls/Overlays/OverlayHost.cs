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

using Microsoft.UI.Xaml.Input;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Layout;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;
using Windows.Foundation;
using Windows.System;

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>
/// The root of the window: the user interface, and above it what is drawn on top of it: dialogs, menus, tooltips and the
/// snack bar. They are laid out in the window, where an element is wherever it is on the screen, so that they can be put
/// at their anchors and kept inside the window.
/// </summary>
internal sealed class OverlayHost : Grid
{
    private readonly Grid dialogLayer = new();
    private readonly Canvas menuLayer = new();
    private readonly Canvas tipLayer = new() { IsHitTestVisible = false };
    private readonly Grid snackLayer = new() { IsHitTestVisible = false };
    private readonly List<DialogView> dialogs = [];
    private readonly Stack<MenuPopup> menus = new();
    private FrameworkElement? tipAnchor;
    private Border? tip;
    private SnackbarView? snack;

    public OverlayHost(UIElement content)
    {
        Current = this;
        Children.Add(content);
        Children.Add(dialogLayer);
        Children.Add(menuLayer);
        Children.Add(tipLayer);
        Children.Add(snackLayer);
        KeyDown += OnKeyDown;
    }

    /// <summary>Gets the host of the window.</summary>
    public static OverlayHost? Current { get; private set; }

    /// <summary>Gets or sets the space at the bottom of the window that the snack bar keeps clear of, which the navigation bar takes.</summary>
    public double BottomInset { get; set; }

    /// <summary>Gets a value indicating whether a dialog is open.</summary>
    public bool HasDialog => dialogs.Count > 0;

    /// <summary>Gets a value indicating whether a menu is open.</summary>
    public bool HasMenu => menus.Count > 0;

    /// <summary>Gets the dialog that is on top, or <see langword="null"/>.</summary>
    public DialogView? TopDialog => dialogs.Count > 0 ? dialogs[^1] : null;

    /// <summary>
    /// Gets what the user can reach while a menu or a dialog is open, which hold everything else back, or
    /// <see langword="null"/> when none is.
    /// </summary>
    public FrameworkElement? ModalScope => menus.Count > 0 ? menus.Peek().Surface : dialogs.Count > 0 ? dialogs[^1].Surface : null;

    /// <summary>Opens a dialog on top of everything else.</summary>
    /// <param name="spec">What the dialog shows.</param>
    /// <returns>The dialog, which closes it.</returns>
    public DialogView ShowDialog(DialogSpec spec)
    {
        if (spec.Id.Length > 0 && dialogs.FirstOrDefault(open => open.Id == spec.Id) is { } existing)
        {
            return existing;
        }

        DialogView view = new(this, spec);
        dialogs.Add(view);
        dialogLayer.Children.Add(view);
        view.Open();
        return view;
    }

    /// <summary>Opens a menu below an element.</summary>
    /// <param name="anchor">The element.</param>
    /// <param name="items">The entries.</param>
    /// <param name="options">How the menu is placed.</param>
    /// <returns>The menu, which closes it.</returns>
    public MenuPopup ShowMenu(FrameworkElement anchor, IReadOnlyList<MenuItem> items, MenuOptions options)
    {
        MenuPopup menu = new(this, anchor, items, options);
        menus.Push(menu);
        menuLayer.Children.Add(menu.Scrim);
        menuLayer.Children.Add(menu.Surface);
        menu.Open();
        return menu;
    }

    /// <summary>Shows a message in the snack bar, in place of the one that shows.</summary>
    /// <param name="message">The message.</param>
    public void ShowSnackbar(Snack message)
    {
        snack?.Dismiss(immediately: true);
        snack = new SnackbarView(this, message);
        snackLayer.Children.Add(snack);
        snack.Open();
    }

    /// <summary>Shows a tooltip below an element.</summary>
    /// <param name="anchor">The element.</param>
    /// <param name="text">The tooltip.</param>
    public void ShowTooltip(FrameworkElement anchor, string text)
    {
        if (menus.Count > 0 || !anchor.IsLoaded)
        {
            return;
        }

        HideTooltip(null);
        AppTheme theme = Ui.Theme;
        TextBlock label = Ui.Text(text, TextRole.BodySmall, theme.InverseOnSurface);
        label.TextWrapping = TextWrapping.Wrap;
        tip = new Border
        {
            Background = theme.InverseSurface,
            CornerRadius = new CornerRadius(Corners.ExtraSmall),
            Padding = new Thickness(8, 4, 8, 4),
            MaxWidth = 280,
            Child = label,
            Opacity = 0,
        };
        tipAnchor = anchor;
        tip.Measure(new Size(280, double.PositiveInfinity));
        Size size = tip.DesiredSize;
        (double x, double y) = Placement.PlaceTooltip(ActualWidth, ActualHeight, BoundsOf(anchor), size.Width, size.Height);
        Canvas.SetLeft(tip, x);
        Canvas.SetTop(tip, y);
        tipLayer.Children.Add(tip);
        Border shown = tip;
        new Animated(MotionScheme.Expressive.FastEffects, value => shown.Opacity = value).To(1);
    }

    /// <summary>Hides the tooltip of an element, or any tooltip.</summary>
    /// <param name="anchor">The element whose tooltip goes, or <see langword="null"/> for any.</param>
    public void HideTooltip(FrameworkElement? anchor)
    {
        if (tip is null || (anchor is not null && !ReferenceEquals(anchor, tipAnchor)))
        {
            return;
        }

        tipLayer.Children.Remove(tip);
        tip = null;
        tipAnchor = null;
    }

    /// <summary>Gets the rectangle of an element in the coordinates of the window.</summary>
    /// <param name="element">The element.</param>
    /// <returns>The rectangle.</returns>
    public Box BoundsOf(FrameworkElement element)
    {
        Rect bounds = element.TransformToVisual(this).TransformBounds(new Rect(0, 0, element.ActualWidth, element.ActualHeight));
        return new Box(bounds.X, bounds.Y, bounds.Width, bounds.Height);
    }

    internal void Closed(DialogView view)
    {
        dialogs.Remove(view);
        dialogLayer.Children.Remove(view);
    }

    internal void Closed(MenuPopup menu)
    {
        if (menus.Count > 0 && ReferenceEquals(menus.Peek(), menu))
        {
            menus.Pop();
        }

        menuLayer.Children.Remove(menu.Scrim);
        menuLayer.Children.Remove(menu.Surface);
    }

    internal void Closed(SnackbarView view)
    {
        snackLayer.Children.Remove(view);
        if (ReferenceEquals(snack, view))
        {
            snack = null;
        }
    }

    private void OnKeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Escape)
        {
            return;
        }

        if (menus.Count > 0)
        {
            menus.Peek().Close();
            e.Handled = true;
        }
        else if (dialogs.Count > 0)
        {
            dialogs[^1].Close();
            e.Handled = true;
        }
    }
}

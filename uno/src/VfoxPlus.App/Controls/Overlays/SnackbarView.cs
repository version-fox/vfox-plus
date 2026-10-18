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
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>A message in the snack bar.</summary>
internal sealed class Snack
{
    /// <summary>Gets the message.</summary>
    public required string Text { get; init; }

    /// <summary>Gets the label of a button next to the message, if any.</summary>
    public string? Action { get; init; }

    /// <summary>Gets what runs when the action button is pressed.</summary>
    public Action? OnAction { get; init; }

    /// <summary>Gets a value indicating whether the snack bar stays until the user dismisses it; otherwise it goes after four seconds.</summary>
    public bool Persist { get; init; }

    /// <summary>Gets the name of the button that dismisses the snack bar, which is added if it is given.</summary>
    public string? CloseLabel { get; init; }
}

/// <summary>
/// A snack bar: a message at the bottom of the window that slides up on a spring, with an action and a button that
/// dismisses it.
/// </summary>
internal sealed class SnackbarView : Border
{
    private static readonly TimeSpan Duration = TimeSpan.FromSeconds(4);

    private readonly OverlayHost host;
    private readonly Snack message;
    private readonly TranslateTransform slide = new();
    private readonly Animated progress;
    private readonly DispatcherTimer timer = new() { Interval = Duration };
    private bool closing;
    private bool finished;

    public SnackbarView(OverlayHost host, Snack message)
    {
        this.host = host;
        this.message = message;
        AppTheme theme = Ui.Theme;
        Grid row = new() { Padding = new Thickness(16, 6, 8, 6), MinHeight = 48 };
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        TextBlock text = Ui.Text(message.Text, TextRole.BodyMedium, theme.InverseOnSurface);
        text.TextWrapping = TextWrapping.Wrap;
        text.VerticalAlignment = VerticalAlignment.Center;
        row.Children.Add(text);
        if (message.Action is { } action)
        {
            M3Button button = new() { Kind = ButtonKind.Text, Label = action, Margin = new Thickness(8, 0, 0, 0), VerticalAlignment = VerticalAlignment.Center };
            button.ContentBrush = theme.InversePrimary;
            button.Click += (_, _) =>
            {
                message.OnAction?.Invoke();
                Dismiss();
            };
            Grid.SetColumn(button, 1);
            row.Children.Add(button);
        }

        if (message.CloseLabel is { } closeLabel)
        {
            M3IconButton close = new() { Symbol = "close", Tooltip = closeLabel, Diameter = 32, IconBrush = theme.InverseOnSurface };
            close.Click += (_, _) => Dismiss();
            Grid.SetColumn(close, 2);
            row.Children.Add(close);
        }

        Background = theme.InverseSurface;
        CornerRadius = new CornerRadius(Corners.Medium);
        Child = row;
        HorizontalAlignment = HorizontalAlignment.Center;
        VerticalAlignment = VerticalAlignment.Bottom;
        RenderTransform = slide;
        Opacity = 0;
        IsHitTestVisible = true;
        AutomationProperties.SetName(this, message.Text);
        Elevation.Set(this, 3);
        progress = new Animated(MotionScheme.Expressive.DefaultSpatial, Apply);
        host.SizeChanged += OnHostSizeChanged;
        timer.Tick += (_, _) => Dismiss();
        Resize();
    }

    /// <summary>Gets the message.</summary>
    public Snack Message => message;

    /// <summary>Slides the snack bar in.</summary>
    public void Open()
    {
        progress.Snap(0);
        progress.To(1);
        if (!message.Persist)
        {
            timer.Start();
        }
    }

    /// <summary>Slides the snack bar out.</summary>
    /// <param name="immediately">Whether to remove it at once.</param>
    public void Dismiss(bool immediately = false)
    {
        timer.Stop();
        if (immediately || closing)
        {
            if (immediately)
            {
                Finish();
            }

            return;
        }

        closing = true;
        progress.To(0);
        if (!progress.IsMoving)
        {
            Finish();
        }
    }

    private void Apply(double value)
    {
        Opacity = Math.Min(1, Math.Max(0, value * 1.5));
        slide.Y = (1 - value) * (ActualHeight + host.BottomInset + 32);
        if (closing && value < 0.02)
        {
            Finish();
        }
    }

    private void Finish()
    {
        if (finished)
        {
            return;
        }

        finished = true;
        timer.Stop();
        host.SizeChanged -= OnHostSizeChanged;
        host.Closed(this);
    }

    private void OnHostSizeChanged(object sender, SizeChangedEventArgs e) => Resize();

    private void Resize()
    {
        double margin = 16;
        Width = Math.Min(480, Math.Max(0, host.ActualWidth - (2 * margin)));
        Margin = new Thickness(margin, margin, margin, margin + host.BottomInset);
    }
}

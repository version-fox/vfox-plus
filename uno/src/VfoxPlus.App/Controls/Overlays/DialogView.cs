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
using Microsoft.UI.Xaml.Automation.Peers;
using Microsoft.UI.Xaml.Input;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>What a dialog shows.</summary>
internal sealed class DialogSpec
{
    /// <summary>Gets the icon above the title, if any.</summary>
    public string? Symbol { get; init; }

    /// <summary>Gets a value indicating whether the icon has the colour of errors instead of the secondary colour, for destructive actions.</summary>
    public bool IconError { get; init; }

    /// <summary>Gets what identifies the dialog: opening a dialog with the id of one that is open leaves that one as it is.</summary>
    public string Id { get; init; } = string.Empty;

    /// <summary>Gets the headline.</summary>
    public required string Title { get; init; }

    /// <summary>Gets the body of the dialog, if any.</summary>
    public UIElement? Content { get; init; }

    /// <summary>Gets the buttons at the bottom, from the left.</summary>
    public IReadOnlyList<UIElement> Actions { get; init; } = [];

    /// <summary>Gets the width the dialog gets at most, in dp.</summary>
    public double MaxWidth { get; init; } = 560;

    /// <summary>Gets a value indicating whether the dialog is as high as the window allows, with the content taking the space between the title and the actions: for content that scrolls.</summary>
    public bool Tall { get; init; }
}

/// <summary>
/// A Material 3 dialog: a modal surface with an icon, a title, content and actions. It scales in on a spring and fades
/// in; a press outside of it, or Escape, closes it.
/// </summary>
internal sealed class DialogView : Grid
{
    private readonly OverlayHost host;
    private readonly DialogSpec spec;
    private readonly Border scrim;
    private readonly Border surface;
    private readonly ScaleTransform scale = new();
    private readonly Animated progress;
    private Control? previousFocus;
    private bool closing;
    private bool finished;

    public DialogView(OverlayHost host, DialogSpec spec)
    {
        this.host = host;
        this.spec = spec;
        AppTheme theme = Ui.Theme;
        scrim = new Border { Background = theme.Scrim32, Opacity = 0 };
        scrim.PointerPressed += (_, e) =>
        {
            e.Handled = true;
            Close();
        };
        surface = new Border
        {
            Background = theme.SurfaceContainerHigh,
            CornerRadius = new CornerRadius(Corners.ExtraLarge),
            Padding = new Thickness(Gaps.Xl),
            HorizontalAlignment = HorizontalAlignment.Center,
            VerticalAlignment = VerticalAlignment.Center,
            RenderTransform = scale,
            RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5),
            Opacity = 0,
            Child = Build(spec),
        };
        surface.PointerPressed += (_, e) => e.Handled = true;
        AutomationProperties.SetName(surface, spec.Title);
        Elevation.Set(surface, 3);
        Children.Add(scrim);
        Children.Add(surface);
        progress = new Animated(MotionScheme.Expressive.DefaultSpatial, Apply);
        host.SizeChanged += OnHostSizeChanged;
        Resize();
    }

    /// <summary>Gets the title of the dialog.</summary>
    public string Title => spec.Title;

    /// <summary>Gets what identifies the dialog.</summary>
    public string Id => spec.Id;

    /// <summary>Gets the dialog itself, which the tests look into.</summary>
    public FrameworkElement Surface => surface;

    /// <summary>Raised when the dialog has closed.</summary>
    public event EventHandler? Closed;

    /// <summary>Opens the dialog: it scales in, and takes the focus.</summary>
    public void Open()
    {
        previousFocus = FocusManager.GetFocusedElement(host.XamlRoot!) as Control;
        progress.Snap(0);
        progress.To(1);
        surface.Loaded += (_, _) => FocusFirst();
        FocusFirst();
    }

    /// <summary>Closes the dialog: it scales out, and the focus goes back where it was.</summary>
    public void Close()
    {
        if (closing)
        {
            return;
        }

        closing = true;
        progress.To(0);
        if (!progress.IsMoving)
        {
            Finish();
        }
    }

    // The icon and the title, the content under them, which takes the space that is left when the dialog is tall, and
    // the actions at the bottom.
    private static Grid Build(DialogSpec spec)
    {
        AppTheme theme = Ui.Theme;
        Grid layout = new();
        layout.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        layout.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        layout.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        StackPanel heading = new() { Orientation = Orientation.Vertical };
        if (spec.Symbol is { } symbol)
        {
            M3Icon icon = new(symbol, 24, spec.IconError ? theme.Error : theme.Secondary) { HorizontalAlignment = HorizontalAlignment.Center };
            heading.Children.Add(icon);
            icon.Margin = new Thickness(0, 0, 0, Gaps.L);
        }

        TextBlock title = Ui.Text(spec.Title, TextRole.HeadlineSmall);
        title.TextWrapping = TextWrapping.Wrap;
        AutomationProperties.SetHeadingLevel(title, AutomationHeadingLevel.Level1);
        if (spec.Symbol is not null)
        {
            title.TextAlignment = TextAlignment.Center;
        }

        heading.Children.Add(title);
        layout.Children.Add(heading);
        if (spec.Content is { } content)
        {
            if (content is FrameworkElement element)
            {
                element.Margin = new Thickness(0, Gaps.L, 0, 0);
            }

            Grid.SetRow(content, 1);
            layout.Children.Add(content);
        }

        if (spec.Actions.Count > 0)
        {
            StackPanel actions = new()
            {
                Orientation = Orientation.Horizontal,
                Spacing = 8,
                HorizontalAlignment = HorizontalAlignment.Right,
                Margin = new Thickness(0, Gaps.Xl, 0, 0),
            };
            foreach (UIElement action in spec.Actions)
            {
                actions.Children.Add(action);
            }

            Grid.SetRow(actions, 2);
            layout.Children.Add(actions);
        }

        return layout;
    }

    private void Apply(double value)
    {
        double effects = Math.Min(1, Math.Max(0, value));
        scrim.Opacity = effects;
        surface.Opacity = effects;
        double factor = 0.8 + (0.2 * value);
        scale.ScaleX = factor;
        scale.ScaleY = factor;
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
        host.SizeChanged -= OnHostSizeChanged;
        host.Closed(this);
        previousFocus?.Focus(FocusState.Programmatic);
        Closed?.Invoke(this, EventArgs.Empty);
    }

    private void OnHostSizeChanged(object sender, SizeChangedEventArgs e) => Resize();

    private void Resize()
    {
        surface.Width = Math.Min(spec.MaxWidth, Math.Max(280, host.ActualWidth - 48));
        surface.MinWidth = Math.Min(surface.Width, 280);
        if (spec.Tall)
        {
            surface.Height = Math.Max(0, host.ActualHeight - 48);
        }

        surface.MaxHeight = Math.Max(0, host.ActualHeight - 48);
    }

    private void FocusFirst()
    {
        if (FocusManager.FindFirstFocusableElement(surface) is Control first)
        {
            first.Focus(FocusState.Programmatic);
        }
    }
}

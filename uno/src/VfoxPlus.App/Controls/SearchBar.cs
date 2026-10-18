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

namespace VfoxPlus.App.Controls;

/// <summary>
/// A Material 3 search bar: a pill with a search icon, the text and a button that clears it.
/// </summary>
internal sealed class M3SearchBar : UserControl
{
    private readonly TextBox input = new();
    private readonly TextBlock hint;
    private readonly M3IconButton clear = new() { Symbol = "close", Visibility = Visibility.Collapsed };
    private readonly Border outline = new() { CornerRadius = new CornerRadius(28), BorderThickness = new Thickness(2), IsHitTestVisible = false, Opacity = 0 };
    private readonly Animated focus;
    private bool silent;

    public M3SearchBar()
    {
        AppTheme theme = Ui.Theme;
        input.Style = (Style)Application.Current.Resources["BareTextBox"];
        input.FontFamily = Fonts.Sans;
        input.FontSize = 16;
        input.Foreground = theme.OnSurface;
        input.VerticalAlignment = VerticalAlignment.Center;
        input.SelectionHighlightColor = new SolidColorBrush(AppTheme.ToColor(theme.Scheme.Primary, 0.3));
        hint = Ui.Text(string.Empty, TextRole.BodyLarge, theme.OnSurfaceVariant);
        hint.IsHitTestVisible = false;
        hint.VerticalAlignment = VerticalAlignment.Center;
        hint.Opacity = 0.7;
        clear.Click += (_, _) => Fill(string.Empty);
        outline.BorderBrush = theme.Primary;

        Border container = new() { Background = theme.SurfaceContainerHigh, CornerRadius = new CornerRadius(28) };
        Grid text = new();
        text.Children.Add(hint);
        text.Children.Add(input);
        M3Icon search = new("search", 24, theme.OnSurfaceVariant) { Margin = new Thickness(16, 0, 16, 0), VerticalAlignment = VerticalAlignment.Center };
        Grid row = new() { Height = 56, MaxWidth = 720 };
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        row.Children.Add(container);
        Grid.SetColumnSpan(container, 3);
        row.Children.Add(search);
        Grid.SetColumn(text, 1);
        row.Children.Add(text);
        Grid.SetColumn(clear, 2);
        clear.Margin = new Thickness(0, 0, 4, 0);
        row.Children.Add(clear);
        row.Children.Add(outline);
        Grid.SetColumnSpan(outline, 3);
        row.PointerPressed += (_, e) =>
        {
            if (!ReferenceEquals(e.OriginalSource, input))
            {
                input.Focus(FocusState.Pointer);
            }
        };
        Content = row;
        IsTabStop = false;
        focus = new Animated(MotionScheme.Expressive.FastEffects, value => outline.Opacity = value);
        input.TextChanged += (_, _) =>
        {
            Refresh();
            if (!silent)
            {
                TextEdited?.Invoke(this, input.Text);
            }
        };
        input.GotFocus += (_, _) => focus.To(1);
        input.LostFocus += (_, _) => focus.To(0);
    }

    /// <summary>Raised with the text whenever the user changes it.</summary>
    public event EventHandler<string>? TextEdited;

    /// <summary>Gets or sets the text. Setting it is not a change of the user: <see cref="TextEdited"/> does not hear of it.</summary>
    public string Text
    {
        get => input.Text;
        set
        {
            if (input.Text == value)
            {
                return;
            }

            silent = true;
            input.Text = value;
            silent = false;
            Refresh();
        }
    }

    /// <summary>Gets the name of the bar: its hint.</summary>
    public string AccessibleName => hint.Text;

    /// <summary>Gets or sets the text shown while the bar is empty.</summary>
    public string Hint
    {
        get => hint.Text;
        set
        {
            hint.Text = value;
            AutomationProperties.SetName(input, value);
        }
    }

    /// <summary>Gets or sets the name of the button that clears the text.</summary>
    public string ClearTooltip
    {
        get => clear.Tooltip;
        set => clear.Tooltip = value;
    }

    /// <summary>Replaces the text as the user typing it would: the tests fill the bar with it.</summary>
    /// <param name="text">The text.</param>
    public void Fill(string text) => input.Text = text;

    private void Refresh()
    {
        hint.Visibility = input.Text.Length == 0 ? Visibility.Visible : Visibility.Collapsed;
        clear.Visibility = input.Text.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
    }
}

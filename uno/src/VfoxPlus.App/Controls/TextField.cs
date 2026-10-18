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
using Microsoft.UI.Xaml.Input;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;
using Windows.System;

namespace VfoxPlus.App.Controls;

/// <summary>
/// A Material 3 filled text field, with a label that floats above the text, helper and error text, and optional
/// icons. A click anywhere in it focuses the text.
/// </summary>
internal sealed class M3TextField : UserControl
{
    private const double LabelSizeRatio = 12.0 / 16.0;

    private readonly Grid root = new();
    private readonly Grid box = new();
    private readonly Grid textArea = new();
    private readonly TextBox input = new();
    private readonly TextBlock label;
    private readonly TextBlock hint;
    private readonly TextBlock note;
    private readonly Border outline = new() { CornerRadius = new CornerRadius(Corners.Medium), IsHitTestVisible = false, Opacity = 0 };
    private readonly Border container = new() { CornerRadius = new CornerRadius(Corners.Medium) };
    private readonly ScaleTransform labelScale = new();
    private readonly TranslateTransform labelShift = new();
    private readonly Animated raise;
    private readonly Animated focus;
    private M3Icon? leading;
    private UIElement? trailing;
    private string labelText = string.Empty;
    private string helper = string.Empty;
    private string error = string.Empty;
    private int lines = 1;
    private int maxLines = 1;
    private bool dense;
    private bool monospace;
    private bool silent;
    private bool focused;
    private double fieldHeight = 56;

    public M3TextField()
    {
        AppTheme theme = Ui.Theme;
        label = Ui.Text(string.Empty, TextRole.BodyLarge, theme.OnSurfaceVariant);
        hint = Ui.Text(string.Empty, TextRole.BodyLarge, theme.OnSurfaceVariant);
        hint.Opacity = 0.7;
        hint.IsHitTestVisible = false;
        note = Ui.Text(string.Empty, TextRole.BodySmall, theme.OnSurfaceVariant);
        note.Margin = new Thickness(16, 4, 16, 0);
        note.Visibility = Visibility.Collapsed;
        note.TextWrapping = TextWrapping.Wrap;
        label.IsHitTestVisible = false;
        label.RenderTransform = new TransformGroup { Children = { labelScale, labelShift } };
        label.RenderTransformOrigin = new Windows.Foundation.Point(0, 0);
        label.HorizontalAlignment = HorizontalAlignment.Left;
        label.VerticalAlignment = VerticalAlignment.Top;

        input.Style = (Style)Application.Current.Resources["BareTextBox"];
        input.FontFamily = Fonts.Sans;
        input.FontSize = 16;
        input.Foreground = theme.OnSurface;
        input.SelectionHighlightColor = new SolidColorBrush(AppTheme.ToColor(theme.Scheme.Primary, 0.3));
        input.VerticalAlignment = VerticalAlignment.Center;
        input.TextChanged += OnTextChanged;
        input.KeyDown += OnKeyDown;
        input.GotFocus += (_, _) => SetFocused(true);
        input.LostFocus += (_, _) =>
        {
            SetFocused(false);
            Blurred?.Invoke(this, Text);
        };

        textArea.Children.Add(hint);
        textArea.Children.Add(input);
        textArea.Children.Add(label);
        container.Background = theme.SurfaceContainerHighest;
        Grid.SetColumnSpan(container, 3);
        Grid.SetColumnSpan(outline, 3);
        box.Children.Add(container);
        box.Children.Add(outline);
        box.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        box.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        box.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        Grid.SetColumn(textArea, 1);
        box.Children.Add(textArea);
        box.PointerPressed += (_, e) =>
        {
            if (!ReferenceEquals(e.OriginalSource, input))
            {
                input.Focus(FocusState.Pointer);
            }
        };

        root.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        root.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        root.Children.Add(box);
        Grid.SetRow(note, 1);
        root.Children.Add(note);
        Content = root;
        IsTabStop = false;
        raise = new Animated(MotionScheme.Expressive.FastSpatial, ApplyRaise);
        focus = new Animated(MotionScheme.Expressive.FastEffects, _ => ApplyOutline());
        IsEnabledChanged += (_, _) => UpdateColors();
        Relayout();
        ApplyRaise(0);
    }

    /// <summary>Raised with the text whenever the user changes it.</summary>
    public event EventHandler<string>? TextEdited;

    /// <summary>Raised with the text when the user presses Enter in a field of one line.</summary>
    public event EventHandler<string>? Submitted;

    /// <summary>Raised with the text when the field loses the focus.</summary>
    public event EventHandler<string>? Blurred;

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
            UpdateRaise(animate: false);
        }
    }

    /// <summary>Gets the name of the field: its label, or its hint when it has none.</summary>
    public string AccessibleName => labelText.Length > 0 ? labelText : hint.Text;

    /// <summary>Gets or sets the label that floats above the text once the field has the focus or a text.</summary>
    public string Label
    {
        get => labelText;
        set
        {
            labelText = value;
            label.Text = value;
            label.Visibility = value.Length == 0 || dense ? Visibility.Collapsed : Visibility.Visible;
            AutomationProperties.SetName(input, value.Length > 0 ? value : hint.Text);
            Relayout();
            UpdateRaise(animate: false);
        }
    }

    /// <summary>Gets or sets the text shown in an empty field that has no label, or that has the focus.</summary>
    public string Hint
    {
        get => hint.Text;
        set
        {
            hint.Text = value;
            AutomationProperties.SetName(input, labelText.Length > 0 ? labelText : value);
            UpdateRaise(animate: false);
        }
    }

    /// <summary>Gets or sets the text below the field.</summary>
    public string Helper
    {
        get => helper;
        set
        {
            helper = value;
            UpdateNote();
        }
    }

    /// <summary>Gets or sets the text below the field in place of the helper, which makes the field the colour of errors.</summary>
    public string Error
    {
        get => error;
        set
        {
            error = value;
            UpdateNote();
            UpdateColors();
            ApplyOutline();
        }
    }

    /// <summary>Gets or sets the icon before the text, if any.</summary>
    public string? LeadingSymbol
    {
        get => leading?.Symbol;
        set
        {
            if (value is null)
            {
                if (leading is not null)
                {
                    box.Children.Remove(leading);
                    leading = null;
                }
            }
            else if (leading is null)
            {
                leading = new M3Icon(value, 24, Ui.Theme.OnSurfaceVariant) { Margin = new Thickness(16, 0, 12, 0), VerticalAlignment = VerticalAlignment.Center };
                box.Children.Add(leading);
            }
            else
            {
                leading.Symbol = value;
            }

            Relayout();
        }
    }

    /// <summary>Gets or sets an element at the end of the field, such as an icon button.</summary>
    public UIElement? Trailing
    {
        get => trailing;
        set
        {
            if (trailing is not null)
            {
                box.Children.Remove(trailing);
            }

            trailing = value;
            if (value is FrameworkElement element)
            {
                element.VerticalAlignment = VerticalAlignment.Center;
                element.Margin = new Thickness(0, 0, 4, 0);
                Grid.SetColumn(element, 2);
                box.Children.Add(element);
            }

            Relayout();
        }
    }

    /// <summary>Gets or sets the number of lines the field shows at least; 1 for a field with a single line.</summary>
    public int Lines
    {
        get => lines;
        set
        {
            lines = Math.Max(1, value);
            maxLines = Math.Max(maxLines, lines);
            Relayout();
        }
    }

    /// <summary>Gets or sets the number of lines the field grows to before it scrolls.</summary>
    public int MaxLines
    {
        get => maxLines;
        set
        {
            maxLines = Math.Max(lines, value);
            Relayout();
        }
    }

    /// <summary>Gets or sets a value indicating whether the text is in the monospaced font.</summary>
    public bool Monospace
    {
        get => monospace;
        set
        {
            monospace = value;
            Relayout();
        }
    }

    /// <summary>Gets or sets a value indicating whether the field is lower, without a floating label.</summary>
    public bool Dense
    {
        get => dense;
        set
        {
            dense = value;
            label.Visibility = labelText.Length == 0 || dense ? Visibility.Collapsed : Visibility.Visible;
            Relayout();
        }
    }

    /// <summary>Gets a value indicating whether the text has the keyboard focus.</summary>
    public bool IsFocused => focused;

    /// <summary>Gives the text the keyboard focus.</summary>
    public void FocusField() => input.Focus(FocusState.Programmatic);

    /// <summary>Replaces the text as the user typing it would: the tests fill fields with it.</summary>
    /// <param name="text">The text.</param>
    /// <param name="submit">Whether to press Enter afterwards.</param>
    public void Fill(string text, bool submit)
    {
        input.Text = text;
        if (submit && lines == 1)
        {
            Submitted?.Invoke(this, input.Text);
        }
    }

    private void SetFocused(bool value)
    {
        focused = value;
        focus.To(value ? 1 : 0);
        UpdateRaise(animate: true);
        UpdateColors();
        ApplyOutline();
    }

    private void OnTextChanged(object sender, TextChangedEventArgs e)
    {
        UpdateRaise(animate: true);
        if (!silent)
        {
            TextEdited?.Invoke(this, input.Text);
        }
    }

    private void OnKeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key == VirtualKey.Enter && lines == 1 && maxLines == 1)
        {
            e.Handled = true;
            Submitted?.Invoke(this, input.Text);
        }
    }

    private bool ShowsLabel => labelText.Length > 0 && !dense;

    private void UpdateRaise(bool animate)
    {
        bool raised = (focused || input.Text.Length > 0) && ShowsLabel;
        if (animate)
        {
            raise.To(raised ? 1 : 0);
        }
        else
        {
            raise.Snap(raised ? 1 : 0);
        }

        hint.Visibility = input.Text.Length == 0 && (!ShowsLabel || focused) && hint.Text.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
    }

    private void UpdateNote()
    {
        string text = error.Length > 0 ? error : helper;
        note.Text = text;
        note.Visibility = text.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        note.Foreground = error.Length > 0 ? Ui.Theme.Error : Ui.Theme.OnSurfaceVariant;
    }

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        bool disabled = !IsEnabled;
        container.Background = disabled ? theme.OnSurface04 : theme.SurfaceContainerHighest;
        input.Foreground = disabled ? theme.OnSurface38 : theme.OnSurface;
        label.Foreground = disabled ? theme.OnSurface38 : error.Length > 0 ? theme.Error : focused ? theme.Primary : theme.OnSurfaceVariant;
        hint.Foreground = disabled ? theme.OnSurface38 : theme.OnSurfaceVariant;
    }

    private void ApplyOutline()
    {
        AppTheme theme = Ui.Theme;
        bool hasError = error.Length > 0;
        outline.BorderBrush = hasError ? theme.Error : theme.Primary;
        outline.BorderThickness = new Thickness(hasError && !focused ? 1 : 2);
        outline.Opacity = hasError ? 1 : focus.Value;
    }

    // The label is as big as the text at rest and 12/16 of it when it has floated up.
    private void ApplyRaise(double progress)
    {
        double scale = 1 - ((1 - LabelSizeRatio) * progress);
        labelScale.ScaleX = scale;
        labelScale.ScaleY = scale;
        double rest = (fieldHeight - 24) / 2;
        labelShift.Y = rest + ((7 - rest) * progress);
    }

    private void Relayout()
    {
        TextStyleFor(input);
        double lineHeight = monospace || dense ? 20 : 24;
        bool single = lines == 1 && maxLines == 1;
        double labelHeight = ShowsLabel ? 16 : 0;
        double height = dense
            ? Math.Max(48, (12 * 2) + (lineHeight * lines))
            : single ? 56 : Math.Max(56, 8 + labelHeight + (lineHeight * lines) + 8);
        fieldHeight = height;
        box.MinHeight = height;
        box.Height = single ? height : double.NaN;
        input.TextWrapping = single ? TextWrapping.NoWrap : TextWrapping.Wrap;
        input.AcceptsReturn = !single;
        input.MinHeight = single ? 0 : lineHeight * lines;
        input.MaxHeight = single ? double.PositiveInfinity : lineHeight * maxLines;
        textArea.Margin = new Thickness(leading is null ? 16 : 0, 0, trailing is null ? 16 : 0, 0);
        if (ShowsLabel)
        {
            input.VerticalAlignment = single ? VerticalAlignment.Bottom : VerticalAlignment.Top;
            input.Margin = new Thickness(0, single ? 0 : 8 + labelHeight, 0, single ? 12 : 8);
            hint.VerticalAlignment = input.VerticalAlignment;
            hint.Margin = input.Margin;
        }
        else
        {
            input.VerticalAlignment = single ? VerticalAlignment.Center : VerticalAlignment.Top;
            input.Margin = new Thickness(0, single ? 0 : 12, 0, single ? 0 : 12);
            hint.VerticalAlignment = input.VerticalAlignment;
            hint.Margin = input.Margin;
        }

        label.Margin = new Thickness(0);
        ApplyRaise(raise?.Value ?? 0);
        UpdateColors();
    }

    private void TextStyleFor(TextBox target)
    {
        target.FontFamily = monospace ? Fonts.Mono : Fonts.Sans;
        target.FontSize = monospace || dense ? 14 : 16;
        hint.FontFamily = target.FontFamily;
        hint.FontSize = target.FontSize;
        hint.LineHeight = 0;
    }
}

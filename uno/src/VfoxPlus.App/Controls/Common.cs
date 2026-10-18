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

using VfoxPlus.App.Services;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>The pieces that several pages show: the states of a view that loads, and a few small elements.</summary>
internal static class Common
{
    /// <summary>The links that the user interface opens.</summary>
    public const string VfoxInstall = "https://vfox.dev/guides/quick-start.html";

    /// <summary>The documentation of vfox.</summary>
    public const string VfoxDocs = "https://vfox.dev/";

    /// <summary>The source code of vfox+.</summary>
    public const string Repository = "https://github.com/version-fox/vfox-plus";

    /// <summary>Explains an error in the middle of the space, with a button that tries again.</summary>
    /// <param name="title">What failed.</param>
    /// <param name="error">The error.</param>
    /// <param name="retry">What the button runs, or <see langword="null"/> for no button.</param>
    /// <returns>The view.</returns>
    public static UIElement ErrorView(string title, Exception error, Action? retry)
    {
        List<UIElement> actions = [];
        if (retry is not null)
        {
            M3Button button = new() { Kind = ButtonKind.Tonal, Symbol = "retry", Label = Env.Current.T("actionRetry") };
            button.Click += (_, _) => retry();
            actions.Add(button);
        }

        return new StatusMessage("error", title, Presentation.Describe(error), isError: true, shape: MaterialShapes.SoftBurst, actions: actions);
    }

    /// <summary>Shows that something is loading, in the middle of the space.</summary>
    /// <param name="label">A line of text under the indicator, or an empty text.</param>
    /// <returns>The view.</returns>
    public static UIElement LoadingView(string label = "")
    {
        StackPanel column = new() { Orientation = Orientation.Vertical, Padding = new Thickness(32), HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center };
        column.Children.Add(new LoadingIndicator(64, contained: true, name: label.Length > 0 ? label : Env.Current.T("loading")) { HorizontalAlignment = HorizontalAlignment.Center });
        if (label.Length > 0)
        {
            TextBlock text = Ui.Text(label, TextRole.BodyMedium, Ui.Theme.OnSurfaceVariant);
            text.TextAlignment = TextAlignment.Center;
            text.Margin = new Thickness(0, 16, 0, 0);
            column.Children.Add(text);
        }

        return column;
    }

    /// <summary>The icon button that loads a resource again.</summary>
    /// <param name="refresh">What it runs.</param>
    /// <returns>The button.</returns>
    public static M3IconButton RefreshButton(Action refresh)
    {
        M3IconButton button = new() { Symbol = "refresh", Tooltip = Env.Current.T("actionRefresh") };
        button.Click += (_, _) => refresh();
        return button;
    }

    /// <summary>A text block that the user can select and copy from, in the monospaced font: a command, the output of a task.</summary>
    /// <param name="text">The text.</param>
    /// <param name="style">The style.</param>
    /// <param name="foreground">The colour.</param>
    /// <returns>The text box.</returns>
    public static TextBox SelectableText(string text, TextStyle style, Brush foreground)
    {
        TextBox box = new()
        {
            Style = (Style)Application.Current.Resources["BareTextBox"],
            IsReadOnly = true,
            Text = text,
            FontFamily = Fonts.Mono,
            FontSize = style.Size,
            Foreground = foreground,
            TextWrapping = TextWrapping.Wrap,
            AcceptsReturn = true,
            SelectionHighlightColor = new SolidColorBrush(AppTheme.ToColor(Ui.Theme.Scheme.Primary, 0.3)),
        };
        return box;
    }
}

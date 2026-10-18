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

using System.Text.RegularExpressions;
using VfoxPlus.App.Services;
using VfoxPlus.App.Theming;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Design.Tokens;
using VfoxPlus.Platform;

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>The dialogs that the pages open: a question to confirm, the project a version is used in, a plugin from a source.</summary>
internal static partial class Dialogs
{
    private static Env Env => Env.Current;

    /// <summary>Asks the user to confirm an action.</summary>
    /// <param name="id">Identifies the question: it is not asked twice at once.</param>
    /// <param name="title">The question.</param>
    /// <param name="body">What it means.</param>
    /// <param name="confirmLabel">The label of the button that confirms.</param>
    /// <param name="symbol">The icon above the title.</param>
    /// <param name="destructive">Whether the action destroys something, which colours the confirmation like an error.</param>
    /// <param name="onConfirm">What runs when the user confirms.</param>
    public static void Confirm(string id, string title, string body, string confirmLabel, string symbol, bool destructive, Action onConfirm)
    {
        DialogView? dialog = null;
        M3Button cancel = new() { Kind = ButtonKind.Text, Label = Env.T("actionCancel") };
        M3Button confirm = new() { Kind = ButtonKind.Filled, Label = confirmLabel, Danger = destructive };
        cancel.Click += (_, _) => dialog!.Close();
        confirm.Click += (_, _) =>
        {
            dialog!.Close();
            onConfirm();
        };
        TextBlock text = Ui.Text(body, TextRole.BodyMedium, Ui.Theme.OnSurfaceVariant);
        text.TextWrapping = TextWrapping.Wrap;
        dialog = Env.Overlays!.ShowDialog(new DialogSpec
        {
            Id = id,
            Symbol = symbol,
            IconError = destructive,
            Title = title,
            Content = text,
            Actions = [cancel, confirm],
        });
    }

    /// <summary>Asks for the project that a version is used in, and uses it there.</summary>
    /// <param name="name">The plugin.</param>
    /// <param name="version">The version.</param>
    public static void AskProjectDirectory(string name, string version)
    {
        DialogView? dialog = null;
        M3TextField field = new() { Label = Env.T("useProjectDirectory") };
        void Submit()
        {
            string path = field.Text.Trim();
            if (path.Length == 0 || !Directory.Exists(path))
            {
                field.Error = Env.T("useProjectInvalid");
                return;
            }

            dialog!.Close();
            Env.Run(() => VfoxCommands.Use(name, version, Core.UseScope.Project, path));
        }

        field.Submitted += (_, _) => Submit();
        field.TextEdited += (_, _) => field.Error = string.Empty;
        M3IconButton browse = BrowseButton();
        browse.Click += (_, _) => Env.Choose(
            new ChooseOptions(Env.T("useProjectDirectory"), Home(), Directory: true),
            path =>
            {
                field.Text = path;
                field.Error = string.Empty;
            });
        TextBlock body = Ui.Text(Env.T("useProjectBody"), TextRole.BodyMedium, Ui.Theme.OnSurfaceVariant);
        body.TextWrapping = TextWrapping.Wrap;
        StackPanel content = new() { Orientation = Orientation.Vertical };
        content.Children.Add(body);
        content.Children.Add(FieldWithBrowse(field, browse, topGap: Gaps.L));
        dialog = Env.Overlays!.ShowDialog(new DialogSpec
        {
            Id = "project-directory",
            Symbol = "projectScope",
            Title = Env.T("useProjectTitle", ("name", name), ("version", version)),
            Content = content,
            Actions = [Cancel(() => dialog!.Close()), Primary(Env.T("useProjectConfirm"), Submit)],
        });
    }

    /// <summary>Asks for the source of a plugin that is not in the registry, and adds it.</summary>
    public static void AskCustomPlugin()
    {
        DialogView? dialog = null;
        M3TextField source = new() { Label = Env.T("pluginsCustomSource"), Helper = Env.T("pluginsCustomSourceHelp") };
        M3TextField alias = new() { Label = Env.T("pluginsCustomAlias"), Helper = Env.T("pluginsCustomAliasHelp") };
        void Submit()
        {
            string address = source.Text.Trim();
            string name = alias.Text.Trim();
            source.Error = address.Length == 0 ? Env.T("pluginsCustomRequired") : string.Empty;
            alias.Error = name.Length > 0 && !PluginName().IsMatch(name) ? Env.T("pluginsCustomInvalidName") : string.Empty;
            if (source.Error.Length > 0 || alias.Error.Length > 0)
            {
                return;
            }

            dialog!.Close();
            _ = Task.Run(() =>
            {
                try
                {
                    Env.Model.AddPluginFromSource(address, name);
                }
                catch (Exception error) when (error is VfoxUsageException or IOException or UnauthorizedAccessException)
                {
                    Env.Dispatch.Post(() => Env.Notify(error.Message));
                }
            });
        }

        source.Submitted += (_, _) => Submit();
        alias.Submitted += (_, _) => Submit();
        source.TextEdited += (_, _) => source.Error = string.Empty;
        alias.TextEdited += (_, _) => alias.Error = string.Empty;
        M3IconButton browse = BrowseButton();
        browse.Click += (_, _) => Env.Choose(
            new ChooseOptions(Env.T("pluginsCustomSource"), Home(), FilterLabel: Env.T("pluginsCustomFileType"), Extensions: ["zip", "lua"]),
            path =>
            {
                source.Text = path;
                source.Error = string.Empty;
            });
        StackPanel content = new() { Orientation = Orientation.Vertical };
        content.Children.Add(FieldWithBrowse(source, browse, topGap: 0));
        alias.Margin = new Thickness(0, Gaps.L, 0, 0);
        content.Children.Add(alias);
        dialog = Env.Overlays!.ShowDialog(new DialogSpec
        {
            Id = "custom-plugin",
            Symbol = "plugins",
            Title = Env.T("pluginsCustomTitle"),
            Content = content,
            Actions = [Cancel(() => dialog!.Close()), Primary(Env.T("pluginsCustomConfirm"), Submit)],
        });
    }

    /// <summary>Shows the licenses of everything the application is built from.</summary>
    public static void ShowLicenses()
    {
        string path = Path.Combine(Env.Assets, "licenses.txt");
        string[] lines;
        try
        {
            lines = File.ReadAllText(path).Replace("\r", string.Empty, StringComparison.Ordinal).Split('\n');
        }
        catch (IOException error)
        {
            Env.Notify(error.Message);
            return;
        }

        // The licenses are long: the list shows them in chunks of lines, made only for the part that shows.
        const int chunk = 40;
        List<string> chunks = [];
        for (int start = 0; start < lines.Length; start += chunk)
        {
            chunks.Add(string.Join('\n', lines.Skip(start).Take(chunk)));
        }

        DialogView? dialog = null;
        VirtualRows<string> rows = new(
            () => new TextBlock
            {
                FontFamily = Fonts.Mono,
                FontSize = 12,
                LineHeight = 18,
                Foreground = Ui.Theme.OnSurfaceVariant,
                TextWrapping = TextWrapping.Wrap,
                IsTextSelectionEnabled = true,
            },
            (row, text, _) => ((TextBlock)row).Text = text,
            text => text,
            spacing: 0);
        ScrollViewer scroll = new() { Content = rows.Repeater, VerticalScrollBarVisibility = ScrollBarVisibility.Auto, HorizontalScrollBarVisibility = ScrollBarVisibility.Disabled };
        rows.SetItems(chunks);
        M3Button close = new() { Kind = ButtonKind.Text, Label = Env.T("actionClose") };
        close.Click += (_, _) => dialog!.Close();
        dialog = Env.Overlays!.ShowDialog(new DialogSpec
        {
            Id = "licenses",
            Title = Env.T("aboutLicenses"),
            MaxWidth = 760,
            Tall = true,
            Content = scroll,
            Actions = [close],
        });
    }

    [GeneratedRegex("^[A-Za-z0-9][A-Za-z0-9._+-]*$")]
    private static partial Regex PluginName();

    private static string Home() => Env.Environment.TryGetValue("HOME", out string? home) ? home : string.Empty;

    private static M3IconButton BrowseButton() => new() { Kind = IconButtonKind.Tonal, Symbol = "folder", Tooltip = Env.T("actionBrowse"), VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(0, Gaps.S, 0, 0) };

    private static Grid FieldWithBrowse(M3TextField field, M3IconButton browse, double topGap)
    {
        Grid row = new() { ColumnSpacing = Gaps.S, Margin = new Thickness(0, topGap, 0, 0) };
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        row.Children.Add(field);
        Grid.SetColumn(browse, 1);
        row.Children.Add(browse);
        return row;
    }

    private static M3Button Cancel(Action close)
    {
        M3Button button = new() { Kind = ButtonKind.Text, Label = Env.T("actionCancel") };
        button.Click += (_, _) => close();
        return button;
    }

    private static M3Button Primary(string label, Action run)
    {
        M3Button button = new() { Kind = ButtonKind.Filled, Label = label };
        button.Click += (_, _) => run();
        return button;
    }
}

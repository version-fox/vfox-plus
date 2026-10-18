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

using VfoxPlus.App.Controls;
using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Services;
using VfoxPlus.App.Theming;
using VfoxPlus.Core;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>An installed version, with the buttons that use it, copy its path and uninstall it.</summary>
internal sealed class InstalledVersionRow : ListItem
{
    private readonly Env env = Env.Current;
    private readonly M3Icon marker = new("versions", 24);
    private readonly StackPanel actions = new() { Orientation = Orientation.Horizontal, Spacing = Gaps.Xs, VerticalAlignment = VerticalAlignment.Center };
    private readonly Tag inUse = new(string.Empty, "current", emphasized: true);
    private readonly M3SplitButton use = new() { Symbol = "check" };
    private readonly M3IconButton more = new() { Symbol = "more" };
    private readonly LoadingIndicator busy = new(40);
    private InstalledSdk sdk = new(string.Empty, []);
    private string version = string.Empty;
    private bool current;

    public InstalledVersionRow()
    {
        WrapTrailing = true;
        Leading = marker;
        Trailing = actions;
        use.Click += (_, _) => UseGlobally();
        Menus.Attach(more, BuildMenu);
    }

    /// <summary>Gets the version that the row shows.</summary>
    public string Version => version;

    /// <summary>Shows a version of a plugin.</summary>
    /// <param name="plugin">The plugin.</param>
    /// <param name="shown">The version.</param>
    /// <param name="where">Where the row is in its list.</param>
    public void Bind(InstalledSdk plugin, string shown, ListPosition where)
    {
        AppTheme theme = Ui.Theme;
        sdk = plugin;
        version = shown;
        Position = where;
        current = plugin.HasCurrent && plugin.Current == shown;
        bool isBusy = env.Model.Tasks.ActiveTask(null, plugin.Name, shown) is not null;
        Title = shown;
        ItemName = shown;
        TitleStyle = current ? Typography.Emphasized(TextRole.BodyLarge) : Typography.Of(TextRole.BodyLarge);
        marker.Symbol = current ? "current" : "versions";
        marker.FillTo(current ? 1 : 0, animate: false);
        marker.Foreground = current ? theme.Primary : theme.OnSurfaceVariant;

        // What is not shown is not in the tree: an indicator that is in the tree is drawn all the time.
        actions.Children.Clear();
        if (isBusy)
        {
            actions.Children.Add(busy);
            return;
        }

        if (current)
        {
            inUse.Text = env.T("versionInUse");
            actions.Children.Add(inUse);
        }
        else
        {
            use.Label = env.T("versionUse");
            use.MenuTooltip = env.T("versionUseOptions");
            use.Items =
            [
                new MenuItem(env.T("versionUseGlobally"), UseGlobally, Symbol: "globalScope"),
                new MenuItem(env.T("versionUseInProject"), () => Dialogs.AskProjectDirectory(sdk.Name, version), Symbol: "projectScope"),
            ];
            actions.Children.Add(use);
        }

        more.Tooltip = env.T("versionMoreActions", ("version", shown));
        actions.Children.Add(more);
    }

    private void UseGlobally() => env.Run(() => VfoxCommands.Use(sdk.Name, version, UseScope.Global));

    private List<MenuItem> BuildMenu()
    {
        string name = sdk.Name;
        string shown = version;
        List<MenuItem> menu = [];
        if (current)
        {
            menu.Add(new MenuItem(env.T("versionUseInProject"), () => Dialogs.AskProjectDirectory(name, shown), Symbol: "projectScope"));
            menu.Add(new MenuItem(env.T("versionUnuse"), () => env.Run(() => VfoxCommands.Unuse(name, UseScope.Global)), Symbol: "cancel"));
        }

        menu.Add(new MenuItem(env.T("versionCopyPath"), () => CopyInstallPath(name, shown), Symbol: "copy"));
        bool wasCurrent = current;
        menu.Add(new MenuItem(env.T("versionUninstall"), () => ConfirmUninstall(name, shown, wasCurrent), Symbol: "uninstall", Danger: true));
        return menu;
    }

    private void ConfirmUninstall(string name, string shown, bool wasCurrent) => Dialogs.Confirm(
        "uninstall",
        env.T("versionUninstallTitle", ("name", name), ("version", shown)),
        env.T(wasCurrent ? "versionUninstallCurrentBody" : "versionUninstallBody"),
        env.T("versionUninstall"),
        "uninstall",
        destructive: true,
        onConfirm: () => env.Run(() => VfoxCommands.Uninstall(name, shown)));

    // Puts the directory a version is installed in on the clipboard.
    private void CopyInstallPath(string name, string shown) => _ = Task.Run(async () =>
    {
        try
        {
            string? path = await env.Model.Client.InstallPathAsync(name, shown).ConfigureAwait(false);
            env.Dispatch.Post(() =>
            {
                if (path is not null)
                {
                    env.Platform.Copy(path);
                    env.Notify(env.T("actionCopied"));
                }
            });
        }
        catch (VfoxException error)
        {
            env.Dispatch.Post(() => env.Notify(Presentation.Describe(error)));
        }
    });
}

/// <summary>A version that can be installed: with its note and the tools that come with it, and the button that installs it.</summary>
internal sealed class AvailableVersionRow : ListItem
{
    private readonly Env env = Env.Current;
    private readonly TextBlock versionText = Ui.Text(string.Empty, TextRole.BodyLarge);
    private readonly Tag note = new(string.Empty, emphasized: true);
    private readonly M3WrapPanel heading = new() { Spacing = Gaps.S, CenterCross = true };
    private readonly Grid trailing = new();
    private readonly M3Button install = new() { Kind = ButtonKind.Tonal, Symbol = "install" };
    private readonly M3Button installed = new() { Kind = ButtonKind.Text, Symbol = "check", IsEnabled = false };
    private InstallProgress? progress;
    private string sdk = string.Empty;
    private string version = string.Empty;

    public AvailableVersionRow()
    {
        WrapTrailing = true;
        ContentPadding = new Thickness(Gaps.L, Gaps.S, Gaps.L, Gaps.S);
        heading.Children.Add(versionText);
        TitleContent = heading;
        trailing.Children.Add(install);
        Trailing = trailing;
        install.Click += (_, _) => env.Run(() => VfoxCommands.Install(sdk, version));
    }

    /// <summary>Gets the version that the row shows.</summary>
    public string Version => version;

    /// <summary>Shows a version of a plugin.</summary>
    /// <param name="plugin">The plugin.</param>
    /// <param name="shown">The version.</param>
    /// <param name="isInstalled">Whether it is installed.</param>
    /// <param name="where">Where the row is in its list.</param>
    public void Bind(string plugin, AvailableVersion shown, bool isInstalled, ListPosition where)
    {
        sdk = plugin;
        version = shown.Version;
        Position = where;
        ItemName = shown.Version;
        versionText.Text = shown.Version;
        if (shown.HasNote)
        {
            note.Text = shown.Note!;
            if (!heading.Children.Contains(note))
            {
                heading.Children.Add(note);
            }
        }
        else
        {
            heading.Children.Remove(note);
        }

        Subtitle = string.Join(" · ", shown.Tools.Select(tool => tool.ToString()));
        VfoxTask? task = env.Model.Tasks.ActiveTask(VfoxOperation.Install, plugin, shown.Version);
        trailing.Children.Clear();
        if (task is not null)
        {
            progress ??= new InstallProgress();
            progress.Bind(task);
            trailing.Children.Add(progress);
        }
        else if (isInstalled)
        {
            installed.Label = env.T("versionInstalled");
            trailing.Children.Add(installed);
        }
        else
        {
            install.Label = env.T("versionInstall");
            trailing.Children.Add(install);
        }
    }
}

/// <summary>How far an installation has come, with a button that cancels it.</summary>
internal sealed class InstallProgress : Grid
{
    private readonly Env env = Env.Current;
    private readonly WavyProgress bar = new();
    private readonly TextBlock status = Ui.Text(string.Empty, TextRole.LabelSmall, Ui.Theme.OnSurfaceVariant);
    private readonly M3IconButton cancel = new() { Symbol = "cancel" };
    private VfoxTask? task;

    public InstallProgress()
    {
        Width = 220;
        ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        status.Margin = new Thickness(0, Gaps.Xs, 0, 0);
        StackPanel text = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Center };
        text.Children.Add(bar);
        text.Children.Add(status);
        Children.Add(text);
        Grid.SetColumn(cancel, 1);
        Children.Add(cancel);
        cancel.Click += (_, _) =>
        {
            if (task is not null)
            {
                env.Model.Tasks.Cancel(task);
            }
        };
    }

    /// <summary>Shows a task.</summary>
    /// <param name="shown">The task that installs a version.</param>
    public void Bind(VfoxTask shown)
    {
        task = shown;
        double? value = shown.Progress;
        bar.Indeterminate = value is null;
        bar.Value = value ?? 0;
        bar.Label = Presentation.TaskTitle(shown.Command, env.T);
        status.Text = Presentation.Progress(shown, env.T);
        cancel.Tooltip = env.T("actionCancel");
    }
}

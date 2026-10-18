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
using VfoxPlus.App.Services;
using VfoxPlus.App.Theming;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>Lists what vfox did and is doing.</summary>
internal sealed class TasksPage : PageBase
{
    private readonly ScrollPage page = new();
    private readonly Border pane;
    private readonly TextBlock title;
    private readonly M3Button clear = new() { Kind = ButtonKind.Text, Symbol = "succeeded" };
    private readonly StackPanel list = new() { Orientation = Orientation.Vertical, Spacing = Gaps.Hairline, Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.Xl) };
    private readonly Dictionary<int, TaskRow> rows = [];
    private List<int> shown = [];
    private StatusMessage? empty;
    private string emptyLanguage = string.Empty;

    public TasksPage()
    {
        clear.Click += (_, _) => Env.Model.Tasks.ClearFinished();
        (Grid header, title, _) = PageParts.Header(string.Empty, string.Empty, clear);
        page.Top.Children.Add(header);
        pane = PageParts.Pane(page);
        Children.Add(pane);
        Watch(Env.Model.Tasks);
        WatchTaskProgress();
    }

    /// <summary>Gets the rows of the tasks, which the tests look into.</summary>
    public IReadOnlyList<TaskRow> Rows => [.. list.Children.OfType<TaskRow>()];

    /// <inheritdoc/>
    public override void Refresh()
    {
        PageParts.SetTitle(title, T("navTasks"));
        clear.Label = T("tasksClearFinished");
        clear.IsEnabled = Env.Model.Tasks.HasFinished;
        IReadOnlyList<VfoxTask> tasks = Env.Model.Tasks.Tasks;
        if (tasks.Count == 0)
        {
            if (empty is null || emptyLanguage != Env.L10n.Language)
            {
                empty = new StatusMessage("tasks", T("tasksEmptyTitle"), T("tasksEmptyBody"), shape: MaterialShapes.Sunny);
                emptyLanguage = Env.L10n.Language;
            }

            page.ShowMessage(empty);
            return;
        }

        for (int i = 0; i < tasks.Count; i++)
        {
            if (!rows.TryGetValue(tasks[i].Id, out TaskRow? row))
            {
                row = new TaskRow();
                rows[tasks[i].Id] = row;
            }

            row.Bind(tasks[i], tasks.Count <= 1 ? ListPosition.Single : i == 0 ? ListPosition.First : i == tasks.Count - 1 ? ListPosition.Last : ListPosition.Middle);
        }

        List<int> ids = [.. tasks.Select(task => task.Id)];
        foreach (int gone in rows.Keys.Except(ids).ToArray())
        {
            rows.Remove(gone);
        }

        if (!ids.SequenceEqual(shown))
        {
            shown = ids;
            list.Children.Clear();
            foreach (int id in ids)
            {
                list.Children.Add(rows[id]);
            }
        }

        page.ShowContent(list);
    }
}

/// <summary>A task: what it does, how far it is, and its output, which opens under it.</summary>
internal sealed class TaskRow : StackPanel
{
    private readonly Env env = Env.Current;
    private readonly ListItem item = new() { Interactive = true, TitleStyle = Typography.Emphasized(TextRole.BodyLarge) };
    private readonly Grid leadingSlot = new() { Width = 32, Height = 32 };
    private readonly TextBlock status = Ui.Text(string.Empty, TextRole.BodyMedium, Ui.Theme.OnSurfaceVariant);
    private readonly StackPanel subtitle = new() { Orientation = Orientation.Vertical };
    private readonly Grid progressHost = new() { Margin = new Thickness(0, Gaps.S, 0, 0) };
    private readonly WavyProgress progress = new() { HorizontalAlignment = HorizontalAlignment.Left };
    private readonly StackPanel actions = new() { Orientation = Orientation.Horizontal, VerticalAlignment = VerticalAlignment.Center };
    private readonly M3IconButton cancel = new() { Symbol = "cancel" };
    private readonly M3IconButton retry = new() { Symbol = "retry" };
    private readonly M3IconButton toggle = new() { Symbol = "expandMore" };
    private readonly Collapsible output = new() { Margin = new Thickness(0, Gaps.Hairline, 0, 0) };
    private readonly Border outputPanel = new() { Padding = new Thickness(Gaps.L, Gaps.S, Gaps.Xs, Gaps.M) };
    private readonly TextBox log;
    private readonly M3IconButton copy = new() { Symbol = "copy", VerticalAlignment = VerticalAlignment.Top };
    private readonly Dictionary<string, UIElement> leadingIcons = [];
    private readonly LoadingIndicator running = new(32);
    private VfoxTask? task;
    private ListPosition position;
    private bool expanded;
    private string logText = string.Empty;
    private ScrollViewer? logScroll;

    public TaskRow()
    {
        Orientation = Orientation.Vertical;
        AppTheme theme = Ui.Theme;
        item.Leading = leadingSlot;
        subtitle.Children.Add(status);
        progressHost.Children.Add(progress);
        status.TextWrapping = TextWrapping.Wrap;
        status.MaxLines = 2;
        status.TextTrimming = TextTrimming.CharacterEllipsis;
        item.SubtitleContent = subtitle;
        actions.Children.Add(toggle);
        item.Trailing = actions;
        item.Click += (_, _) => Toggle();
        toggle.Click += (_, _) => Toggle();
        cancel.Click += (_, _) =>
        {
            if (task is not null)
            {
                env.Model.Tasks.Cancel(task);
            }
        };
        retry.Click += (_, _) =>
        {
            if (task is not null)
            {
                env.Model.Tasks.Retry(task);
            }
        };

        log = Common.SelectableText(string.Empty, new TextStyle(12, 18, 400), theme.OnSurface);
        log.MaxHeight = 320;
        log.FontSize = 12;
        Grid panel = new() { ColumnSpacing = Gaps.Xs };
        panel.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        panel.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        panel.Children.Add(log);
        Grid.SetColumn(copy, 1);
        panel.Children.Add(copy);
        outputPanel.Child = panel;
        outputPanel.Background = theme.SurfaceContainerHighest;
        output.Children.Add(outputPanel);
        copy.Click += (_, _) =>
        {
            env.Platform.Copy(logText);
            env.Notify(env.T("actionCopied"));
        };
        Children.Add(item);
        Children.Add(output);
    }

    /// <summary>Gets the task that the row shows.</summary>
    public VfoxTask? Task => task;

    /// <summary>Gets the item that shows the task, which the tests choose.</summary>
    public ListItem Item => item;

    /// <summary>Gets a value indicating whether the output shows.</summary>
    public bool IsExpanded => expanded;

    /// <summary>Shows a task.</summary>
    /// <param name="shown">The task.</param>
    /// <param name="where">Where the row is in its list.</param>
    public void Bind(VfoxTask shown, ListPosition where)
    {
        AppTheme theme = Ui.Theme;
        task = shown;
        position = where;
        VfoxTaskStatus state = shown.Status;
        double? value = shown.Progress;
        item.Title = Presentation.TaskTitle(shown.Command, env.T);
        item.ItemName = item.Title;

        leadingSlot.Children.Clear();
        leadingSlot.Children.Add(Leading(state));
        string text = state switch
        {
            VfoxTaskStatus.Queued => env.T("taskStatusQueued"),
            VfoxTaskStatus.Running => value is null ? env.T("taskStatusRunning") : env.T("taskDownloading", ("percent", (int)Math.Round(value.Value * 100))),
            VfoxTaskStatus.Succeeded => env.T("taskStatusSucceeded"),
            VfoxTaskStatus.Failed => env.T("taskStatusFailed") + " · " + shown.ErrorMessage,
            _ => env.T("taskStatusCancelled"),
        };
        status.Text = text;
        status.Foreground = state == VfoxTaskStatus.Failed ? theme.Error : theme.OnSurfaceVariant;

        bool showsProgress = state == VfoxTaskStatus.Running && shown.Command.Operation == VfoxOperation.Install;
        if (showsProgress)
        {
            progress.Indeterminate = value is null;
            progress.Value = value ?? 0;
            progress.Label = item.Title;
            if (!subtitle.Children.Contains(progressHost))
            {
                subtitle.Children.Add(progressHost);
            }
        }
        else
        {
            subtitle.Children.Remove(progressHost);
        }

        actions.Children.Clear();
        if (state.IsActive())
        {
            cancel.Tooltip = env.T("actionCancel");
            actions.Children.Add(cancel);
        }
        else if (state is VfoxTaskStatus.Failed or VfoxTaskStatus.Cancelled)
        {
            retry.Tooltip = env.T("actionRetry");
            actions.Children.Add(retry);
        }

        toggle.Tooltip = env.T(expanded ? "taskHideOutput" : "taskShowOutput");
        toggle.IconAngle = expanded ? 180 : 0;
        actions.Children.Add(toggle);
        copy.Tooltip = env.T("taskCopyOutput");
        ApplyPosition();
        if (expanded)
        {
            ShowLog();
        }
    }

    /// <summary>Opens or closes the output of the task.</summary>
    public void Toggle()
    {
        expanded = !expanded;
        output.SetOpen(expanded);
        toggle.Tooltip = env.T(expanded ? "taskHideOutput" : "taskShowOutput");
        toggle.IconAngle = expanded ? 180 : 0;
        ApplyPosition();
        if (expanded)
        {
            ShowLog();
        }
    }

    // The icon of the state of a task, or the indicator that it is running.
    private UIElement Leading(VfoxTaskStatus state)
    {
        AppTheme theme = Ui.Theme;
        if (state == VfoxTaskStatus.Running)
        {
            return running;
        }

        string key = state.ToString();
        if (!leadingIcons.TryGetValue(key, out UIElement? icon))
        {
            M3Icon made = state switch
            {
                VfoxTaskStatus.Queued => new M3Icon("queued", 24, theme.OnSurfaceVariant),
                VfoxTaskStatus.Succeeded => new M3Icon("succeeded", 24, theme.Primary),
                VfoxTaskStatus.Failed => new M3Icon("error", 24, theme.Error),
                _ => new M3Icon("cancel", 24, theme.OnSurfaceVariant),
            };
            if (state is VfoxTaskStatus.Succeeded or VfoxTaskStatus.Failed)
            {
                made.FillTo(1, animate: false);
            }

            made.HorizontalAlignment = HorizontalAlignment.Center;
            made.VerticalAlignment = VerticalAlignment.Center;
            icon = made;
            leadingIcons[key] = icon;
        }

        return icon;
    }

    // An open task is followed by its output, so its bottom corners are not the corners of the list.
    private void ApplyPosition()
    {
        ListPosition shown = position;
        if (expanded)
        {
            shown = position switch
            {
                ListPosition.Last => ListPosition.Middle,
                ListPosition.Single => ListPosition.First,
                _ => position,
            };
        }

        item.Position = shown;
        bool last = position is ListPosition.Last or ListPosition.Single;
        outputPanel.CornerRadius = new CornerRadius(Corners.ExtraSmall, Corners.ExtraSmall, last ? Corners.LargeIncreased : Corners.ExtraSmall, last ? Corners.LargeIncreased : Corners.ExtraSmall);
    }

    // What vfox printed, under the command, which can be copied.
    private void ShowLog()
    {
        if (task is null)
        {
            return;
        }

        IReadOnlyList<string> lines = task.Log;
        string command = "$ " + task.Command.CommandLine;
        string text = lines.Count > 0 ? command + "\n" + string.Join('\n', lines) : command + "\n" + env.T("taskNoOutput");
        if (text == logText)
        {
            return;
        }

        logText = text;
        log.Text = text;

        // The output follows the end of the log while it grows.
        env.Dispatch.Post(() =>
        {
            logScroll ??= Find<ScrollViewer>(log);
            logScroll?.ChangeView(null, logScroll.ScrollableHeight, null, disableAnimation: true);
        });
    }

    private static T? Find<T>(DependencyObject parent)
        where T : class, DependencyObject
    {
        for (int i = 0; i < VisualTreeHelper.GetChildrenCount(parent); i++)
        {
            DependencyObject child = VisualTreeHelper.GetChild(parent, i);
            if (child is T found)
            {
                return found;
            }

            if (Find<T>(child) is { } deeper)
            {
                return deeper;
            }
        }

        return null;
    }
}

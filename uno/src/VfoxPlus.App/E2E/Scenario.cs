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

using System.Diagnostics;
using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Services;
using VfoxPlus.State;

namespace VfoxPlus.App.E2E;

/// <summary>Reports that a check of a scenario failed.</summary>
internal sealed class ScenarioException : Exception
{
    public ScenarioException(string message)
        : base(message)
    {
    }
}

/// <summary>
/// What a scenario can use: it looks at the screen, and does what a person would do, through the tree of the user
/// interface. It runs on a thread of its own, and every access to the user interface goes through the thread of the
/// window.
/// </summary>
internal sealed class Scenario
{
    /// <summary>How long a scenario waits for something to show before it fails.</summary>
    public static readonly TimeSpan DefaultWait = TimeSpan.FromSeconds(30);

    private readonly Window window;

    public Scenario(Window window, Env env, OverlayHost host)
    {
        this.window = window;
        Env = env;
        Host = host;
    }

    /// <summary>Gets the environment of the application.</summary>
    public Env Env { get; }

    /// <summary>Gets the root of the window.</summary>
    public OverlayHost Host { get; }

    /// <summary>Gets the state of the application.</summary>
    public AppModel Model => Env.Model;

    /// <summary>Writes a line for the harness that runs the application, which reads the standard output.</summary>
    /// <param name="message">The line, after <c>e2e: </c>.</param>
    public static void Report(string message) => Console.Out.WriteLine("e2e: " + message);

    /// <summary>Fails the scenario unless a condition holds.</summary>
    /// <param name="condition">The condition.</param>
    /// <param name="description">What it says.</param>
    public static void Check(bool condition, string description)
    {
        if (!condition)
        {
            throw new ScenarioException("Check failed: " + description);
        }
    }

    /// <summary>Waits for the user interface to settle, as a person would.</summary>
    /// <param name="milliseconds">How long.</param>
    public static void Pause(int milliseconds) => Thread.Sleep(milliseconds);

    /// <summary>Reports progress.</summary>
    /// <param name="message">What the scenario is doing.</param>
    public void Step(string message) => Report(message);

    /// <summary>Runs a function on the thread of the window, and waits for it.</summary>
    /// <typeparam name="T">What it returns.</typeparam>
    /// <param name="function">The function.</param>
    /// <returns>What it returned.</returns>
    public T OnWindow<T>(Func<T> function)
    {
        TaskCompletionSource<T> done = new(TaskCreationOptions.RunContinuationsAsynchronously);
        Env.Dispatch.Post(() =>
        {
            try
            {
                done.SetResult(function());
            }
            catch (Exception error)
            {
                done.SetException(error);
            }
        });
        if (!done.Task.Wait(TimeSpan.FromSeconds(30)))
        {
            throw new ScenarioException("the window does not take calls");
        }

        return done.Task.GetAwaiter().GetResult();
    }

    /// <summary>Runs an action on the thread of the window, and waits for it.</summary>
    /// <param name="action">The action.</param>
    public void OnWindow(Action action) => OnWindow(() =>
    {
        action();
        return 0;
    });

    /// <summary>Returns what the user can reach: the nodes of the dialog or the menu that is open, or else all the others.</summary>
    /// <returns>The nodes.</returns>
    public List<Node> Nodes() => OnWindow(() => Tree.Collect(Host));

    /// <summary>Polls until a condition holds.</summary>
    /// <param name="description">What the scenario waits for.</param>
    /// <param name="condition">The condition.</param>
    /// <param name="timeout">How long to wait, or <see cref="DefaultWait"/>.</param>
    public void WaitUntil(string description, Func<bool> condition, TimeSpan? timeout = null)
    {
        DateTime deadline = DateTime.UtcNow + (timeout ?? DefaultWait);
        while (!condition())
        {
            if (DateTime.UtcNow > deadline)
            {
                throw new ScenarioException($"Timed out waiting for {description}");
            }

            Thread.Sleep(50);
        }
    }

    /// <summary>Returns the texts on the screen.</summary>
    /// <returns>The texts.</returns>
    public List<string> Texts() => [.. Nodes().Where(node => node.Role == NodeRole.Text).Select(node => node.Text)];

    /// <summary>Waits until a text is on the screen.</summary>
    /// <param name="text">The text.</param>
    public void WaitForText(string text) => WaitUntil($"the text \"{text}\"", () => Texts().Contains(text));

    /// <summary>Returns the first node that matches.</summary>
    /// <param name="match">Tells whether a node matches, given all the nodes.</param>
    /// <returns>The node, or <see langword="null"/>.</returns>
    public Node? Find(Func<Node, List<Node>, bool> match)
    {
        List<Node> all = Nodes();
        return all.FirstOrDefault(node => match(node, all));
    }

    /// <summary>Waits until a node matches, and returns it.</summary>
    /// <param name="description">What the scenario waits for.</param>
    /// <param name="match">Tells whether a node matches, given all the nodes.</param>
    /// <returns>The node.</returns>
    public Node Await(string description, Func<Node, List<Node>, bool> match)
    {
        Node? found = null;
        WaitUntil(description, () => (found = Find(match)) is not null);
        return found!;
    }

    /// <summary>Waits for what the user did to take effect: a handled click changes the state, which refreshes the page.</summary>
    public void Settle()
    {
        for (int i = 0; i < 3; i++)
        {
            OnWindow(() => { });
            Thread.Sleep(30);
        }
    }

    /// <summary>Does what a click on a node does.</summary>
    /// <param name="node">The node.</param>
    public void Click(Node node)
    {
        Check(node.Click is not null, $"\"{node.Name}\" can be clicked");
        OnWindow(node.Click!);
        Settle();
    }

    /// <summary>Clicks the button that is called a name: its label or its tooltip.</summary>
    /// <param name="name">The name.</param>
    public void ClickButton(string name) =>
        Click(Await($"the button \"{name}\"", (node, _) => IsButton(node, name)));

    /// <summary>Clicks the list item whose title starts with a text.</summary>
    /// <param name="title">The start of the title.</param>
    public void ClickItem(string title) =>
        Click(Await($"the item \"{title}\"", (node, _) => node.Role == NodeRole.ListItem && node.Click is not null && node.Name.StartsWith(title, StringComparison.Ordinal)));

    /// <summary>Clicks a button in the list item that is called a name.</summary>
    /// <param name="row">The name of the item.</param>
    /// <param name="name">The name of the button.</param>
    public void ClickButtonNear(string row, string name) =>
        Click(Await($"the button \"{name}\" next to \"{row}\"", (node, all) =>
            IsButton(node, name) && all.Any(item => item.Role == NodeRole.ListItem && item.Name == row && node.IsInside(item))));

    /// <summary>Types into the text field that is labelled or hinted a text, or into the one in the list item with that title.</summary>
    /// <param name="label">The label, the hint, or the title of the item.</param>
    /// <param name="value">The text.</param>
    /// <param name="submit">Whether to press Enter afterwards.</param>
    public void Fill(string label, string value, bool submit)
    {
        Node node = Await($"the field \"{label}\"", (candidate, all) =>
            candidate.Role == NodeRole.TextField && candidate.Fill is not null
            && (candidate.Name == label || all.Any(item => item.Role == NodeRole.ListItem && item.Name == label && candidate.IsInside(item))));
        OnWindow(() => node.Fill!(value, submit));
        Settle();
    }

    /// <summary>Chooses a destination of the navigation rail or bar.</summary>
    /// <param name="label">The label of the destination.</param>
    public void Navigate(string label) =>
        Click(Await($"the destination \"{label}\"", (node, _) => node.Role == NodeRole.Button && node.Name == label && node.Enabled));

    /// <summary>Opens the menu of the button that is called a tooltip, and chooses an entry.</summary>
    /// <param name="menuTooltip">The name of the button.</param>
    /// <param name="item">The label of the entry.</param>
    public void ChooseMenuItem(string menuTooltip, string item)
    {
        ClickButton(menuTooltip);
        Click(Await($"the menu item \"{item}\"", (node, _) => node.Role == NodeRole.MenuItem && node.Name == item && node.Enabled));
    }

    /// <summary>
    /// Presses the last of the buttons of a dialog, which are the actions at its bottom (the confirmation), or the one
    /// before it (the cancellation).
    /// </summary>
    /// <param name="confirm">Whether to confirm.</param>
    public void TapDialog(bool confirm)
    {
        WaitUntil("a dialog", () => Find((node, _) => node.Role == NodeRole.Dialog) is not null);
        List<Node> all = Nodes();
        Node dialog = all.First(node => node.Role == NodeRole.Dialog);
        List<Node> buttons = [.. all.Where(node => node.Role == NodeRole.Button && node.Enabled && node.IsInside(dialog))];
        Check(buttons.Count > 0, "the dialog has buttons");
        Click(confirm || buttons.Count == 1 ? buttons[^1] : buttons[^2]);
    }

    /// <summary>Waits until the task with a title (in English) has succeeded; fails if the task failed.</summary>
    /// <param name="title">The title.</param>
    public void WaitForTask(string title) => WaitUntil($"the task \"{title}\" to finish", () =>
    {
        foreach (VfoxTask task in Model.Tasks.Tasks)
        {
            if (Presentation.TaskTitle(task.Command, Env.T) != title)
            {
                continue;
            }

            if (task.Status == VfoxTaskStatus.Failed)
            {
                throw new ScenarioException($"{title} failed: {task.ErrorMessage}");
            }

            return task.Status == VfoxTaskStatus.Succeeded;
        }

        return false;
    }, TimeSpan.FromSeconds(60));

    /// <summary>Returns the task that was queued last.</summary>
    /// <returns>The task.</returns>
    public VfoxTask NewestTask()
    {
        IReadOnlyList<VfoxTask> tasks = Model.Tasks.Tasks;
        Check(tasks.Count > 0, "there is a task");
        return tasks[0];
    }

    /// <summary>Runs vfox as the tests see it: in the sandbox, without colours.</summary>
    /// <param name="args">The arguments.</param>
    /// <returns>What vfox printed.</returns>
    public string Vfox(params string[] args)
    {
        ProcessStartInfo info = new(Env.Environment.GetValueOrDefault("VFOX_PLUS_E2E_VFOX", "vfox"))
        {
            WorkingDirectory = "/",
            RedirectStandardOutput = true,
            UseShellExecute = false,
        };
        foreach (string arg in args)
        {
            info.ArgumentList.Add(arg);
        }

        info.Environment["NO_COLOR"] = "1";
        using Process process = Process.Start(info)!;
        string output = process.StandardOutput.ReadToEnd();
        process.WaitForExit();
        return output;
    }

    /// <summary>Waits for the page to settle and asks the harness to capture the screen.</summary>
    /// <param name="name">The name of the picture.</param>
    public void Shot(string name)
    {
        Pause(1500);
        Step("SHOT " + name);
        Pause(1500);
    }

    /// <summary>Changes the size of the window.</summary>
    /// <param name="width">The width in dp.</param>
    /// <param name="height">The height in dp.</param>
    public void Resize(int width, int height)
    {
        OnWindow(() =>
        {
            double scale = window.Content?.XamlRoot?.RasterizationScale ?? 1;
            window.AppWindow.Resize(new Windows.Graphics.SizeInt32 { Width = (int)(width * scale), Height = (int)(height * scale) });
        });
        Pause(500);
    }

    /// <summary>Reports what is on the screen: the texts and the buttons, to find out why a scenario failed.</summary>
    public void Dump()
    {
        List<string> texts = [];
        List<string> widgets = [];
        foreach (Node node in Nodes())
        {
            if (node.Role == NodeRole.Text)
            {
                texts.Add($"\"{node.Text}\"");
            }
            else if (node.Name.Length > 0)
            {
                widgets.Add($"{node.Role} \"{node.Name}\"{(node.Enabled ? string.Empty : " (disabled)")}");
            }
        }

        Report("on screen, texts: " + string.Join(", ", texts));
        Report("on screen, widgets: " + string.Join(", ", widgets));
    }

    private static bool IsButton(Node node, string name) =>
        node.Role is NodeRole.Button or NodeRole.MenuItem && node.Name == name && node.Enabled;
}

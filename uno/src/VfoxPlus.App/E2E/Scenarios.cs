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

using VfoxPlus.Core.Settings;
using VfoxPlus.State;

namespace VfoxPlus.App.E2E;

/// <summary>
/// The end-to-end scenarios, the same as those of the other implementations (<c>web/src/e2e/scenarios.ts</c>,
/// <c>flutter/integration_test</c>).
/// </summary>
internal static class Scenarios
{
    /// <summary>The scenarios by name.</summary>
    public static readonly IReadOnlyDictionary<string, Action<Scenario>> All = new Dictionary<string, Action<Scenario>>
    {
        ["fake"] = Fake,
        ["real"] = Real,
        ["measure"] = Measure,
        ["screenshots"] = Screenshots,
    };

    // Against shared/fixtures/fake-vfox.sh, which replays what vfox printed.
    private static void Fake(Scenario s)
    {
        s.WaitForText("Versions in use");
        s.Step("home");
        string executable = s.Model.Vfox.Executable ?? string.Empty;
        s.WaitForText("vfox 1.0.13 · " + executable);
        Scenario.Check(executable == s.Env.Environment["VFOX_PLUS_E2E_VFOX"], "the executable from the settings is used");

        s.Navigate("SDKs");
        s.ClickItem("nodejs");
        s.WaitForText("Node.js runtime environment.");
        s.Step("detail");

        // Select 22.21.0 as the global default.
        s.ClickButton("Use");
        s.WaitForTask("Use nodejs@22.21.0");

        // Install a version from the available tab.
        s.ClickButton("Available");
        s.WaitForText("872 versions");
        s.Fill("Filter versions", "24.11.0", submit: false);
        s.WaitForText("1 version");
        s.ClickButtonNear("24.11.0", "Install");
        s.WaitForTask("Install nodejs@24.11.0");

        IReadOnlyList<VfoxTask> tasks = s.Model.Tasks.Tasks;
        Scenario.Check(tasks.All(task => task.Status == VfoxTaskStatus.Succeeded), "every task succeeded");
        Scenario.Check(tasks[0].Progress == 1, "the download progress reached 100 %");
        Scenario.Check(string.Join('\n', tasks[0].Log).Contains("Install nodejs@24.11.0 success!", StringComparison.Ordinal), "the output of vfox is kept");
    }

    // Against the real vfox, with the offline plugin of shared/test-plugin.
    private static void Real(Scenario s)
    {
        string server = s.Env.Environment["VFOX_PLUS_TEST_SERVER"];
        string home = s.Env.Environment["HOME"];
        string localPlugin = s.Env.Environment["VFOX_PLUS_E2E_LOCAL_PLUGIN"];
        using HttpClient client = new();
        void Control(string action)
        {
            try
            {
                client.GetStringAsync(server + "/control/" + action).GetAwaiter().GetResult();
            }
            catch (HttpRequestException)
            {
                // The test server is gone: the scenario fails on what it checks next.
            }
        }

        s.WaitForText("Versions in use");
        Scenario.Check(s.Model.Installed.Value?.Count == 0, "no plugin is installed yet");

        // Point vfox to the local registry through the vfox configuration editor.
        s.Step("configure the registry");
        s.Navigate("Settings");
        s.Fill("Plugin registry", server + "/registry", submit: true);
        s.WaitUntil("the registry to be saved", () => s.Model.VfoxConfig.ValueFor("registry.address") == server + "/registry");
        Scenario.Check(s.Vfox("config", "registry.address").Trim() == server + "/registry", "vfox stored the registry");

        // Add the plugin from the registry.
        s.Step("add a plugin from the registry");
        s.Navigate("Plugins");
        s.WaitForText("testsdk");
        s.ClickButton("Add");
        s.WaitForTask("Add plugin testsdk");
        s.WaitForText("Added");

        // Add a local copy of the plugin under another name. vfox moves local .lua files away, so the application must
        // hand it a copy.
        s.Step("add a plugin from a local file");
        s.ClickButton("Add from source…");
        s.WaitForText("Add a plugin from source");
        s.Fill("Source", localPlugin, submit: false);
        s.Fill("Name (optional)", "localsdk", submit: false);
        s.ClickButton("Add plugin");
        s.WaitForTask("Add plugin localsdk");
        Scenario.Check(File.Exists(localPlugin), "the source file survives vfox add --source");

        // Search and install both versions.
        s.Step("install");
        s.Navigate("SDKs");
        s.ClickItem("testsdk");
        s.WaitForText("An offline SDK for the vfox-plus end-to-end tests.");
        s.ClickButton("Available");
        s.WaitForText("2 versions");
        s.WaitForText("LTS");
        s.WaitForText("testtool 2.0");
        Scenario.Check(s.Texts().Count(text => text == "LTS") == 1, "the note is shown once");
        s.ClickButtonNear("2.0.0", "Install");
        s.WaitForTask("Install testsdk@2.0.0");
        Scenario.Check(s.NewestTask().Progress == 1, "vfox reports the download progress on standard error");

        // Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox leaves the partial installation behind,
        // but must neither list it nor refuse to install the version again.
        s.Step("cancel an installation");
        Control("hold");
        s.ClickButtonNear("1.0.0", "Install");
        s.WaitUntil("the download to start", () => s.NewestTask().Progress is > 0);
        s.ClickButton("Cancel");
        s.WaitUntil("vfox to exit", () => s.NewestTask().FinishedAt is not null);
        Scenario.Check(s.NewestTask().Status == VfoxTaskStatus.Cancelled, "the task is cancelled");
        Scenario.Check(!s.Vfox("list", "testsdk").Contains("1.0.0", StringComparison.Ordinal), "vfox does not list the partial version");
        Control("release");
        Scenario.Pause(500);
        s.ClickButtonNear("1.0.0", "Install");
        s.WaitForTask("Install testsdk@1.0.0");

        // Make 2.0.0 the global default.
        s.Step("use");
        s.ClickButton("Installed (2)");
        s.WaitForText("1.0.0");
        s.ClickButtonNear("2.0.0", "Use");
        s.WaitForTask("Use testsdk@2.0.0");
        s.WaitForText("In use: 2.0.0");
        string toml = File.ReadAllText(home + "/.vfox/.vfox.toml");
        Scenario.Check(toml.Contains("testsdk = \"2.0.0\"", StringComparison.Ordinal), "the global .vfox.toml pins 2.0.0");
        Scenario.Check(new FileInfo(home + "/.vfox/sdks/testsdk").LinkTarget is not null || Directory.Exists(home + "/.vfox/sdks/testsdk"), "the SDK is linked");

        // Uninstall 1.0.0.
        s.Step("uninstall");
        s.ChooseMenuItem("More actions for 1.0.0", "Uninstall");
        s.WaitForText("Uninstall testsdk 1.0.0?");
        s.TapDialog(confirm: true);
        s.WaitForTask("Uninstall testsdk@1.0.0");
        s.WaitForText("Installed (1)");

        // Remove the second plugin.
        s.Step("remove");
        s.ClickItem("localsdk");
        s.WaitForText("localsdk");
        s.ChooseMenuItem("More actions", "Remove plugin");
        s.WaitForText("Remove localsdk?");
        s.TapDialog(confirm: true);
        s.WaitForTask("Remove plugin localsdk");
        s.WaitForText("Select an SDK to manage its versions.");
        s.WaitUntil("the removed plugin to disappear from the list", () => s.Model.Installed.Value?.All(sdk => sdk.Name != "localsdk") ?? false);

        // Every task but the cancelled one succeeded, and vfox agrees with what the application shows.
        List<VfoxTaskStatus> statuses = [.. s.Model.Tasks.Tasks.Where(task => task.Status != VfoxTaskStatus.Succeeded).Select(task => task.Status)];
        Scenario.Check(statuses.SequenceEqual([VfoxTaskStatus.Cancelled]), "only the cancelled task did not succeed: " + string.Join(", ", statuses));
        Scenario.Check(s.Vfox("current") == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0");
        Scenario.Check(s.Vfox("list", "testsdk").Contains("2.0.0", StringComparison.Ordinal), "vfox lists testsdk 2.0.0");

        // The copy of the local plugin is deleted once its task has finished.
        string temporary = home + "/.vfox/tmp";
        string[] leftovers = Directory.Exists(temporary)
            ? [.. Directory.EnumerateFileSystemEntries(temporary).Select(Path.GetFileName).Where(name => name!.StartsWith("vfox-plus-plugin-", StringComparison.Ordinal)).Select(name => name!)]
            : [];
        Scenario.Check(leftovers.Length == 0, "the copy of the local plugin is deleted: " + string.Join(", ", leftovers));
    }

    // Opens the Home page and the versions of Node.js, so that the harness can measure the memory of the application at
    // both points (see docs/new-implementation.md).
    private static void Measure(Scenario s)
    {
        s.WaitForText("Versions in use");
        Scenario.Pause(3000);
        s.Step("MEASURE home");
        Scenario.Pause(5000);
        s.Navigate("SDKs");
        s.ClickItem("nodejs");
        s.ClickButton("Available");
        s.WaitForText("872 versions");
        Scenario.Pause(3000);
        s.Step("MEASURE nodejs");
        Scenario.Pause(5000);
        s.Navigate("Home");
    }

    // Shows every page against the fake vfox; the harness captures the screen at every `e2e: SHOT <name>` (see
    // shared/tools/screenshots.sh). With VFOX_PLUS_FAKE_DELAY, it shows an installation in progress instead.
    private static void Screenshots(Scenario s)
    {
        s.WaitForText("Versions in use");
        if (s.Env.Environment.GetValueOrDefault("VFOX_PLUS_FAKE_DELAY", string.Empty).Length > 0)
        {
            s.Navigate("SDKs");
            s.ClickItem("nodejs");
            s.ClickButton("Available");
            s.WaitForText("872 versions");
            s.ClickButtonNear("26.12.0", "Install");
            s.WaitUntil("the download to start", () => s.NewestTask().Progress is > 0);
            s.Shot("installing");
            return;
        }

        s.Shot("home");
        s.Navigate("SDKs");
        s.ClickItem("nodejs");
        s.WaitForText("Node.js runtime environment.");
        s.Shot("sdks");
        s.ClickButton("Available");
        s.WaitForText("872 versions");
        s.Shot("available");
        s.Fill("Filter versions", "24.11.0", submit: false);
        s.WaitForText("1 version");
        s.ClickButtonNear("24.11.0", "Install");
        s.WaitForTask("Install nodejs@24.11.0");
        s.Navigate("Tasks");
        s.ClickButton("Show output");
        s.Shot("tasks");
        s.Navigate("Plugins");
        s.WaitForText("dotnet");
        s.Shot("plugins");
        s.ClickButton("Add from source…");
        s.WaitForText("Add a plugin from source");
        s.Shot("dialog");
        s.TapDialog(confirm: false);
        s.Navigate("Settings");
        s.WaitForText("Plugin registry");
        s.Shot("settings");
        s.Model.Settings.Update(current => current with { Theme = ThemePreference.Dark, RailExpanded = true });
        s.Navigate("Home");
        s.Shot("dark-expanded");
        s.Navigate("SDKs");
        s.ClickItem("maven");
        s.WaitForText("3.8.9");
        s.Shot("dark-sdks");
        s.Model.Settings.Update(current => current with { Theme = ThemePreference.Light, RailExpanded = false, Language = LanguagePreference.SimplifiedChinese });
        s.Navigate("首页");
        s.WaitForText("正在使用的版本");
        s.Shot("zh-home");
        s.Navigate("设置");
        s.WaitForText("插件仓库");
        s.Shot("zh-settings");
        s.Model.Settings.Update(current => current with { Language = LanguagePreference.English });
        s.Resize(412, 800);
        s.Navigate("Home");
        s.WaitForText("Versions in use");
        s.Shot("compact-home");

        // Maven is still selected on the SDKs page, which compact windows would show; the versions in use on the Home
        // page lead to their SDK instead.
        s.ClickItem("nodejs");
        s.WaitForText("Node.js runtime environment.");
        s.Shot("compact-detail");
    }
}

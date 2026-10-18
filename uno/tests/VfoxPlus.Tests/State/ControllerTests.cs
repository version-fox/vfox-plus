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

using System.Text.Json.Nodes;
using VfoxPlus.Core;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Settings;
using VfoxPlus.Core.Vfox;
using VfoxPlus.State;
using VfoxPlus.Tests.Support;
using Xunit;

namespace VfoxPlus.Tests.State;

public sealed class ControllerTests : IDisposable
{
    private readonly List<string> directories = [];
    private readonly List<IDisposable> disposables = [];

    public void Dispose()
    {
        foreach (IDisposable disposable in disposables)
        {
            disposable.Dispose();
        }

        foreach (string directory in directories)
        {
            Directory.Delete(directory, recursive: true);
        }
    }

    // ---- AsyncResource, notifications ----------------------------------------------------------------

    [Fact]
    public async Task AsyncResourceKeepsThePreviousValueWhileReloadingAndRecordsErrors()
    {
        int calls = 0;
        AsyncResource<int> resource = new(() =>
        {
            int call = Interlocked.Increment(ref calls);
            return call == 2 ? Task.FromException<int>(new InvalidOperationException("boom")) : Task.FromResult(call);
        });
        Assert.False(resource.HasValue);
        Assert.False(resource.IsLoading);

        await resource.LoadAsync();
        Assert.True(resource.HasValue);
        Assert.Equal(1, resource.Value);

        Task reload = resource.LoadAsync();
        Assert.True(resource.IsLoading);
        await reload;
        Assert.Equal("boom", resource.Error?.Message);
        Assert.Equal(1, resource.Value);
        Assert.False(resource.IsLoading);

        await resource.LoadAsync();
        Assert.Equal(3, resource.Value);
        Assert.Null(resource.Error);
    }

    [Theory]
    [InlineData(true)]
    [InlineData(false)]
    public async Task AsyncResourceLetsTheLatestOfOverlappingLoadsWin(bool newerFirst)
    {
        TaskCompletionSource<string>[] gates = [new(), new()];
        int calls = 0;
        AsyncResource<string> resource = new(() => gates[Interlocked.Increment(ref calls) - 1].Task);
        Task older = resource.LoadAsync();
        await Wait.UntilAsync("the first load", () => Volatile.Read(ref calls) == 1);
        Task newer = resource.LoadAsync();
        await Wait.UntilAsync("the second load", () => Volatile.Read(ref calls) == 2);
        if (newerFirst)
        {
            gates[1].SetResult("new");
            await newer;
            gates[0].SetResult("old");
            await older;
        }
        else
        {
            gates[0].SetResult("old");
            await older;
            gates[1].SetResult("new");
            await newer;
        }

        Assert.Equal("new", resource.Value);
        Assert.False(resource.IsLoading);
    }

    [Fact]
    public async Task AsyncResourceIgnoresTheFailureOfAReplacedLoad()
    {
        TaskCompletionSource<string>[] gates = [new(), new()];
        int calls = 0;
        AsyncResource<string> resource = new(() => gates[Interlocked.Increment(ref calls) - 1].Task);
        Task older = resource.LoadAsync();
        await Wait.UntilAsync("the first load", () => Volatile.Read(ref calls) == 1);
        Task newer = resource.LoadAsync();
        await Wait.UntilAsync("the second load", () => Volatile.Read(ref calls) == 2);
        gates[1].SetResult("fine");
        await newer;
        gates[0].SetException(new InvalidOperationException("boom"));
        await older;
        Assert.Equal("fine", resource.Value);
        Assert.Null(resource.Error);
    }

    [Fact]
    public async Task AsyncResourceEnsuresItIsLoadedOnce()
    {
        TaskCompletionSource<string> gate = new();
        int calls = 0;
        AsyncResource<string> resource = new(() =>
        {
            Interlocked.Increment(ref calls);
            return gate.Task;
        });
        Task first = resource.EnsureLoadedAsync();
        await Wait.UntilAsync("the load", () => Volatile.Read(ref calls) == 1);
        Task second = resource.EnsureLoadedAsync();
        gate.SetResult("value");
        await first;
        await second;
        await resource.EnsureLoadedAsync();
        Assert.Equal(1, Volatile.Read(ref calls));

        resource.SetValue("set");
        Assert.Equal("set", resource.Value);
        Assert.Null(resource.Error);
        Assert.False(resource.IsLoading);
    }

    [Fact]
    public async Task SetValueReplacesALoadInFlight()
    {
        TaskCompletionSource<string> gate = new();
        AsyncResource<string> resource = new(() => gate.Task);
        Task load = resource.LoadAsync();
        await Wait.UntilAsync("loading", () => resource.IsLoading);
        resource.SetValue("optimistic");
        gate.SetResult("loaded");
        await load;
        Assert.Equal("optimistic", resource.Value);
    }

    [Fact]
    public async Task ListenersAreNotifiedUntilTheyStopOrTheObjectIsDisposed()
    {
        AsyncResource<int> resource = new(() => Task.FromResult(1));
        int notified = 0;
        IDisposable stop = Notifier.ListenAll([resource, null], () => Interlocked.Increment(ref notified));
        await resource.LoadAsync();
        Assert.Equal(2, Volatile.Read(ref notified));

        stop.Dispose();
        stop.Dispose();
        await resource.LoadAsync();
        Assert.Equal(2, Volatile.Read(ref notified));

        resource.AddListener(() => Interlocked.Increment(ref notified));
        resource.Dispose();
        Assert.True(resource.IsDisposed);
        await resource.LoadAsync();
        Assert.Equal(2, Volatile.Read(ref notified));
    }

    [Fact]
    public void EmitterDeliversToListenersUntilTheyStop()
    {
        Emitter<string> emitter = new();
        List<string> first = [];
        List<string> second = [];
        IDisposable stopFirst = emitter.Listen(first.Add);
        emitter.Listen(second.Add);
        emitter.Emit("a");
        stopFirst.Dispose();
        emitter.Emit("b");
        emitter.Close();
        emitter.Emit("c");
        Assert.Equal(["a"], first);
        Assert.Equal(["a", "b"], second);
    }

    // ---- SettingsController -----------------------------------------------------------------------------

    [Fact]
    public void SettingsControllerPersistsChangesAndSkipsNoOps()
    {
        MemorySettingsStore store = new(new AppSettings { Theme = ThemePreference.Dark });
        SettingsController controller = new(store);
        Assert.Equal(ThemePreference.System, controller.Settings.Theme);
        controller.Load();
        Assert.Equal(ThemePreference.Dark, controller.Settings.Theme);

        int changes = 0;
        IDisposable stop = controller.AddListener(() => changes++);
        controller.Update(current => current);
        Assert.Equal(0, store.SaveCount);
        Assert.Equal(0, changes);

        controller.Update(current => current with
        {
            Theme = ThemePreference.Light,
            ExtraEnvironment = new Dictionary<string, string> { ["A"] = "1" },
            Unknown = new Dictionary<string, JsonNode?> { ["x"] = JsonValue.Create(1) },
        });
        AppSettings saved = store.Load();
        Assert.Equal(1, store.SaveCount);
        Assert.Equal(1, changes);
        Assert.Equal(ThemePreference.Light, saved.Theme);
        Assert.Equal("1", saved.ExtraEnvironment["A"]);
        stop.Dispose();
    }

    [Fact]
    public void SettingsControllerSurvivesAStoreThatCannotSave()
    {
        SettingsController controller = new(new FailingStore());
        controller.Update(current => current with { RailExpanded = true });
        Assert.True(controller.Settings.RailExpanded);
    }

    // ---- VfoxController -----------------------------------------------------------------------------------

    private static (VfoxController Controller, VfoxClient Client) VfoxControllerFor(FakeProcessRunner runner, string found)
    {
        VfoxClient client = new(runner, new VfoxLaunchConfig { Executable = "vfox" });
        Dictionary<string, string> environment = new() { ["HOME"] = "/home/u" };
        return (new VfoxController(client, new VfoxLocator(environment, path => path == found), environment), client);
    }

    [Fact]
    public async Task VfoxUsesTheDetectedExecutableAndReadsTheVersion()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        (VfoxController controller, VfoxClient client) = VfoxControllerFor(runner, "/usr/local/bin/vfox");
        Assert.Equal(VfoxAvailability.Unknown, controller.Availability);

        await controller.ApplyAsync(new AppSettings { ExtraEnvironment = new Dictionary<string, string> { ["A"] = "1" } });
        Assert.Equal(VfoxAvailability.Ready, controller.Availability);
        Assert.Equal("1.0.13", controller.Version);
        Assert.Equal("/usr/local/bin/vfox", controller.Executable);
        Assert.False(controller.IsCustomExecutable);
        Assert.Null(controller.Error);
        VfoxLaunchConfig config = client.LaunchConfig;
        Assert.Equal("/usr/local/bin/vfox", config.Executable);
        Assert.Equal("1", config.ExtraEnvironment["A"]);
        Assert.Equal("/home/u", config.BaseEnvironment["HOME"]);
        Assert.Equal("/usr/local/bin/vfox", Assert.Single(runner.Invocations).Executable);
    }

    [Fact]
    public async Task VfoxPrefersTheExecutableChosenByTheUser()
    {
        (VfoxController controller, _) = VfoxControllerFor(FakeProcessRunner.ForFixtures(), "/usr/local/bin/vfox");
        await controller.ApplyAsync(new AppSettings { VfoxExecutable = "/opt/vfox" });
        Assert.Equal("/opt/vfox", controller.Executable);
        Assert.True(controller.IsCustomExecutable);
        Assert.Equal(VfoxAvailability.Ready, controller.Availability);
    }

    [Fact]
    public async Task VfoxReportsAMissingAndABrokenExecutable()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        (VfoxController controller, VfoxClient client) = VfoxControllerFor(runner, string.Empty);
        runner.SetMissingExecutable(true);
        await controller.ApplyAsync(new AppSettings());
        Assert.Equal(VfoxAvailability.NotFound, controller.Availability);
        Assert.Null(controller.Executable);
        Assert.Equal(VfoxClient.ExecutableName, client.LaunchConfig.Executable);
        Assert.Null(controller.Version);

        runner.SetMissingExecutable(false);
        runner.On("--version", new ProcessOutput(0, "hello", string.Empty));
        await controller.CheckAsync();
        Assert.Equal(VfoxAvailability.Broken, controller.Availability);
        Assert.IsType<VfoxException>(controller.Error);
    }

    [Fact]
    public async Task VfoxReportsUnexpectedFailuresAsBroken()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        (VfoxController controller, _) = VfoxControllerFor(runner, "/usr/bin/vfox");
        runner.SetFailure(new InvalidOperationException("boom"));
        await controller.CheckAsync();
        Assert.Equal(VfoxAvailability.Broken, controller.Availability);
        Assert.Equal("boom", controller.Error?.Message);
    }

    // ---- SdkDetailController ------------------------------------------------------------------------------

    [Fact]
    public async Task SdkDetailFiltersTheAvailableVersionsByTextNoteAndInstallation()
    {
        SdkDetailController controller = new(new VfoxClient(FakeProcessRunner.ForFixtures()), "nodejs");
        disposables.Add(controller);
        int changes = 0;
        controller.AddListener(() => Interlocked.Increment(ref changes));
        await controller.Versions.LoadAsync();
        Assert.True(Volatile.Read(ref changes) > 0, "loading the versions is announced by the controller");

        HashSet<string> installed = ["24.11.1"];
        Assert.Equal(872, controller.VisibleVersions(installed).Count);

        controller.SetFilter(VersionFilter.WithNote);
        IReadOnlyList<AvailableVersion> withNote = controller.VisibleVersions(installed);
        Assert.NotEmpty(withNote);
        Assert.All(withNote, version => Assert.True(version.HasNote));

        controller.SetFilter(VersionFilter.All);
        controller.SetQuery(" 24.11. ");
        Assert.Equal(" 24.11. ", controller.Query);
        Assert.Equal(["24.11.1", "24.11.0"], controller.VisibleVersions(installed).Select(version => version.Version));

        controller.SetQuery("lts");
        Assert.True(controller.VisibleVersions(installed).Count > 10, "the query also searches the notes");

        controller.SetQuery(string.Empty);
        controller.SetFilter(VersionFilter.Installed);
        Assert.Equal(VersionFilter.Installed, controller.Filter);
        Assert.Equal(["24.11.1"], controller.VisibleVersions(installed).Select(version => version.Version));

        int before = Volatile.Read(ref changes);
        controller.SetFilter(VersionFilter.Installed);
        controller.SetQuery(string.Empty);
        Assert.Equal(before, Volatile.Read(ref changes));
    }

    [Fact]
    public async Task SdkDetailLoadsThePluginInformation()
    {
        SdkDetailController controller = new(new VfoxClient(FakeProcessRunner.ForFixtures()), "nodejs");
        disposables.Add(controller);
        await controller.Info.LoadAsync();
        Assert.Equal("0.5.1", controller.Info.Value?.Version);
        Assert.Equal("nodejs", controller.Name);
    }

    // ---- VfoxConfigController -----------------------------------------------------------------------------

    [Fact]
    public async Task ConfigSavesReloadsAndExposesValuesByKey()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        ManualResetEventSlim gate = new();
        runner.OnDynamic("config gitignore.enable false", () =>
        {
            gate.Wait();
            return new ProcessOutput(0, string.Empty, string.Empty);
        });
        VfoxConfigController controller = new(new VfoxClient(runner));
        disposables.Add(controller);
        await controller.Entries.LoadAsync();
        Assert.Equal("true", controller.ValueFor("gitignore.enable"));
        Assert.Null(controller.ValueFor("missing"));

        Task set = Task.Run(() => controller.SetAsync("gitignore.enable", "false"));
        await Wait.UntilAsync("saving", () => controller.IsSaving("gitignore.enable"));
        gate.Set();
        await set;
        Assert.False(controller.IsSaving("gitignore.enable"));
        Assert.Equal(2, runner.Calls("config --list").Count);

        runner.On("config --unset proxy.url", new ProcessOutput(0, string.Empty, string.Empty));
        await controller.ResetAsync("proxy.url");
        Assert.Single(runner.Calls("config --unset proxy.url"));
        Assert.Equal(3, runner.Calls("config --list").Count);
    }

    [Fact]
    public async Task ConfigRethrowsRejectedValuesAfterReloading()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures()
            .On("config cache.availableHookDuration x", Fixtures.Output("config-set-invalid-duration", 1));
        VfoxConfigController controller = new(new VfoxClient(runner));
        disposables.Add(controller);
        await Assert.ThrowsAsync<VfoxException>(() => controller.SetAsync("cache.availableHookDuration", "x"));
        Assert.True(controller.Entries.HasValue, "the configuration is reloaded either way");
        Assert.False(controller.IsSaving("cache.availableHookDuration"));
    }

    // ---- AppModel -----------------------------------------------------------------------------------------

    private AppModel NewModel(FakeProcessRunner runner, string home)
    {
        AppModel model = new(new AppModelOptions(
            runner,
            new MemorySettingsStore(new AppSettings()),
            new Dictionary<string, string> { ["HOME"] = home },
            new VfoxLocator(new Dictionary<string, string>(), path => path == "/usr/bin/vfox")));
        disposables.Add(model);
        return model;
    }

    private async Task<AppModel> StartedModelAsync(FakeProcessRunner runner, string home)
    {
        AppModel model = NewModel(runner, home);
        await model.StartAsync();
        await Wait.UntilAsync("the installed SDKs", () => model.Installed.HasValue);
        return model;
    }

    private string TempDirectory()
    {
        string directory = Directory.CreateTempSubdirectory("vfox-plus-test-").FullName;
        directories.Add(directory);
        return directory;
    }

    [Fact]
    public async Task ModelStartsByProbingVfoxAndLoadingTheInstalledSdks()
    {
        AppModel model = await StartedModelAsync(FakeProcessRunner.ForFixtures(), "/home/u");
        Assert.Equal(VfoxAvailability.Ready, model.Vfox.Availability);
        Assert.Equal(["maven", "mvn2", "nodejs"], model.Installed.Value!.Select(sdk => sdk.Name));
        InstalledSdk? nodejs = model.InstalledSdk("nodejs");
        Assert.NotNull(nodejs);
        Assert.Equal("24.11.1", nodejs.Current);
        Assert.Null(model.InstalledSdk("python"));
        SdkDetailController detail = model.SdkDetail("nodejs");
        disposables.Add(detail);
        Assert.Equal("nodejs", detail.Name);
    }

    [Fact]
    public async Task ModelReloadsTheInstalledSdksAfterATask()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        AppModel model = await StartedModelAsync(runner, "/home/u");
        int before = runner.Calls("list").Count;
        VfoxTask task = model.Run(VfoxCommands.Use("nodejs", "22.21.0", UseScope.Global));
        await Wait.UntilAsync("the reload", () => runner.Calls("list").Count == before + 1);
        await Wait.UntilAsync("the task", () => task.Status == VfoxTaskStatus.Succeeded);
    }

    [Fact]
    public async Task ModelReloadsTheRegistryAndTheConfigurationWhenVfoxBecomesAvailableAgain()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        AppModel model = await StartedModelAsync(runner, "/home/u");
        await model.Registry.LoadAsync();
        await model.VfoxConfig.Entries.LoadAsync();
        model.Settings.Update(current => current with { ExtraEnvironment = new Dictionary<string, string> { ["X"] = "1" } });
        await Wait.UntilAsync("the registry reload", () => runner.Calls("available").Count == 2);
        await Wait.UntilAsync("the configuration reload", () => runner.Calls("config --list").Count == 2);
        await Wait.UntilAsync("the environment", () => runner.Invocations.Any(call => call.Environment.GetValueOrDefault("X") == "1"));
    }

    [Fact]
    public async Task ModelLeavesTheRegistryAloneUntilItWasLoaded()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        AppModel model = await StartedModelAsync(runner, "/home/u");
        model.Settings.Update(current => current with { VfoxExecutable = "/opt/vfox" });
        await Wait.UntilAsync("the second probe", () => runner.Calls("--version").Count == 2);
        await Wait.UntilAsync("ready", () => model.Vfox.Availability == VfoxAvailability.Ready);
        Assert.Empty(runner.Calls("available"));
        Assert.Empty(runner.Calls("config --list"));
    }

    [Fact]
    public async Task ModelHandsVfoxACopyOfALocalLuaPluginAndDeletesItAfterwards()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        string home = TempDirectory();
        AppModel model = await StartedModelAsync(runner, home);
        string original = home + "/mine.lua";
        File.WriteAllText(original, "PLUGIN = {}");
        VfoxTask task = model.AddPluginFromSource(original, "mine");
        IReadOnlyList<string> args = task.Command.Arguments;
        string source = args[args.ToList().IndexOf("--source") + 1];
        Assert.StartsWith(home + "/.vfox/tmp/vfox-plus-plugin-", source, StringComparison.Ordinal);
        Assert.EndsWith("/mine.lua", source, StringComparison.Ordinal);
        Assert.Equal(["--alias", "mine"], args.TakeLast(2));

        await Wait.UntilAsync("success", () => task.Status == VfoxTaskStatus.Succeeded);
        await Wait.UntilAsync("the copy to be deleted", () => !File.Exists(source));
        Assert.True(File.Exists(original), "the original must survive");
    }

    [Fact]
    public async Task ModelPassesUrlsToVfoxAsTheyAre()
    {
        AppModel model = await StartedModelAsync(FakeProcessRunner.ForFixtures(), TempDirectory());
        VfoxTask task = model.AddPluginFromSource("https://example.com/p.zip");
        Assert.Equal(["add", "--source", "https://example.com/p.zip"], task.Command.Arguments);
    }

    [Fact]
    public async Task ModelRejectsAnInvalidAlias()
    {
        AppModel model = await StartedModelAsync(FakeProcessRunner.ForFixtures(), TempDirectory());
        Assert.Throws<VfoxUsageException>(() => model.AddPluginFromSource("https://example.com/p.zip", "-bad"));
        Assert.Empty(model.Tasks.Tasks);
    }

    [Fact]
    public async Task ModelAppliesTheSettingsAgainWhenTheExecutableChanges()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        AppModel model = await StartedModelAsync(runner, "/home/u");
        model.Settings.Update(current => current with { VfoxExecutable = "/opt/vfox" });
        await Wait.UntilAsync("the custom executable", () => model.Vfox.Executable == "/opt/vfox" && model.Vfox.Availability == VfoxAvailability.Ready);
        Assert.Equal("/opt/vfox", runner.Invocations[^1].Executable);

        // A change that does not concern vfox applies nothing.
        int probes = runner.Calls("--version").Count;
        model.Settings.Update(current => current with { Theme = ThemePreference.Dark });
        Assert.Equal(probes, runner.Calls("--version").Count);
    }

    [Fact]
    public async Task ModelDoesNotReloadAfterATaskThatNeverRan()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        FakeRunningProcess process = new();
        runner.OnStart("update --all", () => process);
        AppModel model = await StartedModelAsync(runner, "/home/u");
        int before = runner.Calls("list").Count;
        VfoxTask running = model.Run(VfoxCommands.UpdateAllPlugins());
        VfoxTask queued = model.Run(VfoxCommands.UpdatePlugin("b"));
        model.Tasks.Cancel(queued);
        process.Exit(0);
        await Wait.UntilAsync("the running task", () => running.Status == VfoxTaskStatus.Succeeded);
        await Wait.UntilAsync("the reload", () => runner.Calls("list").Count == before + 1);

        // The cancelled task never ran, so it did not change anything.
        await Task.Delay(20);
        Assert.Equal(before + 1, runner.Calls("list").Count);
    }

    [Fact]
    public async Task ModelDeletesPendingPluginCopiesWhenDisposed()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();

        // A task that keeps the queue busy, so that the plugin waits behind it.
        runner.OnStart("update --all", () => new FakeRunningProcess());
        string home = TempDirectory();
        AppModel model = NewModel(runner, home);
        await model.StartAsync();
        string original = home + "/mine.lua";
        File.WriteAllText(original, "PLUGIN = {}");
        VfoxTask busy = model.Run(VfoxCommands.UpdateAllPlugins());
        VfoxTask task = model.AddPluginFromSource(original);
        string source = task.Command.Arguments[^1];
        Assert.True(File.Exists(source), "the copy waits for its turn");
        Assert.Equal(VfoxTaskStatus.Queued, task.Status);
        Assert.NotSame(busy, task);

        model.Dispose();
        Assert.False(File.Exists(source), "the copy must be deleted");
        Assert.True(File.Exists(original), "the original must survive");
    }

    private sealed class FailingStore : ISettingsStore
    {
        public AppSettings Load() => new();

        public void Save(AppSettings settings) => throw new IOException("disk full");
    }
}

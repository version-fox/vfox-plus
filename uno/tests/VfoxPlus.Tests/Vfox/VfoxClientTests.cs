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

using VfoxPlus.Core;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Tests.Support;
using Xunit;

namespace VfoxPlus.Tests.Vfox;

public class VfoxClientTests
{
    private static VfoxLaunchConfig TestConfig() => new()
    {
        Executable = "/usr/local/bin/vfox",
        BaseEnvironment = new Dictionary<string, string>
        {
            ["HOME"] = "/home/user",
            ["PATH"] = "/usr/bin",
            ["__VFOX_SHELL"] = "bash",
            ["__VFOX_PID"] = "42",
            ["__VFOX_CURTMPPATH"] = "/home/user/.vfox/tmp/x",
        },
        ExtraEnvironment = new Dictionary<string, string> { ["VFOX_NODEJS_MIRROR"] = "https://mirror", ["PATH"] = "/opt/bin" },
        NeutralDirectory = "/",
    };

    [Fact]
    public void TheEnvironmentDropsTheSessionAndLetsTheUserOverride()
    {
        VfoxLaunchConfig config = TestConfig();
        Dictionary<string, string> environment = VfoxClient.Environment(config, hookShell: false);
        Assert.DoesNotContain("__VFOX_SHELL", environment.Keys);
        Assert.DoesNotContain("__VFOX_PID", environment.Keys);
        Assert.DoesNotContain("__VFOX_CURTMPPATH", environment.Keys);
        Assert.Equal("1", environment["NO_COLOR"]);
        Assert.Equal("/home/user", environment["HOME"]);
        Assert.Equal("/opt/bin", environment["PATH"]);
        Assert.Equal("https://mirror", environment["VFOX_NODEJS_MIRROR"]);
        Assert.Equal("bash", VfoxClient.Environment(config, hookShell: true)["__VFOX_SHELL"]);
    }

    [Fact]
    public async Task QueriesRunInTheNeutralDirectory()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        VfoxClient client = new(runner, TestConfig());
        Assert.Equal("1.0.13", await client.VersionAsync());
        ProcessCall call = Assert.Single(runner.Invocations);
        Assert.Equal("/usr/local/bin/vfox", call.Executable);
        Assert.Equal("/", call.WorkingDirectory);
        Assert.False(call.Streamed);
        Assert.Equal("1", call.Environment["NO_COLOR"]);
    }

    [Fact]
    public async Task VersionRejectsGarbage()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures().On("--version", new ProcessOutput(0, "hello\n", string.Empty));
        VfoxClient client = new(runner, TestConfig());
        await Assert.ThrowsAsync<VfoxException>(client.VersionAsync);
    }

    [Fact]
    public async Task InstalledSdksMergesTheListAndTheCurrentVersions()
    {
        VfoxClient client = new(FakeProcessRunner.ForFixtures(), TestConfig());
        IReadOnlyList<InstalledSdk> got = await client.InstalledSdksAsync();
        Assert.Equal(["maven", "mvn2", "nodejs"], got.Select(sdk => sdk.Name));
        Assert.Equal(["3.9.16", "3.8.9"], got[0].Versions);
        Assert.Equal("3.9.16", got[0].Current);
        Assert.Empty(got[1].Versions);
        Assert.False(got[1].HasCurrent);
        Assert.Equal(["24.11.1", "22.21.0"], got[2].Versions);
        Assert.Equal("24.11.1", got[2].Current);
    }

    [Fact]
    public async Task NoInstalledPluginsIsNotAnError()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures()
            .On("list", Fixtures.Output("list-empty", 1))
            .On("current", Fixtures.Output("current-empty"));
        Assert.Empty(await new VfoxClient(runner, TestConfig()).InstalledSdksAsync());
    }

    [Fact]
    public async Task AFailingListIsAnErrorWhenPluginsAreInstalled()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures().On("list", new ProcessOutput(1, "boom\n", string.Empty));
        VfoxException error = await Assert.ThrowsAsync<VfoxException>(() => new VfoxClient(runner, TestConfig()).InstalledSdksAsync());
        Assert.Equal("boom", error.Message);
        Assert.Equal(1, error.ExitCode);
        Assert.Equal(["list"], error.Arguments);
    }

    [Fact]
    public async Task PluginInfoPassesTheTemplateAndRejectsOutputWithoutFields()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        VfoxClient client = new(runner, TestConfig());
        PluginInfo info = await client.PluginInfoAsync("nodejs");
        Assert.Equal("0.5.1", info.Version);
        Assert.Equal(["info", "nodejs", "--format"], runner.Invocations[0].Arguments.Take(3));
        await Assert.ThrowsAsync<VfoxException>(() => client.PluginInfoAsync("maven"));
    }

    [Fact]
    public async Task SearchAndAvailableReadTheirFixtures()
    {
        VfoxClient client = new(FakeProcessRunner.ForFixtures(), TestConfig());
        Assert.Contains(await client.SearchVersionsAsync("maven"), version => version.Version == "3.9.16");
        Assert.Equal(43, (await client.AvailablePluginsAsync()).Count);
    }

    [Fact]
    public async Task ConfigurationIsReadSetAndUnset()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures()
            .On("config proxy.url http://proxy:8080", new ProcessOutput(0, string.Empty, string.Empty))
            .On("config --unset proxy.url", new ProcessOutput(0, string.Empty, string.Empty));
        VfoxClient client = new(runner, TestConfig());
        Assert.Equal(10, (await client.ConfigEntriesAsync()).Count);
        await client.SetConfigAsync("proxy.url", "http://proxy:8080");
        await client.UnsetConfigAsync("proxy.url");
        Assert.Equal(
            ["config --list", "config proxy.url http://proxy:8080", "config --unset proxy.url"],
            runner.Calls("config").Select(call => call.CommandLine));
    }

    [Fact]
    public async Task AValueThatVfoxRejectsReportsItsReason()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures()
            .On("config cache.availableHookDuration notaduration", Fixtures.Output("config-set-invalid-duration", 1));
        VfoxException error = await Assert.ThrowsAsync<VfoxException>(() => new VfoxClient(runner, TestConfig()).SetConfigAsync("cache.availableHookDuration", "notaduration"));
        Assert.Contains("time: invalid duration \"notaduration\"", error.Message, StringComparison.Ordinal);
    }

    [Fact]
    public async Task AFailureWithoutOutputReportsTheExitCode()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures().On("config --unset proxy.url", new ProcessOutput(3, string.Empty, string.Empty));
        VfoxException error = await Assert.ThrowsAsync<VfoxException>(() => new VfoxClient(runner, TestConfig()).UnsetConfigAsync("proxy.url"));
        Assert.Equal("vfox exited with code 3", error.Message);
    }

    [Fact]
    public async Task InstallPathIsNullWhenTheVersionIsNotInstalled()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures()
            .On("info nodejs@24.11.1", Fixtures.Output("info-version-path"))
            .On("info nodejs@1.0.0", Fixtures.Output("info-version-not-installed"));
        VfoxClient client = new(runner, TestConfig());
        Assert.Equal("/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1", await client.InstallPathAsync("nodejs", "24.11.1"));
        Assert.Null(await client.InstallPathAsync("nodejs", "1.0.0"));
    }

    [Fact]
    public async Task AMissingExecutableIsReportedAsNotFound()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        runner.SetMissingExecutable(true);
        VfoxNotFoundException error = await Assert.ThrowsAsync<VfoxNotFoundException>(() => new VfoxClient(runner, TestConfig()).VersionAsync());
        Assert.Equal("/usr/local/bin/vfox", error.Executable);
    }

    [Fact]
    public void InstallBuildsAFullyDescribedCommand()
    {
        VfoxCommand command = VfoxCommands.Install("nodejs", "24.11.1");
        Assert.Equal(VfoxOperation.Install, command.Operation);
        Assert.Equal("nodejs@24.11.1", command.Subject);
        Assert.Equal("vfox install --yes nodejs@24.11.1", command.CommandLine);
        Assert.Equal(string.Empty, command.WorkingDirectory);
    }

    [Fact]
    public void UseNeedsAProjectDirectoryForTheProjectScope()
    {
        VfoxCommand global = VfoxCommands.Use("nodejs", "24.11.1", UseScope.Global);
        Assert.Equal(["use", "--global", "nodejs@24.11.1"], global.Arguments);
        Assert.True(global.RequiresHookEnvironment);

        Assert.Equal("/work/app", VfoxCommands.Use("java", "21", UseScope.Project, "/work/app").WorkingDirectory);
        Assert.Throws<VfoxUsageException>(() => VfoxCommands.Use("java", "21", UseScope.Project));
        Assert.Throws<VfoxUsageException>(() => VfoxCommands.Unuse("java", UseScope.Project, " "));
    }

    [Fact]
    public void TheOtherCommandsHaveTheirArguments()
    {
        Assert.Equal(["unuse", "--global", "java"], VfoxCommands.Unuse("java", UseScope.Global).Arguments);
        Assert.Equal(["uninstall", "java@21"], VfoxCommands.Uninstall("java", "21").Arguments);
        Assert.Equal(["remove", "--yes", "java"], VfoxCommands.RemovePlugin("java").Arguments);
        Assert.Equal(["update", "java"], VfoxCommands.UpdatePlugin("java").Arguments);
        VfoxCommand all = VfoxCommands.UpdateAllPlugins();
        Assert.Equal(["update", "--all"], all.Arguments);
        Assert.Equal(string.Empty, all.Subject);
    }

    [Fact]
    public void AddPluginTakesANameOrASource()
    {
        Assert.Equal(["add", "nodejs"], VfoxCommands.AddPlugin("nodejs").Arguments);
        VfoxCommand custom = VfoxCommands.AddPlugin(string.Empty, " https://x/p.zip ", "mine");
        Assert.Equal(["add", "--source", "https://x/p.zip", "--alias", "mine"], custom.Arguments);
        Assert.Equal("mine", custom.Sdk);
        Assert.Equal(string.Empty, VfoxCommands.AddPlugin(string.Empty, "/p.lua").Sdk);
        Assert.Throws<VfoxUsageException>(() => VfoxCommands.AddPlugin("  "));
    }

    [Theory]
    [InlineData("--all", "1")]
    [InlineData("java", "-1")]
    [InlineData("java2", "1 2")]
    [InlineData("java3", "a@b")]
    [InlineData("java4", "")]
    public void NamesAndVersionsThatLookLikeOptionsAreRejected(string name, string version)
    {
        Assert.Throws<VfoxUsageException>(() => VfoxCommands.Install(name, version));
    }

    [Fact]
    public async Task NamesAndKeysAreValidated()
    {
        Assert.True(VfoxCommands.IsValidPluginName("my-sdk.2"));
        Assert.False(VfoxCommands.IsValidPluginName("-x"));
        VfoxClient client = new(FakeProcessRunner.ForFixtures(), TestConfig());
        await Assert.ThrowsAsync<VfoxUsageException>(() => client.SetConfigAsync("proxy..url", "x"));
    }

    [Fact]
    public async Task StartRunsTheCommandInItsDirectoryWithTheShellMarked()
    {
        FakeProcessRunner runner = FakeProcessRunner.ForFixtures();
        VfoxClient client = new(runner, TestConfig());
        VfoxCommand command = VfoxCommands.Use("java", "21", UseScope.Project, "/work/app");
        IRunningProcess process = client.Start(command, new Discard());
        Assert.Equal(0, await process.WaitAsync());
        ProcessCall call = runner.Invocations[0];
        Assert.True(call.Streamed);
        Assert.Equal("/work/app", call.WorkingDirectory);
        Assert.Equal("bash", call.Environment["__VFOX_SHELL"]);

        runner.SetMissingExecutable(true);
        Assert.Throws<VfoxNotFoundException>(() => client.Start(VfoxCommands.UpdateAllPlugins(), new Discard()));
    }

    [Fact]
    public void TheLauncherChangesWhatTheNextInvocationUses()
    {
        VfoxClient client = new(FakeProcessRunner.ForFixtures(), TestConfig());
        client.LaunchConfig = client.LaunchConfig with { Executable = "/opt/vfox" };
        Assert.Equal("/opt/vfox", client.LaunchConfig.Executable);
    }

    [Fact]
    public void TheLocatorSearchesThePathThenTheUsualPlaces()
    {
        VfoxLocator locator = new(new Dictionary<string, string> { ["PATH"] = "/opt/bin::/usr/bin", ["HOME"] = "/home/me" }, _ => false);
        Assert.Equal(
            [
                "/opt/bin", "/usr/bin", "/usr/local/bin", "/home/me/.local/bin", "/home/me/bin", "/home/me/.linuxbrew/bin",
                "/home/linuxbrew/.linuxbrew/bin", "/snap/bin",
            ],
            locator.SearchDirectories);

        HashSet<string> found = ["/home/me/.local/bin/vfox", "/snap/bin/vfox"];
        Assert.Equal("/home/me/.local/bin/vfox", new VfoxLocator(new Dictionary<string, string> { ["HOME"] = "/home/me" }, found.Contains).Locate());
        Assert.Equal(string.Empty, new VfoxLocator(new Dictionary<string, string>(), _ => false).Locate());
    }

    [Fact]
    public void AnExecutableFileIsARegularFileThatAnybodyMayRun()
    {
        string directory = Directory.CreateTempSubdirectory("vfox-plus-test-").FullName;
        try
        {
            string script = Path.Combine(directory, "vfox");
            File.WriteAllText(script, "#!/bin/sh\n");
            Assert.False(VfoxLocator.IsExecutableFile(script));
            File.SetUnixFileMode(script, UnixFileMode.UserRead | UnixFileMode.UserWrite | UnixFileMode.UserExecute);
            Assert.True(VfoxLocator.IsExecutableFile(script));
            Assert.False(VfoxLocator.IsExecutableFile(directory));
            Assert.False(VfoxLocator.IsExecutableFile(Path.Combine(directory, "missing")));
        }
        finally
        {
            Directory.Delete(directory, recursive: true);
        }
    }

    private sealed class Discard : IProcessListener
    {
        public void Stdout(ReadOnlySpan<byte> chunk)
        {
        }

        public void Stderr(ReadOnlySpan<byte> chunk)
        {
        }
    }
}

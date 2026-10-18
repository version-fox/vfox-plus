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
using Tmds.DBus.Protocol;
using VfoxPlus.Platform;
using Xunit;

namespace VfoxPlus.Tests.Platform;

/// <summary>
/// Tests the portal against a desktop portal that is made up in the test: a private session bus (<c>dbus-daemon</c>)
/// with a service that answers as the real one does, so that the messages that go over the bus are checked.
/// </summary>
public sealed class PortalTests : IAsyncLifetime, IDisposable
{
    private const string PortalPath = "/org/freedesktop/portal/desktop";

    private readonly FakePortal fake = new();
    private Process? daemon;
    private DBusConnection? service;
    private string address = string.Empty;

    public async Task InitializeAsync()
    {
        ProcessStartInfo info = new("dbus-daemon") { RedirectStandardOutput = true, UseShellExecute = false };
        info.ArgumentList.Add("--session");
        info.ArgumentList.Add("--nofork");
        info.ArgumentList.Add("--nopidfile");
        info.ArgumentList.Add("--print-address=1");
        daemon = Process.Start(info) ?? throw new InvalidOperationException("dbus-daemon did not start");
        address = (await daemon.StandardOutput.ReadLineAsync().WaitAsync(TimeSpan.FromSeconds(10)))?.Trim() ?? string.Empty;
        Assert.StartsWith("unix:", address, StringComparison.Ordinal);

        service = new DBusConnection(address);
        await service.ConnectAsync();
        fake.Connection = service;
        service.AddMethodHandler(fake);
        await service.RequestNameAsync("org.freedesktop.portal.Desktop", RequestNameOptions.None);
    }

    public Task DisposeAsync() => Task.CompletedTask;

    public void Dispose()
    {
        service?.Dispose();
        if (daemon is { HasExited: false })
        {
            daemon.Kill();
            daemon.WaitForExit();
        }

        daemon?.Dispose();
    }

    [Fact]
    public async Task ReadsTheAppearanceOfTheDesktop()
    {
        fake.ColorScheme = 1;
        fake.Accent = (0.2, 0.4, 0.8);
        fake.Animations = false;
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Appearance appearance = await portal.ReadAppearanceAsync();

        Assert.Equal(new Appearance(Dark: true, Accent: 0xFF3366CC, ReducedMotion: true), appearance);
    }

    [Fact]
    public async Task ALightDesktopAndAnimationsThatAreOn()
    {
        fake.ColorScheme = 2;
        fake.Animations = true;
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Appearance appearance = await portal.ReadAppearanceAsync();

        Assert.Equal(new Appearance(Dark: false, Accent: null, ReducedMotion: false), appearance);
    }

    [Fact]
    public async Task WhatTheDesktopDoesNotSayIsLeftOut()
    {
        fake.ColorScheme = 0;
        fake.Accent = (-1, -1, -1);
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Appearance appearance = await portal.ReadAppearanceAsync();

        Assert.Equal(new Appearance(), appearance);
    }

    [Fact]
    public async Task AnOlderPortalAnswersReadInsteadOfReadOne()
    {
        fake.ReadOneMissing = true;
        fake.ColorScheme = 1;
        fake.Accent = (1, 0, 0.5);
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Appearance appearance = await portal.ReadAppearanceAsync();

        Assert.Equal(new Appearance(Dark: true, Accent: 0xFFFF0080, ReducedMotion: null), appearance);
    }

    [Fact]
    public async Task TellsWhenASettingChanges()
    {
        using Portal portal = (await Portal.ConnectAsync(address))!;
        TaskCompletionSource changed = new(TaskCreationOptions.RunContinuationsAsynchronously);
        await portal.WatchAsync(() => changed.TrySetResult());

        fake.EmitSettingChanged("org.freedesktop.appearance", "color-scheme");

        await changed.Task.WaitAsync(TimeSpan.FromSeconds(10));
    }

    [Fact]
    public async Task ChoosesAFileAndConvertsItsAddressToAPath()
    {
        fake.Answer = (0, "file:///tmp/my%20plugin.lua");
        string folder = Path.GetTempPath().TrimEnd('/');
        using Portal portal = (await Portal.ConnectAsync(address))!;

        string? path = await portal.ChooseAsync(new ChooseOptions("Source", folder, FilterLabel: "vfox plugins", Extensions: ["zip", "lua"]));

        Assert.Equal("/tmp/my plugin.lua", path);
        Assert.Equal("Source", fake.Title);
        Assert.True(fake.Options!["modal"].GetBool());
        Assert.False(fake.Options.ContainsKey("directory"));
        Assert.StartsWith("vfoxplus", fake.Options["handle_token"].GetString(), StringComparison.Ordinal);

        // The folder is a byte string with its terminating zero.
        byte[] current = fake.Options["current_folder"].GetArray<byte>();
        Assert.Equal(0, current[^1]);
        Assert.Equal(folder, System.Text.Encoding.UTF8.GetString(current[..^1]));

        // The filter: a name and the patterns of the extensions.
        VariantValue filters = fake.Options["filters"];
        Assert.Equal(1, filters.Count);
        VariantValue filter = filters.GetItem(0);
        Assert.Equal("vfox plugins", filter.GetItem(0).GetString());
        VariantValue patterns = filter.GetItem(1);
        Assert.Equal(2, patterns.Count);
        Assert.Equal("*.zip", patterns.GetItem(0).GetItem(1).GetString());
        Assert.Equal("*.lua", patterns.GetItem(1).GetItem(1).GetString());
    }

    [Fact]
    public async Task ChoosesADirectoryWithoutFilters()
    {
        fake.Answer = (0, "file:///home/me/project");
        using Portal portal = (await Portal.ConnectAsync(address))!;

        string? path = await portal.ChooseAsync(new ChooseOptions("Project", "/does/not/exist", Directory: true, Extensions: ["zip"]));

        Assert.Equal("/home/me/project", path);
        Assert.True(fake.Options!["directory"].GetBool());
        Assert.False(fake.Options.ContainsKey("filters"));
        Assert.False(fake.Options.ContainsKey("current_folder"));
    }

    [Fact]
    public async Task ACancelledChooserGivesAnEmptyPath()
    {
        fake.Answer = (1, string.Empty);
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Assert.Equal(string.Empty, await portal.ChooseAsync(new ChooseOptions("Source")));
    }

    [Fact]
    public async Task ADesktopWithoutAFileChooserGivesNoAnswer()
    {
        fake.Answer = null;
        using Portal portal = (await Portal.ConnectAsync(address))!;

        Assert.Null(await portal.ChooseAsync(new ChooseOptions("Source")));
    }

    [Fact]
    public async Task WithoutABusThereIsNoPortal()
    {
        Assert.Null(await Portal.ConnectAsync("unix:path=/nonexistent/vfox-plus-test-bus"));
        Assert.Null(await Portal.ConnectAsync(string.Empty));
    }

    /// <summary>A desktop portal that answers what the test made it say.</summary>
    private sealed class FakePortal : IPathMethodHandler
    {
        public DBusConnection? Connection { get; set; }

        public uint ColorScheme { get; set; }

        public (double Red, double Green, double Blue)? Accent { get; set; }

        public bool? Animations { get; set; }

        public bool ReadOneMissing { get; set; }

        // The answer of the file chooser: the code and the address; null for a portal without a chooser.
        public (uint Code, string Uri)? Answer { get; set; }

        public string? Title { get; private set; }

        public Dictionary<string, VariantValue>? Options { get; private set; }

        public string Path => PortalPath;

        public bool HandlesChildPaths => false;

        public ValueTask HandleMethodAsync(MethodContext context)
        {
            Message request = context.Request;
            string member = request.InterfaceAsString + "." + request.MemberAsString;
            switch (member)
            {
                case "org.freedesktop.portal.Settings.ReadOne" when !ReadOneMissing:
                    Reply(context, wrapped: false, request);
                    break;
                case "org.freedesktop.portal.Settings.Read":
                    Reply(context, wrapped: true, request);
                    break;
                case "org.freedesktop.portal.FileChooser.OpenFile" when Answer is not null:
                    OpenFile(context, request);
                    break;
                default:
                    context.ReplyUnknownMethodError();
                    break;
            }

            return ValueTask.CompletedTask;
        }

        public void EmitSettingChanged(string ns, string key)
        {
            using MessageWriter writer = Connection!.GetMessageWriter();
            writer.WriteSignalHeader(null, PortalPath, "org.freedesktop.portal.Settings", "SettingChanged", "ssv");
            writer.WriteString(ns);
            writer.WriteString(key);
            writer.WriteVariantUInt32(1);
            Connection.TrySendMessage(writer.CreateMessage());
        }

        // Answers a read of a setting: the value in a variant, which the first version of the interface wraps again.
        private void Reply(MethodContext context, bool wrapped, Message request)
        {
            Reader reader = request.GetBodyReader();
            string ns = reader.ReadString();
            string key = reader.ReadString();
            bool known = (ns, key) switch
            {
                ("org.freedesktop.appearance", "color-scheme") => true,
                ("org.freedesktop.appearance", "accent-color") => Accent is not null,
                ("org.gnome.desktop.interface", "enable-animations") => Animations is not null,
                _ => false,
            };
            if (!known)
            {
                context.ReplyError("org.freedesktop.portal.Error.NotFound", "No such setting");
                return;
            }

            using MessageWriter writer = context.CreateReplyWriter("v");
            if (wrapped)
            {
                writer.WriteSignature("v");
            }

            switch (key)
            {
                case "color-scheme":
                    writer.WriteVariantUInt32(ColorScheme);
                    break;
                case "accent-color":
                    writer.WriteSignature("(ddd)");
                    writer.WriteStructureStart();
                    writer.WriteDouble(Accent!.Value.Red);
                    writer.WriteDouble(Accent.Value.Green);
                    writer.WriteDouble(Accent.Value.Blue);
                    break;
                default:
                    writer.WriteVariantBool(Animations!.Value);
                    break;
            }

            context.Reply(writer.CreateMessage());
        }

        private void OpenFile(MethodContext context, Message request)
        {
            Reader reader = request.GetBodyReader();
            reader.ReadString();
            Title = reader.ReadString();
            Options = reader.ReadDictionaryOfStringToVariantValue();
            string sender = request.SenderAsString!.TrimStart(':').Replace('.', '_');
            string handle = $"{PortalPath}/request/{sender}/{Options["handle_token"].GetString()}";
            using (MessageWriter reply = context.CreateReplyWriter("o"))
            {
                reply.WriteObjectPath(handle);
                context.Reply(reply.CreateMessage());
            }

            // The answer comes as a signal on the request.
            using MessageWriter signal = Connection!.GetMessageWriter();
            signal.WriteSignalHeader(null, handle, "org.freedesktop.portal.Request", "Response", "ua{sv}");
            signal.WriteUInt32(Answer!.Value.Code);
            ArrayStart results = signal.WriteDictionaryStart();
            if (Answer.Value.Uri.Length > 0)
            {
                signal.WriteDictionaryEntryStart();
                signal.WriteString("uris");
                signal.WriteSignature("as");
                string[] uris = [Answer.Value.Uri];
                signal.WriteArray(uris);
            }

            signal.WriteDictionaryEnd(results);
            Connection.TrySendMessage(signal.CreateMessage());
        }
    }
}

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

using Tmds.DBus.Protocol;
using VfoxPlus.Design.Platform;

namespace VfoxPlus.Platform;

/// <summary>Talks to the XDG desktop portal on the session bus: the appearance of the desktop, and its file choosers.</summary>
public sealed class Portal : IDisposable
{
    private const string Destination = "org.freedesktop.portal.Desktop";
    private const string ObjectPath = "/org/freedesktop/portal/desktop";
    private static readonly TimeSpan CallTimeout = TimeSpan.FromSeconds(2);

    private readonly DBusConnection connection;
    private IDisposable? watch;

    private Portal(DBusConnection connection) => this.connection = connection;

    /// <summary>Connects to a bus. Returns <see langword="null"/> when there is none, as in a container without a desktop.</summary>
    /// <param name="busAddress">The address of the bus; the session bus by default.</param>
    /// <returns>The portal.</returns>
    public static async Task<Portal?> ConnectAsync(string? busAddress = null)
    {
        if ((busAddress ?? DBusAddress.Session) is not { Length: > 0 } address)
        {
            return null;
        }

        DBusConnection connection = new(address);
        try
        {
            using CancellationTokenSource timeout = new(CallTimeout);
            await Limit(connection.ConnectAsync(), timeout.Token).ConfigureAwait(false);
            return new Portal(connection);
        }
        catch (Exception error) when (error is DBusConnectionException or IOException or OperationCanceledException or InvalidOperationException or System.Net.Sockets.SocketException)
        {
            connection.Dispose();
            return null;
        }
    }

    /// <summary>Reads what the desktop says about how applications should look.</summary>
    /// <returns>The appearance.</returns>
    public async Task<Appearance> ReadAppearanceAsync()
    {
        bool? dark = null;
        uint? accent = null;
        bool? reduced = null;
        if (await ReadAsync("org.freedesktop.appearance", "color-scheme").ConfigureAwait(false) is { } scheme && TryGet(() => scheme.GetUInt32(), out uint code))
        {
            dark = PortalValues.ParseColorScheme(code);
        }

        if (await ReadAsync("org.freedesktop.appearance", "accent-color").ConfigureAwait(false) is { } color && TryGet(() => Components(color), out double[]? components))
        {
            accent = PortalValues.ParseAccent(components!);
        }

        if (await ReadAsync("org.gnome.desktop.interface", "enable-animations").ConfigureAwait(false) is { } animations && TryGet(() => animations.GetBool(), out bool enabled))
        {
            reduced = !enabled;
        }

        return new Appearance(dark, accent, reduced);
    }

    /// <summary>Calls an action whenever the portal reports a changed setting, until the portal is closed.</summary>
    /// <param name="onChange">The action; it runs on a thread of the pool.</param>
    /// <returns>A task that completes once the portal is watched.</returns>
    public async Task WatchAsync(Action onChange)
    {
        MatchRule rule = new()
        {
            Type = MessageType.Signal,
            Interface = "org.freedesktop.portal.Settings",
            Member = "SettingChanged",
            Path = ObjectPath,
        };
        watch = await connection.AddMatchAsync<int>(
            rule,
            static (Message message, object? state) => 0,
            (Exception? error, int value, object? readerState, object? handlerState) =>
            {
                if (error is null)
                {
                    onChange();
                }
            },
            ObserverFlags.None,
            readerState: null,
            handlerState: null,
            emitOnCapturedContext: false).ConfigureAwait(false);
    }

    /// <summary>
    /// Lets the user choose a file or a directory with the chooser of the desktop. It waits until the user answers.
    /// </summary>
    /// <param name="options">What to choose.</param>
    /// <returns>The path, an empty string if the user cancelled, or <see langword="null"/> when the desktop has no file chooser.</returns>
    public async Task<string?> ChooseAsync(ChooseOptions options)
    {
        string token = "vfoxplus" + DateTime.UtcNow.Ticks.ToString(System.Globalization.CultureInfo.InvariantCulture);
        string sender = (connection.UniqueName ?? string.Empty).TrimStart(':').Replace('.', '_');
        string request = $"/org/freedesktop/portal/desktop/request/{sender}/{token}";
        TaskCompletionSource<string> answer = new(TaskCreationOptions.RunContinuationsAsynchronously);
        MatchRule rule = new()
        {
            Type = MessageType.Signal,
            Interface = "org.freedesktop.portal.Request",
            Member = "Response",
            Path = request,
        };

        // The answer arrives as a signal on the request, which must be listened to before the request is made.
        using IDisposable match = await connection.AddMatchAsync<string>(
            rule,
            static (Message message, object? state) => ReadResponse(message),
            (Exception? error, string uri, object? readerState, object? handlerState) =>
            {
                if (error is not null)
                {
                    answer.TrySetException(error);
                }
                else
                {
                    answer.TrySetResult(uri);
                }
            },
            ObserverFlags.None,
            readerState: null,
            handlerState: null,
            emitOnCapturedContext: false).ConfigureAwait(false);

        try
        {
            MessageBuffer call;
            MessageWriter writer = connection.GetMessageWriter();
            try
            {
                writer.WriteMethodCallHeader(Destination, ObjectPath, "org.freedesktop.portal.FileChooser", "OpenFile", "ssa{sv}");
                writer.WriteString(string.Empty);
                writer.WriteString(options.Title);
                WriteOptions(ref writer, options, token);
                call = writer.CreateMessage();
            }
            finally
            {
                writer.Dispose();
            }

            await Limit(connection.CallMethodAsync(call), CallTimeout).ConfigureAwait(false);
        }
        catch (Exception error) when (error is DBusErrorReplyException or DBusConnectionException or TimeoutException or IOException)
        {
            return null;
        }

        string uri = await answer.Task.WaitAsync(TimeSpan.FromMinutes(30)).ConfigureAwait(false);
        return PathOf(uri);
    }

    /// <inheritdoc/>
    public void Dispose()
    {
        watch?.Dispose();
        connection.Dispose();
    }

    // The calls of the library return a task, or a value task, depending on its version.
    private static Task Limit(Task task, TimeSpan timeout) => task.WaitAsync(timeout);

    private static Task Limit(ValueTask task, TimeSpan timeout) => task.AsTask().WaitAsync(timeout);

    private static Task Limit(ValueTask task, CancellationToken cancellation) => task.AsTask().WaitAsync(cancellation);

    private static Task Limit(Task task, CancellationToken cancellation) => task.WaitAsync(cancellation);

    private static Task<T> Limit<T>(Task<T> task, TimeSpan timeout) => task.WaitAsync(timeout);

    private static Task<T> Limit<T>(ValueTask<T> task, TimeSpan timeout) => task.AsTask().WaitAsync(timeout);

    private static double[] Components(VariantValue value) =>
        [value.GetItem(0).GetDouble(), value.GetItem(1).GetDouble(), value.GetItem(2).GetDouble()];

    private static bool TryGet<T>(Func<T> get, out T value)
    {
        try
        {
            value = get();
            return true;
        }
        catch (Exception error) when (error is InvalidCastException or ArgumentException or InvalidOperationException or IndexOutOfRangeException)
        {
            value = default!;
            return false;
        }
    }

    // The answer of a request: the code (0 for a chosen file) and the first of the addresses that were chosen; an empty
    // text when the user cancelled.
    private static string ReadResponse(Message message)
    {
        Reader reader = message.GetBodyReader();
        uint code = reader.ReadUInt32();
        Dictionary<string, VariantValue> results = reader.ReadDictionaryOfStringToVariantValue();
        if (code != 0 || !results.TryGetValue("uris", out VariantValue uris))
        {
            return string.Empty;
        }

        string[] chosen = uris.GetArray<string>();
        return chosen.Length > 0 ? chosen[0] : string.Empty;
    }

    private static string PathOf(string uri) =>
        Uri.TryCreate(uri, UriKind.Absolute, out Uri? address) && address.IsFile ? address.LocalPath : uri;

    // A message writer is a struct that must not be copied, or what is written to the copy is lost.
    private static void WriteOptions(ref MessageWriter writer, ChooseOptions options, string token)
    {
        ArrayStart dictionary = writer.WriteDictionaryStart();
        writer.WriteDictionaryEntryStart();
        writer.WriteString("handle_token");
        writer.WriteVariantString(token);
        writer.WriteDictionaryEntryStart();
        writer.WriteString("modal");
        writer.WriteVariantBool(true);
        if (options.Directory)
        {
            writer.WriteDictionaryEntryStart();
            writer.WriteString("directory");
            writer.WriteVariantBool(true);
        }

        if (options.Initial.Length > 0 && Directory.Exists(options.Initial))
        {
            // The folder is a byte string with its terminating zero.
            writer.WriteDictionaryEntryStart();
            writer.WriteString("current_folder");
            writer.WriteSignature("ay");
            byte[] folder = [.. System.Text.Encoding.UTF8.GetBytes(options.Initial), 0];
            writer.WriteArray(folder);
        }

        if (!options.Directory && options.Extensions is { Count: > 0 } extensions)
        {
            // The filters: an array of a name and the patterns it has, each with a kind of pattern (0 for a glob).
            writer.WriteDictionaryEntryStart();
            writer.WriteString("filters");
            writer.WriteSignature("a(sa(us))");
            ArrayStart filters = writer.WriteArrayStart(DBusType.Struct);
            writer.WriteStructureStart();
            writer.WriteString(options.FilterLabel);
            ArrayStart patterns = writer.WriteArrayStart(DBusType.Struct);
            foreach (string extension in extensions)
            {
                writer.WriteStructureStart();
                writer.WriteUInt32(0);
                writer.WriteString("*." + extension);
            }

            writer.WriteArrayEnd(patterns);
            writer.WriteArrayEnd(filters);
        }

        writer.WriteDictionaryEnd(dictionary);
    }

    // Reads a setting of the portal: the version 2 of its interface has ReadOne, which returns the value, and the first
    // version has Read, which wraps it once more.
    private async Task<VariantValue?> ReadAsync(string ns, string key)
    {
        foreach (string member in new[] { "ReadOne", "Read" })
        {
            try
            {
                MessageBuffer call;
                using (MessageWriter writer = connection.GetMessageWriter())
                {
                    writer.WriteMethodCallHeader(Destination, ObjectPath, "org.freedesktop.portal.Settings", member, "ss");
                    writer.WriteString(ns);
                    writer.WriteString(key);
                    call = writer.CreateMessage();
                }

                VariantValue value = await Limit(
                    connection.CallMethodAsync(call, static (Message message, object? state) => message.GetBodyReader().ReadVariantValue(), null),
                    CallTimeout).ConfigureAwait(false);
                return member == "Read" ? value.GetVariantValue() : value;
            }
            catch (Exception error) when (error is DBusErrorReplyException or DBusConnectionException or TimeoutException or IOException or InvalidOperationException)
            {
                // The setting does not exist, or the portal is older: try the next way.
            }
        }

        return null;
    }
}

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
using VfoxPlus.Platform;
using Windows.ApplicationModel.DataTransfer;

namespace VfoxPlus.App.Services;

/// <summary>What the desktop offers the application: its appearance, links, the clipboard and the file choosers.</summary>
internal interface IPlatform
{
    /// <summary>Raised when the appearance of the desktop changed.</summary>
    event Action? AppearanceChanged;

    /// <summary>Gets what the desktop says about how applications should look.</summary>
    Appearance Appearance { get; }

    /// <summary>Opens a web address in the default browser of the desktop.</summary>
    /// <param name="address">The address.</param>
    void OpenUrl(string address);

    /// <summary>Puts text on the clipboard.</summary>
    /// <param name="text">The text.</param>
    void Copy(string text);

    /// <summary>Lets the user choose a file or a directory.</summary>
    /// <param name="options">What to choose.</param>
    /// <returns>The path, or an empty string if the user cancelled.</returns>
    /// <exception cref="PlatformException">There is no file chooser at all.</exception>
    Task<string> ChooseAsync(ChooseOptions options);
}

/// <summary>Reports that the desktop cannot do what is asked.</summary>
internal sealed class PlatformException : Exception
{
    public PlatformException(string message)
        : base(message)
    {
    }
}

/// <summary>The desktop of Linux: the XDG desktop portal where it can, and the programs that desktops have otherwise.</summary>
internal sealed class LinuxPlatform : IPlatform
{
    private Portal? portal;
    private Appearance appearance = new();

    public LinuxPlatform() => _ = Task.Run(ConnectAsync);

    /// <inheritdoc/>
    public event Action? AppearanceChanged;

    /// <inheritdoc/>
    public Appearance Appearance => Volatile.Read(ref appearance);

    /// <inheritdoc/>
    public void OpenUrl(string address)
    {
        try
        {
            Process.Start(new ProcessStartInfo("xdg-open") { ArgumentList = { address }, UseShellExecute = false })?.Dispose();
        }
        catch (Exception error) when (error is System.ComponentModel.Win32Exception or InvalidOperationException)
        {
            // Without xdg-open there is nothing to open the address with.
        }
    }

    /// <inheritdoc/>
    public void Copy(string text)
    {
        DataPackage package = new();
        package.SetText(text);
        Clipboard.SetContent(package);
        Clipboard.Flush();
    }

    /// <inheritdoc/>
    public async Task<string> ChooseAsync(ChooseOptions options)
    {
        // The chooser of the portal, if the desktop has one, and else a program that desktops without one usually have.
        if (portal is not null && await portal.ChooseAsync(options).ConfigureAwait(false) is { } path)
        {
            return path;
        }

        return await ChooseWithProgramAsync(options).ConfigureAwait(false);
    }

    // Connects to the portal in the background: the desktop answers when it does, and until then the defaults apply.
    private async Task ConnectAsync()
    {
        Portal? connected = await Portal.ConnectAsync().ConfigureAwait(false);
        if (connected is null)
        {
            return;
        }

        portal = connected;
        await RefreshAsync().ConfigureAwait(false);
        await connected.WatchAsync(() => _ = RefreshAsync()).ConfigureAwait(false);
    }

    private async Task RefreshAsync()
    {
        if (portal is null)
        {
            return;
        }

        Appearance read = await portal.ReadAppearanceAsync().ConfigureAwait(false);
        if (read != appearance)
        {
            Volatile.Write(ref appearance, read);
            AppearanceChanged?.Invoke();
        }
    }

    // zenity or kdialog, which the desktop environments that lack a file chooser portal usually have.
    private static async Task<string> ChooseWithProgramAsync(ChooseOptions options)
    {
        string? zenity = Find("zenity");
        string? kdialog = zenity is null ? Find("kdialog") : null;
        if (zenity is null && kdialog is null)
        {
            throw new PlatformException("No file chooser is available.");
        }

        ProcessStartInfo info = new(zenity ?? kdialog!) { UseShellExecute = false, RedirectStandardOutput = true };
        string[] patterns = [.. (options.Extensions ?? []).Select(extension => "*." + extension)];
        if (zenity is not null)
        {
            info.ArgumentList.Add("--file-selection");
            info.ArgumentList.Add("--title=" + options.Title);
            if (options.Directory)
            {
                info.ArgumentList.Add("--directory");
            }
            else if (patterns.Length > 0)
            {
                info.ArgumentList.Add($"--file-filter={options.FilterLabel} | {string.Join(' ', patterns)}");
            }

            if (options.Initial.Length > 0)
            {
                info.ArgumentList.Add("--filename=" + options.Initial.TrimEnd('/') + "/");
            }
        }
        else
        {
            string start = options.Initial.Length > 0 ? options.Initial : ".";
            info.ArgumentList.Add("--title");
            info.ArgumentList.Add(options.Title);
            info.ArgumentList.Add(options.Directory ? "--getexistingdirectory" : "--getopenfilename");
            info.ArgumentList.Add(start);
            if (!options.Directory && patterns.Length > 0)
            {
                info.ArgumentList.Add($"{string.Join(' ', patterns)}|{options.FilterLabel}");
            }
        }

        using Process process = Process.Start(info)!;
        string output = await process.StandardOutput.ReadToEndAsync().ConfigureAwait(false);
        await process.WaitForExitAsync().ConfigureAwait(false);

        // A cancelled chooser exits with a non-zero code.
        return process.ExitCode == 0 ? output.Trim() : string.Empty;
    }

    private static string? Find(string program)
    {
        string path = Environment.GetEnvironmentVariable("PATH") ?? string.Empty;
        return path.Split(':', StringSplitOptions.RemoveEmptyEntries)
            .Select(directory => Path.Combine(directory, program))
            .FirstOrDefault(File.Exists);
    }
}

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

namespace VfoxPlus.App.Services;

/// <summary>A top-level page of the navigation.</summary>
internal enum Destination
{
    /// <summary>The overview.</summary>
    Home,

    /// <summary>The installed plugins and their versions.</summary>
    Sdks,

    /// <summary>The plugin registry.</summary>
    Plugins,

    /// <summary>The running and finished operations.</summary>
    Tasks,

    /// <summary>The preferences.</summary>
    Settings,
}

/// <summary>Where the user is.</summary>
internal sealed class Navigation
{
    /// <summary>Raised when the page or the selected plugin changed.</summary>
    public event Action? Changed;

    /// <summary>Gets the page that shows.</summary>
    public Destination Current { get; private set; }

    /// <summary>Gets the plugin whose versions the SDKs page shows, or an empty string for none.</summary>
    public string SelectedSdk { get; private set; } = string.Empty;

    /// <summary>Opens a page.</summary>
    /// <param name="destination">The page.</param>
    public void Go(Destination destination)
    {
        if (Current != destination)
        {
            Current = destination;
            Changed?.Invoke();
        }
    }

    /// <summary>Opens the SDKs page with a plugin selected, or without any.</summary>
    /// <param name="name">The plugin, or an empty string.</param>
    public void ShowSdk(string name)
    {
        Current = Destination.Sdks;
        SelectedSdk = name;
        Changed?.Invoke();
    }

    /// <summary>Selects a plugin on the SDKs page, or none.</summary>
    /// <param name="name">The plugin, or an empty string.</param>
    public void SelectSdk(string name)
    {
        if (SelectedSdk != name)
        {
            SelectedSdk = name;
            Changed?.Invoke();
        }
    }
}

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

using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Services;
using VfoxPlus.App.Shell;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Settings;
using VfoxPlus.State;
using Windows.Foundation;
using Windows.Graphics;
using Windows.UI.ViewManagement;

namespace VfoxPlus.App;

/// <summary>The application.</summary>
public partial class App : Application
{
    private Window? window;

    /// <summary>Initializes a new instance of the <see cref="App"/> class.</summary>
    public App() => InitializeComponent();

    /// <inheritdoc/>
    protected override void OnLaunched(LaunchActivatedEventArgs args)
    {
        window = new Window { Title = "vfox+" };
        window.AppWindow.Resize(new SizeInt32 { Width = 1280, Height = 820 });
        ApplicationView.GetForCurrentView().SetPreferredMinSize(new Size(360, 480));
        UiThread dispatch = new(window.DispatcherQueue);
        Dictionary<string, string> environment = [];
        foreach (System.Collections.DictionaryEntry entry in Environment.GetEnvironmentVariables())
        {
            environment[(string)entry.Key] = (string?)entry.Value ?? string.Empty;
        }

        string home = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
        AppModel model = new(new AppModelOptions(
            new OsProcessRunner(),
            new JsonFileSettingsStore(JsonFileSettingsStore.SettingsPath(environment, home)),
            environment));
        string assets = Path.Combine(AppContext.BaseDirectory, "Assets", "Generated");
        Env env = new(model, new LinuxPlatform(), dispatch, environment, assets);
        window.Title = env.T("appTitle");
        env.L10n.Changed += () => dispatch.Post(() => window.Title = env.T("appTitle"));
        AppShell shell = new();
        OverlayHost host = new(shell);
        env.Overlays = host;
        window.Content = host;

        // The size of a window is in pixels of the screen, and the scale of the display is only known once the content
        // is loaded: the window is 1280 × 820 dp, as in the other implementations.
        RoutedEventHandler? resize = null;
        resize = (_, _) =>
        {
            host.Loaded -= resize;
            double scale = host.XamlRoot?.RasterizationScale ?? 1;
            if (Math.Abs(scale - 1) > 0.01)
            {
                window.AppWindow.Resize(new SizeInt32 { Width = (int)Math.Round(1280 * scale), Height = (int)Math.Round(820 * scale) });
            }
        };
        host.Loaded += resize;
        _ = model.StartAsync();

#if E2E
        E2E.Agent.Start(window, env, host);
#endif

        window.Activate();
    }
}

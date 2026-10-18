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
using VfoxPlus.Design.Shapes;
using VfoxPlus.Platform;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>Tells the user that vfox cannot be used, and what to do about it.</summary>
internal sealed class UnavailablePage : PageBase
{
    private readonly Border pane;

    public UnavailablePage()
    {
        pane = PageParts.Pane(new Grid());
        Children.Add(pane);
        Watch(Env.Model.Vfox);
    }

    /// <inheritdoc/>
    public override void Refresh()
    {
        VfoxController vfox = Env.Model.Vfox;
        UIElement view;
        switch (vfox.Availability)
        {
            case VfoxAvailability.NotFound or VfoxAvailability.Broken:
                bool missing = vfox.Availability == VfoxAvailability.NotFound;
                string executable = vfox.Executable ?? "vfox";
                M3Button guide = new() { Kind = ButtonKind.Filled, Symbol = "openInNew", Label = T("vfoxInstallGuide") };
                guide.Click += (_, _) => Env.OpenUrl(Common.VfoxInstall);
                M3Button locate = new() { Kind = ButtonKind.Tonal, Symbol = "folder", Label = T("vfoxLocate") };
                locate.Click += (_, _) => Env.Choose(
                    new ChooseOptions(T("vfoxLocate"), "/usr/local/bin"),
                    path => Env.Model.Settings.Update(settings => settings with { VfoxExecutable = path }));
                M3Button retry = new() { Kind = ButtonKind.Outlined, Symbol = "retry", Label = T("actionRetry") };
                retry.Click += (_, _) => _ = Env.Model.Vfox.ApplyAsync(Env.Model.Settings.Settings);
                UIElement[] actions = [guide, locate, retry];
                view = missing
                    ? new StatusMessage("notFound", T("vfoxMissingTitle"), T("vfoxMissingBody"), shape: MaterialShapes.Clover4, actions: actions)
                    : new StatusMessage("warning", T("vfoxBrokenTitle"), T("vfoxBrokenBody", ("path", executable)), isError: true, shape: MaterialShapes.SoftBurst, actions: actions);
                break;
            default:
                view = Common.LoadingView(T("vfoxCheckingTitle"));
                break;
        }

        pane.Child = view;
    }
}

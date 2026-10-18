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

using Uno.UI.Hosting;

namespace VfoxPlus.App;

internal static class Program
{
    [STAThread]
    public static void Main()
    {
        // Skia draws with the processor. The user interface is small, and the graphics stack that Uno would otherwise
        // load (Mesa, with LLVM for a software GL) takes more memory than the application does: 165 MB instead of 295 MB
        // on the Home page, in the sandbox, which has no GPU. See docs/architecture.md.
        Uno.UI.FeatureConfiguration.Rendering.UseOpenGLOnX11 = false;
        Uno.UI.FeatureConfiguration.Rendering.UseVulkanOnX11 = false;

        UnoPlatformHost host = UnoPlatformHostBuilder.Create()
            .App(() => new App())
            .UseX11()
            .Build();
        host.Run();
    }
}

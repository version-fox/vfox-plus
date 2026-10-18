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

namespace VfoxPlus.Tests.Support;

/// <summary>Where the shared resources of the repository are.</summary>
internal static class TestPaths
{
    /// <summary>Gets the <c>shared</c> directory of the repository.</summary>
    public static string SharedDirectory { get; } = FindShared();

    private static string FindShared()
    {
        DirectoryInfo? directory = new(AppContext.BaseDirectory);
        while (directory is not null)
        {
            string candidate = Path.Combine(directory.FullName, "shared", "fixtures");
            if (Directory.Exists(candidate))
            {
                return Path.Combine(directory.FullName, "shared");
            }

            directory = directory.Parent;
        }

        throw new DirectoryNotFoundException("The shared directory of the repository is not above " + AppContext.BaseDirectory);
    }
}

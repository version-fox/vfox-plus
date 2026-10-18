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

namespace VfoxPlus.Platform;

/// <summary>What the desktop says about how applications should look. A value is <see langword="null"/> when the desktop does not say.</summary>
/// <param name="Dark">Whether the desktop prefers a dark appearance.</param>
/// <param name="Accent">The accent colour of the desktop as ARGB.</param>
/// <param name="ReducedMotion">Whether the user asked for less motion.</param>
public sealed record Appearance(bool? Dark = null, uint? Accent = null, bool? ReducedMotion = null);

/// <summary>What the user chooses.</summary>
/// <param name="Title">The title of the chooser.</param>
/// <param name="Initial">The directory the chooser starts in.</param>
/// <param name="Directory">Whether to choose a directory instead of a file.</param>
/// <param name="FilterLabel">The name of the files shown, such as "vfox plugins".</param>
/// <param name="Extensions">The extensions of the files shown, such as <c>zip</c> and <c>lua</c>.</param>
public sealed record ChooseOptions(string Title, string Initial = "", bool Directory = false, string FilterLabel = "", IReadOnlyList<string>? Extensions = null);

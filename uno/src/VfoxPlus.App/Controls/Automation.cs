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

using Microsoft.UI.Xaml.Automation;

namespace VfoxPlus.App.Controls;

/// <summary>
/// What the user interface tells assistive technologies and the tests about its elements: a name, and whether an
/// element is selected.
/// </summary>
internal static class Automation
{
    /// <summary>Names an element.</summary>
    /// <param name="element">The element.</param>
    /// <param name="name">The name.</param>
    public static void Name(DependencyObject element, string name) => AutomationProperties.SetName(element, name);

    /// <summary>Returns the name of an element.</summary>
    /// <param name="element">The element.</param>
    /// <returns>The name.</returns>
    public static string NameOf(DependencyObject element) => AutomationProperties.GetName(element);

    /// <summary>Tells whether an element is selected.</summary>
    /// <param name="element">The element.</param>
    /// <param name="selected">Whether it is.</param>
    public static void Select(DependencyObject element, bool selected) =>
        AutomationProperties.SetItemStatus(element, selected ? "selected" : string.Empty);

    /// <summary>Returns whether an element is selected.</summary>
    /// <param name="element">The element.</param>
    /// <returns><see langword="true"/> if it is.</returns>
    public static bool IsSelected(DependencyObject element) => AutomationProperties.GetItemStatus(element) == "selected";
}

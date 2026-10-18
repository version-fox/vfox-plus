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

using System.Runtime.CompilerServices;
using Microsoft.UI.Xaml.Input;
using VfoxPlus.App.Controls.Overlays;

namespace VfoxPlus.App.Controls;

/// <summary>
/// The tooltips of the application: a plain tooltip of Material 3 below an element, which shows when the pointer
/// rests on the element for half a second.
/// </summary>
internal static class Tip
{
    private static readonly TimeSpan Delay = TimeSpan.FromMilliseconds(500);
    private static readonly ConditionalWeakTable<FrameworkElement, State> States = [];

    /// <summary>Gives an element a tooltip.</summary>
    /// <param name="element">The element.</param>
    /// <param name="text">The tooltip, or an empty text for none.</param>
    public static void SetText(FrameworkElement element, string text)
    {
        if (!States.TryGetValue(element, out State? state))
        {
            if (text.Length == 0)
            {
                return;
            }

            state = new State(element);
            States.Add(element, state);
        }

        state.Text = text;
    }

    private sealed class State
    {
        private readonly FrameworkElement element;
        private readonly DispatcherTimer timer = new() { Interval = Delay };

        public State(FrameworkElement element)
        {
            this.element = element;
            timer.Tick += (_, _) =>
            {
                timer.Stop();
                if (Text.Length > 0)
                {
                    OverlayHost.Current?.ShowTooltip(element, Text);
                }
            };
            element.PointerEntered += (_, _) => timer.Start();
            element.PointerExited += (_, _) => Hide();

            // A press is handled by the control, which does not stop the tooltip from going.
            element.AddHandler(UIElement.PointerPressedEvent, new PointerEventHandler((_, _) => Hide()), handledEventsToo: true);
            element.Unloaded += (_, _) => Hide();
        }

        public string Text { get; set; } = string.Empty;

        private void Hide()
        {
            timer.Stop();
            OverlayHost.Current?.HideTooltip(element);
        }
    }
}

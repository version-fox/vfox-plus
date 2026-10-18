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
using VfoxPlus.App.Controls;
using VfoxPlus.App.Controls.Overlays;
using Windows.Foundation;

namespace VfoxPlus.App.E2E;

/// <summary>Reads what is on the screen from the tree of the user interface: what a scenario sees.</summary>
internal static class Tree
{
    /// <summary>
    /// Collects the nodes that the user can reach: those of the dialog or the menu that is open, which hold the rest
    /// back, or else all the others. It must run on the thread of the window.
    /// </summary>
    /// <param name="host">The root of the window.</param>
    /// <returns>The nodes, in the order of the tree.</returns>
    public static List<Node> Collect(OverlayHost host)
    {
        host.UpdateLayout();
        List<Node> nodes = [];
        if (host.ModalScope is { } scope)
        {
            if (!host.HasMenu && host.TopDialog is { } dialog)
            {
                nodes.Add(new Node(NodeRole.Dialog, dialog.Title, string.Empty, true, RectOf(host, scope)));
            }

            Walk(scope, host, nodes);
        }
        else
        {
            Walk(host, host, nodes);
        }

        return nodes;
    }

    private static void Walk(DependencyObject element, OverlayHost host, List<Node> nodes)
    {
        if (element is UIElement shown && shown.Visibility != Visibility.Visible)
        {
            return;
        }

        switch (element)
        {
            // Icons are drawn with a font: their text is not a text of the user interface. A text box is read through
            // the field around it.
            case M3Icon or TextBox:
                return;
            case ViewSwitcher switcher:
                if (switcher.Current is { } view)
                {
                    Walk(view, host, nodes);
                }

                return;
            case M3TextField field:
                nodes.Add(new Node(NodeRole.TextField, field.AccessibleName, field.Text, field.IsEnabled, RectOf(host, field), null, field.Fill));
                break;
            case M3SearchBar bar:
                nodes.Add(new Node(NodeRole.TextField, bar.AccessibleName, bar.Text, bar.IsEnabled, RectOf(host, bar), null, (text, _) => bar.Fill(text)));
                break;
            case ListItem item:
                nodes.Add(new Node(NodeRole.ListItem, AutomationProperties.GetName(item), string.Empty, item.IsEnabled, RectOf(host, item), item.Interactive ? item.Invoke : null));
                break;
            case M3Card { Interactive: true } card:
                nodes.Add(new Node(NodeRole.Button, card.CardName, string.Empty, card.IsEnabled, RectOf(host, card), card.Invoke));
                break;
            case Pressable pressable:
                nodes.Add(new Node(RoleOf(pressable), AutomationProperties.GetName(pressable), string.Empty, pressable.IsEnabled, RectOf(host, pressable), pressable.Invoke));
                break;
            case TextBlock { Text.Length: > 0 } block:
                nodes.Add(new Node(NodeRole.Text, string.Empty, block.Text, true, RectOf(host, block)));
                return;
            default:
                break;
        }

        int count = VisualTreeHelper.GetChildrenCount(element);
        for (int i = 0; i < count; i++)
        {
            Walk(VisualTreeHelper.GetChild(element, i), host, nodes);
        }
    }

    private static NodeRole RoleOf(Pressable pressable) => pressable.Role switch
    {
        "switch" => NodeRole.Switch,
        "menuitem" => NodeRole.MenuItem,
        "item" => NodeRole.ListItem,
        _ => NodeRole.Button,
    };

    private static Rect RectOf(OverlayHost host, FrameworkElement element) =>
        element.TransformToVisual(host).TransformBounds(new Rect(0, 0, element.ActualWidth, element.ActualHeight));
}

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

namespace VfoxPlus.App.Controls;

/// <summary>
/// A long list of rows that are made only for the part of the list that shows: the rows are recycled while the list
/// scrolls, and bound to the item they show. The list belongs in a scroll viewer, whose viewport it follows.
/// </summary>
/// <typeparam name="TItem">What a row shows.</typeparam>
internal sealed class VirtualRows<TItem>
    where TItem : class
{
    private readonly Action<FrameworkElement, TItem, ListPosition> bind;
    private readonly Func<TItem, string> key;
    private readonly Dictionary<FrameworkElement, int> shown = [];
    private List<TItem> items = [];

    public VirtualRows(Func<FrameworkElement> create, Action<FrameworkElement, TItem, ListPosition> bind, Func<TItem, string> key, double spacing)
    {
        this.bind = bind;
        this.key = key;
        Repeater = new ItemsRepeater
        {
            Layout = new StackLayout { Orientation = Orientation.Vertical, Spacing = spacing },
            ItemTemplate = new DataTemplate(() => create()),
        };
        Repeater.ElementPrepared += (_, e) =>
        {
            FrameworkElement row = (FrameworkElement)e.Element;
            shown[row] = e.Index;
            Bind(row, e.Index);
        };
        Repeater.ElementClearing += (_, e) => shown.Remove((FrameworkElement)e.Element);
    }

    /// <summary>Gets the element that draws the rows.</summary>
    public ItemsRepeater Repeater { get; }

    /// <summary>Gets the rows that exist now, which the tests look into.</summary>
    public IReadOnlyCollection<FrameworkElement> Rows => shown.Keys;

    /// <summary>Gets the number of items.</summary>
    public int Count => items.Count;

    /// <summary>
    /// Shows other items. When they are the same ones, in the same order, the rows only show them again, and the list
    /// keeps its place; otherwise it starts over.
    /// </summary>
    /// <param name="newItems">The items.</param>
    public void SetItems(IReadOnlyList<TItem> newItems)
    {
        bool same = items.Count == newItems.Count;
        for (int i = 0; same && i < items.Count; i++)
        {
            same = key(items[i]) == key(newItems[i]);
        }

        items = [.. newItems];
        if (same)
        {
            foreach ((FrameworkElement row, int index) in shown.ToArray())
            {
                Bind(row, index);
            }
        }
        else
        {
            Repeater.ItemsSource = items;
        }
    }

    private void Bind(FrameworkElement row, int index)
    {
        if (index < 0 || index >= items.Count)
        {
            return;
        }

        ListPosition position = items.Count <= 1 ? ListPosition.Single : index == 0 ? ListPosition.First : index == items.Count - 1 ? ListPosition.Last : ListPosition.Middle;
        bind(row, items[index], position);
    }
}

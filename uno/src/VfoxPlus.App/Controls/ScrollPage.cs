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
/// The body of a page: a top part (the header and the toolbar), and either content that scrolls together with the
/// top part, or a message (a list that is loading, empty or failed) that takes the space under it.
/// </summary>
internal sealed class ScrollPage : Grid
{
    private readonly ScrollViewer scroll = new() { VerticalScrollBarVisibility = ScrollBarVisibility.Auto, HorizontalScrollBarVisibility = ScrollBarVisibility.Disabled };
    private readonly StackPanel stack = new() { Orientation = Orientation.Vertical };
    private readonly Grid fixedLayout = new() { Visibility = Visibility.Collapsed };
    private UIElement? body;
    private UIElement? message;

    public ScrollPage()
    {
        fixedLayout.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        fixedLayout.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        scroll.Content = stack;
        stack.Children.Add(Top);
        Children.Add(scroll);
        Children.Add(fixedLayout);
    }

    /// <summary>Gets the top part, which the page fills with its header and its toolbar.</summary>
    public StackPanel Top { get; } = new() { Orientation = Orientation.Vertical };

    /// <summary>Gets the scroll viewer of the content.</summary>
    public ScrollViewer Scroller => scroll;

    /// <summary>Shows content under the top part; both scroll.</summary>
    /// <param name="content">The content.</param>
    public void ShowContent(UIElement content)
    {
        if (!stack.Children.Contains(Top))
        {
            fixedLayout.Children.Remove(Top);
            stack.Children.Insert(0, Top);
        }

        if (!ReferenceEquals(body, content))
        {
            if (body is not null)
            {
                stack.Children.Remove(body);
            }

            body = content;
            stack.Children.Add(content);
        }

        scroll.Visibility = Visibility.Visible;
        fixedLayout.Visibility = Visibility.Collapsed;
    }

    /// <summary>Shows a message under the top part, which stays where it is.</summary>
    /// <param name="view">The message.</param>
    public void ShowMessage(UIElement view)
    {
        if (!fixedLayout.Children.Contains(Top))
        {
            stack.Children.Remove(Top);
            Grid.SetRow(Top, 0);
            fixedLayout.Children.Add(Top);
        }

        if (!ReferenceEquals(message, view))
        {
            if (message is not null)
            {
                fixedLayout.Children.Remove(message);
            }

            message = view;
            Grid.SetRow(view, 1);
            fixedLayout.Children.Add(view);
        }

        scroll.Visibility = Visibility.Collapsed;
        fixedLayout.Visibility = Visibility.Visible;
    }
}

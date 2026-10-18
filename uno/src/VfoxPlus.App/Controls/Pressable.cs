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

using Microsoft.UI.Xaml.Input;
using Windows.System;

namespace VfoxPlus.App.Controls;

/// <summary>
/// An area that reacts to the pointer and the keyboard: a button, a list item, a switch. It tells its state (hovered,
/// pressed, focused with the keyboard) to the control that draws it, and raises <see cref="Click"/> for a click, the
/// Enter key and the Space bar.
/// </summary>
internal abstract class Pressable : UserControl
{
    private bool hovered;
    private bool pressed;
    private bool keyboardFocus;
    private bool spacePressed;

    protected Pressable()
    {
        IsTabStop = true;
        UseSystemFocusVisuals = false;
        PointerEntered += (_, _) => Set(ref hovered, true);
        PointerExited += (_, _) => Set(ref hovered, false);
        PointerPressed += OnPointerPressed;
        PointerReleased += OnPointerReleased;
        PointerCanceled += (_, _) => Release();
        PointerCaptureLost += (_, _) => Release();
        GotFocus += (_, _) => Set(ref keyboardFocus, FocusState == FocusState.Keyboard);
        LostFocus += (_, _) => Set(ref keyboardFocus, false);
        KeyDown += OnKeyDown;
        KeyUp += OnKeyUp;
        IsEnabledChanged += (_, _) =>
        {
            if (!IsEnabled)
            {
                hovered = false;
                pressed = false;
                spacePressed = false;
                keyboardFocus = false;
            }

            StateChanged();
        };
    }

    /// <summary>Raised when the area is clicked, by the pointer or the keyboard.</summary>
    public event EventHandler? Click;

    /// <summary>Gets what the area is for, in the words of the tests: <c>button</c>, <c>switch</c>, <c>item</c>.</summary>
    public virtual string Role => "button";

    /// <summary>Gets a value indicating whether the pointer is over the area.</summary>
    protected bool IsHovered => hovered;

    /// <summary>Gets a value indicating whether the area is held down, by the pointer or the Space bar.</summary>
    protected bool IsPressed => pressed || spacePressed;

    /// <summary>Gets a value indicating whether the keyboard focus is on the area, which is when it shows a ring.</summary>
    protected bool HasKeyboardFocus => keyboardFocus;

    /// <summary>Clicks the area as a click of the user would, which is how the tests operate it.</summary>
    public void Invoke()
    {
        if (IsEnabled)
        {
            OnClicked();
            Click?.Invoke(this, EventArgs.Empty);
        }
    }

    /// <summary>Called when the area is clicked, before <see cref="Click"/> is raised.</summary>
    protected virtual void OnClicked()
    {
    }

    /// <summary>Called when the state of the area changed: it is hovered, pressed, focused or enabled.</summary>
    protected virtual void StateChanged()
    {
    }

    private static bool IsActivationKey(VirtualKey key) => key is VirtualKey.Enter or VirtualKey.Space;

    private void Set(ref bool field, bool value)
    {
        if (field != value)
        {
            field = value;
            StateChanged();
        }
    }

    private void OnPointerPressed(object sender, PointerRoutedEventArgs e)
    {
        if (!IsEnabled || !e.GetCurrentPoint(this).Properties.IsLeftButtonPressed)
        {
            return;
        }

        CapturePointer(e.Pointer);
        e.Handled = true;
        Focus(FocusState.Pointer);
        Set(ref pressed, true);
    }

    private void OnPointerReleased(object sender, PointerRoutedEventArgs e)
    {
        bool wasPressed = pressed;
        ReleasePointerCapture(e.Pointer);
        e.Handled = true;
        Release();
        if (wasPressed && IsInside(e.GetCurrentPoint(this).Position))
        {
            Invoke();
        }
    }

    private void Release() => Set(ref pressed, false);

    private bool IsInside(Windows.Foundation.Point point) =>
        point.X >= 0 && point.Y >= 0 && point.X <= ActualWidth && point.Y <= ActualHeight;

    private void OnKeyDown(object sender, KeyRoutedEventArgs e)
    {
        if (!IsEnabled || !IsActivationKey(e.Key) || e.KeyStatus.WasKeyDown)
        {
            return;
        }

        e.Handled = true;
        if (e.Key == VirtualKey.Enter)
        {
            Invoke();
        }
        else
        {
            Set(ref spacePressed, true);
        }
    }

    private void OnKeyUp(object sender, KeyRoutedEventArgs e)
    {
        if (e.Key != VirtualKey.Space || !spacePressed)
        {
            return;
        }

        e.Handled = true;
        Set(ref spacePressed, false);
        Invoke();
    }
}

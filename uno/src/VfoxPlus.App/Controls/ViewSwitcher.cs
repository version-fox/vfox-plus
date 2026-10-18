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

using VfoxPlus.App.Motion;
using VfoxPlus.Design.Motion;

namespace VfoxPlus.App.Controls;

/// <summary>How a view gives way to another one (see <c>docs/design-system.md#motion</c>).</summary>
internal enum Transition
{
    /// <summary>
    /// Fades the old view out, then fades the new one in while it grows a little: for views that have nothing to do with
    /// each other, such as the pages of the navigation.
    /// </summary>
    FadeThrough,

    /// <summary>
    /// Slides the views along the horizontal axis while they fade: for views that belong together, such as a list and
    /// the details of one of its items.
    /// </summary>
    SharedAxisX,
}

/// <summary>
/// Shows one view, and when another takes its place, lets the old one give way on a transition, which lasts as long as
/// the slow effects spring needs to settle.
/// </summary>
internal sealed class ViewSwitcher : Grid
{
    private readonly CompositeTransform oldTransform = new();
    private readonly CompositeTransform newTransform = new();
    private UIElement? current;
    private UIElement? previous;
    private Transition transition;
    private bool reverse;
    private TimeSpan started;
    private bool running;

    public ViewSwitcher()
    {
        // The views are clipped to the switcher, which the page panes around it do anyway.
        HorizontalAlignment = HorizontalAlignment.Stretch;
        VerticalAlignment = VerticalAlignment.Stretch;
    }

    /// <summary>Gets the view that is shown, or that is on its way in.</summary>
    public UIElement? Current => current;

    /// <summary>Shows a view.</summary>
    /// <param name="view">The view.</param>
    /// <param name="kind">How it takes the place of the view that is shown.</param>
    /// <param name="goingBack">Whether the user goes back, which mirrors the movement of the horizontal axis.</param>
    public void Show(UIElement view, Transition kind = Transition.FadeThrough, bool goingBack = false)
    {
        if (ReferenceEquals(view, current))
        {
            return;
        }

        FinishTransition();
        previous = current;
        current = view;
        transition = kind;
        reverse = goingBack;
        if (!Children.Contains(view))
        {
            Children.Add(view);
        }

        view.Visibility = Visibility.Visible;
        if (previous is null || FrameClock.Instance.ReducedMotion)
        {
            FinishTransition();
            return;
        }

        previous.IsHitTestVisible = false;
        previous.RenderTransform = oldTransform;
        previous.RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5);
        view.RenderTransform = newTransform;
        view.RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5);
        started = FrameClock.Instance.Now;
        running = true;
        Apply(0);
        FrameClock.Instance.Run(OnFrame);
    }

    /// <summary>Takes the views away without a transition: the switcher shows nothing until it is told to show a view.</summary>
    public void Reset()
    {
        running = false;
        Children.Clear();
        current = null;
        previous = null;
    }

    private static double Accelerate(double x) => Progress.CubicBezier(0.4, 0, 1, 1, x);

    private static double Decelerate(double x) => Progress.CubicBezier(0, 0, 0.2, 1, x);

    private static double FastOutSlowIn(double x) => Progress.CubicBezier(0.4, 0, 0.2, 1, x);

    private bool OnFrame(double seconds)
    {
        if (!running)
        {
            return false;
        }

        double duration = MotionScheme.Expressive.SlowEffects.SettleTime.TotalSeconds;
        double t = (FrameClock.Instance.Now - started).TotalSeconds / duration;
        if (t >= 1 || t < 0)
        {
            FinishTransition();
            return false;
        }

        Apply(t);
        return true;
    }

    private void Apply(double t)
    {
        if (previous is null || current is null)
        {
            return;
        }

        double fadeOut = 1 - Accelerate(Progress.Interval(t, 0, 0.3, x => x));
        double fadeIn = Progress.Interval(t, 0.3, 1, Decelerate);
        previous.Opacity = fadeOut;
        current.Opacity = fadeIn;
        if (transition == Transition.SharedAxisX)
        {
            double travel = 30 * (reverse ? -1 : 1);
            double move = FastOutSlowIn(t);
            oldTransform.TranslateX = -travel * move;
            newTransform.TranslateX = travel * (1 - move);
            oldTransform.ScaleX = 1;
            oldTransform.ScaleY = 1;
            newTransform.ScaleX = 1;
            newTransform.ScaleY = 1;
        }
        else
        {
            double scale = 0.92 + (0.08 * fadeIn);
            newTransform.ScaleX = scale;
            newTransform.ScaleY = scale;
            oldTransform.TranslateX = 0;
            newTransform.TranslateX = 0;
        }
    }

    private void FinishTransition()
    {
        running = false;
        if (previous is not null)
        {
            Children.Remove(previous);
            previous.Opacity = 1;
            previous.IsHitTestVisible = true;
            previous.RenderTransform = null;
            previous = null;
        }

        if (current is not null)
        {
            current.Opacity = 1;
            current.RenderTransform = null;
        }
    }
}

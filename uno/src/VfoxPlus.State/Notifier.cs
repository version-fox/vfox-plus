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

namespace VfoxPlus.State;

/// <summary>
/// Stops the listener it was returned for, when it is disposed.
/// </summary>
/// <param name="unsubscribe">Removes the listener.</param>
public sealed class Subscription(Action unsubscribe) : IDisposable
{
    /// <inheritdoc/>
    public void Dispose() => unsubscribe();
}

/// <summary>
/// Something whose changes can be observed.
/// </summary>
public interface IListenable
{
    /// <summary>Calls a listener after every change.</summary>
    /// <param name="listener">The listener, which may run on any thread.</param>
    /// <returns>A subscription whose disposal stops the listener.</returns>
    IDisposable AddListener(Action listener);
}

/// <summary>
/// An <see cref="IListenable"/> that notifies its listeners when told to.
/// </summary>
public class Notifier : IListenable, IDisposable
{
    private readonly Lock gate = new();
    private readonly Dictionary<long, Action> listeners = [];
    private long next;
    private bool disposed;

    /// <summary>Gets a value indicating whether the object was disposed.</summary>
    public bool IsDisposed
    {
        get
        {
            lock (gate)
            {
                return disposed;
            }
        }
    }

    /// <summary>Calls the listeners of several objects at once.</summary>
    /// <param name="listenables">The objects.</param>
    /// <param name="listener">The listener.</param>
    /// <returns>A subscription whose disposal stops the listener everywhere.</returns>
    public static IDisposable ListenAll(IEnumerable<IListenable?> listenables, Action listener)
    {
        List<IDisposable> subscriptions = [.. listenables.OfType<IListenable>().Select(listenable => listenable.AddListener(listener))];
        return new Subscription(() => subscriptions.ForEach(subscription => subscription.Dispose()));
    }

    /// <inheritdoc/>
    public IDisposable AddListener(Action listener)
    {
        long id;
        lock (gate)
        {
            id = ++next;
            listeners[id] = listener;
        }

        return new Subscription(() =>
        {
            lock (gate)
            {
                listeners.Remove(id);
            }
        });
    }

    /// <summary>Releases the listeners; the object must not be used afterwards.</summary>
    public virtual void Dispose()
    {
        lock (gate)
        {
            disposed = true;
            listeners.Clear();
        }

        GC.SuppressFinalize(this);
    }

    /// <summary>Calls every listener. It does nothing once disposed.</summary>
    protected void NotifyListeners()
    {
        Action[] current;
        lock (gate)
        {
            if (disposed)
            {
                return;
            }

            current = [.. listeners.Values];
        }

        foreach (Action listener in current)
        {
            listener();
        }
    }
}

/// <summary>
/// A simple source of events that carry a value.
/// </summary>
/// <typeparam name="T">The type of the value.</typeparam>
public sealed class Emitter<T>
{
    private readonly Lock gate = new();
    private readonly Dictionary<long, Action<T>> listeners = [];
    private long next;

    /// <summary>Calls a listener with every emitted value.</summary>
    /// <param name="listener">The listener.</param>
    /// <returns>A subscription whose disposal stops the listener.</returns>
    public IDisposable Listen(Action<T> listener)
    {
        long id;
        lock (gate)
        {
            id = ++next;
            listeners[id] = listener;
        }

        return new Subscription(() =>
        {
            lock (gate)
            {
                listeners.Remove(id);
            }
        });
    }

    /// <summary>Emits a value to every listener.</summary>
    /// <param name="value">The value.</param>
    public void Emit(T value)
    {
        Action<T>[] current;
        lock (gate)
        {
            current = [.. listeners.Values];
        }

        foreach (Action<T> listener in current)
        {
            listener(value);
        }
    }

    /// <summary>Stops every listener.</summary>
    public void Close()
    {
        lock (gate)
        {
            listeners.Clear();
        }
    }
}

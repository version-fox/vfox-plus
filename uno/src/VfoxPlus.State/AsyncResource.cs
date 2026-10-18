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
/// A value that is loaded asynchronously and can be reloaded.
/// </summary>
/// <remarks>
/// While a reload is in flight the previous value stays available, so the user interface can keep showing
/// stale data with a progress indicator instead of flashing an empty state. When loads overlap, only the
/// latest one wins.
/// </remarks>
/// <typeparam name="T">The type of the value.</typeparam>
public sealed class AsyncResource<T> : Notifier
{
    private readonly Lock gate = new();
    private readonly Func<Task<T>> loader;
    private T? value;
    private bool loaded;
    private Exception? error;
    private bool loading;
    private long generation;
    private Task? inFlight;

    /// <summary>Initializes a new instance of the <see cref="AsyncResource{T}"/> class.</summary>
    /// <param name="loader">Loads the value.</param>
    public AsyncResource(Func<Task<T>> loader) => this.loader = loader;

    /// <summary>Gets a value indicating whether a value was loaded at least once.</summary>
    public bool HasValue
    {
        get
        {
            lock (gate)
            {
                return loaded;
            }
        }
    }

    /// <summary>Gets the last successfully loaded value, or the default value of the type if there is none.</summary>
    public T? Value
    {
        get
        {
            lock (gate)
            {
                return value;
            }
        }
    }

    /// <summary>Gets the error of the last load, or <see langword="null"/> if it succeeded.</summary>
    public Exception? Error
    {
        get
        {
            lock (gate)
            {
                return error;
            }
        }
    }

    /// <summary>Gets a value indicating whether a load is in progress.</summary>
    public bool IsLoading
    {
        get
        {
            lock (gate)
            {
                return loading;
            }
        }
    }

    /// <summary>Loads the value, replacing any load that is still in progress.</summary>
    /// <returns>A task that completes when this load settles; a replaced load settles at once.</returns>
    public Task LoadAsync()
    {
        long mine;
        Task task;
        lock (gate)
        {
            mine = ++generation;
            loading = true;
            task = Task.Run(() => RunAsync(mine));
            inFlight = task;
        }

        NotifyListeners();
        return task;
    }

    /// <summary>Loads the value unless it is already loaded or loading.</summary>
    /// <returns>A task that completes when the value is there.</returns>
    public Task EnsureLoadedAsync()
    {
        lock (gate)
        {
            if (loading && inFlight is not null)
            {
                return inFlight;
            }

            if (loaded)
            {
                return Task.CompletedTask;
            }
        }

        return LoadAsync();
    }

    /// <summary>Replaces the value without loading, for example after an optimistic update.</summary>
    /// <param name="newValue">The value.</param>
    public void SetValue(T newValue)
    {
        lock (gate)
        {
            generation++;
            value = newValue;
            loaded = true;
            error = null;
            loading = false;
        }

        NotifyListeners();
    }

    private async Task RunAsync(long mine)
    {
        T? result = default;
        Exception? failure = null;
        try
        {
            result = await loader().ConfigureAwait(false);
        }
        catch (Exception exception)
        {
            failure = exception;
        }

        lock (gate)
        {
            if (mine != generation)
            {
                return;
            }

            if (failure is not null)
            {
                error = failure;
            }
            else
            {
                value = result;
                loaded = true;
                error = null;
            }

            loading = false;
        }

        NotifyListeners();
    }
}

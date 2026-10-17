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

//! Values that are loaded asynchronously.

use std::rc::Rc;

use dioxus::core::current_scope_id;
use dioxus::prelude::*;

use crate::core::command::VfoxError;
use crate::core::process::BoxFuture;

/// Loads the value of an [`AsyncResource`].
pub type Loader<T> = Rc<dyn Fn() -> BoxFuture<Result<T, VfoxError>>>;

/// What an [`AsyncResource`] knows.
#[derive(Clone, Debug, PartialEq)]
pub struct ResourceState<T> {
    /// The last successfully loaded value.
    pub value: Option<T>,
    /// The error of the last load, if it failed.
    pub error: Option<VfoxError>,
    /// Whether a load is in progress.
    pub loading: bool,
}

/// A value that is loaded asynchronously and can be reloaded. While a reload
/// is in flight the previous value stays available, so the user interface
/// keeps showing it with a progress indicator instead of flashing an empty
/// state. When loads overlap, only the latest one wins.
///
/// The state is a signal, which the user interface observes; loads run as
/// tasks of the scope that created the resource.
pub struct AsyncResource<T: 'static> {
    state: Signal<ResourceState<T>>,
    generation: CopyValue<u64>,
    loader: CopyValue<Loader<T>>,
    owner: ScopeId,
}

impl<T> Clone for AsyncResource<T> {
    fn clone(&self) -> Self {
        *self
    }
}

impl<T> Copy for AsyncResource<T> {}

impl<T> PartialEq for AsyncResource<T> {
    fn eq(&self, other: &Self) -> bool {
        self.state == other.state
    }
}

impl<T: Clone + 'static> AsyncResource<T> {
    /// A resource of the current scope that `loader` loads.
    pub fn new(loader: impl Fn() -> BoxFuture<Result<T, VfoxError>> + 'static) -> Self {
        Self {
            state: Signal::new(ResourceState { value: None, error: None, loading: false }),
            generation: CopyValue::new(0),
            loader: CopyValue::new(Rc::new(loader)),
            owner: current_scope_id(),
        }
    }

    /// The last successfully loaded value.
    pub fn value(&self) -> Option<T> {
        self.state.read().value.clone()
    }

    /// Reads the last successfully loaded value without copying it.
    pub fn with<R>(&self, f: impl FnOnce(Option<&T>) -> R) -> R {
        f(self.state.read().value.as_ref())
    }

    /// Whether a value was loaded at least once.
    pub fn has_value(&self) -> bool {
        self.state.read().value.is_some()
    }

    /// The error of the last load, if it failed.
    pub fn error(&self) -> Option<VfoxError> {
        self.state.read().error.clone()
    }

    /// Whether a load is in progress.
    pub fn is_loading(&self) -> bool {
        self.state.read().loading
    }

    /// Starts a load: later results of earlier loads are ignored.
    fn begin(&self) -> u64 {
        let generation = *self.generation.peek() + 1;
        *self.generation.write_unchecked() = generation;
        let mut state = self.state;
        state.write().loading = true;
        generation
    }

    async fn fetch(self, generation: u64) {
        let loader = self.loader.peek().clone();
        let result = loader().await;
        if *self.generation.peek() != generation {
            return;
        }
        let mut state = self.state;
        if let Ok(mut state) = state.try_write() {
            match result {
                Ok(value) => {
                    state.value = Some(value);
                    state.error = None;
                }
                Err(error) => state.error = Some(error),
            }
            state.loading = false;
        }
    }

    /// Loads the value and waits for it, replacing any load that is still in
    /// progress.
    pub async fn refresh(self) {
        let generation = self.begin();
        self.fetch(generation).await;
    }

    /// Loads the value in the background, replacing any load that is still
    /// in progress.
    pub fn load(&self) {
        let generation = self.begin();
        super::spawn_in(self.owner, self.fetch(generation));
    }

    /// Loads the value unless it is loaded or loading already.
    pub fn ensure_loaded(&self) {
        let state = self.state.peek();
        if !state.loading && state.value.is_none() {
            drop(state);
            self.load();
        }
    }

    /// Replaces the value without loading.
    pub fn replace(&self, value: T) {
        *self.generation.write_unchecked() += 1;
        let mut state = self.state;
        state.set(ResourceState { value: Some(value), error: None, loading: false });
    }
}

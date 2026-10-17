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

//! The application state, in signals that the user interface observes: the
//! asynchronous resources, the task queue, the controllers and the model that
//! wires them together. Everything changes on the thread of the user
//! interface; processes are read by tasks of the virtual DOM.

pub mod controllers;
pub mod model;
pub mod resource;
pub mod tasks;

#[cfg(test)]
mod tests;

use dioxus::core::Runtime;
use dioxus::prelude::*;

/// Runs `future` as a task of `scope`, so that it ends with that scope rather
/// than with the component that started it.
pub(crate) fn spawn_in(scope: ScopeId, future: impl Future<Output = ()> + 'static) {
    Runtime::current().in_scope(scope, || spawn(future));
}

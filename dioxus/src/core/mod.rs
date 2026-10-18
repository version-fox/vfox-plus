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

//! Everything that does not depend on the user interface: the models, the
//! parsers of the output of vfox, the vfox client, the process runner, the
//! settings and the translations.

pub mod cli;
pub mod command;
pub mod i18n;
pub mod locator;
pub mod models;
pub mod output;
pub mod plugin_source;
pub mod process;
pub mod settings;
pub mod terminal;

#[cfg(test)]
mod cli_tests;
#[cfg(test)]
mod contract_tests;
#[cfg(test)]
pub mod testing;

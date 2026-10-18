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

// Package mcu generates the colour scheme of the application from a seed
// colour: the tonal spot scheme of the Material colour utilities, in their
// 2021 specification, which is what Flutter's `ColorScheme.fromSeed` and the
// other implementations of vfox-plus use, so that every one of them shows the
// same colours.
//
// The package is a Go port of the parts of the Material Color Utilities
// (https://github.com/material-foundation/material-color-utilities,
// Copyright 2021 Google LLC, Apache License 2.0) that the tonal spot scheme
// needs: the HCT colour space with its solver, tonal palettes, contrast
// maths and the dynamic colour roles of the 2021 specification. Its results
// are checked against the JavaScript implementation, version 0.4.0, which the
// web front end uses (see testdata).
package mcu

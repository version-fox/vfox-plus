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

// Package shapes is a Go port of the library of rounded polygons and of their
// morphs that Android ships as androidx.graphics.shapes, with the shapes of
// Material 3 Expressive that vfox-plus draws (see NOTICE.txt for where it comes
// from). A Polygon is a closed outline of cubic Bézier curves in a unit
// square; a Morph is the transition between two polygons.
package shapes

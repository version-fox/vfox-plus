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

package kit

// The font families of the interface. mygo draws text with the text engine of
// the desktop, which finds these through fontconfig: the sans-serif font of the
// desktop is the font of Material 3 on Linux (Ubuntu Sans on Ubuntu), and what
// it lacks, such as Chinese, comes from the next family and then from the
// fonts the system chooses.
const (
	// UIFamilies is the font of the interface.
	UIFamilies = "Ubuntu Sans, Ubuntu, Noto Sans, sans-serif"
	// MonoFamilies is the monospaced font.
	MonoFamilies = "Ubuntu Sans Mono, Ubuntu Mono, Noto Sans Mono, monospace"
)

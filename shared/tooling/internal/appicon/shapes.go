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

package appicon

// The paths and the colours of the icon, as in shared/packaging/linux/dev.vfox.VfoxPlus.svg
// (the ids are those of the SVG). The plate and the halo are drawn on a canvas of 256 by 256
// units, the fox in its own units, 160 wide, placed by foxX, foxY and foxScale.

const (
	// Canvas is the side of the square canvas of the plate and the halo.
	canvas = 256
	// haloScale is the scale of the halo around the centre of the canvas.
	haloScale = 0.86
	// foxX, foxY and foxScale place the fox on the canvas: x' = foxX + foxScale * x.
	foxX     = 51.2
	foxY     = 56.24
	foxScale = 0.96
)

// The same fox on the adaptive icon of Android (ic_launcher_foreground.xml and
// ic_launcher_background.xml), where the launcher crops the canvas to a circle or a squircle
// of 72 of its 108 dp, so that the fox is smaller and the halo too.
const (
	androidHaloScale = 0.7
	androidFoxX      = 65.6
	androidFoxY      = 70.07
	androidFoxScale  = 0.78
)

// androidBackground is the plate of the adaptive icon, which has no rounded corners.
const androidBackground = "M0,0H256V256H0Z"

var (
	plate = shape{id: "plate-shape", fill: "url(#plate)", d: "M66,8 L190,8 C222,8 248,34 248,66 L248,190 C248,222 222,248 190,248 L66,248 C34,248 8,222 8,190 L8,66 C8,34 34,8 66,8 Z"}
	halo  = shape{id: "halo-shape", fill: "url(#halo)", d: "M128,3.6 C137.3,3.6 146.7,7 154.1,13.7 C161.6,20.5 171.5,24.1 181.7,23.7 C201.6,23 218.9,37.5 221.6,57.3 C223,67.3 228.3,76.5 236.3,82.7 C252.1,94.9 256,117.1 245.3,134 C239.9,142.6 238.1,153 240.2,162.9 C244.5,182.5 233.2,202 214.2,208.1 C204.5,211.2 196.4,218 191.7,227 C182.4,244.7 161.2,252.4 142.7,244.8 C133.3,241 122.7,241 113.3,244.8 C94.8,252.4 73.6,244.7 64.3,227 C59.6,218 51.5,211.2 41.8,208.1 C22.8,202 11.5,182.5 15.8,162.9 C17.9,153 16.1,142.6 10.7,134 C0,117.1 3.9,94.9 19.7,82.7 C27.7,76.5 33,67.3 34.4,57.3 C37.1,37.5 54.4,23 74.3,23.7 C84.5,24.1 94.4,20.5 101.9,13.7 C109.3,7 118.7,3.6 128,3.6 Z"}

	// foxShapes are the parts of the fox from back to front.
	foxShapes = []shape{
		{id: "cheeks", fill: "url(#cream)", d: "M71.5,38 L65.1,34.2 C63.7,33.4 62.4,32.4 61.3,31.3 L45,14.5 C36.6,5.9 22.1,8.9 17.8,20.1 L5.3,53.2 C5.1,53.7 4.9,54.3 4.8,54.9 L-5.3,92.2 C-6.8,97.7 -3.8,103.5 1.5,105.4 L38.4,118.7 C40.8,119.6 42.9,120.9 44.6,122.7 L61.2,140.2 C71.4,150.9 88.6,150.9 98.8,140.2 L115.4,122.7 C117.1,120.9 119.2,119.6 121.6,118.7 L158.5,105.4 C163.8,103.5 166.8,97.7 165.3,92.2 L155.2,54.9 C155.1,54.3 154.9,53.7 154.7,53.2 L142.2,20.1 C137.9,8.9 123.4,5.9 115,14.5 L98.7,31.3 C97.6,32.4 96.3,33.4 94.9,34.2 L88.5,38 C83.3,41.1 76.7,41.1 71.5,38 Z"},
		{id: "fur", fill: "url(#fur)", d: "M71,35.5 L64.7,32.4 C62.9,31.5 61.3,30.3 59.9,28.8 L42,10.5 C33,1.3 17.5,5.2 13.8,17.5 L4.9,47.7 C3.7,51.7 4.4,56 6.7,59.5 L28.3,91.4 C29.4,93.1 30.8,94.7 32.3,96 L68.1,127.5 C74.9,133.5 85.1,133.5 91.9,127.5 L127.7,96 C129.2,94.7 130.6,93.1 131.7,91.4 L153.3,59.5 C155.6,56 156.3,51.7 155.1,47.7 L146.2,17.5 C142.5,5.2 127,1.3 118,10.5 L100.1,28.8 C98.7,30.3 97.1,31.5 95.3,32.4 L89,35.5 C83.3,38.3 76.7,38.3 71,35.5 Z"},
		{id: "ear-left", fill: "#FFE3C4", d: "M32,20 L45.6,31.6 C49.2,34.7 48.7,40.5 44.6,43 L31.4,50.7 C26.4,53.6 20.2,49.5 20.9,43.7 L23.8,23.1 C24.3,19.2 29,17.4 32,20 Z"},
		{id: "ear-right", fill: "#FFE3C4", d: "M128,20 L114.4,31.6 C110.8,34.7 111.3,40.5 115.4,43 L128.6,50.7 C133.6,53.6 139.8,49.5 139.1,43.7 L136.2,23.1 C135.7,19.2 131,17.4 128,20 Z"},
		{id: "eye-left", fill: "#17123F", d: "M40.5,86 C40.5,79.4 44.8,74 50,74 C55.2,74 59.5,79.4 59.5,86 C59.5,92.6 55.2,98 50,98 C44.8,98 40.5,92.6 40.5,86 Z"},
		{id: "eye-right", fill: "#17123F", d: "M100.5,86 C100.5,79.4 104.8,74 110,74 C115.2,74 119.5,79.4 119.5,86 C119.5,92.6 115.2,98 110,98 C104.8,98 100.5,92.6 100.5,86 Z"},
		{id: "spark-left", fill: "#FFFFFF", d: "M49.8,81.6 C49.8,79.7 51.3,78.2 53.2,78.2 C55.1,78.2 56.6,79.7 56.6,81.6 C56.6,83.5 55.1,85 53.2,85 C51.3,85 49.8,83.5 49.8,81.6 Z"},
		{id: "spark-right", fill: "#FFFFFF", d: "M109.8,81.6 C109.8,79.7 111.3,78.2 113.2,78.2 C115.1,78.2 116.6,79.7 116.6,81.6 C116.6,83.5 115.1,85 113.2,85 C111.3,85 109.8,83.5 109.8,81.6 Z"},
		{id: "nose", fill: "#17123F", d: "M77.4,124 L82.6,124 C87.2,124 89.8,129.3 87,133 L86.3,133.9 C83.1,138 76.9,138 73.7,133.9 L73,133 C70.2,129.3 72.8,124 77.4,124 Z"},
	}

	// maskMono is the fur of the monochrome icon: it stops above the nose, which is cut out of the
	// cheeks only. The Android icon is the only one that has it.
	maskMono = "M71,35.5 L64.7,32.4 C62.9,31.5 61.3,30.3 59.9,28.8 L42,10.5 C33,1.3 17.5,5.2 13.8,17.5 L4.9,47.7 C3.7,51.7 4.4,56 6.7,59.5 L26.6,88.9 C28.8,92.3 31.9,95 35.4,96.8 L72.6,116.2 C77.2,118.6 82.8,118.6 87.4,116.2 L124.6,96.8 C128.1,95 131.2,92.3 133.4,88.9 L153.3,59.5 C155.6,56 156.3,51.7 155.1,47.7 L146.2,17.5 C142.5,5.2 127,1.3 118,10.5 L100.1,28.8 C98.7,30.3 97.1,31.5 95.3,32.4 L89,35.5 C83.3,38.3 76.7,38.3 71,35.5 Z"

	gradients = []gradient{
		{id: "plate", x1: 8, y1: 8, x2: 248, y2: 248, stops: []stop{{0, "#5444E4", 1}, {1, "#1B1268", 1}}},
		{id: "halo", x1: 0, y1: 0, x2: 256, y2: 256, stops: []stop{{0, "#FFFFFF", 0.12}, {1, "#FFFFFF", 0.03}}},
		{id: "fur", x1: 0, y1: -10, x2: 0, y2: 140, stops: []stop{{0, "#FFB63F", 1}, {1, "#FF8419", 1}}},
		{id: "cream", x1: 0, y1: -8, x2: 0, y2: 160, stops: []stop{{0, "#FFFFFF", 1}, {1, "#FFE7CB", 1}}},
	}
)

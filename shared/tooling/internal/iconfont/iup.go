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

package iconfont

// IUP (interpolate untouched points) delta inference, mirroring
// fontTools.varLib.iup.iup_delta. Only the inference half is implemented:
// the instancer's IUP *optimization* of surviving tuples is intentionally
// skipped (surviving tuples keep explicit deltas for every point), which is
// rendering-identical and at most slightly larger. See vary.go.

// iupSegment interpolates deltas for coords between reference points
// (rc1, rd1) and (rc2, rd2). Deltas are [2]float64; the computation is done
// in float64 exactly like the Python original (same IEEE 754 operations).
func iupSegment(coords [][2]float64, rc1, rd1, rc2, rd2 [2]float64) [][2]float64 {
	out := make([][2]float64, len(coords))
	for j := 0; j < 2; j++ {
		x1, x2, d1, d2 := rc1[j], rc2[j], rd1[j], rd2[j]
		if x1 == x2 {
			if d1 == d2 {
				for i := range out {
					out[i][j] = d1
				}
			} else {
				for i := range out {
					out[i][j] = 0
				}
			}
			continue
		}
		if x1 > x2 {
			x1, x2 = x2, x1
			d1, d2 = d2, d1
		}
		scale := (d2 - d1) / (x2 - x1)
		for i, p := range coords {
			x := p[j]
			var d float64
			switch {
			case x <= x1:
				d = d1
			case x >= x2:
				d = d2
			default:
				nudge := (x - x1) * scale
				d = d1 + nudge
			}
			out[i][j] = d
		}
	}
	return out
}

// iupContour fills missing (nil) deltas within one contour. Deltas use nil
// for uninferred points.
func iupContour(deltas [][2]float64, have []bool, coords [][2]float64) [][2]float64 {
	n := len(coords)
	var indices []int
	for i := 0; i < n; i++ {
		if have[i] {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		out := make([][2]float64, n)
		return out
	}
	out := make([][2]float64, 0, n)
	it := indices
	start := it[0]
	if start != 0 {
		i1, i2, ri1, ri2 := 0, start, start, it[len(it)-1]
		out = append(out, iupSegment(coords[i1:i2], coords[ri1], deltas[ri1], coords[ri2], deltas[ri2])...)
	}
	out = append(out, deltas[start])
	for k := 1; k < len(it); k++ {
		end := it[k]
		if end-start > 1 {
			i1, i2, ri1, ri2 := start+1, end, start, end
			out = append(out, iupSegment(coords[i1:i2], coords[ri1], deltas[ri1], coords[ri2], deltas[ri2])...)
		}
		out = append(out, deltas[end])
		start = end
	}
	if start != n-1 {
		i1, i2, ri1, ri2 := start+1, n, start, it[0]
		out = append(out, iupSegment(coords[i1:i2], coords[ri1], deltas[ri1], coords[ri2], deltas[ri2])...)
	}
	return out
}

// iupDelta infers missing deltas contour by contour; each phantom point forms
// its own single-point contour. ends are contour end point indices for the
// real outline points. Returns a full delta vector (same length as coords).
func iupDelta(deltas [][2]float64, have []bool, coords [][2]float64, ends []int) [][2]float64 {
	n := len(coords)
	allEnds := append(append([]int(nil), ends...), n-4, n-3, n-2, n-1)
	out := make([][2]float64, 0, n)
	start := 0
	for _, end := range allEnds {
		end++
		out = append(out, iupContour(deltas[start:end], have[start:end], coords[start:end])...)
		start = end
	}
	return out
}

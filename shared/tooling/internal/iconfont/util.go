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

import (
	"encoding/binary"
	"math"
)

// otRound rounds a float to the nearest integer towards +Infinity, matching
// fontTools.misc.roundTools.otRound (floor(value + 0.5)), which is the
// rounding the OpenType specification mandates for variation interpolations.
func otRound(v float64) float64 {
	return math.Floor(v + 0.5)
}

// otRoundInt is otRound converted to int.
func otRoundInt(v float64) int {
	return int(math.Floor(v + 0.5))
}

// f2dot14ToFloat converts an F2DOT14 fixed-point value to float64.
func f2dot14ToFloat(v int16) float64 {
	return float64(v) / 16384.0
}

// floatToF2Dot14 converts a float to F2DOT14 with otRound rounding.
func floatToF2Dot14(v float64) int16 {
	return int16(otRoundInt(v * 16384.0))
}

// floatToF2Dot14Float quantizes a float to exactly representable F2DOT14,
// matching fontTools.misc.fixedTools.floatToFixedToFloat(value, 14).
func floatToF2Dot14Float(v float64) float64 {
	return float64(int16(otRoundInt(v*16384.0))) / 16384.0
}

// fixedToFloat converts 16.16 fixed-point to float64.
func fixedToFloat(v int32) float64 {
	return float64(v) / 65536.0
}

// floatToFixed converts a float to 16.16 fixed-point with otRound rounding.
func floatToFixed(v float64) int32 {
	return int32(otRoundInt(v * 65536.0))
}

func getU16(b []byte, off int) uint16 { return binary.BigEndian.Uint16(b[off:]) }
func getI16(b []byte, off int) int16  { return int16(binary.BigEndian.Uint16(b[off:])) }
func getU32(b []byte, off int) uint32 { return binary.BigEndian.Uint32(b[off:]) }
func getI32(b []byte, off int) int32  { return int32(binary.BigEndian.Uint32(b[off:])) }

func putU16(b []byte, off int, v uint16) { binary.BigEndian.PutUint16(b[off:], v) }
func putU32(b []byte, off int, v uint32) { binary.BigEndian.PutUint32(b[off:], v) }

func appendU16(b []byte, v uint16) []byte {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendU32(b []byte, v uint32) []byte {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], v)
	return append(b, tmp[:]...)
}

func appendI16(b []byte, v int16) []byte { return appendU16(b, uint16(v)) }

// pad4 returns n rounded up to a multiple of 4.
func pad4(n int) int { return (n + 3) &^ 3 }

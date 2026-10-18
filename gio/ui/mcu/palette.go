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

package mcu

import "sync"

// TonalPalette is the colours of one hue and chroma at every tone, from 0 to
// 100.
type TonalPalette struct {
	// Hue in degrees.
	Hue float64
	// Chroma requested for the colours; light and dark tones may have less.
	Chroma float64

	mu    sync.Mutex
	cache map[float64]uint32
}

// NewTonalPalette creates the palette of a hue and a chroma.
func NewTonalPalette(hue, chroma float64) *TonalPalette {
	return &TonalPalette{Hue: hue, Chroma: chroma, cache: make(map[float64]uint32)}
}

// Tone is the colour of the palette at tone (0 to 100) as ARGB.
func (p *TonalPalette) Tone(tone float64) uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if argb, ok := p.cache[tone]; ok {
		return argb
	}
	var argb uint32
	if tone == 99 && isYellow(p.Hue) {
		// Tone 99 of a yellow is not representable: use the average of the
		// neighbours.
		argb = averageARGB(p.toneLocked(98), p.toneLocked(100))
	} else {
		argb = solveToARGB(p.Hue, p.Chroma, tone)
	}
	p.cache[tone] = argb
	return argb
}

func (p *TonalPalette) toneLocked(tone float64) uint32 {
	if argb, ok := p.cache[tone]; ok {
		return argb
	}
	argb := solveToARGB(p.Hue, p.Chroma, tone)
	p.cache[tone] = argb
	return argb
}

// Hct is the colour of the palette at tone, in HCT.
func (p *TonalPalette) Hct(tone float64) Hct { return HctFromARGB(p.Tone(tone)) }

func averageARGB(a, b uint32) uint32 {
	red := jsRound(float64(redOf(a)+redOf(b)) / 2)
	green := jsRound(float64(greenOf(a)+greenOf(b)) / 2)
	blue := jsRound(float64(blueOf(a)+blueOf(b)) / 2)
	return argbFromRGB(int(red), int(green), int(blue))
}

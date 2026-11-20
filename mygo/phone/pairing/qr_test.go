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

package pairing_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

// qrText is a pairing link of 71 bytes, with digits and letters in both cases.
const qrText = "https://192.168.1.10:8369/auth/connect/" + sampleToken

// qrImage renders qrText as a picture of size pixels, with its quiet zone.
func qrImage(t *testing.T, size int) image.Image {
	t.Helper()
	code, err := qrcode.New(qrText, qrcode.Medium)
	if err != nil {
		t.Fatal(err)
	}
	return code.Image(size)
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// toRGBA returns a copy of img with its origin at zero.
func toRGBA(img image.Image) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
	return dst
}

// boxBlur applies a 3x3 mean filter passes times, the softness of a photo that
// is out of focus or taken in a hurry.
func boxBlur(img image.Image, passes int) *image.RGBA {
	src := toRGBA(img)
	for pass := 0; pass < passes; pass++ {
		bounds := src.Bounds()
		dst := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				var r, g, b, n uint32
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						px, py := x+dx, y+dy
						if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
							continue
						}
						c := src.RGBAAt(px, py)
						r += uint32(c.R)
						g += uint32(c.G)
						b += uint32(c.B)
						n++
					}
				}
				dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
			}
		}
		src = dst
	}
	return src
}

// rotate90 turns img a quarter turn clockwise.
func rotate90(img image.Image) *image.RGBA {
	src := toRGBA(img)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.SetRGBA(h-1-y, x, src.RGBAAt(x, y))
		}
	}
	return dst
}

// withBorder surrounds img with a white margin of margin pixels.
func withBorder(img image.Image, margin int) *image.RGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w+2*margin, h+2*margin))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(margin, margin, margin+w, margin+h), img, img.Bounds().Min, draw.Src)
	return dst
}

func TestDecodeQRReadsPNGAndJPEGPictures(t *testing.T) {
	base := qrImage(t, 800)
	cases := []struct {
		name string
		data []byte
	}{
		{"png", encodePNG(t, base)},
		{"jpeg", encodeJPEG(t, base)},
		{"shrunk", encodePNG(t, qrImage(t, 480))},
		{"enlarged", encodePNG(t, qrImage(t, 1600))},
		{"blurred", encodePNG(t, boxBlur(base, 1))},
		{"blurred twice", encodePNG(t, boxBlur(base, 2))},
		{"rotated", encodePNG(t, rotate90(base))},
		{"with a white border", encodePNG(t, withBorder(base, 60))},
		{"jpeg of a rotated, blurred code", encodeJPEG(t, rotate90(boxBlur(base, 1)))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, err := pairing.DecodeQR(tc.data)
			if err != nil {
				t.Fatalf("DecodeQR: %v", err)
			}
			if text != qrText {
				t.Fatalf("DecodeQR = %q, want %q", text, qrText)
			}
		})
	}
}

func TestDecodeQRRejectsDataThatIsNotAPicture(t *testing.T) {
	for name, data := range map[string][]byte{
		"nothing":   nil,
		"text":      []byte(qrText),
		"truncated": encodePNG(t, qrImage(t, 200))[:40],
	} {
		if _, err := pairing.DecodeQR(data); !errors.Is(err, pairing.ErrNotPicture) {
			t.Errorf("%s: DecodeQR = %v, want ErrNotPicture", name, err)
		}
	}
}

func TestDecodeQRReportsAPictureWithoutACode(t *testing.T) {
	blank := image.NewRGBA(image.Rect(0, 0, 300, 300))
	draw.Draw(blank, blank.Bounds(), image.White, image.Point{}, draw.Src)
	for name, data := range map[string][]byte{
		"blank png":  encodePNG(t, blank),
		"blank jpeg": encodeJPEG(t, blank),
	} {
		if _, err := pairing.DecodeQR(data); !errors.Is(err, pairing.ErrNoQRCode) {
			t.Errorf("%s: DecodeQR = %v, want ErrNoQRCode", name, err)
		}
	}
}

// pngHeader returns the signature and the IHDR chunk of a PNG of the given size.
// That is all that DecodeConfig reads, so a picture of any size can be checked
// without holding its pixels.
func pngHeader(width, height uint32) []byte {
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 0, 13)
	ihdr = binary.BigEndian.AppendUint32(ihdr, width)
	ihdr = binary.BigEndian.AppendUint32(ihdr, height)
	ihdr = append(ihdr, 8, 6, 0, 0, 0) // 8-bit RGBA, deflate, adaptive filters, no interlace
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(binary.BigEndian.AppendUint32(nil, uint32(len(ihdr))))
	buf.Write(chunk)
	buf.Write(binary.BigEndian.AppendUint32(nil, crc32.ChecksumIEEE(chunk)))
	return buf.Bytes()
}

func TestDecodeQRRefusesPicturesTooLargeToRead(t *testing.T) {
	if _, err := pairing.DecodeQR(pngHeader(5000, 4000)); !errors.Is(err, pairing.ErrPictureTooLarge) {
		t.Fatalf("a 20 MP picture: DecodeQR = %v, want ErrPictureTooLarge", err)
	}
}

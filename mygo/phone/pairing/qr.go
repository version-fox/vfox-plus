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

package pairing

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // registers the JPEG decoder of image.Decode
	_ "image/png"  // registers the PNG decoder of image.Decode

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

var (
	// ErrNoQRCode reports a picture that reads fine but holds no QR code that can
	// be read.
	ErrNoQRCode = errors.New("no QR code found in the picture")
	// ErrNotPicture reports data that is not a PNG or JPEG picture.
	ErrNotPicture = errors.New("the file is not a PNG or JPEG picture")
	// ErrPictureTooLarge reports a picture with more pixels than DecodeQR reads.
	// Decoding a larger one would take too much memory on a phone; a caller can
	// scale the picture down and try again.
	ErrPictureTooLarge = errors.New("the picture is too large to read; choose a smaller one")
)

// maxPixels is the most pixels that DecodeQR reads: 16 mega-pixels, which covers
// the 12 MP and 16 MP photos of phone cameras.
const maxPixels = 16 << 20

// DecodeQR returns the text of the first QR code in data, which is a PNG or JPEG
// picture. A picture that is not one of those is ErrNotPicture; a picture with no
// readable code is ErrNoQRCode; a picture with more than maxPixels pixels is
// ErrPictureTooLarge. The reader tries hard, because phone photos are often
// small, blurred or tilted. The EXIF orientation of a JPEG is not applied: the
// reader finds a QR code in a rotated picture without it (tested at 90 degrees).
func DecodeQR(data []byte) (string, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotPicture, err)
	}
	if int64(config.Width)*int64(config.Height) > maxPixels {
		return "", ErrPictureTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNotPicture, err)
	}
	return readQR(img)
}

// readQR reads the first QR code of img. The reader panics on some damaged
// pictures; a panic is reported as a picture without a readable code, so that a
// bad file cannot crash the application.
func readQR(img image.Image) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text, err = "", fmt.Errorf("%w (%v)", ErrNoQRCode, recovered)
		}
	}()
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("prepare the picture for reading: %w", err)
	}
	hints := map[gozxing.DecodeHintType]any{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := qrcode.NewQRCodeReader().Decode(bitmap, hints)
	if err != nil {
		return "", fmt.Errorf("%w (%w)", ErrNoQRCode, err)
	}
	return result.GetText(), nil
}

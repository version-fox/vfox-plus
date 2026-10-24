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
	"bytes"
	"io"

	"github.com/andybalholm/brotli"
)

// Brotli codec for the WOFF2 payload, backed by github.com/andybalholm/brotli.
//
// Only the stable API is used (Writer/Reader via NewWriterOptions/NewReader);
// the experimental NewWriterV2 API is deliberately avoided (upstream marks it
// SemVer-unstable). Quality and window size are set explicitly so WOFF2
// output is deterministic in configuration across builds.
const (
	brotliQuality = 11
	brotliLGWin   = 24
)

// encodeBrotli compresses data into a valid Brotli stream.
func encodeBrotli(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := brotli.NewWriterOptions(&buf, brotli.WriterOptions{
		Quality: brotliQuality,
		LGWin:   brotliLGWin,
	})
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decodeBrotli decompresses a Brotli stream produced by encodeBrotli (or any
// compliant encoder).
func decodeBrotli(data []byte) ([]byte, error) {
	r := brotli.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}

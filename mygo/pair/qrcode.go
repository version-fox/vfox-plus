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

package pair

import (
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
)

// QRMatrix encodes text as a QR code, with the medium level of error correction,
// and returns its modules by row, true for a dark one. The matrix includes the
// quiet zone around the code, four modules wide on every side, which a scanner
// needs: draw it on a light ground. It is square.
func QRMatrix(text string) ([][]bool, error) {
	code, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	return code.Bitmap(), nil
}

// QRSVG renders text as the QR code of QRMatrix, as an SVG with a white ground
// and black modules, that scales to any size: one path, a bar for each run of
// dark modules.
func QRSVG(text string) (string, error) {
	matrix, err := QRMatrix(text)
	if err != nil {
		return "", err
	}
	size := strconv.Itoa(len(matrix))
	var out strings.Builder
	out.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ` + size + " " + size + `" shape-rendering="crispEdges" role="img">`)
	out.WriteString(`<rect width="` + size + `" height="` + size + `" fill="#fff"/><path fill="#000" d="`)
	for y, row := range matrix {
		for x := 0; x < len(row); x++ {
			if !row[x] {
				continue
			}
			end := x
			for end < len(row) && row[end] {
				end++
			}
			out.WriteString("M" + strconv.Itoa(x) + " " + strconv.Itoa(y) + "h" + strconv.Itoa(end-x) + "v1h-" + strconv.Itoa(end-x) + "z")
			x = end
		}
	}
	out.WriteString(`"/></svg>`)
	return out.String(), nil
}

// PairingURL renders the pairing link for an explicit host, with the scheme
// and port of the server. The code that a phone scans uses an address of the
// local network here, because the server listens on a wildcard address.
func (s *Server) PairingURL(host, token string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return s.scheme() + "://" + host + s.portSuffix() + "/auth/connect/" + token
}

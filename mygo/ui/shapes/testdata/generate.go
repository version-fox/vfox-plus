//go:build ignore

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

// Writes shapes.json: the outlines of the Material 3 shapes that vfox-plus
// draws, as SVG paths, from the shape component of @m3e/web (MIT, matraic).
// The Go shapes are tested against it (shapes_test.go). Needs no Node.js:
// the bundle is fetched from the npm registry, verified against its published
// integrity hash, and its SHAPE_PATHS table is extracted with a regular
// expression, like the Node.js script this replaces did.
//
//	go run generate.go [@m3e/web@version | path/to/web-2.10.1.tgz | unpacked/dir]
//
// With no argument the pinned version below is downloaded; the output is
// byte-identical to what the script wrote, so regenerating must not change
// the committed file unless the bundle did.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// pinnedVersion is the @m3e/web release the committed shapes.json comes from.
const pinnedVersion = "2.10.1"

var shapePathsPattern = regexp.MustCompile(`const SHAPE_PATHS = (\{[\s\S]*?\n\});`)

// shapeNames lists the shapes vfox-plus draws, in output order.
var shapeNames = []string{
	"4-leaf-clover", "4-sided-cookie", "6-sided-cookie", "8-leaf-clover", "9-sided-cookie", "12-sided-cookie",
	"oval", "pentagon", "pill", "soft-burst", "sunny", "flower", "gem", "arch", "puffy-diamond", "circle",
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var source []byte
	switch {
	case len(args) == 0:
		var err error
		source, err = fetchBundle(pinnedVersion)
		if err != nil {
			return err
		}
	case strings.HasSuffix(args[0], ".tgz"):
		var err error
		source, err = extractShapeJSFromTarball(args[0])
		if err != nil {
			return err
		}
	default:
		var err error
		source, err = os.ReadFile(args[0] + "/dist/shape.js")
		if err != nil {
			return err
		}
	}
	match := shapePathsPattern.FindSubmatch(source)
	if match == nil {
		return fmt.Errorf("SHAPE_PATHS not found")
	}
	paths, err := parseObjectLiteral(match[1])
	if err != nil {
		return err
	}
	var out bytes.Buffer
	out.WriteString("{\n")
	for i, name := range shapeNames {
		path, ok := paths[name]
		if !ok {
			return fmt.Errorf("shape %q not found", name)
		}
		key, err := json.Marshal(name)
		if err != nil {
			return err
		}
		value, err := json.Marshal(path)
		if err != nil {
			return err
		}
		out.Write(key)
		out.WriteString(": ")
		out.Write(value)
		if i+1 < len(shapeNames) {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}
	out.WriteString("}\n")
	_, err = os.Stdout.Write(out.Bytes())
	return err
}

// fetchBundle downloads the tarball of the pinned @m3e/web release from the
// npm registry, checks its published integrity hash and returns dist/shape.js.
func fetchBundle(version string) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	response, err := client.Get("https://registry.npmjs.org/@m3e%2Fweb/" + version)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry answered %d for @m3e/web@%s", response.StatusCode, version)
	}
	var metadata struct {
		Dist struct {
			Tarball   string `json:"tarball"`
			Integrity string `json:"integrity"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	algorithm, digest, _ := strings.Cut(metadata.Dist.Integrity, "-")
	if algorithm != "sha512" || digest == "" {
		return nil, fmt.Errorf("unexpected integrity %q", metadata.Dist.Integrity)
	}
	want, err := base64.StdEncoding.DecodeString(digest)
	if err != nil {
		return nil, fmt.Errorf("bad integrity %q: %w", metadata.Dist.Integrity, err)
	}
	archive, err := download(client, metadata.Dist.Tarball)
	if err != nil {
		return nil, err
	}
	if sum := sha512.Sum512(archive); !bytes.Equal(sum[:], want) {
		return nil, fmt.Errorf("tarball checksum mismatch")
	}
	return shapeJSFromTarball(archive)
}

func download(client *http.Client, url string) ([]byte, error) {
	response, err := client.Get(url) //nolint:gosec // Pinned release tarball.
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %d", url, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// extractShapeJSFromTarball reads dist/shape.js out of a release tarball.
func extractShapeJSFromTarball(path string) ([]byte, error) {
	archive, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return shapeJSFromTarball(archive)
}

func shapeJSFromTarball(archive []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Name == "package/dist/shape.js" || strings.HasSuffix(header.Name, "/dist/shape.js") {
			return io.ReadAll(tarReader)
		}
	}
	return nil, fmt.Errorf("dist/shape.js not in tarball")
}

// parseObjectLiteral parses a flat JavaScript object literal with string
// values — quoted or bare identifier keys, single- or double-quoted strings,
// trailing commas allowed — into a Go map. Anything else is rejected, so a
// change of the bundle format fails loudly instead of silently.
func parseObjectLiteral(input []byte) (map[string]string, error) {
	type token struct {
		kind byte // '{', '}', ':', ',', 's' (string), 'i' (identifier)
		text string
	}
	var tokens []token
	i := 0
	lexString := func(quote byte) (string, error) {
		var out strings.Builder
		for {
			if i >= len(input) {
				return "", fmt.Errorf("unterminated string")
			}
			c := input[i]
			i++
			if c == quote {
				return out.String(), nil
			}
			if c != '\\' {
				out.WriteByte(c)
				continue
			}
			if i >= len(input) {
				return "", fmt.Errorf("unterminated escape")
			}
			e := input[i]
			i++
			switch e {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case 'b':
				out.WriteByte('\b')
			case 'f':
				out.WriteByte('\f')
			case 'v':
				out.WriteByte('\v')
			case '0':
				out.WriteByte(0)
			case '\'', '"', '\\':
				out.WriteByte(e)
			case 'x', 'u':
				size := map[byte]int{'x': 2, 'u': 4}[e]
				if i+size > len(input) {
					return "", fmt.Errorf("truncated hex escape")
				}
				var code int
				for _, h := range input[i : i+size] {
					code <<= 4
					switch {
					case h >= '0' && h <= '9':
						code |= int(h - '0')
					case h >= 'a' && h <= 'f':
						code |= int(h-'a') + 10
					case h >= 'A' && h <= 'F':
						code |= int(h-'A') + 10
					default:
						return "", fmt.Errorf("bad hex escape")
					}
				}
				i += size
				out.WriteRune(rune(code))
			default:
				return "", fmt.Errorf("unsupported escape \\%c", e)
			}
		}
	}
	for i < len(input) {
		c := input[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '{' || c == '}' || c == ':' || c == ',':
			tokens = append(tokens, token{kind: c})
			i++
		case c == '"' || c == '\'':
			i++
			text, err := lexString(c)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: 's', text: text})
		case c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-':
			start := i
			for i < len(input) {
				d := input[i]
				if !(d == '_' || d == '$' || d >= 'a' && d <= 'z' || d >= 'A' && d <= 'Z' || d >= '0' && d <= '9' || d == '-') {
					break
				}
				i++
			}
			tokens = append(tokens, token{kind: 'i', text: string(input[start:i])})
		default:
			return nil, fmt.Errorf("unexpected character %q", c)
		}
	}
	result := make(map[string]string)
	pos := 0
	expect := func(kind byte) (token, error) {
		if pos >= len(tokens) || tokens[pos].kind != kind {
			return token{}, fmt.Errorf("expected %q", string(kind))
		}
		current := tokens[pos]
		pos++
		return current, nil
	}
	if _, err := expect('{'); err != nil {
		return nil, err
	}
	for {
		if pos < len(tokens) && tokens[pos].kind == '}' {
			pos++
			break
		}
		key, err := expect('i')
		if err != nil {
			if key, err = expect('s'); err != nil {
				return nil, fmt.Errorf("expected a key: %w", err)
			}
		}
		if _, err := expect(':'); err != nil {
			return nil, err
		}
		value, err := expect('s')
		if err != nil {
			return nil, fmt.Errorf("expected a string value: %w", err)
		}
		result[key.text] = value.text
		if pos < len(tokens) && tokens[pos].kind == ',' {
			pos++
			continue
		}
		if _, err := expect('}'); err != nil {
			return nil, err
		}
		break
	}
	if pos != len(tokens) {
		return nil, fmt.Errorf("trailing tokens")
	}
	return result, nil
}

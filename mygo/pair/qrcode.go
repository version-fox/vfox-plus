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
	"net"
	"os"
	"strings"

	"github.com/skip2/go-qrcode"
	"golang.org/x/term"
)

// QRQuietZone is the blank margin around the code, in modules.
const QRQuietZone = 2

// ReachablePairingURL renders scheme://host:port/auth/connect/token for the
// first reachable literal IP in hosts: IPv4 first, then IPv6 bracketed,
// skipping loopback and link-local addresses (LocalAddresses already skips
// them, callers with other sources stay safe). It reports "" when no host is
// usable, for example on a machine without a network route.
func ReachablePairingURL(scheme, port, token string, hosts []string) string {
	var fallback string
	for _, host := range hosts {
		trimmed := strings.Trim(strings.TrimSpace(host), "[]")
		ip := net.ParseIP(trimmed)
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			continue
		}
		if ip.To4() != nil {
			return scheme + "://" + trimmed + portSuffixOf(port) + "/auth/connect/" + token
		}
		if fallback == "" {
			fallback = trimmed
		}
	}
	if fallback == "" {
		return ""
	}
	return scheme + "://[" + fallback + "]" + portSuffixOf(port) + "/auth/connect/" + token
}

// portSuffixOf renders port with a leading colon, unless it is empty.
func portSuffixOf(port string) string {
	if port == "" {
		return ""
	}
	return ":" + strings.TrimPrefix(port, ":")
}

// TerminalQRCode renders url as a QR code for terminals: half-block pairs on
// an explicit white background with a black foreground, so the code scans on
// dark and light themes alike. Every line ends with a reset.
func TerminalQRCode(url string) (string, error) {
	code, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return "", err
	}
	bitmap := code.Bitmap()
	size := len(bitmap)
	padded := size + 2*QRQuietZone
	at := func(x, y int) bool {
		x -= QRQuietZone
		y -= QRQuietZone
		if x < 0 || y < 0 || x >= size || y >= size {
			return false
		}
		return bitmap[y][x]
	}
	const whiteBG = "\x1b[47m"
	const blackFG = "\x1b[30m"
	const reset = "\x1b[0m"
	var out strings.Builder
	for y := 0; y < padded; y += 2 {
		out.WriteString(whiteBG)
		black := false
		for x := 0; x < padded; x++ {
			top, bottom := at(x, y), at(x, y+1)
			switch {
			case top && bottom:
				if !black {
					out.WriteString(blackFG)
					black = true
				}
				out.WriteString("█")
			case top:
				if !black {
					out.WriteString(blackFG)
					black = true
				}
				out.WriteString("▀")
			case bottom:
				if !black {
					out.WriteString(blackFG)
					black = true
				}
				out.WriteString("▄")
			default:
				if black {
					out.WriteString(reset + whiteBG)
					black = false
				}
				out.WriteString(" ")
			}
		}
		out.WriteString(reset + "\n")
	}
	return out.String(), nil
}

// PairingURL renders the pairing link for an explicit host, with the scheme
// and port of the server. The QR code of `vfox-plus serve` uses a LAN address
// here, because the display base URL may hold the 0.0.0.0 placeholder.
func (s *Server) PairingURL(host, token string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return s.scheme() + "://" + host + s.portSuffix() + "/auth/connect/" + token
}

// pairingPort reports the listening port of the server, without the colon.
func (s *Server) pairingPort() string {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return ""
	}
	return port
}

// isTerminal reports whether file is a terminal, so that control sequences like
// the QR code are only printed where they render (a character device such as
// /dev/null is not one).
func isTerminal(file *os.File) bool { return term.IsTerminal(int(file.Fd())) }

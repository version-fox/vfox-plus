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

package webheap

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serverSocket is the server end of a WebSocket handshake: enough to test the
// client, and to stand in for the DevTools server of Chrome.
type serverSocket struct {
	conn   net.Conn
	reader *bufio.Reader
}

// acceptSocket completes the handshake. It refuses a client that sends an
// Origin header, which is what Chrome does unless it is told to allow one.
func acceptSocket(w http.ResponseWriter, r *http.Request) (*serverSocket, error) {
	if r.Header.Get("Origin") != "" {
		http.Error(w, "Rejected an incoming WebSocket connection: an Origin is not allowed", http.StatusForbidden)
		return nil, errors.New("the client sent an Origin header")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("the response cannot be hijacked")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		acceptValue(r.Header.Get("Sec-WebSocket-Key")))
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &serverSocket{conn: conn, reader: rw.Reader}, nil
}

// read returns the next frame of the client, which must be masked.
func (s *serverSocket) read() (final bool, opcode byte, payload []byte, err error) {
	var head [2]byte
	if _, err = io.ReadFull(s.reader, head[:]); err != nil {
		return false, 0, nil, err
	}
	final = head[0]&0x80 != 0
	opcode = head[0] & 0x0f
	if head[1]&0x80 == 0 {
		return false, 0, nil, errors.New("the client sent an unmasked frame")
	}
	length := uint64(head[1] & 0x7f)
	switch length {
	case 126:
		var extended [2]byte
		if _, err = io.ReadFull(s.reader, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended[:]))
	case 127:
		var extended [8]byte
		if _, err = io.ReadFull(s.reader, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended[:])
	}
	var mask [4]byte
	if _, err = io.ReadFull(s.reader, mask[:]); err != nil {
		return false, 0, nil, err
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(s.reader, payload); err != nil {
		return false, 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return final, opcode, payload, nil
}

// send writes one frame, unmasked as a server does.
func (s *serverSocket) send(final bool, opcode byte, payload []byte) error {
	first := opcode
	if final {
		first |= 0x80
	}
	frame := []byte{first}
	switch length := len(payload); {
	case length < 126:
		frame = append(frame, byte(length))
	case length <= 0xffff:
		frame = append(frame, 126, byte(length>>8), byte(length))
	default:
		frame = append(frame, 127)
		frame = binary.BigEndian.AppendUint64(frame, uint64(length))
	}
	_, err := s.conn.Write(append(frame, payload...))
	return err
}

// socketServer starts a server whose WebSocket handler is serve, and returns
// the ws:// address of it.
func socketServer(t *testing.T, serve func(*serverSocket)) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := acceptSocket(w, r)
		if err != nil {
			return
		}
		defer func() { _ = socket.conn.Close() }()
		serve(socket)
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func TestWebSocketMessagesOfEveryLength(t *testing.T) {
	address := socketServer(t, func(s *serverSocket) {
		for {
			_, opcode, payload, err := s.read()
			if err != nil || opcode == opClose {
				return
			}
			if s.send(true, opText, payload) != nil {
				return
			}
		}
	})
	ws, err := dialWebSocket(t.Context(), address)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.close()
	// The edges of the three length encodings, and a message larger than a
	// screenshot of a phone.
	for _, length := range []int{0, 1, 125, 126, 127, 65535, 65536, 300000} {
		payload := []byte(strings.Repeat("é", length/2) + strings.Repeat("x", length%2))
		if err := ws.writeFrame(opText, payload); err != nil {
			t.Fatalf("length %d: %v", length, err)
		}
		echo, err := ws.readMessage()
		if err != nil {
			t.Fatalf("length %d: %v", length, err)
		}
		if string(echo) != string(payload) {
			t.Fatalf("length %d: the echo differs (%d bytes)", length, len(echo))
		}
	}
}

func TestWebSocketControlFramesAndFragments(t *testing.T) {
	pongs := make(chan string, 2)
	address := socketServer(t, func(s *serverSocket) {
		frames := []struct {
			final  bool
			opcode byte
			data   string
		}{
			{true, opPing, "first"},
			{false, opText, "hello, "},
			{true, opPing, "second"}, // A control frame may come inside a message.
			{true, opPong, "ignored"},
			{true, opContinuation, "world"},
			{true, opClose, "\x03\xe8"},
		}
		for _, frame := range frames {
			if s.send(frame.final, frame.opcode, []byte(frame.data)) != nil {
				return
			}
		}
		for len(pongs) < cap(pongs) {
			_, opcode, payload, err := s.read()
			if err != nil {
				return
			}
			if opcode == opPong {
				pongs <- string(payload)
			}
		}
	})
	ws, err := dialWebSocket(t.Context(), address)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.close()
	message, err := ws.readMessage()
	if err != nil || string(message) != "hello, world" {
		t.Fatalf("got %q, %v", message, err)
	}
	if _, err := ws.readMessage(); !errors.Is(err, errPeerClosed) {
		t.Fatalf("after the close frame: %v", err)
	}
	if first, second := <-pongs, <-pongs; first != "first" || second != "second" {
		t.Fatalf("pongs %q and %q", first, second)
	}
}

func TestWebSocketRefusesBrokenFrames(t *testing.T) {
	for name, frames := range map[string][][]byte{
		"reserved bits":     {{0xc1, 0x00}},
		"control too long":  {append([]byte{0x89, 126, 0x00, 0x80}, make([]byte, 128)...)},
		"fragmented ping":   {{0x09, 0x00}},
		"lone continuation": {{0x80, 0x00}},
		"unknown opcode":    {{0x83, 0x00}},
		"message in message": {
			{0x01, 0x01, 'a'}, // A text frame that is not final...
			{0x81, 0x01, 'b'}, // ...and another text frame before its end.
		},
	} {
		t.Run(name, func(t *testing.T) {
			address := socketServer(t, func(s *serverSocket) {
				for _, frame := range frames {
					if _, err := s.conn.Write(frame); err != nil {
						return
					}
				}
				_, _, _, _ = s.read() // Stay until the client leaves.
			})
			ws, err := dialWebSocket(t.Context(), address)
			if err != nil {
				t.Fatal(err)
			}
			defer ws.close()
			if message, err := ws.readMessage(); err == nil {
				t.Fatalf("got %q, want an error", message)
			}
		})
	}
}

func TestWebSocketHandshakeFailures(t *testing.T) {
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Rejected an incoming WebSocket connection from the origin", http.StatusForbidden)
	}))
	defer refusing.Close()
	_, err := dialWebSocket(t.Context(), "ws"+strings.TrimPrefix(refusing.URL, "http")+"/devtools/page/1")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "Rejected an incoming") {
		t.Errorf("a refused handshake: %v", err)
	}

	wrongAccept := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("Connection", "Upgrade")
		w.Header().Set("Sec-WebSocket-Accept", "AAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	defer wrongAccept.Close()
	_, err = dialWebSocket(t.Context(), "ws"+strings.TrimPrefix(wrongAccept.URL, "http"))
	if err == nil || !strings.Contains(err.Error(), "accept") {
		t.Errorf("a wrong accept value: %v", err)
	}

	for _, address := range []string{"http://127.0.0.1:1/", "wss://127.0.0.1:1/", "ws:///path", "::bad::"} {
		if _, err := dialWebSocket(t.Context(), address); err == nil {
			t.Errorf("%q: want an error", address)
		}
	}
	// Nothing listens on port 1.
	if _, err := dialWebSocket(t.Context(), "ws://127.0.0.1:1/"); err == nil {
		t.Error("a refused connection: want an error")
	}
}

func TestAcceptValueOfRFC6455(t *testing.T) {
	// The example of RFC 6455 section 1.3.
	if got := acceptValue("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Errorf("got %s", got)
	}
}

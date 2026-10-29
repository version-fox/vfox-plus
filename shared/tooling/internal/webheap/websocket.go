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
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6455 defines the accept value with SHA-1; it protects nothing.
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// A minimal WebSocket client (RFC 6455) with what the Chrome DevTools Protocol
// needs: whole messages in both directions over a plain TCP connection to a
// browser on this machine. There is no TLS, compression or extension. The
// handshake sends no Origin header, as the WebSocket of Node.js did for the
// script that this replaces: Chrome refuses an Origin that
// --remote-allow-origins does not list, and golang.org/x/net/websocket cannot
// leave it out.

const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xa
)

const (
	// handshakeGUID is the constant of RFC 6455 section 1.3 that the accept
	// value of the server is made with.
	handshakeGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	// maxMessage bounds one message; a screenshot is the largest thing that
	// Chrome sends.
	maxMessage       = 64 << 20
	handshakeTimeout = 10 * time.Second
)

// errPeerClosed is what reading returns once the browser has closed the
// connection.
var errPeerClosed = errors.New("the browser closed the WebSocket")

// wsConn is the client end of one WebSocket connection. One goroutine reads
// while any number write.
type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
	write  sync.Mutex
}

// acceptValue is the Sec-WebSocket-Accept value that answers a challenge.
func acceptValue(challenge string) string {
	sum := sha1.Sum([]byte(challenge + handshakeGUID)) //nolint:gosec // See the import.
	return base64.StdEncoding.EncodeToString(sum[:])
}

// dialWebSocket opens the ws:// address.
func dialWebSocket(ctx context.Context, address string) (*wsConn, error) {
	target, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "ws" || target.Host == "" {
		return nil, fmt.Errorf("not a ws:// address: %q", address)
	}
	host := target.Host
	if target.Port() == "" {
		host = net.JoinHostPort(target.Hostname(), "80")
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, err
	}
	client := &wsConn{conn: conn, reader: bufio.NewReader(conn)}
	if err := client.handshake(ctx, target); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *wsConn) handshake(ctx context.Context, target *url.URL) error {
	deadline := time.Now().Add(handshakeTimeout)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	if err := c.conn.SetDeadline(deadline); err != nil {
		return err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	challenge := base64.StdEncoding.EncodeToString(nonce)
	request := "GET " + target.RequestURI() + " HTTP/1.1\r\n" +
		"Host: " + target.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + challenge + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(c.conn, request); err != nil {
		return err
	}
	response, err := http.ReadResponse(c.reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusSwitchingProtocols {
		// Chrome says in the body why it refuses (for example an Origin).
		reason, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("the WebSocket handshake answered %s: %s", response.Status, reason)
	}
	if response.Header.Get("Sec-WebSocket-Accept") != acceptValue(challenge) {
		return errors.New("the WebSocket handshake answered a wrong accept value")
	}
	return c.conn.SetDeadline(time.Time{})
}

// readFrame reads one frame.
func (c *wsConn) readFrame() (final bool, opcode byte, payload []byte, err error) {
	var head [2]byte
	if _, err = io.ReadFull(c.reader, head[:]); err != nil {
		return false, 0, nil, err
	}
	final = head[0]&0x80 != 0
	if head[0]&0x70 != 0 {
		return false, 0, nil, errors.New("the browser set reserved WebSocket bits")
	}
	opcode = head[0] & 0x0f
	masked := head[1]&0x80 != 0
	length := uint64(head[1] & 0x7f)
	switch length {
	case 126:
		var extended [2]byte
		if _, err = io.ReadFull(c.reader, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended[:]))
	case 127:
		var extended [8]byte
		if _, err = io.ReadFull(c.reader, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended[:])
	}
	if length > maxMessage {
		return false, 0, nil, fmt.Errorf("a WebSocket frame of %d bytes is too large", length)
	}
	if opcode >= opClose && (length > 125 || !final) {
		return false, 0, nil, errors.New("the browser sent an invalid control frame")
	}
	var mask [4]byte
	if masked { // A server must not, but nothing here depends on it.
		if _, err = io.ReadFull(c.reader, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(c.reader, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return final, opcode, payload, nil
}

// readMessage returns the next whole text or binary message. It answers pings
// and skips pongs on the way, and fails with errPeerClosed on a close frame.
func (c *wsConn) readMessage() ([]byte, error) {
	var message []byte
	started := false
	for {
		final, opcode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case opPing:
			if err := c.writeFrame(opPong, payload); err != nil {
				return nil, err
			}
			continue
		case opPong:
			continue
		case opClose:
			return nil, errPeerClosed
		case opText, opBinary:
			if started {
				return nil, errors.New("a WebSocket message began inside another one")
			}
			started = true
		case opContinuation:
			if !started {
				return nil, errors.New("a WebSocket continuation has no message")
			}
		default:
			return nil, fmt.Errorf("unknown WebSocket opcode %d", opcode)
		}
		message = append(message, payload...)
		if len(message) > maxMessage {
			return nil, fmt.Errorf("a WebSocket message of more than %d bytes", maxMessage)
		}
		if final {
			return message, nil
		}
	}
}

// writeFrame sends one final frame. A client masks every frame it sends. The
// header and the masked payload go out in one write.
func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > maxMessage {
		return fmt.Errorf("a WebSocket message of %d bytes is too large to send", len(payload))
	}
	var header [14]byte
	header[0] = 0x80 | opcode
	n := 2
	switch length := len(payload); {
	case length < 126:
		header[1] = 0x80 | byte(length)
	case length <= 0xffff:
		header[1] = 0x80 | 126
		binary.BigEndian.PutUint16(header[2:], uint16(length))
		n = 4
	default:
		header[1] = 0x80 | 127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
		n = 10
	}
	mask := header[n : n+4]
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	n += 4
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%4]
	}
	c.write.Lock()
	defer c.write.Unlock()
	buffers := net.Buffers{header[:n], masked}
	_, err := buffers.WriteTo(c.conn)
	return err
}

// close says goodbye (status 1000) and drops the connection. It does not wait.
func (c *wsConn) close() {
	_ = c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	_ = c.writeFrame(opClose, []byte{0x03, 0xe8})
	_ = c.conn.Close()
}

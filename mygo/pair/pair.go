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
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/version-fox/vfox-plus/mygo/state"
)

// DefaultTokenTTL is how long a pairing link works: five minutes.
const DefaultTokenTTL = 5 * time.Minute

// DefaultBind is the interface the server on this computer listens on: the
// IPv4 loopback, so a pairing link never leaves the machine.
const DefaultBind = "127.0.0.1"

// DefaultSessionTTL is how long a device that connects over the network stays
// paired. Zero means the session lives as long as the server.
const DefaultSessionTTL = 24 * time.Hour

// Control is what the server asks of the application besides vfox: how
// the other devices can connect, and the end of the application. The Host
// implements it. Its listeners are told when the status may have changed.
type Control interface {
	state.Listenable
	// Status describes the server for other devices.
	Status() Status
	// RemoteLink mints a pairing link for a device on the network, with host as
	// the address of this computer in it, or the most likely one when host is
	// empty. It fails while no server listens for other devices.
	RemoteLink(host string) (string, error)
	// Quit ends the application.
	Quit()
}

// Options configures a Server.
type Options struct {
	// Bind is the interface to listen on. Empty means DefaultBind. Only
	// loopback addresses are accepted unless AllowNonLoopback is set, for the
	// server that other devices connect to.
	Bind string
	// AllowNonLoopback permits listening on a non-loopback interface. Pairing
	// links then travel the network, so such a server speaks HTTPS only
	// (TLSCertFile and TLSKeyFile).
	AllowNonLoopback bool
	// Port is the port to listen on. Zero picks a free one.
	Port int
	// TokenTTL is how long a pairing link works. Zero means DefaultTokenTTL.
	TokenTTL time.Duration
	// SessionTTL is how long a paired browser stays paired. Zero means the
	// session lives as long as the server. Expired sessions answer 401 and
	// must pair again.
	SessionTTL time.Duration
	// Model is the application state behind the server.
	Model *state.AppModel
	// TLSCertFile and TLSKeyFile switch the server to HTTPS. Empty means
	// plain HTTP, which only loopback binds use.
	TLSCertFile string
	TLSKeyFile  string
	// Control answers what the server asks of the application besides
	// vfox. Nil leaves those routes unavailable.
	Control Control
}

// Server is a pairing server. Create it with NewServer, hand out links with
// NewToken, and close it with Close.
type Server struct {
	listener   net.Listener
	httpServer *http.Server
	handler    http.Handler
	// crossOrigin refuses the cross-origin requests of browsers that change state.
	crossOrigin *http.CrossOriginProtection
	mu          sync.Mutex
	tokens      map[string]time.Time
	sessions    map[string]time.Time
	ttl         time.Duration
	sttl        time.Duration
	model       *state.AppModel
	control     Control
	tls         bool
}

// NewServer starts a pairing server, by default on the loopback interface.
func NewServer(options Options) (*Server, error) {
	if options.Model == nil {
		return nil, errors.New("pair: a server needs the application state to serve")
	}
	bind := options.Bind
	if bind == "" {
		bind = DefaultBind
	}
	if !options.AllowNonLoopback {
		if ip := net.ParseIP(bind); ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("pair: refusing to listen on non-loopback %q", bind)
		}
	}
	if (options.TLSCertFile == "") != (options.TLSKeyFile == "") {
		return nil, fmt.Errorf("pair: a TLS certificate and key go together")
	}
	if options.AllowNonLoopback && options.TLSCertFile == "" {
		return nil, fmt.Errorf("pair: a server that other devices connect to needs a TLS certificate")
	}
	ttl := options.TokenTTL
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	var certificate tls.Certificate
	if options.TLSCertFile != "" {
		var err error
		certificate, err = tls.LoadX509KeyPair(options.TLSCertFile, options.TLSKeyFile)
		if err != nil {
			return nil, err
		}
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(bind, fmt.Sprint(options.Port)))
	if err != nil {
		return nil, err
	}
	server := &Server{
		listener:    listener,
		crossOrigin: http.NewCrossOriginProtection(),
		tokens:      make(map[string]time.Time),
		sessions:    make(map[string]time.Time),
		ttl:         ttl,
		sttl:        options.SessionTTL,
		model:       options.Model,
		control:     options.Control,
	}
	if options.TLSCertFile != "" {
		server.tls = true
		listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/connect/", server.serveConnect)
	mux.HandleFunc("/api/token", server.serveToken)
	mux.HandleFunc("/api/inventory", server.serveInventory)
	mux.HandleFunc("/api/sdks/", server.serveSdks)
	mux.HandleFunc("/api/doctor", server.serveDoctor)
	mux.HandleFunc("/api/tasks", server.serveTasks)
	mux.HandleFunc("/api/tasks/", server.serveTask)
	mux.HandleFunc("/api/plugins", server.servePlugins)
	mux.HandleFunc("/api/settings", server.serveSettings)
	mux.HandleFunc("/api/web", server.serveWeb)
	mux.HandleFunc("/api/web/pairing", server.serveWebPairing)
	mux.HandleFunc("/api/quit", server.serveQuit)
	mux.HandleFunc("/api/config", server.serveConfig)
	mux.HandleFunc("/api/events", server.serveEvents)
	server.handler = mux
	// ReadHeaderTimeout only: the server answers big logs and keeps
	// server-sent events open, so the body and the writes stay unbounded.
	server.httpServer = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		_ = server.httpServer.Serve(listener)
	}()
	return server, nil
}

// Addr is the address the server listens on.
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

// scheme is the URL scheme of the server.
func (s *Server) scheme() string {
	if s.tls {
		return "https"
	}
	return "http"
}

// BaseURL is the URL of the server, without a trailing slash.
func (s *Server) BaseURL() string {
	host, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return ""
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return s.scheme() + "://" + host + ":" + port
}

// NewToken creates a one-time pairing link. Expired unused tokens are swept
// along the way, so minting links for more browsers cannot grow the map
// without bound on a long-lived server.
func (s *Server) NewToken() (token string, url string, err error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(bytes[:])
	now := time.Now()
	s.mu.Lock()
	for known, expiry := range s.tokens {
		if !now.Before(expiry) {
			delete(s.tokens, known)
		}
	}
	s.tokens[token] = now.Add(s.ttl)
	s.mu.Unlock()
	return token, s.BaseURL() + "/auth/connect/" + token, nil
}

// Close stops the server and ends every connection to it, the open streams of
// the paired browsers included. The port is free when it returns.
//
// http.Server.Close closes the listeners that Serve has registered, and the
// goroutine that NewServer made may not have started Serve yet: a server that
// is closed at once has none registered, and its port stays taken until that
// goroutine runs and finds the server closed. The host stops the server for
// other devices and starts it again on the same port when the certificate
// changes, and the second server would fail with "address already in use". So
// the listener is closed here too.
func (s *Server) Close() error {
	err := s.httpServer.Close()
	if closeErr := s.listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && err == nil {
		err = closeErr
	}
	return err
}

// takeToken consumes token. It reports whether the token was valid, unused
// and unexpired.
func (s *Server) takeToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.tokens[token]
	if !ok {
		return false
	}
	delete(s.tokens, token)
	return time.Now().Before(expiry)
}

// newSession issues a session cookie value for a paired browser.
func (s *Server) newSession() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		// rand.Read only fails when the operating system is out of entropy;
		// fall back to a timestamped value rather than refusing to pair.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	session := hex.EncodeToString(bytes[:])
	s.mu.Lock()
	s.sessions[session] = time.Now()
	s.mu.Unlock()
	return session
}

// hasSession reports whether the request carries a paired session that has
// not expired yet. Expired sessions are dropped.
func (s *Server) hasSession(request *http.Request) bool {
	cookie, err := request.Cookie("vfox-plus-session")
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	created, ok := s.sessions[cookie.Value]
	if !ok {
		return false
	}
	if s.sttl > 0 && time.Since(created) > s.sttl {
		delete(s.sessions, cookie.Value)
		return false
	}
	return true
}

// publicURL renders a pairing path with the host the client used, so that a
// browser behind a port mapping gets a link it can open. It falls back to
// the listening address when the request carries no usable host.
func (s *Server) publicURL(request *http.Request, token string) string {
	host := request.Host
	if host == "" {
		return s.BaseURL() + "/auth/connect/" + token
	}
	if _, _, err := net.SplitHostPort(host); err != nil {
		// No port in the Host header: the client reached us on the
		// listening port.
		host += s.portSuffix()
	}
	return s.scheme() + "://" + host + "/auth/connect/" + token
}

// portSuffix renders the listening port for hosts without one.
func (s *Server) portSuffix() string {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil || port == "" || port == "80" {
		return ""
	}
	return ":" + port
}

// serveConnect pairs a browser holding a one-time link, and answers with the
// session cookie and 204 No Content. This server serves no page to send the
// browser to: the interface is the gateway's.
func (s *Server) serveConnect(writer http.ResponseWriter, request *http.Request) {
	token := strings.TrimPrefix(request.URL.Path, "/auth/connect/")
	if token == "" || strings.Contains(token, "/") || !s.takeToken(token) {
		http.Error(writer, "This pairing link is invalid, already used or expired. Open vfox+ on the computer, go to Settings and get a new one.", http.StatusForbidden)
		return
	}
	session := s.newSession()
	// Secure unconditionally: loopback origins (localhost, 127.0.0.1) are
	// trustworthy, so browsers send Secure cookies back over plain HTTP
	// there; every non-loopback bind serves HTTPS, where Secure is required.
	// Static analysis (CodeQL, gosec) only accepts the literal.
	http.SetCookie(writer, &http.Cookie{
		Name:     "vfox-plus-session",
		Value:    session,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writer.WriteHeader(http.StatusNoContent)
}

// serveToken mints another one-time pairing link for an already paired
// browser, so that a second browser can join without returning to the
// computer. It renders the link with the host the caller used, which is the
// reachable address behind a port mapping.
func (s *Server) serveToken(writer http.ResponseWriter, request *http.Request) {
	if !s.hasSession(request) {
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
		return
	}
	token, _, err := s.NewToken()
	if err != nil {
		http.Error(writer, "Cannot mint a link right now.", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(map[string]string{"url": s.publicURL(request, token)})
}

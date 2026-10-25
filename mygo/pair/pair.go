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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// DefaultTokenTTL is how long a pairing link works, like the five minutes of
// `opencode pair`.
const DefaultTokenTTL = 5 * time.Minute

// DefaultBind is the only interface the server listens on: the IPv4
// loopback, so a pairing link never leaves the machine.
const DefaultBind = "127.0.0.1"

// SdkSummary is one installed SDK on the pairing page.
type SdkSummary struct {
	// Name is the plugin name, for example "nodejs".
	Name string `json:"name"`
	// Current is the version in use, if any.
	Current string `json:"current"`
	// HasCurrent reports whether Current is set.
	HasCurrent bool `json:"hasCurrent"`
	// Versions are the installed versions.
	Versions []string `json:"versions"`
}

// Snapshot is the read-only state the paired browser sees.
type Snapshot struct {
	// HasVfox reports whether vfox answered `--version`.
	HasVfox bool `json:"hasVfox"`
	// VfoxVersion is the version of vfox, when it answered.
	VfoxVersion string `json:"vfoxVersion,omitempty"`
	// VfoxError is why vfox is not available, when it is not.
	VfoxError string `json:"vfoxError,omitempty"`
	// SDKs are the installed SDKs, when vfox answered `list`.
	SDKs []SdkSummary `json:"sdks"`
}

// Summarize converts installed SDKs to the shape the page shows.
func Summarize(installed []core.InstalledSdk) []SdkSummary {
	summaries := make([]SdkSummary, 0, len(installed))
	for _, sdk := range installed {
		summaries = append(summaries, SdkSummary{
			Name:       sdk.Name,
			Current:    sdk.Current,
			HasCurrent: sdk.HasCurrent,
			Versions:   append([]string(nil), sdk.Versions...),
		})
	}
	return summaries
}

// DefaultSessionTTL is how long a paired browser stays paired on a
// persistent server. Zero means the session lives as long as the server.
const DefaultSessionTTL = 24 * time.Hour

// Options configures a Server.
type Options struct {
	// Bind is the interface to listen on. Empty means DefaultBind. Only
	// loopback addresses are accepted unless AllowNonLoopback is set, for a
	// server behind a port mapping or on a trusted network.
	Bind string
	// AllowNonLoopback permits listening on a non-loopback interface. Pair
	// links then travel the network: only use it where the network is
	// trusted, and prefer pairing over an encrypted tunnel.
	AllowNonLoopback bool
	// Port is the port to listen on. Zero picks a free one.
	Port int
	// TokenTTL is how long a pairing link works. Zero means DefaultTokenTTL.
	TokenTTL time.Duration
	// SessionTTL is how long a paired browser stays paired. Zero means the
	// session lives as long as the server. Expired sessions answer 401 and
	// must pair again.
	SessionTTL time.Duration
	// Snapshot returns the read-only state the paired browser sees.
	Snapshot func() Snapshot
	// Model is the application state behind the web interface. When nil,
	// the server only pairs and serves the read-only snapshot: the
	// `vfox-plus pair` mode. `vfox-plus serve` hands it a started AppModel.
	Model *state.AppModel
}

// Server is a pairing server. Create it with NewServer, hand out links with
// NewToken, and close it with Close.

// Server is a pairing server. Create it with NewServer, hand out links with
// NewToken, and close it with Close.
type Server struct {
	listener   net.Listener
	httpServer *http.Server
	handler    http.Handler
	mu         sync.Mutex
	tokens     map[string]time.Time
	sessions   map[string]time.Time
	ttl        time.Duration
	sttl       time.Duration
	snapshot   func() Snapshot
	model      *state.AppModel
}

// NewServer starts a pairing server, by default on the loopback interface.
func NewServer(options Options) (*Server, error) {
	bind := options.Bind
	if bind == "" {
		bind = DefaultBind
	}
	if !options.AllowNonLoopback {
		if ip := net.ParseIP(bind); ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("pair: refusing to listen on non-loopback %q", bind)
		}
	}
	ttl := options.TokenTTL
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	snapshot := options.Snapshot
	if snapshot == nil {
		snapshot = func() Snapshot { return Snapshot{} }
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, options.Port))
	if err != nil {
		return nil, err
	}
	server := &Server{
		listener: listener,
		tokens:   make(map[string]time.Time),
		sessions: make(map[string]time.Time),
		ttl:      ttl,
		sttl:     options.SessionTTL,
		snapshot: snapshot,
		model:    options.Model,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.serveIntro)
	mux.HandleFunc("/auth/connect/", server.serveConnect)
	mux.HandleFunc("/api/status", server.serveStatus)
	mux.HandleFunc("/api/token", server.serveToken)
	mux.HandleFunc("/app", server.serveApp)
	mux.HandleFunc("/api/inventory", server.serveInventory)
	mux.HandleFunc("/api/sdks/", server.serveSdks)
	mux.HandleFunc("/api/tasks", server.serveTasks)
	mux.HandleFunc("/api/tasks/", server.serveTask)
	mux.HandleFunc("/api/plugins", server.servePlugins)
	mux.HandleFunc("/api/settings", server.serveSettings)
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

// BaseURL is the URL of the server, without a trailing slash.
func (s *Server) BaseURL() string {
	host, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return ""
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return "http://" + host + ":" + port
}

// DisplayBaseURL is the URL to show to the user. When the server listens on
// a wildcard address, browsers cannot open it as is, so it renders
// 0.0.0.0 as a placeholder for the reachable address.
func (s *Server) DisplayBaseURL() string {
	host, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return s.BaseURL()
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); host == "" || (ip != nil && ip.IsUnspecified()) {
		return "http://0.0.0.0:" + port
	}
	return s.BaseURL()
}

// NewToken creates a one-time pairing link.
func (s *Server) NewToken() (token string, url string, err error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(bytes[:])
	s.mu.Lock()
	s.tokens[token] = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return token, s.BaseURL() + "/auth/connect/" + token, nil
}

// Close stops the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
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
	return "http://" + host + "/auth/connect/" + token
}

// portSuffix renders the listening port for hosts without one.
func (s *Server) portSuffix() string {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil || port == "" || port == "80" {
		return ""
	}
	return ":" + port
}

// serveIntro answers without a token. It never mentions tokens, so that
// scanners learn nothing about pairing links.
func (s *Server) serveIntro(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = introTemplate.Execute(writer, nil)
}

// serveConnect pairs a browser holding a one-time link.
func (s *Server) serveConnect(writer http.ResponseWriter, request *http.Request) {
	token := strings.TrimPrefix(request.URL.Path, "/auth/connect/")
	if token == "" || strings.Contains(token, "/") || !s.takeToken(token) {
		http.Error(writer, "This pairing link is invalid, already used or expired. Run `vfox-plus pair` again for a new one.", http.StatusForbidden)
		return
	}
	session := s.newSession()
	// No Secure attribute on purpose: the server speaks plain HTTP, and
	// Secure would stop browsers from sending the cookie back over HTTP
	// outside localhost, breaking pairing. Loopback traffic never leaves
	// the machine; beyond it, serve warns to prefer an encrypted tunnel.
	http.SetCookie(writer, &http.Cookie{ //nolint:gosec // Plain HTTP by design; Secure would break pairing outside localhost.
		Name:     "vfox-plus-session",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = connectTemplate.Execute(writer, connectData{
		Snapshot: s.snapshot(),
		Chinese:  prefersChinese(request.Header.Get("Accept-Language")),
	})
}

// serveStatus answers the read-only API for paired browsers.
func (s *Server) serveStatus(writer http.ResponseWriter, request *http.Request) {
	if !s.hasSession(request) {
		http.Error(writer, "Not paired. Open a `vfox-plus pair` link first.", http.StatusUnauthorized)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(s.snapshot())
}

// serveToken mints another one-time pairing link for an already paired
// browser, so that a second browser can join without returning to the
// terminal. It renders the link with the host the caller used, which is the
// reachable address behind a port mapping.
func (s *Server) serveToken(writer http.ResponseWriter, request *http.Request) {
	if !s.hasSession(request) {
		http.Error(writer, "Not paired. Open a `vfox-plus pair` link first.", http.StatusUnauthorized)
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

// prefersChinese reports whether the browser prefers Simplified Chinese.
func prefersChinese(header string) bool {
	for _, part := range strings.Split(header, ",") {
		language := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if strings.HasPrefix(language, "zh") {
			return true
		}
		if language == "" {
			continue
		}
		if language == "*" {
			continue
		}
		return false
	}
	return false
}

type connectData struct {
	Snapshot
	Chinese bool
}

var introTemplate = template.Must(template.New("intro").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>vfox-plus pair</title></head>
<body>
<h1>vfox-plus pair</h1>
<p>This page pairs a browser with the vfox-plus instance on this machine.</p>
<p>Run <code>vfox-plus pair</code> in a terminal (bash on Ubuntu, PowerShell on Windows) and open the link it prints. Links work once and expire after a few minutes.</p>
<p>运行 <code>vfox-plus pair</code>（Ubuntu 用 bash，Windows 用 PowerShell），在浏览器打开它输出的一次性链接即可配对。链接一次性有效，几分钟后过期。</p>
</body>
</html>`))

var connectTemplate = template.Must(template.New("connect").Parse(`<!doctype html>
<html lang="{{if .Chinese}}zh-CN{{else}}en{{end}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>vfox-plus {{if .Chinese}}已配对{{else}}paired{{end}}</title></head>
<body>
<h1>vfox-plus {{if .Chinese}}已配对{{else}}paired{{end}}</h1>
{{if .Chinese}}
<p>此浏览器已与本机的 vfox-plus 配对。配对链接一次性有效，请勿分享。</p>
{{else}}
<p>This browser is paired with vfox-plus on this machine. Pairing links work once; do not share them.</p>
{{end}}
<h2>vfox</h2>
{{if .HasVfox}}
<p><code>{{.VfoxVersion}}</code></p>
<h2>{{if .Chinese}}已安装{{else}}Installed{{end}}</h2>
{{if .SDKs}}
<ul>
{{range .SDKs}}<li><code>{{.Name}}</code>{{if .HasCurrent}} — {{.Current}}{{end}} ({{range $i, $v := .Versions}}{{if $i}}, {{end}}{{$v}}{{end}})</li>{{end}}
</ul>
{{else}}
<p>{{if .Chinese}}暂无已安装的 SDK。{{else}}No SDKs installed yet.{{end}}</p>
{{end}}
{{else}}
<p>{{if .Chinese}}未找到可用的 vfox：{{.VfoxError}}。请先安装 vfox（https://vfox.dev/zh-hans/guides/quick-start.html），再重新配对。{{else}}No working vfox: {{.VfoxError}}. Install vfox first (https://vfox.dev/guides/quick-start.html), then pair again.{{end}}</p>
{{end}}
</body>
</html>`))

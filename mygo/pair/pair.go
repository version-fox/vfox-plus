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

// DefaultTokenTTL is how long a pairing link works: five minutes.
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
	// TLSCertFile and TLSKeyFile switch the server to HTTPS. Empty means
	// plain HTTP, which only loopback binds use.
	TLSCertFile string
	TLSKeyFile  string
	// Fingerprint is the SHA-256 fingerprint of the TLS certificate, shown
	// on the intro and pairing pages for comparison with the browser.
	Fingerprint string
	// AdminToken mints pairing links over loopback without a session: it
	// authenticates the local CLI (`pair`, `serve --bind 127.0.0.1`) against
	// the shared server recorded in the service file. It is refused on
	// non-loopback connections even when known, and it never becomes a
	// cookie. Empty disables it.
	AdminToken string
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
	snapshot    func() Snapshot
	model       *state.AppModel
	tls         bool
	// fingerprint is the SHA-256 fingerprint of the TLS certificate,
	// shown on the intro and pairing pages for comparison.
	fingerprint string
	// adminToken mints pairing links over loopback without a session.
	adminToken string
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
		listener:    listener,
		crossOrigin: http.NewCrossOriginProtection(),
		tokens:      make(map[string]time.Time),
		sessions:    make(map[string]time.Time),
		ttl:         ttl,
		sttl:        options.SessionTTL,
		snapshot:    snapshot,
		model:       options.Model,
		fingerprint: options.Fingerprint,
		adminToken:  options.AdminToken,
	}
	if options.TLSCertFile != "" || options.TLSKeyFile != "" {
		if options.TLSCertFile == "" || options.TLSKeyFile == "" {
			_ = listener.Close()
			return nil, fmt.Errorf("pair: a TLS certificate and key go together")
		}
		cert, err := tls.LoadX509KeyPair(options.TLSCertFile, options.TLSKeyFile)
		if err != nil {
			_ = listener.Close()
			return nil, err
		}
		server.tls = true
		listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{cert}})
		server.listener = listener
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.serveIntro)
	mux.HandleFunc("/auth/connect/", server.serveConnect)
	mux.HandleFunc("/api/status", server.serveStatus)
	mux.HandleFunc("/api/token", server.serveToken)
	mux.HandleFunc("/app", server.serveApp)
	mux.HandleFunc("/manifest.webmanifest", server.serveManifest)
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

// DisplayBaseURL is the URL to show to the user. When the server listens on
// a wildcard address, browsers cannot open it as is, so it renders
// 0.0.0.0 as a placeholder for the reachable address.
func (s *Server) DisplayBaseURL() string {
	host, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return s.BaseURL()
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); host == "" || (ip != nil && ip.IsUnspecified()) {
		return s.scheme() + "://0.0.0.0:" + port
	}
	return s.BaseURL()
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

// allowAdmin reports whether request carries the admin bearer over
// loopback. The token lives in a 0600 file on this machine and only ever
// travels loopback; it is refused anywhere else even when correct, and it
// never becomes a session cookie.
func (s *Server) allowAdmin(request *http.Request) bool {
	if s.adminToken == "" {
		return false
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return false
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return false
	}
	scheme, token, ok := strings.Cut(request.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	if len(token) != len(s.adminToken) {
		return false
	}
	match := 0
	for i := 0; i < len(token); i++ {
		match |= int(token[i] ^ s.adminToken[i])
	}
	return match == 0
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

// serveIntro answers without a token. It never mentions tokens, so that
// scanners learn nothing about pairing links.
func (s *Server) serveIntro(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = introTemplate.Execute(writer, map[string]string{"Fingerprint": s.fingerprint})
}

// serveConnect pairs a browser holding a one-time link.
func (s *Server) serveConnect(writer http.ResponseWriter, request *http.Request) {
	token := strings.TrimPrefix(request.URL.Path, "/auth/connect/")
	if token == "" || strings.Contains(token, "/") || !s.takeToken(token) {
		http.Error(writer, "This pairing link is invalid, already used or expired. Run `vfox-plus pair` again for a new one.", http.StatusForbidden)
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
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = connectTemplate.Execute(writer, connectData{
		Snapshot:    s.snapshot(),
		Chinese:     prefersChinese(request.Header.Get("Accept-Language")),
		Fingerprint: s.fingerprint,
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
// reachable address behind a port mapping. The local CLI mints the same way
// with the admin bearer over loopback, without a session.
func (s *Server) serveToken(writer http.ResponseWriter, request *http.Request) {
	if !s.allowAdmin(request) && !s.hasSession(request) {
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
	Chinese     bool
	Fingerprint string
}

var introTemplate = template.Must(template.New("intro").Parse(withFavicon(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>vfox-plus pair</title><!--favicon--></head>
<body>
<h1>vfox-plus pair</h1>
<p>This page pairs a browser with the vfox-plus instance on this machine.</p>
<p>Run <code>vfox-plus pair</code> in a terminal (bash on Ubuntu, PowerShell on Windows) and open the link it prints. Links work once and expire after a few minutes.</p>
<p>运行 <code>vfox-plus pair</code>（Ubuntu 用 bash，Windows 用 PowerShell），在浏览器打开它输出的一次性链接即可配对。链接一次性有效，几分钟后过期。</p>
{{if .Fingerprint}}<p>Certificate fingerprint: <code>{{.Fingerprint}}</code></p>
<p>证书指纹：<code>{{.Fingerprint}}</code>（信任自签名证书前请核对）。</p>{{end}}
</body>
</html>`)))

var connectTemplate = template.Must(template.New("connect").Parse(withFavicon(`<!doctype html>
<html lang="{{if .Chinese}}zh-CN{{else}}en{{end}}">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>vfox-plus {{if .Chinese}}已配对{{else}}paired{{end}}</title><!--favicon--></head>
<body>
<h1>vfox-plus {{if .Chinese}}已配对{{else}}paired{{end}}</h1>
{{if .Chinese}}
<p>此浏览器已与本机的 vfox-plus 配对。配对链接一次性有效，请勿分享。</p>
{{if .Fingerprint}}<p>证书指纹：<code>{{.Fingerprint}}</code>。</p>{{end}}
{{else}}
<p>This browser is paired with vfox-plus on this machine. Pairing links work once; do not share them.</p>
{{if .Fingerprint}}<p>Certificate fingerprint: <code>{{.Fingerprint}}</code>.</p>{{end}}
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
</html>`)))

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

package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"unicode"
	"unicode/utf8"
)

// upstreamKey carries the instance that a request is forwarded to, from the handler that chose it to the
// proxy: the instance and the session it began with are the ones it finishes with, even if the gateway
// is paired again meanwhile.
type upstreamKey struct{}

func upstreamOf(request *http.Request) *upstream {
	up, _ := request.Context().Value(upstreamKey{}).(*upstream)
	return up
}

// instanceRoundTripper sends a request to the instance that the request carries.
type instanceRoundTripper struct{}

func (instanceRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	up := upstreamOf(request)
	if up == nil {
		return nil, errNotConnected
	}
	return up.transport.RoundTrip(request)
}

// newProxy makes the proxy that forwards the page and the routes of the instance.
func (g *Gateway) newProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite:        rewrite,
		ModifyResponse: g.modify,
		ErrorHandler:   g.fail,
		Transport:      instanceRoundTripper{},
		// The events of the page are server-sent: every piece goes on as it comes.
		FlushInterval: -1,
		ErrorLog:      log.New(io.Discard, "", 0),
	}
}

// rewrite points the request at the instance and takes out of it what is the browser's and the gateway's, and
// not for the instance: the cookies and the headers that say where the browser is from or who sent it. The
// instance refuses a state-changing request whose origin is not its own host (the gateway checked the origin
// against its own before it forwarded the request), and it would take the session of the browser for its own.
func rewrite(request *httputil.ProxyRequest) {
	up := upstreamOf(request.In)
	request.SetURL(up.base)
	header := request.Out.Header
	for name := range header {
		if strippedHeader(name) {
			delete(header, name)
		}
	}
	if up.session != "" {
		header.Set("Cookie", instanceCookie+"="+up.session)
	}
}

// strippedHeader says whether a header of the browser stays with the gateway.
func strippedHeader(name string) bool {
	switch name = http.CanonicalHeaderKey(name); name {
	case "Cookie", "Origin", "Referer", "Authorization", "Proxy-Authorization", "Forwarded", "X-Real-Ip":
		return true
	}
	return strings.HasPrefix(name, "X-Forwarded-") || strings.HasPrefix(name, "Sec-Fetch-")
}

// modify keeps from the browser what the instance sets for the gateway, its cookie above all, and learns when
// the instance has ended the session.
func (g *Gateway) modify(response *http.Response) error {
	response.Header.Del("Set-Cookie")
	if response.StatusCode == http.StatusUnauthorized {
		g.invalidate(upstreamOf(response.Request))
	}
	return nil
}

// fail answers a request whose instance could not be reached: the page with a page that says so, and the
// routes of the API in the form that the web interface shows the message of.
func (g *Gateway) fail(writer http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return // The browser left, and there is no one to tell.
	}
	up := upstreamOf(request)
	message := err.Error()
	address := ""
	if up != nil {
		message = up.unreachable(err).Error()
		address = up.base.Host
	}
	if request.URL.Path == "/app" {
		render(writer, http.StatusBadGateway, view{Page: pageFailed, Address: address, Error: sentence(message)})
		return
	}
	writeError(writer, http.StatusBadGateway, sentence(message))
}

// writeError answers in the form that the web interface reads the message of an error from.
func writeError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": message})
}

// sentence makes a message of the errors of the program begin with a capital, as the messages of the
// application do.
func sentence(message string) string {
	first, size := utf8.DecodeRuneInString(message)
	if first == utf8.RuneError || unicode.IsUpper(first) {
		return message
	}
	return string(unicode.ToUpper(first)) + message[size:]
}

// serveInstance forwards the page and the routes of the API to the instance for a browser that is paired
// with the gateway, and sends everyone else to the page of the gateway or answers that they are not paired.
func (g *Gateway) serveInstance(writer http.ResponseWriter, request *http.Request) {
	isPage := request.URL.Path == "/app"
	if !g.browsers.paired(request) {
		if isPage {
			http.Redirect(writer, request, "/", http.StatusFound)
			return
		}
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
		return
	}
	up := g.current()
	if !up.connected() {
		if isPage {
			http.Redirect(writer, request, "/", http.StatusFound)
			return
		}
		// 401, as for a browser that is not paired: the web interface goes to the page of the gateway.
		writeError(writer, http.StatusUnauthorized, "The gateway is not paired with a vfox+. Open its page and pair it.")
		return
	}
	if err := g.crossOrigin.Check(request); err != nil {
		http.Error(writer, "Cross-site requests are not allowed.", http.StatusForbidden)
		return
	}
	g.forward(writer, request, up)
}

// serveManifest forwards the manifest of the web interface, which the instance serves to anyone: a
// browser may fetch it without the cookie, and it names the application and its start page and nothing else.
func (g *Gateway) serveManifest(writer http.ResponseWriter, request *http.Request) {
	up := g.current()
	if up == nil {
		http.NotFound(writer, request)
		return
	}
	g.forward(writer, request, up)
}

// forward sends the request to the instance.
func (g *Gateway) forward(writer http.ResponseWriter, request *http.Request, up *upstream) {
	g.proxy.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), upstreamKey{}, up)))
}

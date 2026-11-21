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
	"bytes"
	_ "embed"
	"io"
	"net/http"
)

// webappHTML is the page of the web interface: the single file that the build of the web interface
// writes (mygo/web, npm run build). The instance serves its API; the page is served here, and the
// routes that the page calls are the instance's, forwarded.
//
//go:embed webapp.html
var webappHTML []byte

// The page around its two placeholders: the favicon link in the head, and the
// icon in the brand mark of the navigation rail. The page is written in
// pieces, so that serving it copies nothing.
var (
	webappHead, webappRest, _   = bytes.Cut(webappHTML, []byte(faviconPlaceholder))
	webappMiddle, webappTail, _ = bytes.Cut(webappRest, []byte(logoPlaceholder))
)

// servePage answers the page of the interface to a browser that is paired with the gateway, while the
// gateway holds a session at the instance. Any other browser is sent to the page of the gateway. The
// page is answered without a request to the instance, which the page then makes by the routes of /api.
func (g *Gateway) servePage(writer http.ResponseWriter, request *http.Request) {
	if !g.browsers.paired(request) || !g.current().connected() {
		http.Redirect(writer, request, "/", http.StatusFound)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = writer.Write(webappHead)
	_, _ = io.WriteString(writer, faviconLink)
	_, _ = writer.Write(webappMiddle)
	_, _ = io.WriteString(writer, faviconURI)
	_, _ = writer.Write(webappTail)
}

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
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

// How the gateway is paired with an instance, as the Android app is: the person pastes a pairing link that
// the instance made for another device. The gateway looks at the certificate of an HTTPS instance first. A
// certificate that the roots of the system vouch for is trusted; one that nobody signed, which is what the
// application makes for itself, is shown to the person as a fingerprint to compare with the one in the
// settings of the application, and trusted only when the person says it is the same, and then it is the
// only one that is. The link is used once, to get the session.

// serveConnect takes the pairing link that was pasted: it pairs the gateway with the instance, or, for a
// certificate to be compared, asks first.
func (g *Gateway) serveConnect(writer http.ResponseWriter, request *http.Request) {
	if !g.authorize(writer, request) {
		return
	}
	link := request.PostFormValue("link")
	base, token, err := parseLink(link)
	if err != nil {
		g.renderHome(writer, http.StatusBadRequest, sentence(err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), requestTimeout)
	defer cancel()
	var pinned *x509.Certificate
	if base.Scheme == "https" {
		certificate, verified, err := certificateOf(ctx, base, g.roots)
		if err != nil {
			g.renderHome(writer, http.StatusBadGateway, sentence(newUpstream(base, nil, g.roots).unreachable(err).Error()))
			return
		}
		if !verified {
			fingerprint := pair.Fingerprint(certificate.Raw)
			// The one that the person compared already, for this address, is not asked about again: it is
			// what a gateway that is paired again after a day does. A certificate that changed is asked about
			// with a warning.
			trusted := g.trusted(base)
			if trusted != fingerprint {
				render(writer, http.StatusOK, view{Page: pageTrust, Address: base.Host, Fingerprint: fingerprint, Link: link, Changed: trusted != ""})
				return
			}
			pinned = certificate
		}
	}
	g.pairAndRemember(ctx, writer, request, base, token, pinned)
}

// serveTrust pairs the gateway with the instance whose certificate the person compared. The certificate is
// looked at again, and it has to be the one that was shown: what the person compared is what is trusted, and
// not what is at the address a moment later.
func (g *Gateway) serveTrust(writer http.ResponseWriter, request *http.Request) {
	if !g.authorize(writer, request) {
		return
	}
	base, token, err := parseLink(request.PostFormValue("link"))
	if err != nil || base.Scheme != "https" {
		g.renderHome(writer, http.StatusBadRequest, "That is not a pairing link of vfox+ over https.")
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), requestTimeout)
	defer cancel()
	shown := request.PostFormValue("fingerprint")
	certificate, _, err := certificateOf(ctx, base, g.roots)
	if err != nil {
		g.renderHome(writer, http.StatusBadGateway, sentence(newUpstream(base, nil, g.roots).unreachable(err).Error()))
		return
	}
	if pair.Fingerprint(certificate.Raw) != shown {
		g.renderHome(writer, http.StatusConflict, "The certificate at "+base.Host+" is not the one that was shown for the comparison. Nothing was trusted; try again.")
		return
	}
	// The certificate is verified for a name that it has, and a certificate that names no host or address has none: it
	// could not be told from another that is served at the address.
	if nameOf(certificate) == "" {
		g.renderHome(writer, http.StatusBadRequest, "The certificate at "+base.Host+" names no host and no address, so the gateway could not tell it from another: it cannot be trusted.")
		return
	}
	g.pairAndRemember(ctx, writer, request, base, token, certificate)
}

// pairAndRemember uses the token of the link, with the instance whose certificate is the one that was pinned or one
// that the roots of the system vouch for, and takes the browser to its interface.
func (g *Gateway) pairAndRemember(ctx context.Context, writer http.ResponseWriter, request *http.Request, base *url.URL, token string, pinned *x509.Certificate) {
	up := newUpstream(base, pinned, g.roots)
	session, err := up.pairWith(ctx, token)
	if err != nil {
		up.transport.CloseIdleConnections()
		status := http.StatusBadGateway
		if errors.Is(err, errRefused) {
			status = http.StatusForbidden
		}
		g.renderHome(writer, status, sentence(err.Error()))
		return
	}
	g.replace(up.withSession(session, time.Now()))
	http.Redirect(writer, request, "/app", http.StatusSeeOther)
}

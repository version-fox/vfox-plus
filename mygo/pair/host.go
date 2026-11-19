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
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// The reasons that a link cannot be made.
var (
	// ErrNoRemote means that no server listens for other devices.
	ErrNoRemote = errors.New("other devices cannot connect")
	// ErrNoAddress means that this computer has no address on a network, or not
	// the one that was asked for.
	ErrNoAddress = errors.New("this computer has no such address on a network")
)

// Status describes the server for other devices, as the settings made it.
type Status struct {
	// Remote reports whether other devices can connect now.
	Remote bool
	// Port is the port that other devices connect to, whether they can now or not.
	Port int
	// Addresses are the addresses of this computer on the networks it is on, the
	// likeliest first: what other devices may connect to. They are listed while
	// Remote is true.
	Addresses []string
	// Fingerprint is the SHA-256 fingerprint of the certificate that other
	// devices get, in the form that browsers show it. It is set while Remote is true.
	Fingerprint string
	// SelfSigned reports whether the application made that certificate itself.
	SelfSigned bool
	// RemoteError says why other devices cannot connect, when they are allowed to
	// and cannot: the port is in use, the certificate does not load.
	RemoteError string
	// TunnelURL is the public address of the tunnel while it runs, and empty
	// otherwise: pairing links then name it instead of this computer.
	TunnelURL string
}

// HostOptions are what a Host needs besides the application state.
type HostOptions struct {
	// SettingsDir is the directory of the settings, where the certificate that
	// the application makes for itself is kept.
	SettingsDir string
	// Quit ends the application, when a paired device asks for it.
	Quit func()
	// Addresses lists the addresses of this computer on the networks it is on, as its interfaces have them when it is nil.
	// A test, or a build whose screens are shown, names them, so that what says them does not depend on the machine that
	// runs it: a computer with no network has none to pair a phone with, and one with several has a long list of them.
	Addresses func() []string
}

// webConfig is what the server for other devices is started for: the part of the
// settings that it depends on.
type webConfig struct {
	remote           bool
	port             int
	certificate, key string
}

func webConfigOf(current settings.AppSettings) webConfig {
	return webConfig{
		remote:      current.WebRemote,
		port:        current.EffectiveWebPort(),
		certificate: current.WebCertificate,
		key:         current.WebKey,
	}
}

// Host runs the server for other devices as its settings say, and follows them:
// with other devices allowed, a server on the network serves them over HTTPS, and
// turning that off stops it at once, together with the connections it holds. The
// application shows no web interface on this computer: the Docker gateway, and a
// phone that pairs with the application, are the clients of that server.
//
// A Host is also a Listenable: it notifies its listeners when its status may have
// changed.
type Host struct {
	state.Notifier
	model   *state.AppModel
	options HostOptions
	// loopback is the server of this computer that only the end-to-end tests have,
	// see loopback_e2e.go: the application has none in the builds that are released.
	loopback loopbackState

	// applyMu serializes Apply.
	applyMu sync.Mutex
	// mu guards the fields below.
	mu      sync.Mutex
	remote  *Server
	applied webConfig
	status  Status
	stop    state.Unsubscribe
	closed  bool
	// tunnel publishes the server for other devices through Cloudflare, while
	// the user keeps the switch on.
	tunnel *Tunnel
}

// NewHost creates the host of the server for other devices. Nothing is served
// until Start.
func NewHost(model *state.AppModel, options HostOptions) *Host {
	return &Host{model: model, options: options}
}

// Start serves other devices as the settings say, and follows the settings from
// now on. The settings must be loaded.
func (h *Host) Start() {
	h.Apply()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.stop != nil {
		return
	}
	h.stop = h.model.Settings.AddListener(func() { go h.Apply() })
}

// Apply makes the server match the settings now: it starts it when other devices
// may connect and stops it when they may not. It does nothing when they already
// match.
func (h *Host) Apply() {
	h.applyMu.Lock()
	defer h.applyMu.Unlock()
	config := webConfigOf(h.model.Settings.Settings())
	h.mu.Lock()
	if h.closed || config == h.applied {
		h.mu.Unlock()
		return
	}
	previous, remote := h.applied, h.remote
	h.mu.Unlock()

	status := Status{Port: config.port}
	// Other devices connect again when anything about the connection changed, and
	// the tunnel publishes that server: it ends with the server, since a tunnel
	// reaches one port.
	changed := previous.remote != config.remote || previous.port != config.port ||
		previous.certificate != config.certificate || previous.key != config.key
	endTunnel := !config.remote || changed
	if remote != nil && (changed || !config.remote) {
		closeServer(remote)
		remote = nil
	}
	if config.remote && remote == nil {
		var err error
		if remote, status.Fingerprint, status.SelfSigned, err = h.startRemote(config); err != nil {
			status.RemoteError = err.Error()
		}
	}
	if endTunnel {
		h.mu.Lock()
		tunnel := h.tunnel
		h.tunnel = nil
		h.mu.Unlock()
		tunnel.Stop()
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		closeServer(remote)
		return
	}
	h.remote, h.applied = remote, config
	if remote != nil {
		status.Remote = true
	} else {
		status.Fingerprint, status.SelfSigned = "", false
	}
	if h.tunnel != nil {
		status.TunnelURL = h.tunnel.URL()
	}
	h.status = status
	h.mu.Unlock()
	h.NotifyListeners()
}

// startRemote starts the server for other devices, with the certificate of the
// user or one that the application makes for itself.
func (h *Host) startRemote(config webConfig) (server *Server, fingerprint string, selfSigned bool, err error) {
	certFile, keyFile := config.certificate, config.key
	switch {
	case certFile != "" && keyFile != "":
		if fingerprint, err = CertFingerprint(certFile); err != nil {
			return nil, "", false, err
		}
	case certFile != "" || keyFile != "":
		return nil, "", false, errors.New("a certificate and its key go together")
	default:
		if certFile, keyFile, fingerprint, err = EnsureServerCert(h.options.SettingsDir, h.localAddresses()); err != nil {
			return nil, "", false, err
		}
		selfSigned = true
	}
	server, err = NewServer(Options{
		Bind:             "0.0.0.0",
		AllowNonLoopback: true,
		Port:             config.port,
		SessionTTL:       DefaultSessionTTL,
		Model:            h.model,
		TLSCertFile:      certFile,
		TLSKeyFile:       keyFile,
		Fingerprint:      fingerprint,
		Control:          h,
	})
	if err != nil {
		return nil, "", false, err
	}
	return server, fingerprint, selfSigned, nil
}

func closeServer(server *Server) {
	if server != nil {
		_ = server.Close()
	}
}

// Close stops the server, together with the connections it holds, and stops
// following the settings.
func (h *Host) Close() {
	h.mu.Lock()
	h.closed = true
	remote, stop, tunnel := h.remote, h.stop, h.tunnel
	h.remote, h.stop, h.tunnel = nil, nil, nil
	h.status = Status{}
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
	tunnel.Stop()
	closeServer(remote)
	h.loopback.close()
}

// Publish starts the public tunnel of the server for other devices, and returns
// the address where it serves it. A tunnel that already runs is kept.
func (h *Host) Publish() (string, error) {
	h.mu.Lock()
	if h.tunnel != nil {
		address := h.tunnel.URL()
		h.mu.Unlock()
		return address, nil
	}
	remote, port := h.remote, h.status.Port
	h.mu.Unlock()
	if remote == nil {
		return "", ErrNoRemote
	}
	tunnel, err := PublishTunnel(context.Background(), port)
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	// The server may have been replaced while the tunnel started.
	if h.closed || h.remote != remote {
		h.mu.Unlock()
		tunnel.Stop()
		return "", ErrNoRemote
	}
	h.tunnel = tunnel
	h.status.TunnelURL = tunnel.URL()
	h.mu.Unlock()
	h.NotifyListeners()
	return tunnel.URL(), nil
}

// Unpublish ends the public tunnel, and its address with it.
func (h *Host) Unpublish() {
	h.mu.Lock()
	tunnel := h.tunnel
	h.tunnel = nil
	h.status.TunnelURL = ""
	h.mu.Unlock()
	tunnel.Stop()
	h.NotifyListeners()
}

// Status implements Control: it describes the server now.
func (h *Host) Status() Status {
	h.mu.Lock()
	status := h.status
	h.mu.Unlock()
	if status.Remote {
		status.Addresses = h.networkAddresses()
	}
	return status
}

// RemoteLink implements Control: it makes a pairing link for a device on the
// network, with host as the address of this computer in it, or the likeliest one
// when host is empty.
func (h *Host) RemoteLink(host string) (string, error) {
	h.mu.Lock()
	remote := h.remote
	h.mu.Unlock()
	if remote == nil {
		return "", ErrNoRemote
	}
	h.mu.Lock()
	tunnelURL := h.status.TunnelURL
	h.mu.Unlock()
	addresses := h.networkAddresses()
	host = strings.TrimSpace(host)
	public := h.model.Settings.Settings().EffectiveWebAddress()
	if tunnelURL != "" {
		public = tunnelURL
	}
	switch {
	case host == "":
		// The address that a phone reaches first: the public one when the user
		// published the server, and else the likeliest address at home.
		if public != "" {
			host = public
		} else if len(addresses) > 0 {
			host = addresses[0]
		}
	case host == public:
		// The configured public address is one of the two answers, with the
		// addresses of this computer.
	case slices.Contains(addresses, host):
	default:
		return "", ErrNoAddress
	}
	if host == "" {
		return "", ErrNoAddress
	}
	token, _, err := remote.NewToken()
	if err != nil {
		return "", err
	}
	return remote.PairingURL(host, token), nil
}

// Quit implements Control: it ends the application.
func (h *Host) Quit() {
	if h.options.Quit != nil {
		h.options.Quit()
	}
}

// NetworkAddresses lists the addresses of this computer on the networks it is
// on, which other devices can connect to: the IPv4 addresses first, then the IPv6
// ones, each without brackets. Loopback and link-local addresses are not among
// them.
func NetworkAddresses() []string { return networkAddresses(LocalAddresses()) }

// localAddresses are the addresses of this computer that the options name, or those of its interfaces.
func (h *Host) localAddresses() []string {
	if h.options.Addresses != nil {
		return h.options.Addresses()
	}
	return LocalAddresses()
}

// networkAddresses are the ones of them that other devices can connect to.
func (h *Host) networkAddresses() []string { return networkAddresses(h.localAddresses()) }

// networkAddresses keeps of addresses what NetworkAddresses lists: the IPv4 addresses first, then the IPv6 ones, each once.
func networkAddresses(addresses []string) []string {
	var v4, v6 []string
	for _, address := range addresses {
		ip := net.ParseIP(address)
		switch {
		case ip == nil || slices.Contains(v4, address) || slices.Contains(v6, address):
		case ip.To4() != nil:
			v4 = append(v4, address)
		default:
			v6 = append(v6, address)
		}
	}
	return append(v4, v6...)
}

// String describes the status for logs and test failures.
func (s Status) String() string {
	return fmt.Sprintf("remote=%t port=%d error=%q", s.Remote, s.Port, s.RemoteError)
}

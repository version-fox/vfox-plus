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
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// The reasons that a link cannot be made.
var (
	// ErrNotServing means that the web interface is off.
	ErrNotServing = errors.New("the web interface is off")
	// ErrNoRemote means that no server listens for other devices.
	ErrNoRemote = errors.New("other devices cannot connect")
	// ErrNoAddress means that this computer has no address on a network, or not
	// the one that was asked for.
	ErrNoAddress = errors.New("this computer has no such address on a network")
)

// Status describes the servers of the web interface, as the settings made them.
type Status struct {
	// Running reports whether the web interface is served on this computer.
	Running bool
	// LocalURL is the address of that server, like http://127.0.0.1:41233.
	LocalURL string
	// Error says why the web interface cannot be served, when it is on and is not.
	Error string
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
}

// HostOptions are what a Host needs besides the application state.
type HostOptions struct {
	// SettingsDir is the directory of the settings, where the certificate that
	// the application makes for itself is kept.
	SettingsDir string
	// Quit ends the application, when a paired device asks for it.
	Quit func()
}

// webConfig is what the servers are started for: the part of the settings that
// they depend on.
type webConfig struct {
	on, remote       bool
	port             int
	certificate, key string
}

func webConfigOf(current settings.AppSettings) webConfig {
	return webConfig{
		on:          current.WebInterface,
		remote:      current.WebInterface && current.WebRemote,
		port:        current.EffectiveWebPort(),
		certificate: current.WebCertificate,
		key:         current.WebKey,
	}
}

// Host runs the web interface of the application as its settings say, and
// follows them: with the web interface on, a server on the loopback interface
// serves the browsers of this computer, and with other devices allowed, a
// second one serves the network over HTTPS. Turning either off stops it at
// once, together with the connections it holds. The servers share the
// application state; each has its own pairing links and sessions.
//
// A Host is also a Listenable: it notifies its listeners when its status may
// have changed.
type Host struct {
	state.Notifier
	model   *state.AppModel
	options HostOptions

	// applyMu serializes Apply.
	applyMu sync.Mutex
	// mu guards the fields below.
	mu      sync.Mutex
	local   *Server
	remote  *Server
	applied webConfig
	status  Status
	stop    state.Unsubscribe
	closed  bool
}

// NewHost creates the host of the web interface. Nothing is served until Start.
func NewHost(model *state.AppModel, options HostOptions) *Host {
	return &Host{model: model, options: options}
}

// Start serves the web interface as the settings say, and follows the settings
// from now on. The settings must be loaded.
func (h *Host) Start() {
	h.Apply()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.stop != nil {
		return
	}
	h.stop = h.model.Settings.AddListener(func() { go h.Apply() })
}

// Apply makes the servers match the settings now: it starts what they ask for
// and stops what they no longer do. It does nothing when they already match.
func (h *Host) Apply() {
	h.applyMu.Lock()
	defer h.applyMu.Unlock()
	config := webConfigOf(h.model.Settings.Settings())
	h.mu.Lock()
	if h.closed || config == h.applied {
		h.mu.Unlock()
		return
	}
	previous, local, remote := h.applied, h.local, h.remote
	h.mu.Unlock()

	status := Status{Port: config.port}
	if !config.on {
		closeServer(local)
		closeServer(remote)
		local, remote = nil, nil
	} else {
		if local == nil {
			var err error
			if local, err = NewServer(Options{Model: h.model, Control: h}); err != nil {
				status.Error = err.Error()
			}
		}
		// Other devices connect again when anything about the connection changed.
		changed := previous.remote != config.remote || previous.port != config.port ||
			previous.certificate != config.certificate || previous.key != config.key
		if (changed || !config.remote) && remote != nil {
			closeServer(remote)
			remote = nil
		}
		if config.remote && remote == nil && local != nil {
			var err error
			if remote, status.Fingerprint, status.SelfSigned, err = h.startRemote(config); err != nil {
				status.RemoteError = err.Error()
			}
		}
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		closeServer(local)
		closeServer(remote)
		return
	}
	h.local, h.remote, h.applied = local, remote, config
	if local != nil {
		status.Running, status.LocalURL = true, local.BaseURL()
	}
	if remote != nil {
		status.Remote = true
	} else {
		status.Fingerprint, status.SelfSigned = "", false
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
		if certFile, keyFile, fingerprint, err = EnsureServerCert(h.options.SettingsDir, LocalAddresses()); err != nil {
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

// Close stops the servers, together with the connections they hold, and stops
// following the settings.
func (h *Host) Close() {
	h.mu.Lock()
	h.closed = true
	local, remote, stop := h.local, h.remote, h.stop
	h.local, h.remote, h.stop = nil, nil, nil
	h.status = Status{}
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
	closeServer(local)
	closeServer(remote)
}

// Status implements Control: it describes the servers now.
func (h *Host) Status() Status {
	h.mu.Lock()
	status := h.status
	h.mu.Unlock()
	if status.Remote {
		status.Addresses = NetworkAddresses()
	}
	return status
}

// LocalLink makes a pairing link for a browser on this computer.
func (h *Host) LocalLink() (string, error) {
	h.mu.Lock()
	local := h.local
	h.mu.Unlock()
	if local == nil {
		return "", ErrNotServing
	}
	_, link, err := local.NewToken()
	return link, err
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
	addresses := NetworkAddresses()
	if host == "" && len(addresses) > 0 {
		host = addresses[0]
	}
	if host == "" || !slices.Contains(addresses, host) {
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
func NetworkAddresses() []string {
	var v4, v6 []string
	for _, address := range LocalAddresses() {
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
	return fmt.Sprintf("running=%t local=%q remote=%t port=%d error=%q remoteError=%q", s.Running, s.LocalURL, s.Remote, s.Port, s.Error, s.RemoteError)
}

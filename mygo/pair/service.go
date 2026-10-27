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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/state"
)

// ServiceFileName is the discovery file of the shared background server, in
// the settings directory. At most one server owns the application state per
// user: the desktop application, a background daemon or a loopback `serve`
// registers itself here, and `pair` mints its links from whichever is live
// instead of starting another model.
const ServiceFileName = "server.json"

// Service owners recorded in the discovery file.
const (
	ServiceOwnerDesktop = "desktop"
	ServiceOwnerService = "service"
)

// ServiceInfo describes the live shared server.
type ServiceInfo struct {
	// PID is the process that owns the server.
	PID int `json:"pid"`
	// Port is the loopback port the server listens on.
	Port int `json:"port"`
	// AdminToken mints pairing links over loopback; it never leaves the
	// machine (the file holds mode 0600) and never becomes a cookie.
	AdminToken string `json:"adminToken"`
	// Owner is ServiceOwnerDesktop or ServiceOwnerService.
	Owner string `json:"owner"`
	// StartedAt is when the server was registered, in RFC 3339.
	StartedAt string `json:"startedAt"`
}

// ServiceFilePath returns the discovery file in the settings directory.
func ServiceFilePath(settingsDir string) string {
	return filepath.Join(settingsDir, ServiceFileName)
}

// WriteServiceInfo records info with mode 0600: the admin token inside must
// not be readable by other users.
func WriteServiceInfo(path string, info *ServiceInfo) error {
	content, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}

// ReadServiceInfo reads the discovery file.
func ReadServiceInfo(path string) (*ServiceInfo, error) {
	content, err := os.ReadFile(path) //nolint:gosec // The path is built from the settings directory.
	if err != nil {
		return nil, err
	}
	var info ServiceInfo
	if err := json.Unmarshal(content, &info); err != nil {
		return nil, err
	}
	if info.Port <= 0 || info.AdminToken == "" {
		return nil, fmt.Errorf("pair: %s is not a service file", path)
	}
	return &info, nil
}

// ProbeService reports whether info answers on loopback.
func ProbeService(info *ServiceInfo) bool {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", info.Port))
	if err != nil {
		return false
	}
	defer func() { _ = response.Body.Close() }()
	return response.StatusCode == http.StatusOK
}

// DiscoverService returns the live shared server, if any. A stale file is
// removed, so a crashed server does not block the next one.
func DiscoverService(settingsDir string) *ServiceInfo {
	path := ServiceFilePath(settingsDir)
	info, err := ReadServiceInfo(path)
	if err != nil {
		return nil
	}
	if !ProbeService(info) {
		_ = os.Remove(path)
		return nil
	}
	return info
}

// NewAdminToken creates a bearer token for the service file.
func NewAdminToken() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes[:])
}

// MintToken asks a live service for a one-time pairing link. The admin bearer
// only travels over loopback; the file holding it has mode 0600.
func MintToken(info *ServiceInfo) (string, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/token", info.Port), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+info.AdminToken)
	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pair: the shared server answered %d", response.StatusCode)
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.URL == "" {
		return "", fmt.Errorf("pair: the shared server sent no link")
	}
	return body.URL, nil
}

// MintURL renders a pairing URL for base with token, for links the server did
// not mint itself (kept next to MintToken for callers).
func MintURL(base, token string) string {
	return strings.TrimSuffix(base, "/") + "/auth/connect/" + token
}

// ServeEmbedded starts the loopback server for the desktop application: the
// desktop owns the single shared server while it runs, so that `pair` and
// browsers attach to it instead of starting another model. The caller enables it from the remoteAccess setting, which is off by
// default: with no live server around, ServeEmbedded itself always serves.
// When another live server owns the file — a daemon or an earlier desktop
// that did not clean up — or the service is disabled, it does nothing and
// returns a no-op cleanup. Call cleanup when the desktop exits.
func ServeEmbedded(model *state.AppModel, settingsDir string) (func(), error) {
	noop := func() {}
	if ServiceDisabled(settingsDir) || DiscoverService(settingsDir) != nil {
		return noop, nil
	}
	token := NewAdminToken()
	server, err := NewServer(Options{Model: model, AdminToken: token})
	if err != nil {
		return noop, err
	}
	info := &ServiceInfo{
		PID:        os.Getpid(),
		Port:       serverPort(server),
		AdminToken: token,
		Owner:      ServiceOwnerDesktop,
		StartedAt:  time.Now().Format(time.RFC3339),
	}
	path := ServiceFilePath(settingsDir)
	if err := WriteServiceInfo(path, info); err != nil {
		_ = server.Close()
		return noop, err
	}
	return func() {
		_ = server.Close()
		_ = os.Remove(path)
	}, nil
}

// serverPort reports the listening port of server, or 0.
func serverPort(server *Server) int {
	_, port, err := net.SplitHostPort(server.Addr().String())
	if err != nil {
		return 0
	}
	var number int
	_, _ = fmt.Sscanf(port, "%d", &number)
	return number
}

// ServiceDisabled reports whether the user disabled the shared background
// service. Pairing requires the shared service and is unavailable while it is
// disabled; the desktop and `serve` keep working with their own models.
func ServiceDisabled(settingsDir string) bool {
	content, err := os.ReadFile(serviceConfigPath(settingsDir)) //nolint:gosec // Built from the settings directory.
	if err != nil {
		return false
	}
	var config struct {
		Disabled bool `json:"disabled"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return false
	}
	return config.Disabled
}

// SetServiceDisabled records whether the shared background service is
// disabled.
func SetServiceDisabled(settingsDir string, disabled bool) error {
	content, err := json.MarshalIndent(struct {
		Disabled bool `json:"disabled"`
	}{Disabled: disabled}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(settingsDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(serviceConfigPath(settingsDir), content, 0o600)
}

func serviceConfigPath(settingsDir string) string {
	return filepath.Join(settingsDir, "service.json")
}

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

package screens

import (
	"os"
	"path/filepath"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// configDirName is the directory of the app's files, under the user's configuration
// directory. On Android that directory is the app's private files directory: mygo
// points XDG_CONFIG_HOME at it before Go starts (see HostImpl's startGo), so
// os.UserConfigDir needs no Android code here.
const configDirName = "vfox-plus"

// The files of the saved computer. The record is not secret; the session is a
// credential, so it is kept in a file of its own, which the store makes private.
const (
	serverFile  = "server.json"
	sessionFile = "session.json"
)

// newService returns the service over the files of the app's configuration
// directory.
func newService() (*app.Service, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, configDirName)
	return app.NewService(app.Options{
		Servers:  store.NewFileServerStore(filepath.Join(dir, serverFile)),
		Sessions: store.NewFileSessionStore(filepath.Join(dir, sessionFile)),
	})
}

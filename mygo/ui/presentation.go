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

package ui

import (
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

// taskTitle names a task: what vfox is doing, in the language of the user
// interface.
func (a *App) taskTitle(command vfox.VfoxCommand) string {
	subject := command.Subject()
	switch command.Operation {
	case vfox.OperationInstall:
		return a.T("taskInstall", "subject", subject)
	case vfox.OperationUninstall:
		return a.T("taskUninstall", "subject", subject)
	case vfox.OperationUse:
		return a.T("taskUse", "subject", subject)
	case vfox.OperationUnuse:
		return a.T("taskUnuse", "subject", subject)
	case vfox.OperationAddPlugin:
		if subject == "" {
			return a.T("taskAddPluginFromSource")
		}
		return a.T("taskAddPlugin", "subject", subject)
	case vfox.OperationRemovePlugin:
		return a.T("taskRemovePlugin", "subject", subject)
	case vfox.OperationUpdatePlugin:
		return a.T("taskUpdatePlugin", "subject", subject)
	default:
		return a.T("taskUpdateAllPlugins")
	}
}

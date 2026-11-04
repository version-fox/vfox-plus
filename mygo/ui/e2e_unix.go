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

//go:build e2e && !windows

package ui

import (
	"os"
	"os/signal"
	"syscall"
)

// closeOnSignal closes the window when the process gets SIGUSR1, which is how the tools
// of the end-to-end tests do what the user does with the close button.
func (a *App) closeOnSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	go func() {
		for range signals {
			a.closeWindow()
		}
	}()
}

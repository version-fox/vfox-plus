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
	"errors"
	"strconv"

	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// errorView explains an error in the middle of the space, with a button that
// tries again.
func (a *App) errorView(k kit.Ctx, title string, err error, retry func()) {
	var actions []func()
	if retry != nil {
		actions = append(actions, func() {
			kit.Button(k, kit.ButtonSpec{
				ID: "retry", Kind: kit.TonalButton, Icon: "retry", Label: a.T("actionRetry"), OnClick: retry,
			})
		})
	}
	kit.StatusMessage(k, kit.StatusSpec{
		Icon: "error", Shape: shapes.ShapeSoftBurst, IsError: true, Title: title, Body: describeError(err), Actions: actions,
	})
}

// loadingView shows that something is loading, in the middle of the space.
func (a *App) loadingView(k kit.Ctx, label string) {
	accessible := label
	if accessible == "" {
		accessible = a.T("loading")
	}
	kit.LoadingMessage(k, label, accessible)
}

// loadingOrError shows the error when there is one, and else that something is
// loading.
func (a *App) loadingOrError(k kit.Ctx, title string, err error, retry func()) {
	if err != nil {
		a.errorView(k, title, err, retry)
		return
	}
	a.loadingView(k, "")
}

// refreshButton is the icon button that loads a resource again.
func (a *App) refreshButton(k kit.Ctx, id string, loading bool, refresh func()) {
	kit.IconButton(k, kit.IconButtonSpec{
		ID: id, Icon: "refresh", Tooltip: a.T("actionRefresh"), Disabled: loading, OnClick: refresh,
	})
}

// avatar is the avatar of a plugin in a list.
func avatar(k kit.Ctx, name string, size float32) {
	kit.Avatar(k, kit.AvatarSpec{Name: name, Size: size})
}

// run queues a command, or tells the user why it cannot be queued.
func (a *App) run(command vfox.VfoxCommand, err error) {
	if err != nil {
		a.notify(describeError(err))
		return
	}
	a.model.Run(command)
}

// describeError is the message of an error that the user can read.
func describeError(err error) string {
	var failure *vfox.VfoxError
	if errors.As(err, &failure) {
		return failure.Message
	}
	return err.Error()
}

// itoa formats a number.
func itoa(value int) string { return strconv.Itoa(value) }

// orQuestionMark is the text of a value that may be unknown.
func orQuestionMark(value func() (string, bool)) string {
	if text, ok := value(); ok {
		return text
	}
	return "?"
}

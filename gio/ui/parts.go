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
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// errorView explains an error in the middle of the space, with a button that
// tries again.
func (a *App) errorView(title string, err error, retry func()) kit.Widget {
	var actions []kit.Widget
	if retry != nil {
		actions = append(actions, kit.Button{
			ID: "retry", Kind: kit.TonalButton, Icon: "retry", Label: a.T("actionRetry"), OnClick: retry,
		}.W())
	}
	return kit.StatusMessage{
		Icon: "error", Shape: shapes.ShapeSoftBurst, IsError: true, Title: title, Body: describeError(err), Actions: actions,
	}.W()
}

// loadingView shows that something is loading, in the middle of the space.
func (a *App) loadingView(label string) kit.Widget {
	accessible := label
	if accessible == "" {
		accessible = a.T("loading")
	}
	return kit.LoadingMessage(label, accessible)
}

// loadingOrError shows the error when there is one, and else that something
// is loading.
func (a *App) loadingOrError(title string, err error, retry func()) kit.Widget {
	if err != nil {
		return a.errorView(title, err, retry)
	}
	return a.loadingView("")
}

// refreshButton is the icon button that loads a resource again.
func (a *App) refreshButton(id string, loading bool, refresh func()) kit.Widget {
	return kit.IconButton{ID: id, Icon: "refresh", Tooltip: a.T("actionRefresh"), Disabled: loading, OnClick: refresh}.W()
}

// avatar is the avatar of a plugin in a list.
func avatar(name string, size float32) kit.Widget {
	return kit.Avatar{Name: name, Size: size}.W("avatar:" + name)
}

// run queues a command, or tells the user why it cannot be queued.
func (a *App) run(command vfox.VfoxCommand, err error) {
	if err != nil {
		a.notify(describeError(err))
		return
	}
	a.model.Run(command)
}

// column lays out widgets one under the other, as wide as the space.
func column(gap float32, children ...kit.Widget) kit.Widget {
	flex := make([]kit.FlexChild, 0, len(children))
	for _, child := range children {
		if child != nil {
			flex = append(flex, kit.Rigid(child))
		}
	}
	return kit.Column(kit.FlexOpts{Gap: gap, Cross: kit.CrossStretch, Min: true}, flex...)
}

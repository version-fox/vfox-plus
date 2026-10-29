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

package main

import (
	"github.com/version-fox/vfox-plus/shared/tooling/internal/build"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/builddeb"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/capture"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/coverage"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/docscheck"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/e2erun"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/fakevfox"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/fetch"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/golicenses"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/iconfont"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/installpkgs"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/measureapk"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/measureserve"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/memoryreport"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/prepare"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/renamefont"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/runtask"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/sandbox"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/screenshots"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/signmsix"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/smoketest"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/validatemsix"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/webheap"
)

func init() {
	commands["golicenses"] = golicenses.Run
	commands["iconfont"] = iconfont.Run
	commands["renamefont"] = renamefont.Run
	commands["build"] = build.Run
	commands["prepare"] = prepare.Run
	commands["coverage"] = coverage.Run
	commands["measure"] = measure.Run
	commands["measure-serve"] = measureserve.Run
	commands["measure-apk"] = measureapk.Run
	commands["webheap"] = webheap.Run
	commands["screenshots"] = screenshots.Run
	commands["smoke-test"] = smoketest.Run
	commands["e2e-run"] = e2erun.Run
	commands["capture"] = capture.Run
	commands["fake-vfox"] = fakevfox.Run
	commands["build-deb"] = builddeb.Run
	commands["build-msix"] = buildmsix.Run
	commands["sign-msix"] = signmsix.Run
	commands["validate-msix"] = validatemsix.Run
	commands["docs-check"] = docscheck.Run
	commands["sandbox"] = sandbox.Run
	commands["runtask"] = runtask.Run
	commands["fetch"] = fetch.Run
	commands["installpkgs"] = installpkgs.Run
	commands["memory-report"] = memoryreport.Run
}

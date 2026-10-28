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

// Command vfxtool replaces the Python and shell tools of the repository
// (shared/tools/*.py, shared/tools/*.sh, mygo/tools/*, shared/e2e/run.sh,
// shared/fixtures/*, shared/packaging/linux/build-deb.sh,
// .github/scripts/*.py and .github/actions/install-linux-packages/*.sh)
// with a single Go program. Run `vfxtool <command> --help` for each one.
//
// When executed under the file name "vfox", it answers vfox commands like
// the fake vfox instead: the end-to-end tools install a copy of this binary
// under that name as the stand-in for vfox.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/fakevfox"
)

func main() {
	if filepath.Base(os.Args[0]) == "vfox" {
		if err := fakevfox.Run(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "vfox: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	run, ok := commands[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "vfxtool: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "vfxtool %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: vfxtool <command> [options]

Commands:
  golicenses    collect Go module licenses (replaces shared/tools/go_licenses.py)
  iconfont      download/subset the icon font (replaces shared/tools/icon_font.py)
  renamefont    rename a font family (replaces mygo/tools/rename_font.py)
  build         build the mygo application (replaces mygo/tools/build.sh)
  prepare       generate embedded assets (replaces mygo/tools/prepare.sh)
  coverage      gate unit-test coverage (replaces mygo/tools/coverage.sh)
  measure       measure memory use (replaces shared/tools/measure.sh)
  measure-serve measure the memory use of the headless server
  measure-apk   measure the memory use of the Android app on an emulator
  screenshots   capture page screenshots (replaces shared/tools/screenshots.sh)
  smoke-test    start a graphical application and check that it stays up
  e2e-run       run an end-to-end scenario (replaces shared/e2e/run.sh)
  capture       capture vfox CLI fixtures (replaces shared/fixtures/capture.sh)
  fake-vfox     replay captured vfox output (replaces shared/fixtures/fake-vfox.sh)
  build-deb     package a .deb and .tar.gz (replaces shared/packaging/linux/build-deb.sh)
  build-msix    package unsigned .msix and .msixbundle for Windows 11 (see shared/packaging/windows)
  validate-msix validate .msix and .msixbundle files without the Windows SDK
  docs-check    verify every Markdown document has a Chinese counterpart
  sandbox       host side of the Docker sandbox (replaces mygo/sandbox/sandbox.sh + shared/sandbox/sandbox-lib.sh)
  runtask       one sandbox task in a container (replaces mygo/sandbox/run-task.sh + shared/sandbox/task-lib.sh)
  fetch         fetch one apt archive (replaces .github/actions/install-linux-packages/fetch.sh)
  installpkgs   install Ubuntu packages with a prefetch (replaces .../install-linux-packages/install.sh)
  memory-report comment the memory use on a pull request (replaces .github/scripts/memory_report.py)
`)
}

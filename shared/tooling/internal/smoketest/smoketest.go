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

// Package smoketest implements `vfxtool smoke-test`: starts a graphical
// application on a virtual display and checks that it still runs after a
// timeout, replacing the smoke-test step of .github/workflows/mygo.yml.
package smoketest

import (
	"errors"
	"flag"
	"fmt"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/display"
)

// Run implements `smoke-test [--screen GEOM] [--timeout DURATION] [--log FILE] <binary> [args...]`.
func Run(args []string) error {
	fs := flag.NewFlagSet("smoke-test", flag.ContinueOnError)
	screen := fs.String("screen", "", "the -screen argument of Xvfb (default "+display.DefaultScreen+")")
	timeout := fs.Duration("timeout", 0, "how long the application may take to stay up (default "+display.DefaultTimeout.String()+")")
	logPath := fs.String("log", "", "collect the output of the application here (default: discard)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: smoke-test [--screen GEOM] [--timeout DURATION] [--log FILE] <binary> [args...]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return errors.New("Usage: smoke-test [--screen GEOM] [--timeout DURATION] [--log FILE] <binary> [args...]")
	}
	return display.SmokeTest(rest[0], display.Options{
		Screen:  *screen,
		Timeout: *timeout,
		LogPath: *logPath,
		Args:    rest[1:],
	})
}

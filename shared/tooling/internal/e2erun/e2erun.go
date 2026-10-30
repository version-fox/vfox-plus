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

// Package e2erun runs the end-to-end scenarios of an application built with the
// tag e2e (vfxtool build --e2e) against a fresh HOME: it starts the application,
// drives it through the automation plugin of mygo as a person using assistive
// technology would (see package appsession), and checks what the window shows,
// what the fake or the real vfox says and what is on the disk.
package e2erun

import (
	"errors"
	"fmt"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
)

const usage = "Usage: e2e-run <fake|real|measure> <application command...>"

// IsValidScenario reports whether s names a known end-to-end scenario.
func IsValidScenario(s string) bool {
	switch s {
	case "fake", "real", "measure":
		return true
	default:
		return false
	}
}

// ValidateScenario returns an error unless s is a known scenario.
func ValidateScenario(s string) error {
	if !IsValidScenario(s) {
		return fmt.Errorf("Unknown scenario: %s", s)
	}
	return nil
}

// Run implements `e2e-run <fake|real|measure> <app command...>`. The measure
// scenario delegates to the measure package in-process.
func Run(args []string) error {
	if len(args) < 2 {
		return errors.New(usage)
	}
	scenario, command := args[0], args[1:]
	if err := ValidateScenario(scenario); err != nil {
		return err
	}
	switch scenario {
	case "measure":
		return measure.Run(command)
	case "fake":
		return runFake(command)
	default:
		return runReal(command)
	}
}

// runFake runs the application against the fake vfox, which replays the captured
// output of the fixtures.
func runFake(command []string) error {
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	return appsession.RunScenario(appsession.Config{Command: command, Fixtures: appsession.FixturesDir(root)}, "fake", fakeScenario)
}

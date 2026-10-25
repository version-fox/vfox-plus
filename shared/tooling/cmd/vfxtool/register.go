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

// Package registry wires vfxtool subcommands. Each subcommand lives in
// its own internal package and registers itself here.
package main

import (
	"flag"
	"fmt"
)

var commands = map[string]func([]string) error{}

func register(name string, run func(*flag.FlagSet, []string) error, synopsis string) {
	commands[name] = func(args []string) error {
		fs := flag.NewFlagSet("vfxtool "+name, flag.ContinueOnError)
		fs.Usage = func() { fmt.Fprintf(fs.Output(), "Usage: vfxtool %s %s\n", name, synopsis) }
		return run(fs, args)
	}
}

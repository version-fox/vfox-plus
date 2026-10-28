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

// Command vfox-plus-serve is the headless vfox-plus: pairing, serving and
// the background service without the desktop user interface. It behaves
// exactly like `vfox-plus pair|serve|service`, but it does not link the mygo
// user interface framework, so the binary — and the Docker image built from
// it — stays small. Anything interactive stays in the desktop application.
package main

import (
	"fmt"
	"os"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage())
		return 2
	}
	switch args[0] {
	case "pair":
		return pair.Main(args[1:])
	case "serve":
		return pair.ServeMain(args[1:])
	case "service":
		return pair.ServiceMain(args[1:])
	case "help", "--help", "-h":
		fmt.Println(usage())
		return 0
	default:
		fmt.Fprintln(os.Stderr, usage())
		return 2
	}
}

func usage() string {
	return `Usage: vfox-plus-serve [pair|serve|service]

  pair            Pair a browser with this machine over loopback.
  serve           Keep the pairing server up for containers and remote instances.
  service         Manage the shared background server (start, stop, status, set).`
}
